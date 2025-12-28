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

	"github.com/gorilla/mux"
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
	Handler func(app *App, c *wsrooms.Conn, msg *wsrooms.Message, matches []string)
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

func (app *App) CompileRoutes() error {
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

func (app *App) SetupRoutes() (*mux.Router, error) {
	router := mux.NewRouter().StrictSlash(false)
	router.HandleFunc("/login", app.Login).Methods("POST")
	router.HandleFunc("/register", app.Register).Methods("POST")
	router.HandleFunc("/logout", app.Logout).Methods("POST")
	router.HandleFunc("/ws", wsrooms.SocketHandler(app.GetSessionValues)).Methods("GET")
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router.PathPrefix("/").HandlerFunc(app.baseHandler).Methods("GET")
	if err := app.CompileRoutes(); err != nil {
		return nil, fmt.Errorf("compile routes: %w", err)
	}
	return router, nil
}

func (app *App) AddRoute(pattern string, handler func(app *App, c *wsrooms.Conn, msg *wsrooms.Message, matches []string)) error {
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
    data, _ := app.GetSessionValues(r)
    // Render the template
	if err := app.Templates.ExecuteTemplate(w, "base", data); err != nil {
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

func (app *App) matchAddedRoute(c *wsrooms.Conn, msg *wsrooms.Message, path string) bool {
	for _, added := range app.Added {
		if subs := added.Pattern.FindStringSubmatch(path); subs != nil {
			added.Handler(app, c, msg, subs)
			return true
		}
	}
	return false
}

func (app *App) MatchCompiledRoute(path string) (*CompiledRoute, []string) {
	for _, cr := range app.CompiledRoutes {
		if subs := cr.Pattern.FindStringSubmatch(path); subs != nil {
			return &cr, subs
		}
	}
	return nil, nil
}

func SelectRouteConfig(route Route, privilege string) RouteConfig {
	// Admin override
	if privilege == "admin" &&
		(route.Admin.Template != "" || route.Admin.Controllers != "") {
		return route.Admin
	}
	// Authorized users
	if privilege != "" && route.Authorized.Privilege != "" {
		for _, allowed := range strings.Split(route.Authorized.Privilege, ",") {
			if strings.TrimSpace(allowed) == privilege {
				return route.Authorized
			}
		}
	}
	// Default
	return route.RouteConfig
}

func (app *App) ResolveRouteData(ctx context.Context, cfg RouteConfig, subs []string) (interface{}, error) {
	table := resolveDynamic(cfg.Table, subs)
	key := resolveDynamic(cfg.Key, subs)
	if table == "" {
		return nil, nil
	}
	if !IsValidTableName(table) {
		return nil, fmt.Errorf("invalid table name: %q", table)
	}
	if key != "" {
		return app.GetRow(ctx, table, key)
	}
	return app.GetRows(ctx, table)
}

func (app *App) ProcessRequest(c *wsrooms.Conn, msg *wsrooms.Message) {
	path := string(msg.Payload)
	// 1. Added routes (custom handlers)
	if app.matchAddedRoute(c, msg, path) {
		return
	}
	// 2. Match configured routes
	cr, subs := app.MatchCompiledRoute(path)
	if cr == nil {
		log.Printf("No route matched: %s", path)
		return
	}
	// 3. Select route config by privilege
	privilege := c.Cookie["privilege"]
	cfg := SelectRouteConfig(cr.Config, privilege)
	// 4. Load data (if any)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	data, err := app.ResolveRouteData(ctx, cfg, subs)
	if err != nil {
		log.Printf("Route data error (%s): %v", path, err)
		return
	}
	// 5. Render response
	controllers := strings.Split(cfg.Controllers, ",")
	app.Render(c, msg, cfg.Template, controllers, data)
}
