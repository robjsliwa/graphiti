package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestSessionStore() *SessionStore {
	return NewSessionStore("test-secret-key-at-least-32-chars!", 24*time.Hour, false)
}

func setSessionCookie(r *http.Request, store *SessionStore, session *Session) {
	w := httptest.NewRecorder()
	_ = store.Set(w, session)
	for _, c := range w.Result().Cookies() {
		r.AddCookie(c)
	}
}

func TestAuthMiddleware_UnauthenticatedRedirectsToLogin(t *testing.T) {
	store := newTestSessionStore()
	handler := AuthMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	location := rec.Header().Get("Location")
	if location != "/auth/login" {
		t.Errorf("Location = %q, want %q", location, "/auth/login")
	}
}

func TestAuthMiddleware_AuthenticatedPassesThrough(t *testing.T) {
	store := newTestSessionStore()
	var gotSession *Session

	handler := AuthMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSession = GetSession(r)
		w.WriteHeader(http.StatusOK)
	}))

	session := &Session{
		UserID:    "user-1",
		Username:  "testuser",
		Email:     "test@example.com",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	setSessionCookie(req, store, session)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotSession == nil {
		t.Fatal("session should be present in context")
	}
	if gotSession.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", gotSession.UserID, "user-1")
	}
	if gotSession.Username != "testuser" {
		t.Errorf("Username = %q, want %q", gotSession.Username, "testuser")
	}
}

func TestAuthMiddleware_AuthPathBypassesAuth(t *testing.T) {
	store := newTestSessionStore()
	called := false

	handler := AuthMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	paths := []string{"/auth/login", "/auth/callback", "/auth/logout"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if !called {
				t.Error("handler should have been called for auth path")
			}
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

func TestAuthMiddleware_StaticPathBypassesAuth(t *testing.T) {
	store := newTestSessionStore()
	called := false

	handler := AuthMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	paths := []string{"/static/js/canvas.js", "/static/css/style.css", "/static/img/logo.png"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if !called {
				t.Error("handler should have been called for static path")
			}
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

func TestAuthMiddleware_ExpiredSessionRedirects(t *testing.T) {
	store := newTestSessionStore()

	handler := AuthMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	session := &Session{
		UserID:    "user-1",
		Username:  "testuser",
		Email:     "test@example.com",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // expired
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	setSessionCookie(req, store, session)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	location := rec.Header().Get("Location")
	if location != "/auth/login" {
		t.Errorf("Location = %q, want %q", location, "/auth/login")
	}
}

func TestGetSession_NoSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	session := GetSession(req)
	if session != nil {
		t.Error("expected nil session when no session in context")
	}
}
