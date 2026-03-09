package auth

import (
	"context"
	"time"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

// FakeAuthAdapter auto-authenticates a hardcoded dev user for local development.
type FakeAuthAdapter struct {
	devUser *domain.User
}

// NewFakeAuth creates a new FakeAuthAdapter with a hardcoded dev user.
func NewFakeAuth() *FakeAuthAdapter {
	return &FakeAuthAdapter{
		devUser: &domain.User{
			ID:             "dev-user-001",
			Username:       "dev",
			Email:          "dev@local",
			AuthProvider:   "fake",
			AuthProviderID: "fake-001",
			CreatedAt:      time.Now(),
			LastLoginAt:    time.Now(),
		},
	}
}

// compile-time interface check
var _ driven.AuthProvider = (*FakeAuthAdapter)(nil)

// GetAuthURL returns a fake auth callback URL for local development.
func (f *FakeAuthAdapter) GetAuthURL(state string) string {
	return "/auth/callback?code=fake-code&state=" + state
}

// ExchangeCode always returns a valid TokenPair with 24h expiry.
func (f *FakeAuthAdapter) ExchangeCode(_ context.Context, _ string) (*driven.TokenPair, error) {
	return &driven.TokenPair{
		AccessToken:  "fake-access-token",
		RefreshToken: "fake-refresh-token",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}, nil
}

// GetUserInfo returns the hardcoded dev user info.
func (f *FakeAuthAdapter) GetUserInfo(_ context.Context, _ string) (*domain.UserInfo, error) {
	return &domain.UserInfo{
		ID:       f.devUser.AuthProviderID,
		Username: f.devUser.Username,
		Email:    f.devUser.Email,
		Avatar:   "",
		Provider: f.devUser.AuthProvider,
	}, nil
}

// RefreshToken returns a new valid token pair with 24h expiry.
func (f *FakeAuthAdapter) RefreshToken(_ context.Context, _ string) (*driven.TokenPair, error) {
	return &driven.TokenPair{
		AccessToken:  "fake-access-token-refreshed",
		RefreshToken: "fake-refresh-token-refreshed",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}, nil
}
