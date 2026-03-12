package driven

import (
	"context"

	"graphiti/internal/domain"
)

type DeployTarget interface {
	Deploy(ctx context.Context, payload domain.DeployPayload) (*domain.DeployResult, error)
}

// DeployStatusChecker is an optional interface that deploy targets can implement
// to support verifying whether a workflow is still deployed on the engine.
type DeployStatusChecker interface {
	CheckDeployStatus(ctx context.Context, workflowID string, version int) (*domain.DeployStatusResult, error)
}
