package memory

import (
	"context"
	"errors"
	"sync"
)

// TokenStore is an in-memory token validator for development and testing.
type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]string // token -> userID
}

// NewTokenStore creates a new in-memory token store.
func NewTokenStore() *TokenStore {
	return &TokenStore{tokens: make(map[string]string)}
}

// AddToken registers a token for a user.
func (s *TokenStore) AddToken(token, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = userID
}

// ValidateToken implements driven.TokenValidator.
func (s *TokenStore) ValidateToken(_ context.Context, token string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userID, ok := s.tokens[token]
	if !ok {
		return "", errors.New("invalid token")
	}
	return userID, nil
}
