package frame

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
)

const sessionTTL = 24 * time.Hour

func NewSessionStore(name, hashKey, blockKey string, secure bool) (sessions.Store, error) {
	if len(hashKey) < 32 {
		return nil, fmt.Errorf("hash key must be at least 32 bytes (got %d)", len(hashKey))
	}
	if len(blockKey) < 32 {
		return nil, fmt.Errorf("block key must be at least 32 bytes (got %d)", len(blockKey))
	}
	store := sessions.NewCookieStore([]byte(hashKey), []byte(blockKey))
	store.Options = &sessions.Options{
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
		Path:     "/",
	}
	return store, nil
}

func (app *App) GetSessionValues(r *http.Request) (map[string]string, error) {
	session, err := app.SessionStore.Get(r, app.Name)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string)
	if alias, ok := session.Values["alias"].(string); ok && alias != "" {
		values["alias"] = alias
	}
	if priv, ok := session.Values["privilege"].(string); ok && priv != "" {
		values["privilege"] = priv
	}
	log.Println(values)
	return values, nil
}

func (app *App) SetSession(w http.ResponseWriter, r *http.Request, alias, privilege string) error {
	session, err := app.SessionStore.Get(r, app.Name)
	if err != nil {
		return err
	}
	session.Values["alias"] = alias
	session.Values["privilege"] = privilege
	return session.Save(r, w)
}

func (app *App) clearSession(w http.ResponseWriter, r *http.Request) error {
	session, err := app.SessionStore.Get(r, app.Name)
	if err != nil {
		return err
	}
	session.Options.MaxAge = -1
	return session.Save(r, w)
}
