package frame

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	PasswordHash string `json:"password_hash"`
	Privilege    string `json:"privilege"`
}

const (
	PrivilegeUser = "user"
	sessionTTL    = 24 * time.Hour
)

func validateAlias(alias string) bool {
	if len(alias) < 3 || len(alias) > 64 {
		return false
	}
	for _, r := range alias {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

func (app *App) ReadCookie(r *http.Request) map[string]string {
	cookie, err := r.Cookie(app.Name)
	if err != nil {
		return nil
	}
	var value map[string]string
	if err := app.SecureCookie.Decode(app.Name, cookie.Value, &value); err != nil {
		log.Printf("Cookie decode error: %v", err)
		return nil
	}
	return value
}

func (app *App) SetCookie(w http.ResponseWriter, r *http.Request, value map[string]string, logout bool) {
	var encoded string
	if !logout && value != nil {
		var err error
		encoded, err = app.SecureCookie.Encode(app.Name, value)
		if err != nil {
			log.Printf("Cookie encode error: %v", err)
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}
	}

	cookie := &http.Cookie{
		Name:     app.Name,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	}

	if logout || value == nil {
		cookie.MaxAge = -1
		cookie.Expires = time.Now().Add(-24 * time.Hour)
	} else {
		cookie.MaxAge = int(sessionTTL.Seconds())
		cookie.Expires = time.Now().Add(sessionTTL)
	}

	http.SetCookie(w, cookie)
}

func (app *App) register(w http.ResponseWriter, r *http.Request) {
	alias := strings.TrimSpace(r.FormValue("alias"))
	password := r.FormValue("password")

	if !validateAlias(alias) || password == "" {
		http.Error(w, "Invalid alias or password", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var exists bool
	err := app.Driver.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM auth WHERE key = $1)`, alias).Scan(&exists)
	if err != nil {
		log.Printf("Database error checking alias existence: %v", err)
		http.Error(w, "Registration unavailable", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "Alias already taken", http.StatusConflict)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Password hashing failed: %v", err)
		http.Error(w, "Registration unavailable", http.StatusInternalServerError)
		return
	}

	auth := Auth{
		PasswordHash: string(hashed),
		Privilege:    PrivilegeUser,
	}

	data, err := json.Marshal(auth)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		http.Error(w, "Registration unavailable", http.StatusInternalServerError)
		return
	}

	_, err = app.Driver.ExecContext(ctx,
		`INSERT INTO auth (key, value) VALUES ($1, $2)`, alias, data)
	if err != nil {
		log.Printf("Database insert error for alias %q: %v", alias, err)
		http.Error(w, "Registration unavailable", http.StatusInternalServerError)
		return
	}

	app.SetCookie(w, r, map[string]string{
		"alias":     alias,
		"privilege": auth.Privilege,
	}, false)
}

func (app *App) login(w http.ResponseWriter, r *http.Request) {
	alias := strings.TrimSpace(r.FormValue("alias"))
	password := r.FormValue("password")

	if alias == "" || password == "" {
		http.Error(w, "Missing credentials", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var data []byte
	err := app.Driver.QueryRowContext(ctx,
		`SELECT value FROM auth WHERE key = $1`, alias).Scan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		} else {
			log.Printf("Database error during login: %v", err)
			http.Error(w, "Login unavailable", http.StatusInternalServerError)
		}
		return
	}

	var auth Auth
	if err := json.Unmarshal(data, &auth); err != nil {
		log.Printf("Failed to unmarshal auth data for alias %q", alias)
		http.Error(w, "Login unavailable", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(auth.PasswordHash), []byte(password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	app.SetCookie(w, r, map[string]string{
		"alias":     alias,
		"privilege": auth.Privilege,
	}, false)
}

func (app *App) logout(w http.ResponseWriter, r *http.Request) {
	app.SetCookie(w, r, nil, true)
}
