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

// AuthMiddleware returns middleware that checks for a valid session cookie
// or bearer token. Requests to /auth/* and /static/* paths bypass authentication.
// Unauthenticated requests to /api/* return 401 JSON; others redirect to /auth/login.
// If a userRepo is provided, the middleware also verifies the user exists in the DB
// (handles stale cookies after DB reset).
func AuthMiddleware(sessionStore *SessionStore, userRepo ...driven.UserRepository) func(http.Handler) http.Handler {
	return AuthMiddlewareWithToken(sessionStore, nil, userRepo...)
}

// AuthMiddlewareWithToken is like AuthMiddleware but also accepts bearer tokens
// via the Authorization header when a TokenValidator is provided.
func AuthMiddlewareWithToken(sessionStore *SessionStore, tokenValidator driven.TokenValidator, userRepo ...driven.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Skip auth for public paths
			if strings.HasPrefix(path, "/auth/") || strings.HasPrefix(path, "/static/") {
				next.ServeHTTP(w, r)
				return
			}

			// Try bearer token first (for API clients)
			if token := extractBearerToken(r); token != "" && tokenValidator != nil {
				userID, err := tokenValidator.ValidateToken(r.Context(), token)
				if err != nil {
					writeJSONError(w, http.StatusUnauthorized, "invalid token")
					return
				}
				session := &Session{UserID: userID}
				ctx := context.WithValue(r.Context(), userContextKey, session)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Fall back to session cookie
			session, err := sessionStore.Get(r)
			if err != nil {
				if strings.HasPrefix(path, "/api/") {
					writeJSONError(w, http.StatusUnauthorized, "unauthorized")
					return
				}
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			// Verify user still exists in DB (handles DB reset with stale cookie)
			if len(userRepo) > 0 && userRepo[0] != nil {
				if _, err := userRepo[0].GetByID(r.Context(), session.UserID); err != nil {
					sessionStore.Clear(w)
					if strings.HasPrefix(path, "/api/") {
						writeJSONError(w, http.StatusUnauthorized, "unauthorized")
						return
					}
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

// extractBearerToken extracts the token from an Authorization: Bearer header.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
