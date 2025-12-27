package frame

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

const privilegeUser = "user"

// validateAlias checks if an alias meets format requirements.
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

// isAliasAvailable returns true if the alias is not taken.
func (app *App) isAliasAvailable(ctx context.Context, alias string) (bool, error) {
	var exists bool
	err := app.Driver.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM auth WHERE key = $1)`, alias).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check alias existence: %w", err)
	}
	return !exists, nil
}

// createUser hashes the password and inserts a new user into the auth table.
func (app *App) createUser(ctx context.Context, alias, password string) (*Auth, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	auth := &Auth{
		PasswordHash: string(hashed),
		Privilege:    privilegeUser,
	}
	data, err := json.Marshal(auth)
	if err != nil {
		return nil, fmt.Errorf("marshal auth data: %w", err)
	}
	_, err = app.Driver.ExecContext(ctx, `INSERT INTO auth (key, value) VALUES ($1, $2)`, alias, data)
	if err != nil {
		return nil, fmt.Errorf("insert user record: %w", err)
	}
	return auth, nil
}

// verifyCredentials validates alias/password and returns the user auth data on success.
func (app *App) verifyCredentials(ctx context.Context, alias, password string) (*Auth, error) {
	var data []byte
	err := app.Driver.QueryRowContext(ctx, `SELECT value FROM auth WHERE key = $1`, alias).Scan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid alias or password")
		}
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	var auth Auth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, fmt.Errorf("corrupt auth data for alias %q", alias)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(auth.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid alias or password")
	}
	return &auth, nil
}

// ———————— HTTP Handlers ————————

func (app *App) register(w http.ResponseWriter, r *http.Request) {
	alias := strings.TrimSpace(r.FormValue("alias"))
	password := r.FormValue("password")
	if !validateAlias(alias) || password == "" {
		http.Error(w, "Invalid alias or password", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	available, err := app.isAliasAvailable(ctx, alias)
	if err != nil {
		log.Printf("Registration DB error: %v", err)
		http.Error(w, "Registration unavailable", http.StatusInternalServerError)
		return
	}
	if !available {
		http.Error(w, "Alias already taken", http.StatusConflict)
		return
	}
	auth, err := app.createUser(ctx, alias, password)
	if err != nil {
		log.Printf("Failed to create user %q: %v", alias, err)
		http.Error(w, "Registration unavailable", http.StatusInternalServerError)
		return
	}
	if err := app.setSession(w, r, alias, auth.Privilege); err != nil {
		log.Printf("Session setup failed for %q: %v", alias, err)
		http.Error(w, "Login failed", http.StatusInternalServerError)
		return
	}
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
	auth, err := app.verifyCredentials(ctx, alias, password)
	if err != nil {
		log.Printf("Login failed for %q: %v", alias, err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	if err := app.setSession(w, r, alias, auth.Privilege); err != nil {
		log.Printf("Session setup failed for %q: %v", alias, err)
		http.Error(w, "Login failed", http.StatusInternalServerError)
		return
	}
}

func (app *App) logout(w http.ResponseWriter, r *http.Request) {
	if err := app.clearSession(w, r); err != nil {
		log.Printf("Session clear error: %v", err)
		// Proceed anyway — user should be logged out from perspective
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
