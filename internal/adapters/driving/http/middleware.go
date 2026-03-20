package http

import (
	"context"
	"net/http"
	"strings"

	"graphiti/internal/ports/driven"
)

type contextKey string

const userContextKey contextKey = "user_session"

// ExportedUserContextKey is the context key for the user session,
// exported for use by the graphiti library package.
var ExportedUserContextKey = userContextKey

// AuthMiddleware returns middleware that checks for a valid session cookie.
// Requests to /auth/* and /static/* paths bypass authentication.
// Unauthenticated requests are redirected to /auth/login.
// If a userRepo is provided, the middleware also verifies the user exists in the DB
// (handles stale cookies after DB reset).
func AuthMiddleware(sessionStore *SessionStore, userRepo ...driven.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Skip auth for public paths
			if strings.HasPrefix(path, "/auth/") || strings.HasPrefix(path, "/static/") {
				next.ServeHTTP(w, r)
				return
			}

			session, err := sessionStore.Get(r)
			if err != nil {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			// Verify user still exists in DB (handles DB reset with stale cookie)
			if len(userRepo) > 0 && userRepo[0] != nil {
				if _, err := userRepo[0].GetByID(r.Context(), session.UserID); err != nil {
					sessionStore.Clear(w)
					http.Redirect(w, r, "/auth/login", http.StatusFound)
					return
				}
			}

			ctx := context.WithValue(r.Context(), userContextKey, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetSession extracts the Session from the request context.
// Returns nil if no session is present.
func GetSession(r *http.Request) *Session {
	session, _ := r.Context().Value(userContextKey).(*Session)
	return session
}
