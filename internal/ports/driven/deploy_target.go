package driven

import (
	"context"

	"graphiti/internal/domain"
)

type DeployTarget interface {
	Deploy(ctx context.Context, payload domain.DeployPayload) (*domain.DeployResult, error)
}
