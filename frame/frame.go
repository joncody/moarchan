package frame

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

type DBConfig struct {
	User     string
	Password string
	Name     string
}

type AppConfig struct {
	Name         string
	HashKey      string
	BlockKey     string
	Port         string
	SSLPort      string
	ViewsPattern string
	Database     DBConfig
}

type App struct {
	AppConfig
	SessionStore   *SessionStore
	Templates      *template.Template
	Driver         *sql.DB
	Added          []AddedRoute
	Routes         []Route
	CompiledRoutes []CompiledRoute
	Router         *http.ServeMux
	Hub            *SSEHub
	DataProvider   DataProvider
	Storage        Storage
	RateLimiter    *RateLimiter
	knownTables    sync.Map
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// Automatically parse local .env file if it exists
func loadDotEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return // .env is optional
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}

func NewApp() (*App, error) {
	// Parse local .env if present
	loadDotEnv(".env")

	app := &App{
		AppConfig: AppConfig{
			Name:         getEnv("APP_NAME", "frame"),
			Port:         getEnv("PORT", "9001"),
			SSLPort:      getEnv("SSL_PORT", "0"),
			HashKey:      getEnv("SESSION_HASH_KEY", "12345678901234567890123456789012"),
			BlockKey:     getEnv("SESSION_BLOCK_KEY", "abcdefghijklmnopqrstuvwx12345678"),
			ViewsPattern: getEnv("VIEWS_PATTERN", "./static/views/*"),
			Database: DBConfig{
				User:     getEnv("POSTGRES_USER", "postgres"),
				Password: getEnv("POSTGRES_PASSWORD", "postgres"),
				Name:     getEnv("POSTGRES_DB", "moarchan"),
			},
		},
		Hub:         NewSSEHub(),
		Routes:      make([]Route, 0),
		RateLimiter: NewRateLimiter(2.0, 10), // 2 tokens/sec, burst of 10
	}

	dbUser := app.AppConfig.Database.User
	dbPass := app.AppConfig.Database.Password
	dbName := app.AppConfig.Database.Name
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5432")
	sslMode := getEnv("POSTGRES_SSLMODE", "disable")

	dbstring := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPass, dbName, sslMode,
	)

	db, err := sql.Open("postgres", dbstring)
	if err != nil {
		return nil, fmt.Errorf("open DB: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	app.Driver = db

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping DB (connection string: %s): %w", dbstring, err)
	}

	if err := app.PrepareTables(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("prepare tables: %w", err)
	}

	// Initialize schema migrations tracking table
	if err := app.InitSchemaMigrations(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema migrations: %w", err)
	}

	// Initialize distributed PostgreSQL LISTEN / NOTIFY real-time SSE listener
	app.Hub.InitDBListener(db, dbstring)

	// Initialize file storage backend (local disk by default)
	storageBase := getEnv("UPLOAD_PATH", "./static/images/uploads")
	storageURLPrefix := getEnv("UPLOAD_URL_PREFIX", "/static/images/uploads")
	storage, err := NewLocalDiskStorage(storageBase, storageURLPrefix)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("init storage: %w", err)
	}
	app.Storage = storage

	secure := app.AppConfig.SSLPort != "" && app.AppConfig.SSLPort != "0"
	store, err := NewSessionStore(app.AppConfig.Name, app.AppConfig.HashKey, app.AppConfig.BlockKey, secure)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("session store: %w", err)
	}
	app.SessionStore = store

	app.Templates, err = template.New("").Funcs(TemplateFuncs).ParseGlob(app.AppConfig.ViewsPattern)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	app.Router, err = app.SetupRoutes()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("setup routes: %w", err)
	}

	return app, nil
}

func (app *App) Start() error {
	if err := app.CompileRoutes(); err != nil {
		return fmt.Errorf("compile routes on start: %w", err)
	}

	// Pipeline: Recovery -> Logging -> SecurityHeaders -> RateLimit -> CSRF -> Router
	// SecurityHeadersMiddleware is executed first so all responses (including 403 & 429) have defensive headers
	var handler http.Handler = app.Router
	handler = CSRFMiddleware(handler)
	handler = RateLimitMiddleware(app.RateLimiter)(handler)
	handler = SecurityHeadersMiddleware(handler)
	handler = LoggingMiddleware(handler)
	handler = RecoveryMiddleware(handler)

	addr := ":" + app.AppConfig.Port
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting HTTP/2-ready server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	return app.Close()
}

func (app *App) Close() error {
	if app.RateLimiter != nil {
		app.RateLimiter.Stop()
	}
	if app.Hub != nil {
		app.Hub.Close()
	}
	if app.Driver != nil {
		return app.Driver.Close()
	}
	return nil
}

// ──────────────────────────── Schema Migrations ────────────────────────────

// InitSchemaMigrations creates the schema_migrations tracking table if it
// does not already exist. This table records which versioned migrations
// have been applied, preventing re-execution on subsequent startups.
func (app *App) InitSchemaMigrations(ctx context.Context) error {
	const query = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version     TEXT PRIMARY KEY,
	description TEXT NOT NULL,
	applied_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);`
	if _, err := app.Driver.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	return nil
}

// RunMigration executes a migration function exactly once, identified by a
// unique version string. If the version has already been applied, the
// migration is skipped. This replaces the fragile idempotent-DDL pattern.
func (app *App) RunMigration(ctx context.Context, version, description string, fn func(tx *sql.Tx) error) error {
	var exists bool
	err := app.Driver.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check migration %q: %w", version, err)
	}
	if exists {
		return nil // Already applied
	}

	tx, err := app.Driver.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %q: %w", version, err)
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return fmt.Errorf("execute migration %q (%s): %w", version, description, err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, description) VALUES ($1, $2)`,
		version, description,
	); err != nil {
		return fmt.Errorf("record migration %q: %w", version, err)
	}

	return tx.Commit()
}
