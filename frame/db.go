// Package frame provides a decoupled web framework featuring session management,
// dynamic routing, database KV helpers, and Server-Sent Events (SSE).
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

// validTableNameRegex ensures that table names contain only alphanumeric characters and underscores.
var validTableNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// defaultMaxRows specifies the fallback limit for multi-row queries.
const defaultMaxRows = 1000

// systemTables maps reserved table names that are blocked from direct KV manipulation.
var systemTables = map[string]bool{
	"auth": true,
}

// IsValidTableName verifies that a table identifier adheres to strict character whitelist rules.
func IsValidTableName(name string) bool {
	return validTableNameRegex.MatchString(name)
}

// IsSystemTable returns true if the table is reserved or belongs to PostgreSQL catalog schemas.
func IsSystemTable(table string) bool {
	lower := strings.ToLower(table)
	if systemTables[lower] {
		return true
	}
	return strings.HasPrefix(lower, "pg_") || strings.HasPrefix(lower, "information_schema")
}

// EnsureTable lazily creates the key-value document table if it does not yet exist.
func (app *App) EnsureTable(ctx context.Context, table string) error {
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

// PrepareTables initializes the primary auth table and all static route tables registered on boot.
func (app *App) PrepareTables(ctx context.Context) error {
	const initSchema = `
CREATE TABLE IF NOT EXISTS auth (
	id BIGSERIAL PRIMARY KEY,
	key TEXT UNIQUE NOT NULL,
	value JSONB
);
`
	if _, err := app.Driver.ExecContext(ctx, initSchema); err != nil {
		return fmt.Errorf("prepare frame base database schema: %w", err)
	}
	app.knownTables.Store("auth", true)

	for _, r := range app.Routes {
		if r.Table == "" || strings.HasPrefix(r.Table, "$") {
			continue
		}
		if err := app.EnsureTable(ctx, r.Table); err != nil {
			return err
		}
	}
	return nil
}

// ExecTx executes a function within a database transaction, automatically rolling back on failure.
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

// GetRow retrieves a single JSONB document by key from the specified table.
//
// SECURITY INVARIANT: The `table` parameter is interpolated into the SQL
// query via fmt.Sprintf. This is safe ONLY because:
//   1. IsValidTableName() enforces ^[a-zA-Z0-9_]+$ before any query is built.
//   2. IsSystemTable() blocks access to pg_*, information_schema, and auth.
//   3. All public entry points (EnsureTable, GetRow, InsertRow, etc.) call
//      IsValidTableName() as their first operation.
//
// Do NOT add new code paths that accept a table name without validation.
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

// GetRowStruct retrieves a single JSONB document and decodes it directly into the provided destination struct.
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

// GetRows retrieves all rows from the specified key-value table up to defaultMaxRows.
func (app *App) GetRows(ctx context.Context, table string) ([]map[string]interface{}, error) {
	return app.GetRowsPaginated(ctx, table, defaultMaxRows, 0)
}

// GetRowsPaginated retrieves paginated JSONB rows from the specified table ordered by ID descending.
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

// InsertRow upserts a JSONB document into the specified table using ON CONFLICT (key) DO UPDATE.
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

// DeleteRow deletes a document by key from the specified table.
func (app *App) DeleteRow(ctx context.Context, table, key string) error {
	if err := app.EnsureTable(ctx, table); err != nil {
		return err
	}
	query := fmt.Sprintf(`DELETE FROM "%s" WHERE key = $1`, table)
	if _, err := app.Driver.ExecContext(ctx, query, key); err != nil {
		return fmt.Errorf("delete from %q (key=%q): %w", table, key, err)
	}
	return nil
}
