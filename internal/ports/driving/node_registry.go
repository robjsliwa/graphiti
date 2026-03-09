package driving

import (
	"context"

	"graphiti/internal/domain"
)

type NodeRegistryService interface {
	GetAllDefinitions(ctx context.Context) ([]*domain.NodeDefinition, error)
	GetByCategory(ctx context.Context, category string) ([]*domain.NodeDefinition, error)
	GetByID(ctx context.Context, id string) (*domain.NodeDefinition, error)
	Search(ctx context.Context, query string) ([]*domain.NodeDefinition, error)
}
