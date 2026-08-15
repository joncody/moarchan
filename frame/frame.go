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

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
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
	SessionStore   sessions.Store
	Templates      *template.Template
	Driver         *sql.DB
	Added          []AddedRoute
	Routes         []Route
	CompiledRoutes []CompiledRoute
	Router         *mux.Router
	Hub            *SSEHub
	DataProvider   DataProvider
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
		Hub:    NewSSEHub(),
		Routes: make([]Route, 0),
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

	// Initialize distributed PostgreSQL LISTEN / NOTIFY real-time SSE listener
	app.Hub.InitDBListener(db, dbstring)

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

	addr := ":" + app.AppConfig.Port
	srv := &http.Server{
		Addr:              addr,
		Handler:           app.Router,
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
	if app.Hub != nil {
		app.Hub.Close()
	}
	if app.Driver != nil {
		return app.Driver.Close()
	}
	return nil
}
