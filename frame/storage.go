package frame

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage abstracts file persistence so the application can swap between
// local disk and S3-compatible object stores (e.g. LocalStack, MinIO, AWS)
// without changing handler logic. The default implementation is LocalDiskStorage.
type Storage interface {
	// Save writes data to the given relative path.
	Save(ctx context.Context, relPath string, data io.Reader) error
	// Delete removes the object at the given relative path.
	Delete(ctx context.Context, relPath string) error
	// PublicURL returns the externally-accessible URL for the given relative path.
	PublicURL(relPath string) string
}

// ──────────────────────────── Local Disk Storage ────────────────────────────

// LocalDiskStorage implements Storage by writing to the local filesystem.
// File writes are handled concurrently without global locking because modern
// OS filesystems manage concurrent writes to distinct file paths safely.
type LocalDiskStorage struct {
	basePath  string
	urlPrefix string
}

func NewLocalDiskStorage(basePath, urlPrefix string) (*LocalDiskStorage, error) {
	abs, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("resolve storage base path: %w", err)
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}
	if urlPrefix == "" {
		urlPrefix = "/static/images/uploads"
	}
	return &LocalDiskStorage{
		basePath:  abs,
		urlPrefix: urlPrefix,
	}, nil
}

func (s *LocalDiskStorage) Save(ctx context.Context, relPath string, data io.Reader) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullPath := filepath.Join(s.basePath, relPath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create parent directory %q: %w", dir, err)
	}

	out, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create file %q: %w", fullPath, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, data); err != nil {
		return fmt.Errorf("write file %q: %w", fullPath, err)
	}
	return nil
}

func (s *LocalDiskStorage) Delete(ctx context.Context, relPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullPath := filepath.Join(s.basePath, relPath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file %q: %w", fullPath, err)
	}
	return nil
}

func (s *LocalDiskStorage) PublicURL(relPath string) string {
	return fmt.Sprintf("%s/%s", s.urlPrefix, relPath)
}
