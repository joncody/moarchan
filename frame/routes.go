// Package frame provides a decoupled web framework featuring session management,
// dynamic routing, database KV helpers, and Server-Sent Events (SSE).
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
)

var (
	// ErrUnauthorized indicates missing authentication credentials.
	ErrUnauthorized = errors.New("401 Unauthorized: authentication required")
	// ErrForbidden indicates insufficient user privileges for the requested route.
	ErrForbidden = errors.New("403 Forbidden: insufficient privilege")
)

// RouteConfig specifies data bindings, HTML templates, and JS controllers for a route tier.
type RouteConfig struct {
	Table       string
	Key         string
	Template    string
	Controllers string
	Privilege   string
}

// Route represents a declared URL pattern alongside role-based route configurations.
type Route struct {
	Route      string
	Admin      RouteConfig
	Authorized RouteConfig
	RouteConfig
}

// AddedRoute represents a custom regex pattern mapped to a procedural HTTP handler.
type AddedRoute struct {
	Pattern *regexp.Regexp
	Handler func(app *App, w http.ResponseWriter, r *http.Request, matches []string)
}

// CompiledRoute is a compiled regular expression paired with its route definition.
type CompiledRoute struct {
	Pattern *regexp.Regexp
	Config  Route
}

// RoutePayload represents the JSON payload delivered to the SPA client on page transitions.
type RoutePayload struct {
	Template    string   `json:"template"`
	Controllers []string `json:"controllers"`
}

// DataProvider resolves dynamic application data for matched routes before rendering.
type DataProvider func(ctx context.Context, app *App, cfg RouteConfig, subs []string) (interface{}, error)

// RouteBuilder provides a fluent interface for configuring programmatic routes.
type RouteBuilder struct {
	route *Route
}

// Route initializes a new fluent route definition for the given URL regex pattern.
func (app *App) Route(pattern string) *RouteBuilder {
	r := Route{
		Route: pattern,
	}
	app.Routes = append(app.Routes, r)
	return &RouteBuilder{
		route: &app.Routes[len(app.Routes)-1],
	}
}

// Template binds an HTML template name to the route.
func (b *RouteBuilder) Template(tmpl string) *RouteBuilder {
	b.route.RouteConfig.Template = tmpl
	return b
}

// Controller binds one or more JavaScript client controllers to the route.
func (b *RouteBuilder) Controller(ctrls ...string) *RouteBuilder {
	b.route.RouteConfig.Controllers = strings.Join(ctrls, ",")
	return b
}

// Table binds a database table or capture token (e.g. "$1") to the route.
func (b *RouteBuilder) Table(table string) *RouteBuilder {
	b.route.RouteConfig.Table = table
	return b
}

// Key binds a lookup key or capture token (e.g. "$2") to the route.
func (b *RouteBuilder) Key(key string) *RouteBuilder {
	b.route.RouteConfig.Key = key
	return b
}

// Privilege sets the required access tier for the route.
func (b *RouteBuilder) Privilege(priv string) *RouteBuilder {
	b.route.RouteConfig.Privilege = priv
	return b
}

var (
	// keyCleanRegex strips characters not permitted in URL keys.
	keyCleanRegex = regexp.MustCompile(`[^a-z0-9_\-\s]+`)
	// reservedPath matches static assets and API routes that bypass the SPA root handler.
	reservedPath = regexp.MustCompile(`^/(api/|login|register|logout|static/|favicon\.ico)`)
)

// ToKey converts a human-readable title into a clean URL slug.
//
// Example:
//
//	ToKey("Anime & Manga - General") -> "anime-manga_-_general"
func ToKey(s string) string {
	s = strings.ToLower(s)
	s = keyCleanRegex.ReplaceAllString(s, "")
	s = strings.Replace(s, " - ", "_-_", -1)
	s = strings.Replace(s, " ", "-", -1)
	return strings.Trim(s, "-")
}

// titleCaseWords capitalizes the initial letter of each word in a string.
func titleCaseWords(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// FromKey converts a URL slug back into a title-cased display title.
//
// Example:
//
//	FromKey("anime-manga_-_general") -> "Anime Manga - General"
func FromKey(s string) string {
	s = strings.Replace(s, "_-_", " __SEPARATOR__ ", -1)
	s = strings.Replace(s, "-", " ", -1)
	s = strings.Replace(s, " __SEPARATOR__ ", " - ", -1)
	return titleCaseWords(s)
}

// TemplateFuncs defines helper functions available within Go HTML templates.
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

// CompileRoutes compiles all declared route patterns into anchored regular expressions.
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

// SetupRoutes builds the standard HTTP ServeMux routing tree.
func (app *App) SetupRoutes() (*http.ServeMux, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /login", app.Login)
	mux.HandleFunc("POST /register", app.Register)
	mux.HandleFunc("POST /logout", app.Logout)
	mux.HandleFunc("GET /api/render", app.RenderHandler)
	mux.HandleFunc("GET /api/stream", app.SSEHandler)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	mux.HandleFunc("GET /", app.baseHandler)

	return mux, nil
}

// AddRoute registers an imperatively handled route with regex parameter matching.
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

// baseHandler renders the top-level HTML base template containing the SPA shell.
func (app *App) baseHandler(w http.ResponseWriter, r *http.Request) {
	if reservedPath.MatchString(r.URL.Path) {
		http.NotFound(w, r)
		return
	}

	data, err := app.GetSessionValues(r)
	if err != nil {
		data = make(map[string]string)
	}

	// Inject CSRF token into template context
	data["csrf_token"] = GetCSRFToken(r)

	if err := app.Templates.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("Template error in baseHandler: %v", err)
		http.Error(w, "Render failed", http.StatusInternalServerError)
	}
}

// resolveDynamic resolves positional capture tokens (e.g., "$1") against URL submatches.
func resolveDynamic(field string, subs []string) string {
	if !strings.HasPrefix(field, "$") {
		return field
	}
	if n, err := strconv.Atoi(field[1:]); err == nil && n >= 0 && n < len(subs) {
		return subs[n]
	}
	return ""
}

// matchAddedRoute attempts to dispatch the request to any manually registered AddedRoutes.
func (app *App) matchAddedRoute(w http.ResponseWriter, r *http.Request, path string) bool {
	for _, added := range app.Added {
		if subs := added.Pattern.FindStringSubmatch(path); subs != nil {
			added.Handler(app, w, r, subs)
			return true
		}
	}
	return false
}

// MatchCompiledRoute tests a request path against compiled regular expression routes.
func (app *App) MatchCompiledRoute(path string) (*CompiledRoute, []string) {
	for _, cr := range app.CompiledRoutes {
		if subs := cr.Pattern.FindStringSubmatch(path); subs != nil {
			return &cr, subs
		}
	}
	return nil, nil
}

// SelectRouteConfig determines the appropriate route configuration based on user privilege.
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

// ResolveRouteData invokes the application DataProvider or fallback KV store to populate template data.
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

// RenderHandler serves dynamic HTML templates and controllers as JSON for client-side SPA navigation.
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

	templateData := map[string]interface{}{
		"alias":      sessionValues["alias"],
		"privilege":  sessionValues["privilege"],
		"csrf_token": GetCSRFToken(r),
	}
	if data != nil {
		if dataMap, ok := data.(map[string]interface{}); ok {
			for k, v := range dataMap {
				templateData[k] = v
			}
		} else if dataSlice, ok := data.([]map[string]interface{}); ok {
			templateData["threads"] = dataSlice
			templateData["data"] = dataSlice
		} else {
			templateData["data"] = data
		}
	}

	var buf bytes.Buffer
	if err := app.Templates.ExecuteTemplate(&buf, cfg.Template, templateData); err != nil {
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
