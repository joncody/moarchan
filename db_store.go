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

func InitMoarchanSchema(ctx context.Context, app *frame.App) error {
	const query = `
		CREATE TABLE IF NOT EXISTS boards (
			id BIGSERIAL PRIMARY KEY,
			slug TEXT UNIQUE NOT NULL
		);

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
		);

		-- Safe non-destructive migrations for existing tables
		ALTER TABLE threads ADD COLUMN IF NOT EXISTS password_hash TEXT;
		ALTER TABLE threads ADD COLUMN IF NOT EXISTS bumped_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

		-- Backfill bumped_at from created_at for any legacy threads
		UPDATE threads SET bumped_at = created_at WHERE bumped_at IS NULL;

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
		);

		-- Safe non-destructive migration for posts table
		ALTER TABLE posts ADD COLUMN IF NOT EXISTS password_hash TEXT;

		-- Indexes
		CREATE INDEX IF NOT EXISTS idx_threads_topic_bumped ON threads(topic, bumped_at DESC);
		CREATE INDEX IF NOT EXISTS idx_posts_thread_hash ON posts(thread_hash);
	`
	if _, err := app.Driver.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("init moarchan schema: %w", err)
	}
	return nil
}

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

func resolveSub(field string, subs []string) string {
	if len(field) > 1 && field[0] == '$' {
		idx := int(field[1] - '0')
		if idx >= 0 && idx < len(subs) {
			return subs[idx]
		}
	}
	return field
}

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

type DeletedPostInfo struct {
	Hash      string
	Topic     string
	IsThread  bool
	FileNames []string
}

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
