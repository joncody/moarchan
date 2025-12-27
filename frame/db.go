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

var validTableNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func IsValidTableName(name string) bool {
	return validTableNameRegex.MatchString(name)
}

func (app *App) PrepareTables(ctx context.Context) error {
	const queryTemplate = `
		CREATE TABLE IF NOT EXISTS %s (
			id BIGSERIAL PRIMARY KEY,
			key TEXT UNIQUE NOT NULL,
			value JSONB
		)`

	tables := []string{"auth"}
	for _, r := range app.Routes {
		if r.Table == "" || strings.HasPrefix(r.Table, "$") {
			continue
		}
		tables = append(tables, r.Table)
	}

	for _, table := range tables {
		if !IsValidTableName(table) {
			return fmt.Errorf("invalid table name: %q", table)
		}
		query := fmt.Sprintf(queryTemplate, table)
		if _, err := app.Driver.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("create table %q: %w", table, err)
		}
	}
	return nil
}

func (app *App) GetRow(ctx context.Context, table, key string) (map[string]interface{}, error) {
	if !IsValidTableName(table) {
		return nil, fmt.Errorf("invalid table name: %q", table)
	}
	var value []byte
	query := fmt.Sprintf(`SELECT value FROM %s WHERE key = $1`, table)
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

func (app *App) GetRows(ctx context.Context, table string) ([]map[string]interface{}, error) {
	if !IsValidTableName(table) {
		return nil, fmt.Errorf("invalid table name: %q", table)
	}
	query := fmt.Sprintf(`SELECT value FROM %s`, table)
	rows, err := app.Driver.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query table %q: %w", table, err)
	}
	defer rows.Close()

	var results []map[string]interface{}
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
	if !IsValidTableName(table) {
		return fmt.Errorf("invalid table name: %q", table)
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value for %q/%q: %w", table, key, err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		table)

	if _, err := app.Driver.ExecContext(ctx, query, key, data); err != nil {
		return fmt.Errorf("upsert into %q (key=%q): %w", table, key, err)
	}
	return nil
}
