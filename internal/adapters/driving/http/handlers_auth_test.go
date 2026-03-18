package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"graphiti/internal/adapters/driven/auth"
	"graphiti/web/templates"
)

func TestHandleLogin_RendersBranding(t *testing.T) {
	// Use fake auth which redirects — so we need a real OAuth-style provider
	// to actually render the login page. Instead, test with GitHub auth
	// (which renders the page rather than redirecting).
	// We'll create a GitHub auth that will render the login page.
	authProvider := auth.NewGitHubAuth(auth.GitHubConfig{
		ClientID: "test-client-id",
	})

	sessionStore := NewSessionStore("test-secret", time.Hour, false)

	branding := templates.Branding{
		AppName:     "Acme Workflows",
		Tagline:     "Internal Pipeline Builder",
		TitleSuffix: "Acme",
	}

	handler := handleLogin(authProvider, sessionStore, branding)

	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Acme Workflows") {
		t.Errorf("expected body to contain %q", "Acme Workflows")
	}
	if !strings.Contains(body, "Internal Pipeline Builder") {
		t.Errorf("expected body to contain %q", "Internal Pipeline Builder")
	}
}
