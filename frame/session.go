package frame

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const sessionTTL = 24 * time.Hour

type SessionStore struct {
	name   string
	aead   cipher.AEAD
	secure bool
	maxAge int
}

func NewSessionStore(name, hashKey, blockKey string, secure bool) (*SessionStore, error) {
	if len(hashKey) < 16 || len(blockKey) < 16 {
		return nil, errors.New("session keys must be at least 16 bytes")
	}

	// Derive a consistent 32-byte AES-256 key from blockKey and hashKey
	derivedKey := sha256.Sum256([]byte(blockKey + hashKey))

	block, err := aes.NewCipher(derivedKey[:])
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm aead: %w", err)
	}

	return &SessionStore{
		name:   name,
		aead:   gcm,
		secure: secure,
		maxAge: int(sessionTTL.Seconds()),
	}, nil
}

func (s *SessionStore) Encrypt(values map[string]string) (string, error) {
	data, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal session: %w", err)
	}

	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	sealed := s.aead.Seal(nonce, nonce, data, nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (s *SessionStore) Decrypt(encoded string) (map[string]string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	nonceSize := s.aead.NonceSize()
	if len(raw) < nonceSize {
		return nil, errors.New("session payload too short")
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	data, err := s.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt session: %w", err)
	}

	var values map[string]string
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return values, nil
}

func (app *App) GetSessionValues(r *http.Request) (map[string]string, error) {
	if app.SessionStore == nil {
		return make(map[string]string), nil
	}
	cookie, err := r.Cookie(app.SessionStore.name)
	if err != nil {
		return make(map[string]string), nil
	}
	values, err := app.SessionStore.Decrypt(cookie.Value)
	if err != nil {
		return make(map[string]string), nil
	}
	return values, nil
}

func (app *App) SetSession(w http.ResponseWriter, r *http.Request, alias, privilege string) error {
	if app.SessionStore == nil {
		return errors.New("session store not initialized")
	}
	values := map[string]string{
		"alias":     alias,
		"privilege": privilege,
	}
	encoded, err := app.SessionStore.Encrypt(values)
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     app.SessionStore.name,
		Value:    encoded,
		Path:     "/",
		MaxAge:   app.SessionStore.maxAge,
		HttpOnly: true,
		Secure:   app.SessionStore.secure,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
	return nil
}

func (app *App) clearSession(w http.ResponseWriter, r *http.Request) error {
	if app.SessionStore == nil {
		return errors.New("session store not initialized")
	}
	cookie := &http.Cookie{
		Name:     app.SessionStore.name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   app.SessionStore.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
	}
	http.SetCookie(w, cookie)
	return nil
}
