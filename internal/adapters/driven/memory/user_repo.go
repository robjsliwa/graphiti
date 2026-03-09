package memory

import (
	"context"
	"fmt"
	"sync"

	"graphiti/internal/domain"
)

// UserRepo is an in-memory implementation of UserRepository for tests.
type UserRepo struct {
	mu    sync.RWMutex
	users map[string]*domain.User
}

// NewUserRepo creates a new in-memory user repository.
func NewUserRepo() *UserRepo {
	return &UserRepo{
		users: make(map[string]*domain.User),
	}
}

func (r *UserRepo) Upsert(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.ID] = user
	return nil
}

func (r *UserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[id]
	if !ok {
		return nil, fmt.Errorf("user %s not found", id)
	}
	return user, nil
}

func (r *UserRepo) GetByProviderID(_ context.Context, provider, providerID string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.AuthProvider == provider && user.AuthProviderID == providerID {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found for provider %s/%s", provider, providerID)
}
