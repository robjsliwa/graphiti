package driven

import (
	"context"

	"graphiti/internal/domain"
)

type UserRepository interface {
	Upsert(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByProviderID(ctx context.Context, provider, providerID string) (*domain.User, error)
}
