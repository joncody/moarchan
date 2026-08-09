package frame

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var validTableNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

const defaultMaxRows = 1000

var systemTables = map[string]bool{
	"auth": true,
}

// IsValidTableName validates table identifier formatting.
func IsValidTableName(name string) bool {
	return validTableNameRegex.MatchString(name)
}

// IsSystemTable returns true if the table is restricted system metadata.
func IsSystemTable(table string) bool {
	lower := strings.ToLower(table)
	if systemTables[lower] {
		return true
	}
	return strings.HasPrefix(lower, "pg_") || strings.HasPrefix(lower, "information_schema")
}

// EnsureTable initializes a table if not already created, caching known tables in memory.
func (app *App) EnsureTable(ctx context.Context, table string) error {
	// Check memory cache first to avoid runtime DDL lock overhead.
	if _, loaded := app.knownTables.Load(table); loaded {
		return nil
	}

	if !IsValidTableName(table) {
		return fmt.Errorf("invalid table name: %q", table)
	}

	const queryTemplate = `
		CREATE TABLE IF NOT EXISTS "%s" (
			id BIGSERIAL PRIMARY KEY,
			key TEXT UNIQUE NOT NULL,
			value JSONB
		)`
	query := fmt.Sprintf(queryTemplate, table)
	if _, err := app.Driver.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("ensure table %q: %w", table, err)
	}

	app.knownTables.Store(table, true)
	return nil
}

func (app *App) PrepareTables(ctx context.Context) error {
	tables := []string{"auth"}
	for _, r := range app.Routes {
		if r.Table == "" || strings.HasPrefix(r.Table, "$") {
			continue
		}
		tables = append(tables, r.Table)
	}
	for _, table := range tables {
		if err := app.EnsureTable(ctx, table); err != nil {
			return err
		}
	}
	return nil
}

// ExecTx executes a function inside a database transaction with automatic rollback.
func (app *App) ExecTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := app.Driver.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (app *App) GetRow(ctx context.Context, table, key string) (map[string]interface{}, error) {
	if err := app.EnsureTable(ctx, table); err != nil {
		return nil, err
	}
	var value []byte
	query := fmt.Sprintf(`SELECT value FROM "%s" WHERE key = $1`, table)
	err := app.Driver.QueryRowContext(ctx, query, key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("row not found in %q with key %q", table, key)
		}
		return nil, fmt.Errorf("query row in %q: %w", table, err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(value, &result); err != nil {
		return nil, fmt.Errorf("unmarshal JSON from %q (key=%q): %w", table, key, err)
	}
	return result, nil
}

func (app *App) GetRowStruct(ctx context.Context, table, key string, dest interface{}) error {
	if err := app.EnsureTable(ctx, table); err != nil {
		return err
	}
	var value []byte
	query := fmt.Sprintf(`SELECT value FROM "%s" WHERE key = $1`, table)
	err := app.Driver.QueryRowContext(ctx, query, key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("row not found in %q with key %q", table, key)
		}
		return fmt.Errorf("query row in %q: %w", table, err)
	}
	if err := json.Unmarshal(value, dest); err != nil {
		return fmt.Errorf("unmarshal JSON from %q (key=%q): %w", table, key, err)
	}
	return nil
}

func (app *App) GetRows(ctx context.Context, table string) ([]map[string]interface{}, error) {
	return app.GetRowsPaginated(ctx, table, defaultMaxRows, 0)
}

func (app *App) GetRowsPaginated(ctx context.Context, table string, limit, offset int) ([]map[string]interface{}, error) {
	if err := app.EnsureTable(ctx, table); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 1000 {
		limit = defaultMaxRows
	}
	if offset < 0 {
		offset = 0
	}

	// Deterministic ordering by primary key ID (newest first)
	query := fmt.Sprintf(`SELECT value FROM "%s" ORDER BY id DESC LIMIT $1 OFFSET $2`, table)
	rows, err := app.Driver.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query table %q: %w", table, err)
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var value []byte
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("scan row in %q: %w", table, err)
		}
		var entry map[string]interface{}
		if err := json.Unmarshal(value, &entry); err != nil {
			return nil, fmt.Errorf("unmarshal row in %q: %w", table, err)
		}
		results = append(results, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error in %q: %w", table, err)
	}
	return results, nil
}

func (app *App) InsertRow(ctx context.Context, table, key string, value interface{}) error {
	if err := app.EnsureTable(ctx, table); err != nil {
		return err
	}
	var data []byte
	var err error
	switch v := value.(type) {
	case []byte:
		data = v
	case json.RawMessage:
		data = v
	default:
		data, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("marshal value for %q/%q: %w", table, key, err)
		}
	}
	query := fmt.Sprintf(`
		INSERT INTO "%s" (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		table)

	if _, err := app.Driver.ExecContext(ctx, query, key, data); err != nil {
		return fmt.Errorf("upsert into %q (key=%q): %w", table, key, err)
	}
	return nil
}
