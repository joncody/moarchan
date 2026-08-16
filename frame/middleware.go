package frame

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterInterceptor) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Flush implements the http.Flusher interface to support SSE over wrapped response writers
func (w *responseWriterInterceptor) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap implements standard Go 1.20+ ResponseWriter unwrapping
func (w *responseWriterInterceptor) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC RECOVERY] %v\n%s", err, string(debug.Stack()))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":  "Internal Server Error",
					"status": 500,
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		interceptor := &responseWriterInterceptor{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(interceptor, r)
		duration := time.Since(start)
		log.Printf("[HTTP] %s %s | %d | %v", r.Method, r.URL.Path, interceptor.statusCode, duration)
	})
}

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}

// ──────────────────────────── Rate Limiting ────────────────────────────

type rateLimitEntry struct {
	tokens    float64
	lastCheck time.Time
}

// RateLimiter implements a token bucket algorithm keyed by client IP.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*rateLimitEntry
	rate     float64 // tokens added per second
	burst    int     // maximum tokens (bucket capacity)
	cleanup  time.Duration
	stopChan chan struct{}
	stopOnce sync.Once
}

func NewRateLimiter(rate float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*rateLimitEntry),
		rate:     rate,
		burst:    burst,
		cleanup:  5 * time.Minute,
		stopChan: make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stopChan:
			return
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-rl.cleanup)
			for ip, entry := range rl.buckets {
				if entry.lastCheck.Before(cutoff) {
					delete(rl.buckets, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.buckets[ip]
	if !exists {
		rl.buckets[ip] = &rateLimitEntry{
			tokens:    float64(rl.burst) - 1,
			lastCheck: now,
		}
		return true
	}

	elapsed := now.Sub(entry.lastCheck).Seconds()
	entry.tokens += elapsed * rl.rate
	if entry.tokens > float64(rl.burst) {
		entry.tokens = float64(rl.burst)
	}
	entry.lastCheck = now

	if entry.tokens < 1 {
		return false
	}
	entry.tokens--
	return true
}

func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.stopChan)
	})
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	if real := r.Header.Get("X-Real-IP"); real != "" {
		return strings.TrimSpace(real)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimitMiddleware returns 429 Too Many Requests when the client exceeds
// the configured rate. Applied only to state-changing methods.
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
				ip := clientIP(r)
				if !limiter.Allow(ip) {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Retry-After", "60")
					w.WriteHeader(http.StatusTooManyRequests)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":  "Rate limit exceeded. Please slow down.",
						"status": 429,
					})
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ──────────────────────────── CSRF Protection ────────────────────────────

type csrfContextKey string

const (
	CSRFTokenContextKey csrfContextKey = "csrf_token"
	csrfCookieName                     = "moarchan_csrf"
	csrfHeaderName                     = "X-CSRF-Token"
	csrfTokenLength                    = 32
)

func generateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate csrf token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// CSRFMiddleware implements the double-submit cookie pattern.
// On GET requests it ensures a CSRF cookie exists and injects the token
// into the request context for template rendering. On state-changing
// requests (POST/PUT/DELETE) it verifies the X-CSRF-Token header matches
// the cookie value using constant-time comparison.
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""

		cookie, err := r.Cookie(csrfCookieName)
		if err == nil && cookie.Value != "" {
			token = cookie.Value
		}

		// Generate a new token if none exists
		if token == "" {
			newToken, genErr := generateCSRFToken()
			if genErr != nil {
				log.Printf("[CSRF] Token generation error: %v", genErr)
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			token = newToken
			http.SetCookie(w, &http.Cookie{
				Name:     csrfCookieName,
				Value:    token,
				Path:     "/",
				MaxAge:   86400,
				HttpOnly: false, // Must be readable by JavaScript
				Secure:   false, // Set true in production with TLS
				SameSite: http.SameSiteLaxMode,
			})
		}

		// Verify on state-changing requests
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
			headerToken := r.Header.Get(csrfHeaderName)
			// Only inspect PostForm if content-type is form-urlencoded to avoid consuming multipart streams
			if headerToken == "" && strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
				headerToken = r.PostFormValue("_csrf")
			}
			if headerToken == "" || subtle.ConstantTimeCompare([]byte(headerToken), []byte(token)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":  "CSRF token validation failed",
					"status": 403,
				})
				return
			}
		}

		// Inject token into context for downstream handlers/templates
		ctx := context.WithValue(r.Context(), CSRFTokenContextKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCSRFToken retrieves the CSRF token from the request context.
func GetCSRFToken(r *http.Request) string {
	if token, ok := r.Context().Value(CSRFTokenContextKey).(string); ok {
		return token
	}
	return ""
}
