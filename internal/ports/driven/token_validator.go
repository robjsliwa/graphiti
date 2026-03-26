package driven

import "context"

// TokenValidator verifies API bearer tokens and returns the associated user ID.
type TokenValidator interface {
	// ValidateToken checks a bearer token and returns the user ID it belongs to.
	// Returns an error if the token is invalid, expired, or revoked.
	ValidateToken(ctx context.Context, token string) (userID string, err error)
}
