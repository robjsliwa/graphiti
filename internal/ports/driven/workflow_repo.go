package driven

import (
	"context"

	"graphiti/internal/domain"
)

type WorkflowFilter struct {
	UserID string
	Status domain.WorkflowStatus
}

type WorkflowRepository interface {
	Create(ctx context.Context, wf *domain.Workflow) error
	GetByID(ctx context.Context, id string) (*domain.Workflow, error)
	List(ctx context.Context, filter WorkflowFilter) ([]*domain.WorkflowSummary, error)
	Update(ctx context.Context, wf *domain.Workflow) error
	Delete(ctx context.Context, id string) error
	GetVersionHistory(ctx context.Context, id string) ([]*domain.WorkflowVersion, error)
}
