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

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/joncody/wsrooms"
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

type App struct {
	Name           string `json:"name"`
	HashKey        string `json:"hashkey"`
	BlockKey       string `json:"blockkey"`
	SecureCookie   *securecookie.SecureCookie
	Templates      *template.Template
	Port           string   `json:"port"`
	SSLPort        string   `json:"sslport"`
	Database       DBConfig `json:"database"`
	Driver         *sql.DB
	Routes         []Route `json:"routes"`
	Added          []AddedRoute
	CompiledRoutes []CompiledRoute
	Router         *mux.Router
}

func NewApp(configPath string) (*App, error) {
	app := &App{
		Name:    "frame",
		Port:    "8080",
		SSLPort: "0",
		Database: DBConfig{
			User:     "dbuser",
			Password: "dbpass",
			Name:     "dbname",
		},
		Router: mux.NewRouter().StrictSlash(false),
	}

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("read config %q: %w", configPath, err)
		}
		if err := json.Unmarshal(data, app); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}

	if err := app.initSecureCookie(); err != nil {
		return nil, err
	}

	var err error
	app.Templates, err = template.New("").Funcs(TemplateFuncs).ParseGlob("./static/views/*")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	if err := app.setupRoutes(); err != nil {
		return nil, fmt.Errorf("setup routes: %w", err)
	}

	wsrooms.Emitter.On("request", app.processRequest)
	return app, nil
}

func (app *App) initSecureCookie() error {
	if len(app.HashKey) < MinHashKeyLen {
		return fmt.Errorf("hash key must be ≥%d bytes", MinHashKeyLen)
	}
	if len(app.BlockKey) < MinBlockKeyLen {
		return fmt.Errorf("block key must be ≥%d bytes", MinBlockKeyLen)
	}
	app.SecureCookie = securecookie.New([]byte(app.HashKey), []byte(app.BlockKey))
	return nil
}

func (app *App) Start() error {
	dbstring := fmt.Sprintf(
		"user=%s password=%s dbname=%s sslmode=disable",
		app.Database.User, app.Database.Password, app.Database.Name,
	)

	db, err := sql.Open("postgres", dbstring)
	if err != nil {
		return fmt.Errorf("open DB: %w", err)
	}
	app.Driver = db

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("ping DB: %w", err)
	}

	if err := app.PrepareTables(ctx); err != nil {
		db.Close()
		return fmt.Errorf("prepare tables: %w", err)
	}

	if app.SSLPort != "" && app.SSLPort != "0" {
		go func() {
			addr := ":" + app.SSLPort
			if err := http.ListenAndServeTLS(addr, "server.crt", "server.key", app.Router); err != nil {
				logFatalIfErr(fmt.Errorf("HTTPS server on %s failed: %w", addr, err))
			}
		}()
	}

	addr := ":" + app.Port
	return fmt.Errorf("HTTP server on %s failed: %w", addr, http.ListenAndServe(addr, app.Router))
}

func (app *App) Close() error {
	if app.Driver != nil {
		return app.Driver.Close()
	}
	return nil
}

func logFatalIfErr(err error) {
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
