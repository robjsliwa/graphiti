package sqlite_test

import (
	"context"
	"testing"
	"time"

	sqliteadapter "graphiti/internal/adapters/driven/sqlite"
	"graphiti/internal/domain"
)

func TestUserRepo_Upsert_CreateNew(t *testing.T) {
	db := newTestDB(t)
	repo := sqliteadapter.NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{
		ID:             "user-1",
		Username:       "alice",
		Email:          "alice@example.com",
		AvatarURL:      "https://example.com/alice.png",
		AuthProvider:   "github",
		AuthProviderID: "gh-123",
		CreatedAt:      time.Now().Truncate(time.Second).UTC(),
		LastLoginAt:    time.Now().Truncate(time.Second).UTC(),
	}

	if err := repo.Upsert(ctx, user); err != nil {
		t.Fatalf("upsert (create): %v", err)
	}

	got, err := repo.GetByID(ctx, "user-1")
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Username != "alice" {
		t.Errorf("Username: got %s, want alice", got.Username)
	}
	if got.Email != "alice@example.com" {
		t.Errorf("Email: got %s, want alice@example.com", got.Email)
	}
	if got.AuthProvider != "github" {
		t.Errorf("AuthProvider: got %s, want github", got.AuthProvider)
	}
}

func TestUserRepo_Upsert_UpdateExisting(t *testing.T) {
	db := newTestDB(t)
	repo := sqliteadapter.NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{
		ID:             "user-1",
		Username:       "alice",
		Email:          "alice@example.com",
		AuthProvider:   "github",
		AuthProviderID: "gh-123",
		CreatedAt:      time.Now().Truncate(time.Second).UTC(),
		LastLoginAt:    time.Now().Truncate(time.Second).UTC(),
	}
	if err := repo.Upsert(ctx, user); err != nil {
		t.Fatalf("upsert (create): %v", err)
	}

	// Update with same provider credentials but different username/email.
	updated := &domain.User{
		ID:             "user-1",
		Username:       "alice-updated",
		Email:          "newalice@example.com",
		AvatarURL:      "https://example.com/new-alice.png",
		AuthProvider:   "github",
		AuthProviderID: "gh-123",
		CreatedAt:      user.CreatedAt,
		LastLoginAt:    time.Now().Truncate(time.Second).UTC(),
	}
	if err := repo.Upsert(ctx, updated); err != nil {
		t.Fatalf("upsert (update): %v", err)
	}

	got, err := repo.GetByID(ctx, "user-1")
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Username != "alice-updated" {
		t.Errorf("Username after update: got %s, want alice-updated", got.Username)
	}
	if got.Email != "newalice@example.com" {
		t.Errorf("Email after update: got %s, want newalice@example.com", got.Email)
	}
	if got.AvatarURL != "https://example.com/new-alice.png" {
		t.Errorf("AvatarURL after update: got %s, want https://example.com/new-alice.png", got.AvatarURL)
	}
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := sqliteadapter.NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestUserRepo_GetByProviderID(t *testing.T) {
	db := newTestDB(t)
	repo := sqliteadapter.NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{
		ID:             "user-1",
		Username:       "bob",
		Email:          "bob@example.com",
		AuthProvider:   "github",
		AuthProviderID: "gh-456",
		CreatedAt:      time.Now().Truncate(time.Second).UTC(),
		LastLoginAt:    time.Now().Truncate(time.Second).UTC(),
	}
	if err := repo.Upsert(ctx, user); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := repo.GetByProviderID(ctx, "github", "gh-456")
	if err != nil {
		t.Fatalf("get by provider id: %v", err)
	}
	if got.ID != "user-1" {
		t.Errorf("ID: got %s, want user-1", got.ID)
	}
	if got.Username != "bob" {
		t.Errorf("Username: got %s, want bob", got.Username)
	}
}

func TestUserRepo_GetByProviderID_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := sqliteadapter.NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.GetByProviderID(ctx, "github", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent provider ID")
	}
}
