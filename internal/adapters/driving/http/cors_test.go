package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newCORSTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestCORSMiddleware_PreflightAllowed(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://portal.example.com"}}
	handler := CORSMiddleware(cfg)(newCORSTestHandler())

	req := httptest.NewRequest(http.MethodOptions, "/api/workflows", nil)
	req.Header.Set("Origin", "https://portal.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://portal.example.com" {
		t.Errorf("ACAO = %q, want %q", got, "https://portal.example.com")
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("ACMA = %q, want %q", got, "86400")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
}

func TestCORSMiddleware_PreflightDenied(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://portal.example.com"}}
	handler := CORSMiddleware(cfg)(newCORSTestHandler())

	req := httptest.NewRequest(http.MethodOptions, "/api/workflows", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO = %q, want empty", got)
	}
}

func TestCORSMiddleware_GetWithAllowedOrigin(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://portal.example.com"}}
	handler := CORSMiddleware(cfg)(newCORSTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
	req.Header.Set("Origin", "https://portal.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://portal.example.com" {
		t.Errorf("ACAO = %q, want %q", got, "https://portal.example.com")
	}
}

func TestCORSMiddleware_NonAPIRoute(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://portal.example.com"}}
	handler := CORSMiddleware(cfg)(newCORSTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://portal.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("non-API route should not get CORS headers, got ACAO = %q", got)
	}
}

func TestCORSMiddleware_WildcardOrigin(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"*"}}
	handler := CORSMiddleware(cfg)(newCORSTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
	req.Header.Set("Origin", "https://any-origin.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://any-origin.example.com" {
		t.Errorf("ACAO = %q, want %q", got, "https://any-origin.example.com")
	}
}

func TestCORSMiddleware_AllowCredentials(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://portal.example.com"},
		AllowCredentials: true,
	}
	handler := CORSMiddleware(cfg)(newCORSTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
	req.Header.Set("Origin", "https://portal.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("ACAC = %q, want %q", got, "true")
	}
}

func TestCORSMiddleware_EmptyOrigins(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{}}
	handler := CORSMiddleware(cfg)(newCORSTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
	req.Header.Set("Origin", "https://portal.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("empty origins should disable CORS, got ACAO = %q", got)
	}
}

func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://portal.example.com"}}
	handler := CORSMiddleware(cfg)(newCORSTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
	// No Origin header
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("no Origin header should skip CORS, got ACAO = %q", got)
	}
}

func TestCORSMiddleware_VaryOriginHeader(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://portal.example.com"}}
	handler := CORSMiddleware(cfg)(newCORSTestHandler())

	// Allowed origin should have Vary: Origin
	req := httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
	req.Header.Set("Origin", "https://portal.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want %q", got, "Origin")
	}

	// Denied origin should also have Vary: Origin
	req = httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want %q for denied origin", got, "Origin")
	}
}

func TestCORSMiddleware_WorkflowsRoute(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://portal.example.com"}}
	handler := CORSMiddleware(cfg)(newCORSTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/workflows/123", nil)
	req.Header.Set("Origin", "https://portal.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("non-API route /workflows/ should not get CORS headers, got ACAO = %q", got)
	}
}
