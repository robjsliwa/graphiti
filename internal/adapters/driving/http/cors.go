package http

import (
	"net/http"
	"strings"
)

// CORSConfig controls Cross-Origin Resource Sharing behavior.
type CORSConfig struct {
	// AllowedOrigins is a list of origins permitted to make cross-origin requests.
	// Use ["*"] to allow all origins (not recommended for production).
	// Empty slice disables CORS headers entirely (same-origin only).
	AllowedOrigins []string

	// AllowCredentials indicates whether the request can include credentials.
	AllowCredentials bool
}

// CORSMiddleware returns middleware that adds CORS headers to /api/ routes.
func CORSMiddleware(cfg CORSConfig) func(http.Handler) http.Handler {
	if len(cfg.AllowedOrigins) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	originSet := make(map[string]bool, len(cfg.AllowedOrigins))
	allowAll := false
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			allowAll = true
		}
		originSet[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply CORS to API routes
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Always set Vary: Origin to prevent cache poisoning
			w.Header().Add("Vary", "Origin")

			if allowAll || originSet[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
				w.Header().Set("Access-Control-Expose-Headers", "X-Request-Id")
				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				// Handle allowed preflight
				if r.Method == http.MethodOptions {
					w.Header().Set("Access-Control-Max-Age", "86400")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			} else if r.Method == http.MethodOptions {
				// Denied origin preflight — reject immediately
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
