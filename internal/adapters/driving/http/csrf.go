package http

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const csrfTokenName = "graphiti_csrf"
const csrfHeaderName = "X-CSRF-Token"

// CSRFMiddleware provides CSRF protection for state-changing requests.
// It generates a token stored in a cookie and requires it to be sent back
// in a header or form field for POST/PATCH/DELETE requests.
func CSRFMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For GET/HEAD/OPTIONS, set the token cookie if not present
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				if _, err := r.Cookie(csrfTokenName); err != nil {
					token := generateCSRFToken()
					http.SetCookie(w, &http.Cookie{
						Name:     csrfTokenName,
						Value:    token,
						Path:     "/",
						HttpOnly: false, // must be readable by JS
						SameSite: http.SameSiteStrictMode,
					})
				}
				next.ServeHTTP(w, r)
				return
			}

			// For state-changing methods, verify the token
			cookie, err := r.Cookie(csrfTokenName)
			if err != nil {
				http.Error(w, "CSRF token missing", http.StatusForbidden)
				return
			}

			// Check header first, then form value
			token := r.Header.Get(csrfHeaderName)
			if token == "" {
				token = r.FormValue("_csrf")
			}

			if token != cookie.Value {
				http.Error(w, "CSRF token invalid", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// RateLimiter provides simple per-IP rate limiting.
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// RateLimitMiddleware applies rate limiting to the wrapped handler.
func (rl *RateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		// Strip port
		if idx := lastIndex(ip, ':'); idx >= 0 {
			ip = ip[:idx]
		}

		rl.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)

		// Clean old entries
		var recent []time.Time
		for _, t := range rl.requests[ip] {
			if t.After(cutoff) {
				recent = append(recent, t)
			}
		}

		if len(recent) >= rl.limit {
			rl.mu.Unlock()
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		rl.requests[ip] = append(recent, now)
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func lastIndex(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
