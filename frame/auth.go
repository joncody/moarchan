// auth.go
package frame

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Auth struct {
	Alias     string `json:"alias,omitempty"`
	Passhash  string `json:"passhash"`
	Salt      string `json:"salt"`
	Hash      string `json:"hash"`
	Privilege string `json:"privilege"`
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
	var err error

	if logout || value == nil {
		encoded = ""
	} else {
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
		cookie.Expires = time.Unix(0, 0)
		cookie.MaxAge = -1
	} else {
		cookie.MaxAge = 60 * 60 * 24
		cookie.Expires = time.Now().Add(24 * time.Hour)
	}

	http.SetCookie(w, cookie)
}

func validateAlias(alias string) bool {
	alias = strings.TrimSpace(alias)
	return len(alias) >= 3 && len(alias) <= 64 && !strings.ContainsAny(alias, " <>@#$%^&*()+=[]{}|\\:;\"'?,./")
}

func (app *App) register(w http.ResponseWriter, r *http.Request) {
	alias := strings.TrimSpace(r.FormValue("alias"))
	passhash := r.FormValue("passhash")

	if !validateAlias(alias) || passhash == "" {
		http.Error(w, "Invalid alias or password", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var existing []byte
	err := app.Driver.QueryRowContext(ctx, `SELECT value FROM auth WHERE key = $1`, alias).Scan(&existing)
	if err == nil {
		http.Error(w, "Alias already taken", http.StatusConflict)
		return
	}
	if err != nil && !isErrNoRows(err) {
		log.Printf("DB error checking alias %q: %v", alias, err)
		http.Error(w, "Registration failed", http.StatusInternalServerError)
		return
	}

	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		log.Printf("Salt generation failed: %v", err)
		http.Error(w, "Registration failed", http.StatusInternalServerError)
		return
	}
	salt := fmt.Sprintf("%x", sha1.Sum(random))
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(alias+passhash+salt)))

	auth := Auth{
		Passhash:  passhash,
		Salt:      salt,
		Hash:      hash,
		Privilege: "user",
	}

	data, err := json.Marshal(auth)
	if err != nil {
		log.Printf("Marshal error: %v", err)
		http.Error(w, "Registration failed", http.StatusInternalServerError)
		return
	}

	_, err = app.Driver.ExecContext(ctx, `INSERT INTO auth (key, value) VALUES ($1, $2)`, alias, data)
	if err != nil {
		log.Printf("Insert error for %q: %v", alias, err)
		http.Error(w, "Registration failed", http.StatusInternalServerError)
		return
	}

	app.SetCookie(w, r, map[string]string{
		"alias":     alias,
		"privilege": auth.Privilege,
	}, false)

	w.WriteHeader(http.StatusOK)
}

func (app *App) login(w http.ResponseWriter, r *http.Request) {
	alias := strings.TrimSpace(r.FormValue("alias"))
	passhash := r.FormValue("passhash")

	if alias == "" || passhash == "" {
		http.Error(w, "Missing credentials", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var data []byte
	err := app.Driver.QueryRowContext(ctx, `SELECT value FROM auth WHERE key = $1`, alias).Scan(&data)
	if err != nil {
		if isErrNoRows(err) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		} else {
			log.Printf("DB error during login for %q: %v", alias, err)
			http.Error(w, "Login failed", http.StatusInternalServerError)
		}
		return
	}

	var auth Auth
	if err := json.Unmarshal(data, &auth); err != nil {
		log.Printf("Unmarshal error for %q: %v", alias, err)
		http.Error(w, "Login failed", http.StatusInternalServerError)
		return
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(alias+passhash+auth.Salt)))
	if hash != auth.Hash {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	app.SetCookie(w, r, map[string]string{
		"alias":     alias,
		"privilege": auth.Privilege,
	}, false)

	w.WriteHeader(http.StatusOK)
}

func (app *App) logout(w http.ResponseWriter, r *http.Request) {
	app.SetCookie(w, r, nil, true)
	w.WriteHeader(http.StatusOK)
}

func isErrNoRows(err error) bool {
	return err != nil && err == sql.ErrNoRows
}
