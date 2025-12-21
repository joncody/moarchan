package wsframe

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

type DBConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (wfa *App) prepareTables() {
	const query = `CREATE TABLE IF NOT EXISTS %s (
        id BIGSERIAL PRIMARY KEY,
        key TEXT UNIQUE NOT NULL,
        value JSON
    )`
	tables := []string{"auth"}
	for _, r := range wfa.Routes {
		if r.Table == "" || strings.HasPrefix(r.Table, "$") {
			continue
		}
		tables = append(tables, r.Table)
	}
	for _, table := range tables {
		if _, err := wfa.Driver.Exec(fmt.Sprintf(query, table)); err != nil {
			log.Fatalf("Error creating table %s: %v", table, err)
		}
	}
}

func (wfa *App) GetRow(table, key string) map[string]interface{} {
	var value []byte
	result := make(map[string]interface{})
	row := wfa.Driver.QueryRow(fmt.Sprintf(`SELECT value FROM %s WHERE key = $1`, table), key)
	if err := row.Scan(&value); err != nil {
		log.Printf("Scan error in GetRow(%s, %s): %v", table, key, err)
		return result
	}
	if err := json.Unmarshal(value, &result); err != nil {
		log.Printf("Unmarshal error in GetRow(%s, %s): %v", table, key, err)
	}
	return result
}

func (wfa *App) GetRows(table string) []map[string]interface{} {
	var results []map[string]interface{}
	rows, err := wfa.Driver.Query(fmt.Sprintf(`SELECT value FROM %s`, table))
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var value []byte
		entry := make(map[string]interface{})
		if err := rows.Scan(&value); err != nil {
			log.Printf("Scan error in GetRows(%s): %v", table, err)
			continue
		}
		if err := json.Unmarshal(value, &entry); err != nil {
			log.Printf("Unmarshal error in GetRows(%s): %v", table, err)
			continue
		}
		results = append(results, entry)
	}
	return results
}

func (wfa *App) InsertRow(table, key, value string) error {
	query := fmt.Sprintf(`
        INSERT INTO %s (key, value)
        VALUES ($1, $2)
        ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, table)

	if _, err := wfa.Driver.Exec(query, key, value); err != nil {
		log.Printf("InsertRow error (%s, %s): %v", table, key, err)
		return err
	}
	return nil
}
