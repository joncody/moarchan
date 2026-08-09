package frame

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/joncody/roomer"
	_ "github.com/lib/pq"
)

type DBConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type AppConfig struct {
	Name         string   `json:"name"`
	HashKey      string   `json:"hashkey"`
	BlockKey     string   `json:"blockkey"`
	Port         string   `json:"port"`
	SSLPort      string   `json:"sslport"`
	ViewsPattern string   `json:"views_pattern,omitempty"`
	Database     DBConfig `json:"database"`
	Routes       []Route  `json:"routes"`
}

type App struct {
	AppConfig
	SessionStore   sessions.Store
	Templates      *template.Template
	Driver         *sql.DB
	Added          []AddedRoute
	CompiledRoutes []CompiledRoute
	Router         *mux.Router
	knownTables    sync.Map // In-memory cache of verified database tables
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func (app *App) Start() error {
	dbUser := getEnv("POSTGRES_USER", app.AppConfig.Database.User)
	dbPass := getEnv("POSTGRES_PASSWORD", app.AppConfig.Database.Password)
	dbName := getEnv("POSTGRES_DB", app.AppConfig.Database.Name)
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5432")
	sslMode := getEnv("POSTGRES_SSLMODE", "disable")

	dbstring := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPass, dbName, sslMode,
	)
	db, err := sql.Open("postgres", dbstring)
	if err != nil {
		return fmt.Errorf("open DB: %w", err)
	}

	// Configure DB Connection Pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	app.Driver = db

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("ping DB: %w", err)
	}

	if err := app.PrepareTables(ctx); err != nil {
		db.Close()
		return fmt.Errorf("prepare tables: %w", err)
	}

	addr := ":" + app.AppConfig.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      app.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting HTTP server on %s", addr)
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
	if app.Driver != nil {
		return app.Driver.Close()
	}
	return nil
}

func NewApp(configPath string) (*App, error) {
	app := &App{
		AppConfig: AppConfig{
			Name:         "frame",
			Port:         "8080",
			SSLPort:      "0",
			ViewsPattern: "./static/views/*",
			Database: DBConfig{
				User:     "dbuser",
				Password: "dbpass",
				Name:     "dbname",
			},
		},
	}
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("read config %q: %w", configPath, err)
		}
		var cfg AppConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
		app.AppConfig = cfg
	}

	if app.AppConfig.ViewsPattern == "" {
		app.AppConfig.ViewsPattern = "./static/views/*"
	}

	// Session keys override via environment variables
	hashKey := getEnv("SESSION_HASH_KEY", app.AppConfig.HashKey)
	blockKey := getEnv("SESSION_BLOCK_KEY", app.AppConfig.BlockKey)

	secure := app.AppConfig.SSLPort != "" && app.AppConfig.SSLPort != "0"
	store, err := NewSessionStore(app.AppConfig.Name, hashKey, blockKey, secure)
	if err != nil {
		return nil, fmt.Errorf("session store: %w", err)
	}
	app.SessionStore = store

	app.Templates, err = template.New("").Funcs(TemplateFuncs).ParseGlob(app.AppConfig.ViewsPattern)
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	app.Router, err = app.SetupRoutes()
	if err != nil {
		return nil, fmt.Errorf("setup routes: %w", err)
	}

	if err := roomer.RegisterHandler("request", app.ProcessRequest); err != nil {
		log.Fatal("Failed to register handler:", err)
	}

	return app, nil
}
