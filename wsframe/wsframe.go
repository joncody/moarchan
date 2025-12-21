package wsframe

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/joncody/wsrooms"
	_ "github.com/lib/pq"
)

type App struct {
	Name         string `json:"name"`
	HashKey      string `json:"hashkey"`
	BlockKey     string `json:"blockkey"`
	SecureCookie *securecookie.SecureCookie
	Templates    *template.Template
	Port         string   `json:"port"`
	SSLPort      string   `json:"sslport"`
	Database     DBConfig `json:"database"`
	Driver       *sql.DB
	Routes       []Route `json:"routes"`
	Added        []AddedRoute
	Router       *mux.Router
}

func (wfa *App) Start() {
	dbstring := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", wfa.Database.User, wfa.Database.Password, wfa.Database.Name)
	wfa.SecureCookie = securecookie.New([]byte(wfa.HashKey), []byte(wfa.BlockKey))
	driver, err := sql.Open("postgres", dbstring)
	if err != nil {
		log.Fatal(err)
	}
	defer driver.Close()
	wfa.Driver = driver
	wfa.prepareTables()
	if wfa.SSLPort != "0" {
		go log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%s", wfa.SSLPort), "server.crt", "server.key", wfa.Router))
	}
	log.Fatal(http.ListenAndServe(":"+wfa.Port, wfa.Router))
}

func NewApp(cj string) *App {
	app := &App{
		Name:      "wsframe",
		Templates: template.Must(template.New("").Funcs(TemplateFuncs).ParseGlob("./static/views/*")),
		HashKey:   "very-secret",
		BlockKey:  "a-lotvery-secret",
		Port:      "8080",
		SSLPort:   "0",
		Database: DBConfig{
			User:     "dbuser",
			Password: "dbpass",
			Name:     "dbname",
		},
		Router: mux.NewRouter().StrictSlash(false),
	}
	cjb, err := ioutil.ReadFile(cj)
	if err != nil {
		log.Fatal(err)
	}
	if err = json.Unmarshal(cjb, &app); err != nil {
		log.Fatal(err)
	}
	app.setupRoutes()
	wsrooms.Emitter.On("request", app.processRequest)
	return app
}
