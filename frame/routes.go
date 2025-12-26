// routes.go
package frame

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joncody/wsrooms"
)

type RouteConfig struct {
	Table       string `json:"table"`
	Key         string `json:"key"`
	Template    string `json:"template"`
	Controllers string `json:"controllers"`
	Privilege   string `json:"privilege,omitempty"`
}

type Route struct {
	Route      string      `json:"route"`
	Admin      RouteConfig `json:"admin"`
	Authorized RouteConfig `json:"authorized"`
	RouteConfig
}

type AddedRoute struct {
	Pattern *regexp.Regexp
	Handler func(c *wsrooms.Conn, msg *wsrooms.Message, matches []string)
}

type CompiledRoute struct {
	Pattern *regexp.Regexp
	Config  Route
}

type RoutePayload struct {
	Template    string   `json:"template"`
	Controllers []string `json:"controllers"`
}

var (
	keyCleanRegex = regexp.MustCompile(`[^a-z0-9_\-\s]+`)
	reservedPath  = regexp.MustCompile(`^/(ws|login|register|logout|static/|favicon\.ico)`)
)

func ToKey(s string) string {
	s = strings.ToLower(s)
	s = keyCleanRegex.ReplaceAllString(s, "")
	s = strings.Replace(s, " - ", "_-_", -1)
	s = strings.Replace(s, " ", "-", -1)
	return strings.Trim(s, "-")
}

func FromKey(s string) string {
	s = strings.Replace(s, "-", " ", -1)
	s = strings.Replace(s, "_ _", " - ", -1)
	return strings.Title(s)
}

var TemplateFuncs = template.FuncMap{
	"unescaped": func(x string) interface{} { return template.HTML(x) },
	"sha1sum":   func(x string) string { return fmt.Sprintf("%x", sha1.Sum([]byte(x))) },
	"subtract":  func(a, b int) int { return a - b },
	"add":       func(a, b int) int { return a + b },
	"multiply":  func(a, b int) int { return a * b },
	"divide":    func(a, b int) int { return a / b },
	"usd":       func(x int) string { return fmt.Sprintf("$%.2f", float64(x)/100) },
	"css":       func(s string) template.CSS { return template.CSS(s) },
	"tokey":     ToKey,
	"fromkey":   FromKey,
}

func (app *App) setupRoutes() error {
	app.Router.HandleFunc("/login", app.login).Methods("POST")
	app.Router.HandleFunc("/register", app.register).Methods("POST")
	app.Router.HandleFunc("/logout", app.logout).Methods("POST")
	app.Router.HandleFunc("/ws", wsrooms.SocketHandler(app.ReadCookie)).Methods("GET")
	app.Router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	app.Router.PathPrefix("/").HandlerFunc(app.baseHandler).Methods("GET")

	if err := app.compileRoutes(); err != nil {
		return fmt.Errorf("compile routes: %w", err)
	}
	return nil
}

func (app *App) compileRoutes() error {
	compiled := make([]CompiledRoute, 0, len(app.Routes))
	for _, r := range app.Routes {
		patternStr := r.Route
		if !strings.HasPrefix(patternStr, "^") {
			patternStr = "^" + patternStr
		}
		if !strings.HasSuffix(patternStr, "$") {
			patternStr += "$"
		}
		re, err := regexp.Compile(patternStr)
		if err != nil {
			return fmt.Errorf("invalid route pattern %q: %w", r.Route, err)
		}
		compiled = append(compiled, CompiledRoute{
			Pattern: re,
			Config:  r,
		})
	}
	app.CompiledRoutes = compiled
	return nil
}

func (app *App) AddRoute(pattern string, handler func(c *wsrooms.Conn, msg *wsrooms.Message, matches []string)) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid route pattern %q: %w", pattern, err)
	}
	app.Added = append(app.Added, AddedRoute{
		Pattern: re,
		Handler: handler,
	})
	return nil
}

func (app *App) baseHandler(w http.ResponseWriter, r *http.Request) {
	if reservedPath.MatchString(r.URL.Path) {
		http.NotFound(w, r)
		return
	}
	cook := app.ReadCookie(r)
	if err := app.Templates.ExecuteTemplate(w, "base", cook); err != nil {
		log.Printf("Template error in baseHandler: %v", err)
		http.Error(w, "Render failed", http.StatusInternalServerError)
	}
}

func (app *App) Render(c *wsrooms.Conn, msg *wsrooms.Message, tmpl string, controllers []string, data interface{}) {
	var buf bytes.Buffer
	if err := app.Templates.ExecuteTemplate(&buf, tmpl, data); err != nil {
		log.Printf("Render error (%s): %v", tmpl, err)
		return
	}

	cleanCtrls := make([]string, 0, len(controllers))
	for _, ctrl := range controllers {
		if trimmed := strings.TrimSpace(ctrl); trimmed != "" {
			cleanCtrls = append(cleanCtrls, trimmed)
		}
	}

	resp := RoutePayload{
		Template:    buf.String(),
		Controllers: cleanCtrls,
	}

	payload, err := json.Marshal(resp)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}

	msg.Event = "response"
	msg.EventLength = len(msg.Event)
	msg.Payload = payload
	msg.PayloadLength = len(payload)
	c.Send <- msg.Bytes()
}

func resolveDynamic(field string, subs []string) string {
	if !strings.HasPrefix(field, "$") {
		return field
	}
	if n, err := strconv.Atoi(field[1:]); err == nil && n >= 0 && n < len(subs) {
		return subs[n]
	}
	return ""
}

func (app *App) processRequest(c *wsrooms.Conn, msg *wsrooms.Message) {
	path := string(msg.Payload)

	for _, added := range app.Added {
		if subs := added.Pattern.FindStringSubmatch(path); subs != nil {
			added.Handler(c, msg, subs)
			return
		}
	}

	for _, cr := range app.CompiledRoutes {
		if subs := cr.Pattern.FindStringSubmatch(path); subs != nil {
			route := cr.Config
			cfg := route.RouteConfig
			priv, _ := c.Cookie["privilege"]

			if priv == "admin" && (route.Admin.Template != "" || route.Admin.Controllers != "") {
				cfg = route.Admin
			} else if priv != "" && route.Authorized.Privilege != "" {
				for _, allowed := range strings.Split(route.Authorized.Privilege, ",") {
					if strings.TrimSpace(allowed) == priv {
						cfg = route.Authorized
						break
					}
				}
			}

			table := resolveDynamic(cfg.Table, subs)
			key := resolveDynamic(cfg.Key, subs)

			if table != "" && !IsValidTableName(table) {
				log.Printf("Blocked invalid table %q in route", table)
				return
			}

			var data interface{}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var errDB error
			if table != "" {
				if key != "" {
					data, errDB = app.GetRow(ctx, table, key)
				} else {
					data, errDB = app.GetRows(ctx, table)
				}
				if errDB != nil {
					log.Printf("DB error for %s: %v", path, errDB)
					//	return
				}
			}

			controllers := strings.Split(cfg.Controllers, ",")
			app.Render(c, msg, cfg.Template, controllers, data)
			return
		}
	}

	log.Printf("No route matched: %s", path)
}
