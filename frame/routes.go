package frame

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var (
	ErrUnauthorized = errors.New("401 Unauthorized: authentication required")
	ErrForbidden    = errors.New("403 Forbidden: insufficient privilege")
)

type RouteConfig struct {
	Table       string
	Key         string
	Template    string
	Controllers string
	Privilege   string
}

type Route struct {
	Route      string
	Admin      RouteConfig
	Authorized RouteConfig
	RouteConfig
}

type AddedRoute struct {
	Pattern *regexp.Regexp
	Handler func(app *App, w http.ResponseWriter, r *http.Request, matches []string)
}

type CompiledRoute struct {
	Pattern *regexp.Regexp
	Config  Route
}

type RoutePayload struct {
	Template    string   `json:"template"`
	Controllers []string `json:"controllers"`
}

type DataProvider func(ctx context.Context, app *App, cfg RouteConfig, subs []string) (interface{}, error)

// Fluent Route Builder Pattern
type RouteBuilder struct {
	route *Route
}

func (app *App) Route(pattern string) *RouteBuilder {
	r := Route{
		Route: pattern,
	}
	app.Routes = append(app.Routes, r)
	return &RouteBuilder{
		route: &app.Routes[len(app.Routes)-1],
	}
}

func (b *RouteBuilder) Template(tmpl string) *RouteBuilder {
	b.route.RouteConfig.Template = tmpl
	return b
}

func (b *RouteBuilder) Controller(ctrls ...string) *RouteBuilder {
	b.route.RouteConfig.Controllers = strings.Join(ctrls, ",")
	return b
}

func (b *RouteBuilder) Table(table string) *RouteBuilder {
	b.route.RouteConfig.Table = table
	return b
}

func (b *RouteBuilder) Key(key string) *RouteBuilder {
	b.route.RouteConfig.Key = key
	return b
}

func (b *RouteBuilder) Privilege(priv string) *RouteBuilder {
	b.route.RouteConfig.Privilege = priv
	return b
}

var (
	keyCleanRegex = regexp.MustCompile(`[^a-z0-9_\-\s]+`)
	reservedPath  = regexp.MustCompile(`^/(api/|login|register|logout|static/|favicon\.ico)`)
)

func ToKey(s string) string {
	s = strings.ToLower(s)
	s = keyCleanRegex.ReplaceAllString(s, "")
	s = strings.Replace(s, " - ", "_-_", -1)
	s = strings.Replace(s, " ", "-", -1)
	return strings.Trim(s, "-")
}

func titleCaseWords(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func FromKey(s string) string {
	s = strings.Replace(s, "_-_", " __SEPARATOR__ ", -1)
	s = strings.Replace(s, "-", " ", -1)
	s = strings.Replace(s, " __SEPARATOR__ ", " - ", -1)
	return titleCaseWords(s)
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

	router.Use(RecoveryMiddleware)
	router.Use(LoggingMiddleware)
	router.Use(SecurityHeadersMiddleware)

	router.HandleFunc("/login", app.Login).Methods("POST")
	router.HandleFunc("/register", app.Register).Methods("POST")
	router.HandleFunc("/logout", app.Logout).Methods("POST")

	router.HandleFunc("/api/render", app.RenderHandler).Methods("GET")
	router.HandleFunc("/api/stream", app.SSEHandler).Methods("GET")

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router.PathPrefix("/").HandlerFunc(app.baseHandler).Methods("GET")

	return router, nil
}

func (app *App) AddRoute(pattern string, handler func(app *App, w http.ResponseWriter, r *http.Request, matches []string)) error {
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
	data, err := app.GetSessionValues(r)
	if err != nil {
		data = make(map[string]string)
	}
	if err := app.Templates.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("Template error in baseHandler: %v", err)
		http.Error(w, "Render failed", http.StatusInternalServerError)
	}
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

func (app *App) matchAddedRoute(w http.ResponseWriter, r *http.Request, path string) bool {
	for _, added := range app.Added {
		if subs := added.Pattern.FindStringSubmatch(path); subs != nil {
			added.Handler(app, w, r, subs)
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

func SelectRouteConfig(route Route, privilege string) (RouteConfig, error) {
	if route.Admin.Template != "" || route.Admin.Controllers != "" {
		if privilege == "admin" {
			return route.Admin, nil
		}
		if privilege == "" {
			return RouteConfig{}, ErrUnauthorized
		}
		return RouteConfig{}, ErrForbidden
	}

	if route.Authorized.Privilege != "" {
		if privilege == "" {
			return RouteConfig{}, ErrUnauthorized
		}
		for _, allowed := range strings.Split(route.Authorized.Privilege, ",") {
			if strings.TrimSpace(allowed) == privilege {
				return route.Authorized, nil
			}
		}
		return RouteConfig{}, ErrForbidden
	}

	return route.RouteConfig, nil
}

func (app *App) ResolveRouteData(ctx context.Context, cfg RouteConfig, subs []string) (interface{}, error) {
	if app.DataProvider != nil {
		return app.DataProvider(ctx, app, cfg, subs)
	}

	table := resolveDynamic(cfg.Table, subs)
	key := resolveDynamic(cfg.Key, subs)
	if table == "" {
		return nil, nil
	}
	if IsSystemTable(table) {
		return nil, fmt.Errorf("access to system table %q is restricted", table)
	}
	if !IsValidTableName(table) {
		return nil, fmt.Errorf("invalid table name: %q", table)
	}
	if key != "" {
		return app.GetRow(ctx, table, key)
	}
	return app.GetRows(ctx, table)
}

func (app *App) RenderHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		path = "/"
	}

	if app.matchAddedRoute(w, r, path) {
		return
	}

	cr, subs := app.MatchCompiledRoute(path)
	if cr == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  fmt.Sprintf("No route matched: %s", path),
			"status": 404,
		})
		return
	}

	sessionValues, _ := app.GetSessionValues(r)
	privilege := sessionValues["privilege"]

	cfg, err := SelectRouteConfig(cr.Config, privilege)
	if err != nil {
		status := http.StatusForbidden
		if errors.Is(err, ErrUnauthorized) {
			status = http.StatusUnauthorized
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  err.Error(),
			"status": status,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	data, err := app.ResolveRouteData(ctx, cfg, subs)
	if err != nil {
		log.Printf("Route data error (%s): %v", path, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  fmt.Sprintf("Data not found for route: %s", path),
			"status": 404,
		})
		return
	}

	var buf bytes.Buffer
	if err := app.Templates.ExecuteTemplate(&buf, cfg.Template, data); err != nil {
		log.Printf("Render error (%s): %v", cfg.Template, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "Internal template render error",
			"status": 500,
		})
		return
	}

	controllers := strings.Split(cfg.Controllers, ",")
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
