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

// Pre-computed dummy hash to enforce constant-time execution during lookup failures.
const dummyHash = "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUU123456"

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

// IsAliasAvailable returns true if the alias is not taken.
func (app *App) IsAliasAvailable(ctx context.Context, alias string) (bool, error) {
	var exists bool
	err := app.Driver.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM auth WHERE key = $1)`, alias).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check alias existence: %w", err)
	}
	return !exists, nil
}

// CreateUser hashes the password and inserts a new user into the auth table.
func (app *App) CreateUser(ctx context.Context, alias, password string) (*Auth, error) {
	if len(password) < 8 || len(password) > 72 {
		return nil, errors.New("password must be between 8 and 72 characters")
	}

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

	// Directly insert and rely on unique constraints to prevent TOCTOU race conditions.
	_, err = app.Driver.ExecContext(ctx, `INSERT INTO auth (key, value) VALUES ($1, $2)`, alias, data)
	if err != nil {
		return nil, fmt.Errorf("user already exists or database error: %w", err)
	}
	return auth, nil
}

// VerifyCredentials validates alias/password using constant-time checks to prevent user enumeration.
func (app *App) VerifyCredentials(ctx context.Context, alias, password string) (*Auth, error) {
	if len(password) == 0 || len(password) > 72 {
		return nil, errors.New("invalid alias or password")
	}

	var data []byte
	err := app.Driver.QueryRowContext(ctx, `SELECT value FROM auth WHERE key = $1`, alias).Scan(&data)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("lookup user: %w", err)
	}

	var auth Auth
	targetHash := dummyHash

	if err == nil {
		if jsonErr := json.Unmarshal(data, &auth); jsonErr == nil {
			targetHash = auth.PasswordHash
		}
	}

	// Always execute CompareHashAndPassword to maintain constant execution time.
	bcryptErr := bcrypt.CompareHashAndPassword([]byte(targetHash), []byte(password))

	if err != nil || bcryptErr != nil {
		return nil, errors.New("invalid alias or password")
	}

	return &auth, nil
}

// ———————— HTTP Handlers ————————

func (app *App) Register(w http.ResponseWriter, r *http.Request) {
	alias := strings.TrimSpace(r.FormValue("alias"))
	password := r.FormValue("password")
	if !validateAlias(alias) || len(password) < 8 || len(password) > 72 {
		http.Error(w, "Invalid alias format or password length (8-72 chars required)", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	auth, err := app.CreateUser(ctx, alias, password)
	if err != nil {
		log.Printf("Registration failed for %q: %v", alias, err)
		http.Error(w, "Alias taken or registration unavailable", http.StatusConflict)
		return
	}
	if err := app.SetSession(w, r, alias, auth.Privilege); err != nil {
		log.Printf("Session setup failed for %q: %v", alias, err)
		http.Error(w, "Login failed", http.StatusInternalServerError)
		return
	}
}

func (app *App) Login(w http.ResponseWriter, r *http.Request) {
	alias := strings.TrimSpace(r.FormValue("alias"))
	password := r.FormValue("password")
	if alias == "" || password == "" {
		http.Error(w, "Missing credentials", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	auth, err := app.VerifyCredentials(ctx, alias, password)
	if err != nil {
		log.Printf("Login failed for alias %q: %v", alias, err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	if err := app.SetSession(w, r, alias, auth.Privilege); err != nil {
		log.Printf("Session setup failed for %q: %v", alias, err)
		http.Error(w, "Login failed", http.StatusInternalServerError)
		return
	}
}

func (app *App) Logout(w http.ResponseWriter, r *http.Request) {
	if err := app.clearSession(w, r); err != nil {
		log.Printf("Session clear error: %v", err)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
