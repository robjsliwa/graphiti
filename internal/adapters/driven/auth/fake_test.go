package auth

import (
	"context"
	"testing"
	"time"

	"graphiti/internal/ports/driven"
)

func TestFakeAuthAdapter_ImplementsAuthProvider(t *testing.T) {
	var _ driven.AuthProvider = (*FakeAuthAdapter)(nil)
}

func TestFakeAuthAdapter_GetAuthURL(t *testing.T) {
	adapter := NewFakeAuth()

	tests := []struct {
		name  string
		state string
		want  string
	}{
		{"with state", "abc123", "/auth/callback?code=fake-code&state=abc123"},
		{"empty state", "", "/auth/callback?code=fake-code&state="},
		{"special chars", "x=1&y=2", "/auth/callback?code=fake-code&state=x=1&y=2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.GetAuthURL(tt.state)
			if got != tt.want {
				t.Errorf("GetAuthURL(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestFakeAuthAdapter_ExchangeCode(t *testing.T) {
	adapter := NewFakeAuth()
	ctx := context.Background()
	before := time.Now()

	pair, err := adapter.ExchangeCode(ctx, "any-code")
	if err != nil {
		t.Fatalf("ExchangeCode returned error: %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if pair.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	if pair.ExpiresAt.Before(before.Add(23 * time.Hour)) {
		t.Error("ExpiresAt should be approximately 24 hours from now")
	}
}

func TestFakeAuthAdapter_GetUserInfo(t *testing.T) {
	adapter := NewFakeAuth()
	ctx := context.Background()

	info, err := adapter.GetUserInfo(ctx, "any-token")
	if err != nil {
		t.Fatalf("GetUserInfo returned error: %v", err)
	}

	if info.Username != "dev" {
		t.Errorf("Username = %q, want %q", info.Username, "dev")
	}
	if info.Email != "dev@local" {
		t.Errorf("Email = %q, want %q", info.Email, "dev@local")
	}
	if info.Provider != "fake" {
		t.Errorf("Provider = %q, want %q", info.Provider, "fake")
	}
	if info.ID != "fake-001" {
		t.Errorf("ID = %q, want %q", info.ID, "fake-001")
	}
}

func TestFakeAuthAdapter_RefreshToken(t *testing.T) {
	adapter := NewFakeAuth()
	ctx := context.Background()
	before := time.Now()

	pair, err := adapter.RefreshToken(ctx, "any-refresh-token")
	if err != nil {
		t.Fatalf("RefreshToken returned error: %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if pair.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	if pair.ExpiresAt.Before(before.Add(23 * time.Hour)) {
		t.Error("ExpiresAt should be approximately 24 hours from now")
	}
}
