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
	"github.com/joncody/roomer"
)

var (
	ErrUnauthorized = errors.New("401 Unauthorized: authentication required")
	ErrForbidden    = errors.New("403 Forbidden: insufficient privilege")
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
	Handler func(app *App, c *roomer.Conn, msg *roomer.Message, matches []string)
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
	router.HandleFunc("/login", app.Login).Methods("POST")
	router.HandleFunc("/register", app.Register).Methods("POST")
	router.HandleFunc("/logout", app.Logout).Methods("POST")
	router.HandleFunc("/ws", roomer.SocketHandlerWithOptions(
		roomer.WithAuthorize(app.GetSessionValues),
		roomer.WithMaxMessageSize(32*1024*1024), // 32MB max message limit
		roomer.WithBufferSizes(16384, 16384),
	)).Methods("GET")
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router.PathPrefix("/").HandlerFunc(app.baseHandler).Methods("GET")
	if err := app.CompileRoutes(); err != nil {
		return nil, fmt.Errorf("compile routes: %w", err)
	}
	return router, nil
}

func (app *App) AddRoute(pattern string, handler func(app *App, c *roomer.Conn, msg *roomer.Message, matches []string)) error {
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

func (app *App) Render(c *roomer.Conn, msg *roomer.Message, tmpl string, controllers []string, data interface{}) {
	var buf bytes.Buffer
	if err := app.Templates.ExecuteTemplate(&buf, tmpl, data); err != nil {
		log.Printf("Render error (%s): %v", tmpl, err)
		app.sendErrorResponse(c, msg, http.StatusInternalServerError, "Internal template render error")
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
		app.sendErrorResponse(c, msg, http.StatusInternalServerError, "Internal response serialization error")
		return
	}
	c.SendToClient(c.ID, "response", payload)
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

func (app *App) matchAddedRoute(c *roomer.Conn, msg *roomer.Message, path string) bool {
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

func SelectRouteConfig(route Route, privilege string) (RouteConfig, error) {
	// Admin override
	if route.Admin.Template != "" || route.Admin.Controllers != "" {
		if privilege == "admin" {
			return route.Admin, nil
		}
		if privilege == "" {
			return RouteConfig{}, ErrUnauthorized
		}
		return RouteConfig{}, ErrForbidden
	}

	// Authorized users
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

	// Default public config
	return route.RouteConfig, nil
}

func (app *App) ResolveRouteData(ctx context.Context, cfg RouteConfig, subs []string) (interface{}, error) {
	table := resolveDynamic(cfg.Table, subs)
	key := resolveDynamic(cfg.Key, subs)
	if table == "" {
		return nil, nil
	}
	if IsSystemTable(table) {
		return nil, fmt.Errorf("access to system table %q via dynamic route is restricted", table)
	}
	if !IsValidTableName(table) {
		return nil, fmt.Errorf("invalid table name: %q", table)
	}
	if key != "" {
		return app.GetRow(ctx, table, key)
	}
	return app.GetRows(ctx, table)
}

func (app *App) sendErrorResponse(c *roomer.Conn, msg *roomer.Message, status int, text string) {
	resp := map[string]interface{}{
		"error":  text,
		"status": status,
	}
	payload, _ := json.Marshal(resp)
	c.SendToClient(c.ID, "error", payload)
}

func (app *App) ProcessRequest(c *roomer.Conn, msg *roomer.Message) error {
	path := strings.TrimSpace(string(msg.Payload))

	// 1. Added routes (custom handlers)
	if app.matchAddedRoute(c, msg, path) {
		return nil
	}

	// 2. Match configured routes
	cr, subs := app.MatchCompiledRoute(path)
	if cr == nil {
		app.sendErrorResponse(c, msg, http.StatusNotFound, fmt.Sprintf("No route matched: %s", path))
		return fmt.Errorf("no route matched: %s", path)
	}

	// 3. Select route config by privilege
	privilege := c.Claims["privilege"]
	cfg, err := SelectRouteConfig(cr.Config, privilege)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			app.sendErrorResponse(c, msg, http.StatusUnauthorized, "Authentication required")
		} else {
			app.sendErrorResponse(c, msg, http.StatusForbidden, "Access denied")
		}
		return fmt.Errorf("route permission error (%s): %w", path, err)
	}

	// 4. Load data (if any)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	data, err := app.ResolveRouteData(ctx, cfg, subs)
	if err != nil {
		log.Printf("Route data error (%s): %v", path, err)
		app.sendErrorResponse(c, msg, http.StatusNotFound, fmt.Sprintf("Data not found for route: %s", path))
		return fmt.Errorf("data error (%s): %w", path, err)
	}

	// 5. Render response
	controllers := strings.Split(cfg.Controllers, ",")
	app.Render(c, msg, cfg.Template, controllers, data)
	return nil
}
