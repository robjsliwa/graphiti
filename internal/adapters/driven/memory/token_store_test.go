package memory

import (
	"context"
	"testing"
)

func TestTokenStore_ValidateToken(t *testing.T) {
	store := NewTokenStore()
	store.AddToken("valid-token", "user-123")

	userID, err := store.ValidateToken(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != "user-123" {
		t.Errorf("userID = %q, want %q", userID, "user-123")
	}
}

func TestTokenStore_ValidateToken_Invalid(t *testing.T) {
	store := NewTokenStore()

	_, err := store.ValidateToken(context.Background(), "invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestTokenStore_AddMultipleTokens(t *testing.T) {
	store := NewTokenStore()
	store.AddToken("token-a", "user-1")
	store.AddToken("token-b", "user-2")

	userID, err := store.ValidateToken(context.Background(), "token-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != "user-1" {
		t.Errorf("userID = %q, want %q", userID, "user-1")
	}

	userID, err = store.ValidateToken(context.Background(), "token-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != "user-2" {
		t.Errorf("userID = %q, want %q", userID, "user-2")
	}
}
