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
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/joncody/roomer"
	_ "github.com/lib/pq"
)

// DBConfig holds database connection info.
type DBConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

const (
	MinHashKeyLen  = 32
	MinBlockKeyLen = 32
)

type AppConfig struct {
	Name     string   `json:"name"`
	HashKey  string   `json:"hashkey"`
	BlockKey string   `json:"blockkey"`
	Port     string   `json:"port"`
	SSLPort  string   `json:"sslport"`
	Database DBConfig `json:"database"`
	Routes   []Route  `json:"routes"`
}

type App struct {
	AppConfig
	SessionStore   sessions.Store
	Templates      *template.Template
	Driver         *sql.DB
	Added          []AddedRoute
	CompiledRoutes []CompiledRoute
	Router         *mux.Router
}

func logFatalIfErr(err error) {
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func (app *App) Start() error {
	// Build DB connection string
	dbstring := fmt.Sprintf(
		"user=%s password=%s dbname=%s sslmode=disable",
		app.AppConfig.Database.User, app.AppConfig.Database.Password, app.AppConfig.Database.Name,
	)
	// Open DB
	db, err := sql.Open("postgres", dbstring)
	if err != nil {
		return fmt.Errorf("open DB: %w", err)
	}
	app.Driver = db
	// Ping DB to ensure connectivity
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("ping DB: %w", err)
	}
	// Prepare tables
	if err := app.PrepareTables(ctx); err != nil {
		db.Close()
		return fmt.Errorf("prepare tables: %w", err)
	}
	if app.AppConfig.SSLPort != "" && app.AppConfig.SSLPort != "0" {
		go func() {
			addr := ":" + app.AppConfig.SSLPort
			log.Printf("Starting HTTPS server on %s", addr)
			if err := http.ListenAndServeTLS(addr, "server.crt", "server.key", app.Router); err != nil {
				logFatalIfErr(fmt.Errorf("HTTPS server failed on %s: %w", addr, err))
			}
		}()
	}

	addr := ":" + app.AppConfig.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: app.Router,
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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
			Name:    "frame",
			Port:    "8080",
			SSLPort: "0",
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
	// Setup session store
	secure := app.AppConfig.SSLPort != "" && app.AppConfig.SSLPort != "0"
	store, err := NewSessionStore(app.AppConfig.Name, app.AppConfig.HashKey, app.AppConfig.BlockKey, secure)
	if err != nil {
		return nil, fmt.Errorf("session store: %w", err)
	}
	app.SessionStore = store
	// Parse templates
	app.Templates, err = template.New("").Funcs(TemplateFuncs).ParseGlob("./static/views/*")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	// Setup routes
	app.Router, err = app.SetupRoutes()
	if err != nil {
		return nil, fmt.Errorf("setup routes: %w", err)
	}
	// WebSocket event
	if err := roomer.RegisterHandler("request", app.ProcessRequest); err != nil {
		log.Fatal("Failed to register handler:", err)
	}
	return app, nil
}
