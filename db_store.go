// Package main implements the domain-specific database queries, schema migrations,
// and relational data providers for MoarChan.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"moarchan/frame"

	"golang.org/x/crypto/bcrypt"
)

// InitMoarchanSchema applies all domain schema migrations using the versioned
// migration system in frame. Each migration executes within an atomic transaction
// and is recorded in the schema_migrations table, preventing repetitive and expensive
// table alterations or data backfills on subsequent application boots.
func InitMoarchanSchema(ctx context.Context, app *frame.App) error {
	// Migration 001: Create boards table for board slug routing
	if err := app.RunMigration(ctx, "001_create_boards", "Create boards table", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS boards (
	id BIGSERIAL PRIMARY KEY,
	slug TEXT UNIQUE NOT NULL
);`)
		return err
	}); err != nil {
		return fmt.Errorf("migration 001: %w", err)
	}

	// Migration 002: Create threads table
	if err := app.RunMigration(ctx, "002_create_threads", "Create threads table", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS threads (
	id BIGSERIAL PRIMARY KEY,
	hash TEXT UNIQUE NOT NULL,
	topic TEXT NOT NULL,
	name TEXT NOT NULL,
	subject TEXT,
	options TEXT,
	password_hash TEXT,
	comment TEXT NOT NULL,
	file_name TEXT NOT NULL,
	file_mime TEXT NOT NULL,
	file_size TEXT NOT NULL,
	file_dimensions TEXT NOT NULL,
	timestamp TEXT NOT NULL,
	bumped_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);`)
		return err
	}); err != nil {
		return fmt.Errorf("migration 002: %w", err)
	}

	// Migration 003: Create posts table for thread replies
	if err := app.RunMigration(ctx, "003_create_posts", "Create posts table", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS posts (
	id BIGSERIAL PRIMARY KEY,
	hash TEXT UNIQUE NOT NULL,
	thread_hash TEXT NOT NULL REFERENCES threads(hash) ON DELETE CASCADE,
	topic TEXT NOT NULL,
	name TEXT NOT NULL,
	options TEXT,
	password_hash TEXT,
	comment TEXT NOT NULL,
	file_name TEXT,
	file_mime TEXT,
	file_size TEXT,
	file_dimensions TEXT,
	timestamp TEXT NOT NULL,
	tagging JSONB DEFAULT '[]'::jsonb,
	tagged_by JSONB DEFAULT '[]'::jsonb,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);`)
		return err
	}); err != nil {
		return fmt.Errorf("migration 003: %w", err)
	}

	// Migration 004: Add password_hash to threads for legacy schema compatibility
	if err := app.RunMigration(ctx, "004_threads_password_hash", "Add password_hash column to threads", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `ALTER TABLE threads ADD COLUMN IF NOT EXISTS password_hash TEXT;`)
		return err
	}); err != nil {
		return fmt.Errorf("migration 004: %w", err)
	}

	// Migration 005: Add bumped_at to threads and backfill timestamp from created_at
	if err := app.RunMigration(ctx, "005_threads_bumped_at", "Add bumped_at column and backfill from created_at", func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE threads ADD COLUMN IF NOT EXISTS bumped_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;`); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE threads SET bumped_at = created_at WHERE bumped_at IS NULL;`)
		return err
	}); err != nil {
		return fmt.Errorf("migration 005: %w", err)
	}

	// Migration 006: Add password_hash to posts for legacy schema compatibility
	if err := app.RunMigration(ctx, "006_posts_password_hash", "Add password_hash column to posts", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `ALTER TABLE posts ADD COLUMN IF NOT EXISTS password_hash TEXT;`)
		return err
	}); err != nil {
		return fmt.Errorf("migration 006: %w", err)
	}

	// Migration 007: Create performance indexes for feed sorting and cascade lookups
	if err := app.RunMigration(ctx, "007_create_indexes", "Create performance indexes", func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_threads_topic_bumped ON threads(topic, bumped_at DESC);`); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_posts_thread_hash ON posts(thread_hash);`)
		return err
	}); err != nil {
		return fmt.Errorf("migration 007: %w", err)
	}

	return nil
}

// MoarchanDataProvider serves as the custom DataProvider hook for the frame
// router. It translates dynamic route match parameters into database queries,
// returning either a single thread with all replies or a list of active board threads.
func MoarchanDataProvider(ctx context.Context, app *frame.App, cfg frame.RouteConfig, subs []string) (interface{}, error) {
	topic := resolveSub(cfg.Table, subs)
	key := resolveSub(cfg.Key, subs)

	if topic == "" || topic == "main" {
		return nil, nil
	}

	if key != "" {
		return GetSingleThread(ctx, app.Driver, topic, key)
	}
	return GetTopicThreads(ctx, app.Driver, topic)
}

// resolveSub extracts indexed regex capture groups (e.g., "$1", "$2") from matched route URLs.
func resolveSub(field string, subs []string) string {
	if len(field) > 1 && field[0] == '$' {
		idx := int(field[1] - '0')
		if idx >= 0 && idx < len(subs) {
			return subs[idx]
		}
	}
	return field
}

// GetTopicThreads queries the most recently bumped 100 threads for a given topic board.
// Replies are aggregated directly in PostgreSQL using jsonb_agg to eliminate N+1 query overhead.
func GetTopicThreads(ctx context.Context, db *sql.DB, topic string) ([]map[string]interface{}, error) {
	const query = `
SELECT
	t.hash, t.topic, t.name, COALESCE(t.subject, ''), COALESCE(t.options, ''), t.comment,
	t.file_name, t.file_mime, t.file_size, t.file_dimensions, t.timestamp,
	COALESCE(
		jsonb_agg(
			jsonb_build_object(
				'hash', p.hash,
				'thread', p.thread_hash,
				'topic', p.topic,
				'name', p.name,
				'options', COALESCE(p.options, ''),
				'comment', p.comment,
				'file_name', COALESCE(p.file_name, ''),
				'file_mime', COALESCE(p.file_mime, ''),
				'file_size', COALESCE(p.file_size, ''),
				'file_dimensions', COALESCE(p.file_dimensions, ''),
				'timestamp', p.timestamp,
				'tagging', p.tagging,
				'taggedBy', p.tagged_by
			) ORDER BY p.id ASC
		) FILTER (WHERE p.hash IS NOT NULL),
		'[]'::jsonb
	) as replies
FROM threads t
LEFT JOIN posts p ON t.hash = p.thread_hash
WHERE t.topic = $1
GROUP BY t.id, t.hash, t.bumped_at
ORDER BY t.bumped_at DESC
LIMIT 100
`
	rows, err := db.QueryContext(ctx, query, topic)
	if err != nil {
		return nil, fmt.Errorf("query topic threads for %q: %w", topic, err)
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var hash, top, name, subject, options, comment, fileName, fileMime, fileSize, fileDims, timestamp string
		var repliesRaw []byte

		err := rows.Scan(
			&hash, &top, &name, &subject, &options, &comment,
			&fileName, &fileMime, &fileSize, &fileDims, &timestamp, &repliesRaw,
		)
		if err != nil {
			return nil, fmt.Errorf("scan thread row: %w", err)
		}

		var replies []map[string]interface{}
		if err := json.Unmarshal(repliesRaw, &replies); err != nil {
			replies = make([]map[string]interface{}, 0)
		}

		entry := map[string]interface{}{
			"hash":            hash,
			"topic":           top,
			"name":            name,
			"subject":         subject,
			"options":         options,
			"comment":         comment,
			"file_name":       fileName,
			"file_mime":       fileMime,
			"file_size":       fileSize,
			"file_dimensions": fileDims,
			"timestamp":       timestamp,
			"replies":         replies,
			"taggedBy":        []string{},
			"tagging":         []string{},
		}
		results = append(results, entry)
	}
	return results, nil
}

// GetSingleThread retrieves a specific thread by topic and hash, including all replies
// ordered chronologically by ascending ID.
func GetSingleThread(ctx context.Context, db *sql.DB, topic, threadHash string) (map[string]interface{}, error) {
	const query = `
SELECT
	t.hash, t.topic, t.name, COALESCE(t.subject, ''), COALESCE(t.options, ''), t.comment,
	t.file_name, t.file_mime, t.file_size, t.file_dimensions, t.timestamp,
	COALESCE(
		jsonb_agg(
			jsonb_build_object(
				'hash', p.hash,
				'thread', p.thread_hash,
				'topic', p.topic,
				'name', p.name,
				'options', COALESCE(p.options, ''),
				'comment', p.comment,
				'file_name', COALESCE(p.file_name, ''),
				'file_mime', COALESCE(p.file_mime, ''),
				'file_size', COALESCE(p.file_size, ''),
				'file_dimensions', COALESCE(p.file_dimensions, ''),
				'timestamp', p.timestamp,
				'tagging', p.tagging,
				'taggedBy', p.tagged_by
			) ORDER BY p.id ASC
		) FILTER (WHERE p.hash IS NOT NULL),
		'[]'::jsonb
	) as replies
FROM threads t
LEFT JOIN posts p ON t.hash = p.thread_hash
WHERE t.topic = $1 AND t.hash = $2
GROUP BY t.id, t.hash
`
	var hash, top, name, subject, options, comment, fileName, fileMime, fileSize, fileDims, timestamp string
	var repliesRaw []byte

	err := db.QueryRowContext(ctx, query, topic, threadHash).Scan(
		&hash, &top, &name, &subject, &options, &comment,
		&fileName, &fileMime, &fileSize, &fileDims, &timestamp, &repliesRaw,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("thread %q not found in topic %q", threadHash, topic)
		}
		return nil, fmt.Errorf("query single thread: %w", err)
	}

	var replies []map[string]interface{}
	if err := json.Unmarshal(repliesRaw, &replies); err != nil {
		replies = make([]map[string]interface{}, 0)
	}

	return map[string]interface{}{
		"hash":            hash,
		"topic":           top,
		"name":            name,
		"subject":         subject,
		"options":         options,
		"comment":         comment,
		"file_name":       fileName,
		"file_mime":       fileMime,
		"file_size":       fileSize,
		"file_dimensions": fileDims,
		"timestamp":       timestamp,
		"replies":         replies,
		"taggedBy":        []string{},
		"tagging":         []string{},
	}, nil
}

// GetSinglePost retrieves an individual reply post by its unique hash.
func GetSinglePost(ctx context.Context, db *sql.DB, hash string) (map[string]interface{}, error) {
	const query = `
SELECT
	p.hash, p.thread_hash, p.topic, p.name, COALESCE(p.options, ''), p.comment,
	COALESCE(p.file_name, ''), COALESCE(p.file_mime, ''), COALESCE(p.file_size, ''), COALESCE(p.file_dimensions, ''),
	p.timestamp, p.tagging, p.tagged_by
FROM posts p
WHERE p.hash = $1
`
	var postHash, threadHash, topic, name, options, comment, fileName, fileMime, fileSize, fileDims, timestamp string
	var taggingRaw, taggedByRaw []byte

	err := db.QueryRowContext(ctx, query, hash).Scan(
		&postHash, &threadHash, &topic, &name, &options, &comment,
		&fileName, &fileMime, &fileSize, &fileDims, &timestamp,
		&taggingRaw, &taggedByRaw,
	)
	if err != nil {
		return nil, fmt.Errorf("query single post %q: %w", hash, err)
	}

	var tagging, taggedBy []string
	if err := json.Unmarshal(taggingRaw, &tagging); err != nil {
		tagging = []string{}
	}
	if err := json.Unmarshal(taggedByRaw, &taggedBy); err != nil {
		taggedBy = []string{}
	}

	return map[string]interface{}{
		"hash":            postHash,
		"thread":          threadHash,
		"topic":           topic,
		"name":            name,
		"options":         options,
		"comment":         comment,
		"file_name":       fileName,
		"file_mime":       fileMime,
		"file_size":       fileSize,
		"file_dimensions": fileDims,
		"timestamp":       timestamp,
		"tagging":         tagging,
		"taggedBy":        taggedBy,
	}, nil
}

// DeletedPostInfo encapsulates metadata regarding deleted posts and associated media files
// so that corresponding physical files can be purged from storage.
type DeletedPostInfo struct {
	Hash      string
	Topic     string
	IsThread  bool
	FileNames []string
}

// DeletePostOrFile handles authenticated post and attachment deletions.
// It verifies bcrypt passwords (or admin session overrides), determines whether the
// target is an OP thread or reply, and executes either a metadata clear (fileOnly)
// or cascade record deletion within a database transaction.
func DeletePostOrFile(ctx context.Context, db *sql.DB, hash, password string, isAdmin, fileOnly bool) (*DeletedPostInfo, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin delete tx: %w", err)
	}
	defer tx.Rollback()

	var topic, dbPassHash, opFileName string
	var isThread bool
	var fileNames []string

	// 1. Check if the target is a thread OP
	err = tx.QueryRowContext(ctx, `SELECT topic, COALESCE(password_hash, ''), COALESCE(file_name, '') FROM threads WHERE hash = $1 FOR UPDATE`, hash).
		Scan(&topic, &dbPassHash, &opFileName)
	if err == nil {
		isThread = true
	} else if errors.Is(err, sql.ErrNoRows) {
		// 2. Check if the target is a post reply
		err = tx.QueryRowContext(ctx, `SELECT topic, COALESCE(password_hash, ''), COALESCE(file_name, '') FROM posts WHERE hash = $1 FOR UPDATE`, hash).
			Scan(&topic, &dbPassHash, &opFileName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("post or thread %q not found", hash)
			}
			return nil, fmt.Errorf("query post: %w", err)
		}
	} else {
		return nil, fmt.Errorf("query thread: %w", err)
	}

	// 3. Verify deletion authorization
	if !isAdmin {
		if dbPassHash == "" {
			return nil, errors.New("post cannot be deleted without admin privileges")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(dbPassHash), []byte(password)); err != nil {
			return nil, errors.New("invalid post deletion password")
		}
	}

	if opFileName != "" {
		fileNames = append(fileNames, opFileName)
	}

	if fileOnly {
		// Clear file metadata while preserving the post text
		if isThread {
			_, err = tx.ExecContext(ctx, `UPDATE threads SET file_name = '', file_mime = '', file_size = '', file_dimensions = '' WHERE hash = $1`, hash)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE posts SET file_name = '', file_mime = '', file_size = '', file_dimensions = '' WHERE hash = $1`, hash)
		}
		if err != nil {
			return nil, fmt.Errorf("clear post file metadata: %w", err)
		}
	} else {
		if isThread {
			// Collect reply filenames before cascade delete
			rows, err := tx.QueryContext(ctx, `SELECT file_name FROM posts WHERE thread_hash = $1 AND file_name != ''`, hash)
			if err == nil {
				for rows.Next() {
					var rfn string
					if err := rows.Scan(&rfn); err == nil && rfn != "" {
						fileNames = append(fileNames, rfn)
					}
				}
				rows.Close()
			}
			_, err = tx.ExecContext(ctx, `DELETE FROM threads WHERE hash = $1`, hash)
		} else {
			_, err = tx.ExecContext(ctx, `DELETE FROM posts WHERE hash = $1`, hash)
		}
		if err != nil {
			return nil, fmt.Errorf("delete post row: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit delete tx: %w", err)
	}

	return &DeletedPostInfo{
		Hash:      hash,
		Topic:     topic,
		IsThread:  isThread,
		FileNames: fileNames,
	}, nil
}
