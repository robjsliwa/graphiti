package driven

import (
	"context"

	"graphiti/internal/domain"
)

type ExecutionFilter struct {
	Status domain.ExecutionStatus
	Limit  int
	Offset int
}

type ExecutionRepository interface {
	Create(ctx context.Context, run *domain.ExecutionRun) error
	GetByID(ctx context.Context, id string) (*domain.ExecutionRun, error)
	ListByWorkflow(ctx context.Context, workflowID string, filter ExecutionFilter) ([]*domain.ExecutionRunSummary, error)
	UpdateNodeStatus(ctx context.Context, runID, nodeID string, status domain.NodeExecutionStatus) error
	AppendNodeLog(ctx context.Context, runID, nodeID string, entry domain.LogEntry) error
}
