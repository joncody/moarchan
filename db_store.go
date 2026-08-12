package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"moarchan/frame"
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
			comment TEXT NOT NULL,
			file_name TEXT NOT NULL,
			file_mime TEXT NOT NULL,
			file_size TEXT NOT NULL,
			file_dimensions TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS posts (
			id BIGSERIAL PRIMARY KEY,
			hash TEXT UNIQUE NOT NULL,
			thread_hash TEXT NOT NULL REFERENCES threads(hash) ON DELETE CASCADE,
			topic TEXT NOT NULL,
			name TEXT NOT NULL,
			options TEXT,
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

		CREATE INDEX IF NOT EXISTS idx_threads_topic ON threads(topic);
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
	if topic == "" {
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
			t.hash, t.topic, t.name, t.subject, t.options, t.comment,
			t.file_name, t.file_mime, t.file_size, t.file_dimensions, t.timestamp,
			COALESCE(
				jsonb_object_agg(
					p.hash,
					jsonb_build_object(
						'hash', p.hash,
						'thread', p.thread_hash,
						'topic', p.topic,
						'name', p.name,
						'options', p.options,
						'comment', p.comment,
						'file_name', COALESCE(p.file_name, ''),
						'file_mime', COALESCE(p.file_mime, ''),
						'file_size', COALESCE(p.file_size, ''),
						'file_dimensions', COALESCE(p.file_dimensions, ''),
						'timestamp', p.timestamp,
						'tagging', p.tagging,
						'taggedBy', p.tagged_by
					)
				) FILTER (WHERE p.hash IS NOT NULL),
				'{}'::jsonb
			) as replies
		FROM threads t
		LEFT JOIN posts p ON t.hash = p.thread_hash
		WHERE t.topic = $1
		GROUP BY t.id, t.hash
		ORDER BY t.id DESC
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

		var replies map[string]interface{}
		if err := json.Unmarshal(repliesRaw, &replies); err != nil {
			replies = make(map[string]interface{})
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
			t.hash, t.topic, t.name, t.subject, t.options, t.comment,
			t.file_name, t.file_mime, t.file_size, t.file_dimensions, t.timestamp,
			COALESCE(
				jsonb_object_agg(
					p.hash,
					jsonb_build_object(
						'hash', p.hash,
						'thread', p.thread_hash,
						'topic', p.topic,
						'name', p.name,
						'options', p.options,
						'comment', p.comment,
						'file_name', COALESCE(p.file_name, ''),
						'file_mime', COALESCE(p.file_mime, ''),
						'file_size', COALESCE(p.file_size, ''),
						'file_dimensions', COALESCE(p.file_dimensions, ''),
						'timestamp', p.timestamp,
						'tagging', p.tagging,
						'taggedBy', p.tagged_by
					)
				) FILTER (WHERE p.hash IS NOT NULL),
				'{}'::jsonb
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

	var replies map[string]interface{}
	if err := json.Unmarshal(repliesRaw, &replies); err != nil {
		replies = make(map[string]interface{})
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
