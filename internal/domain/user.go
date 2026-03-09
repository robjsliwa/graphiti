package domain

import "time"

// User represents an authenticated user of the system.
type User struct {
	ID             string
	Username       string
	Email          string
	AvatarURL      string
	AuthProvider   string
	AuthProviderID string
	CreatedAt      time.Time
	LastLoginAt    time.Time
}

// UserInfo is the data returned by an auth provider about a user.
type UserInfo struct {
	ID       string
	Username string
	Email    string
	Avatar   string
	Provider string
}
