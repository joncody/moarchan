package wsframe

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/joncody/wsrooms"
)

type RouteConfig struct {
	Table       string `json:"table"`
	Key         string `json:"key"`
	Template    string `json:"template"`
	Controllers string `json:"controllers"`
	Privilege   string `json:"privilege,omitempty"` // Only used for Authorized routes
}

type Route struct {
	Route      string      `json:"route"`
	Admin      RouteConfig `json:"admin"`
	Authorized RouteConfig `json:"authorized"`
	RouteConfig
}

type AddedRoute struct {
	Route   string
	Handler func(c *wsrooms.Conn, msg *wsrooms.Message, matches []string)
}

type RoutePayload struct {
	Template    string   `json:"template"`
	Controllers []string `json:"controllers"`
}

func ToKey(s string) string {
	respecial := regexp.MustCompile(`([^a-z0-9_\-\s]+)`)
	s = strings.ToLower(s)
	s = respecial.ReplaceAllString(s, "")
	s = strings.Replace(s, " - ", "_-_", -1)
	s = strings.Replace(s, " ", "-", -1)
	return strings.Trim(s, "-")
}

func FromKey(s string) string {
	s = strings.Replace(s, "-", " ", -1)
	s = strings.Replace(s, "_ _", " - ", -1)
	s = strings.Title(s)
	return strings.Trim(s, " ")
}

var TemplateFuncs = template.FuncMap{
	"unescaped": func(x string) interface{} {
		return template.HTML(x)
	},
	"sha1sum": func(x string) string {
		return fmt.Sprintf("%x", sha1.Sum([]byte(x)))
	},
	"subtract": func(a, b int) int {
		return a - b
	},
	"add": func(a, b int) int {
		return a + b
	},
	"multiply": func(a, b int) int {
		return a * b
	},
	"divide": func(a, b int) int {
		return a / b
	},
	"usd": func(x int) string {
		return fmt.Sprintf("$%.2f", float64(x)/float64(100))
	},
	"css": func(s string) template.CSS {
		return template.CSS(s)
	},
	"tokey":   ToKey,
	"fromkey": FromKey,
}

func (app *App) setupRoutes() {
	app.Router.HandleFunc("/login", app.login).Methods("POST")
	app.Router.HandleFunc("/register", app.register).Methods("POST")
	app.Router.HandleFunc("/logout", app.logout).Methods("POST")
	app.Router.HandleFunc("/ws", wsrooms.SocketHandler(app.ReadCookie)).Methods("GET")
	app.Router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	app.Router.PathPrefix("/").Handler(http.HandlerFunc(app.baseHandler)).Methods("GET")
}

func (wfa *App) AddRoute(path string, handler func(c *wsrooms.Conn, msg *wsrooms.Message, matches []string)) {
	wfa.Added = append(wfa.Added, AddedRoute{Route: path, Handler: handler})
}

func (wfa *App) baseHandler(w http.ResponseWriter, r *http.Request) {
	cook := wfa.ReadCookie(r)
	if err := wfa.Templates.ExecuteTemplate(w, "base", cook); err != nil {
		log.Println("Base handler template error:", err)
	}
}

func (wfa *App) Render(c *wsrooms.Conn, msg *wsrooms.Message, template string, controllers []string, data interface{}) {
	var tpl bytes.Buffer

	if err := wfa.Templates.ExecuteTemplate(&tpl, template, data); err != nil {
        log.Println("Template error:", err)
	}
	resp := RoutePayload{
		Template:    tpl.String(),
		Controllers: controllers,
	}
	payload, err := json.Marshal(&resp)
	if err != nil {
        log.Println("Marshal error:", err)
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
	if n, err := strconv.Atoi(field[1:]); err == nil && n < len(subs) {
		return subs[n]
	}
	return ""
}

func (wfa *App) processRequest(c *wsrooms.Conn, msg *wsrooms.Message) {
	var data interface{}
	path := string(msg.Payload)
	for _, added := range wfa.Added {
		if pattern := regexp.MustCompile(added.Route); pattern.MatchString(path) {
			subs := pattern.FindStringSubmatch(path)
			added.Handler(c, msg, subs)
			return
		}
	}
	for _, route := range wfa.Routes {
		pattern := regexp.MustCompile(route.Route)
		subs := pattern.FindStringSubmatch(path)
		if subs == nil {
			continue
		}

		cfg := route.RouteConfig
		priv := c.Cookie["privilege"]

		switch {
		case priv == "admin" && (route.Admin.Template != "" || route.Admin.Controllers != ""):
			cfg = route.Admin
		case priv != "" && route.Authorized.Privilege != "" && strings.Contains(route.Authorized.Privilege, priv):
			cfg = route.Authorized
		}

		table := resolveDynamic(cfg.Table, subs)
		key := resolveDynamic(cfg.Key, subs)

		if table != "" {
			if key != "" {
				data = wfa.GetRow(table, key)
			} else {
				data = wfa.GetRows(table)
			}
		}

		controllers := strings.Split(cfg.Controllers, ",")
		wfa.Render(c, msg, cfg.Template, controllers, data)
		return
	}
}
