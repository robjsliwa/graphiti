package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

// UserRepository implements driven.UserRepository using SQLite.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new SQLite-backed user repository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Upsert creates a new user or updates an existing one matched by auth_provider + auth_provider_id.
func (r *UserRepository) Upsert(ctx context.Context, user *domain.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, username, email, avatar_url, auth_provider, auth_provider_id, created_at, last_login_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(auth_provider, auth_provider_id) DO UPDATE SET
		   username = excluded.username,
		   email = excluded.email,
		   avatar_url = excluded.avatar_url,
		   last_login_at = excluded.last_login_at`,
		user.ID, user.Username, user.Email, user.AvatarURL,
		user.AuthProvider, user.AuthProviderID,
		user.CreatedAt.UTC(), user.LastLoginAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}
	return nil
}

// GetByID retrieves a user by their ID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	var email, avatarURL sql.NullString
	var createdAt, lastLoginAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, email, avatar_url, auth_provider, auth_provider_id, created_at, last_login_at
		 FROM users WHERE id = ?`, id,
	).Scan(&user.ID, &user.Username, &email, &avatarURL,
		&user.AuthProvider, &user.AuthProviderID, &createdAt, &lastLoginAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user %s: not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	user.Email = email.String
	user.AvatarURL = avatarURL.String
	user.CreatedAt, _ = time.Parse("2006-01-02 15:04:05-07:00", createdAt)
	if user.CreatedAt.IsZero() {
		user.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	}
	if lastLoginAt != "" {
		user.LastLoginAt, _ = time.Parse("2006-01-02 15:04:05-07:00", lastLoginAt)
		if user.LastLoginAt.IsZero() {
			user.LastLoginAt, _ = time.Parse("2006-01-02 15:04:05", lastLoginAt)
		}
	}

	return &user, nil
}

// GetByProviderID retrieves a user by their auth provider and provider-specific ID.
func (r *UserRepository) GetByProviderID(ctx context.Context, provider, providerID string) (*domain.User, error) {
	var user domain.User
	var email, avatarURL sql.NullString
	var createdAt, lastLoginAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, email, avatar_url, auth_provider, auth_provider_id, created_at, last_login_at
		 FROM users WHERE auth_provider = ? AND auth_provider_id = ?`, provider, providerID,
	).Scan(&user.ID, &user.Username, &email, &avatarURL,
		&user.AuthProvider, &user.AuthProviderID, &createdAt, &lastLoginAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user with provider %s/%s: not found", provider, providerID)
	}
	if err != nil {
		return nil, fmt.Errorf("query user by provider: %w", err)
	}

	user.Email = email.String
	user.AvatarURL = avatarURL.String
	user.CreatedAt, _ = time.Parse("2006-01-02 15:04:05-07:00", createdAt)
	if user.CreatedAt.IsZero() {
		user.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	}
	if lastLoginAt != "" {
		user.LastLoginAt, _ = time.Parse("2006-01-02 15:04:05-07:00", lastLoginAt)
		if user.LastLoginAt.IsZero() {
			user.LastLoginAt, _ = time.Parse("2006-01-02 15:04:05", lastLoginAt)
		}
	}

	return &user, nil
}

// Ensure UserRepository implements driven.UserRepository.
var _ driven.UserRepository = (*UserRepository)(nil)
