package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"graphiti/internal/ports/driven"
)

func TestGitHubAuthAdapter_ImplementsAuthProvider(t *testing.T) {
	var _ driven.AuthProvider = (*GitHubAuthAdapter)(nil)
}

func TestGitHubAuthAdapter_GetAuthURL(t *testing.T) {
	adapter := NewGitHubAuth(GitHubConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		Scopes:       []string{"user:email", "read:org"},
	})

	url := adapter.GetAuthURL("test-state-123")

	// Verify the URL contains expected components
	tests := []struct {
		name     string
		contains string
	}{
		{"github authorize endpoint", "https://github.com/login/oauth/authorize"},
		{"client_id", "client_id=test-client-id"},
		{"scopes", "scope=user%3Aemail+read%3Aorg"},
		{"state", "state=test-state-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !containsSubstring(url, tt.contains) {
				t.Errorf("GetAuthURL() = %q, want to contain %q", url, tt.contains)
			}
		})
	}
}

func TestGitHubAuthAdapter_GetAuthURL_EmptyScopes(t *testing.T) {
	adapter := NewGitHubAuth(GitHubConfig{
		ClientID: "id",
	})
	url := adapter.GetAuthURL("s")
	if containsSubstring(url, "scope=") {
		t.Errorf("GetAuthURL() with no scopes should not contain scope param, got %q", url)
	}
}

func TestGitHubAuthAdapter_ExchangeCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login/oauth/access_token" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept: application/json, got %s", r.Header.Get("Accept"))
		}

		// Verify form values
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("client_id") != "test-client" {
			t.Errorf("unexpected client_id: %s", r.FormValue("client_id"))
		}
		if r.FormValue("code") != "test-code" {
			t.Errorf("unexpected code: %s", r.FormValue("code"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":  "gho_test_token",
			"refresh_token": "ghr_test_refresh",
			"token_type":    "bearer",
		})
	}))
	defer server.Close()

	adapter := NewGitHubAuth(GitHubConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	})
	adapter.tokenURL = server.URL + "/login/oauth/access_token"

	pair, err := adapter.ExchangeCode(context.Background(), "test-code")
	if err != nil {
		t.Fatalf("ExchangeCode error: %v", err)
	}
	if pair.AccessToken != "gho_test_token" {
		t.Errorf("AccessToken = %q, want %q", pair.AccessToken, "gho_test_token")
	}
	if pair.RefreshToken != "ghr_test_refresh" {
		t.Errorf("RefreshToken = %q, want %q", pair.RefreshToken, "ghr_test_refresh")
	}
	if pair.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestGitHubAuthAdapter_ExchangeCode_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "bad_verification_code",
			"error_description": "The code passed is incorrect or expired.",
		})
	}))
	defer server.Close()

	adapter := NewGitHubAuth(GitHubConfig{ClientID: "id", ClientSecret: "secret"})
	adapter.tokenURL = server.URL + "/login/oauth/access_token"

	_, err := adapter.ExchangeCode(context.Background(), "bad-code")
	if err == nil {
		t.Fatal("expected error for bad code")
	}
}

func TestGitHubAuthAdapter_ExchangeCode_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := NewGitHubAuth(GitHubConfig{ClientID: "id", ClientSecret: "secret"})
	adapter.tokenURL = server.URL + "/login/oauth/access_token"

	_, err := adapter.ExchangeCode(context.Background(), "code")
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestGitHubAuthAdapter_GetUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":         12345,
			"login":      "octocat",
			"email":      "octocat@github.com",
			"avatar_url": "https://avatars.githubusercontent.com/u/12345",
		})
	}))
	defer server.Close()

	adapter := NewGitHubAuth(GitHubConfig{ClientID: "id", ClientSecret: "secret"})
	adapter.apiURL = server.URL

	info, err := adapter.GetUserInfo(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("GetUserInfo error: %v", err)
	}
	if info.ID != "12345" {
		t.Errorf("ID = %q, want %q", info.ID, "12345")
	}
	if info.Username != "octocat" {
		t.Errorf("Username = %q, want %q", info.Username, "octocat")
	}
	if info.Email != "octocat@github.com" {
		t.Errorf("Email = %q, want %q", info.Email, "octocat@github.com")
	}
	if info.Avatar != "https://avatars.githubusercontent.com/u/12345" {
		t.Errorf("Avatar = %q", info.Avatar)
	}
	if info.Provider != "github" {
		t.Errorf("Provider = %q, want %q", info.Provider, "github")
	}
}

func TestGitHubAuthAdapter_GetUserInfo_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	adapter := NewGitHubAuth(GitHubConfig{ClientID: "id", ClientSecret: "secret"})
	adapter.apiURL = server.URL

	_, err := adapter.GetUserInfo(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error for unauthorized")
	}
}

func TestGitHubAuthAdapter_GetUserInfo_WithAllowedOrgs_Allowed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    1,
			"login": "dev",
			"email": "dev@test.com",
		})
	})
	mux.HandleFunc("/user/orgs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"login": "allowed-org"},
			{"login": "other-org"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := NewGitHubAuth(GitHubConfig{
		ClientID:    "id",
		ClientSecret: "secret",
		AllowedOrgs: []string{"allowed-org"},
	})
	adapter.apiURL = server.URL

	info, err := adapter.GetUserInfo(context.Background(), "token")
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if info.Username != "dev" {
		t.Errorf("Username = %q, want %q", info.Username, "dev")
	}
}

func TestGitHubAuthAdapter_GetUserInfo_WithAllowedOrgs_Denied(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    1,
			"login": "dev",
			"email": "dev@test.com",
		})
	})
	mux.HandleFunc("/user/orgs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"login": "wrong-org"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := NewGitHubAuth(GitHubConfig{
		ClientID:    "id",
		ClientSecret: "secret",
		AllowedOrgs: []string{"required-org"},
	})
	adapter.apiURL = server.URL

	_, err := adapter.GetUserInfo(context.Background(), "token")
	if err == nil {
		t.Fatal("expected error for org not allowed")
	}
}

func TestGitHubAuthAdapter_RefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q, want refresh_token", r.FormValue("grant_type"))
		}
		if r.FormValue("refresh_token") != "old-refresh" {
			t.Errorf("refresh_token = %q", r.FormValue("refresh_token"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
		})
	}))
	defer server.Close()

	adapter := NewGitHubAuth(GitHubConfig{ClientID: "id", ClientSecret: "secret"})
	adapter.tokenURL = server.URL + "/login/oauth/access_token"

	pair, err := adapter.RefreshToken(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("RefreshToken error: %v", err)
	}
	if pair.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q", pair.AccessToken)
	}
	if pair.RefreshToken != "new-refresh" {
		t.Errorf("RefreshToken = %q", pair.RefreshToken)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	return fmt.Sprintf("%s", s) != "" && len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
