package driving

import (
	"context"

	"graphiti/internal/domain"
)

type ExecutionService interface {
	GetRun(ctx context.Context, runID string) (*domain.ExecutionRun, error)
	ListRuns(ctx context.Context, workflowID string) ([]*domain.ExecutionRunSummary, error)
	UpdateNodeStatus(ctx context.Context, runID string, status domain.NodeExecutionStatus) error
}
