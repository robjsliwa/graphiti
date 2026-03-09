package driving

import (
	"context"

	"graphiti/internal/domain"
)

type CommandResult struct {
	AffectedNodeIDs []string
	AffectedEdgeIDs []string
	CanUndo         bool
	CanRedo         bool
	Workflow        *domain.Workflow // current workflow state after command
}

type WorkflowService interface {
	CreateWorkflow(ctx context.Context, name, description, userID string) (*domain.Workflow, error)
	GetWorkflow(ctx context.Context, id string) (*domain.Workflow, error)
	ListWorkflows(ctx context.Context, userID string) ([]*domain.WorkflowSummary, error)
	SaveWorkflow(ctx context.Context, wf *domain.Workflow) error
	DeleteWorkflow(ctx context.Context, id string) error
	ExecuteCommand(ctx context.Context, workflowID string, cmd domain.Command) (*CommandResult, error)
	Undo(ctx context.Context, workflowID string) (*CommandResult, error)
	Redo(ctx context.Context, workflowID string) (*CommandResult, error)
	CopyNodes(ctx context.Context, workflowID string, nodeIDs []string) (*domain.ClipboardPayload, error)
	PasteNodes(ctx context.Context, workflowID string, payload *domain.ClipboardPayload, x, y float64) (*CommandResult, error)
	DeployWorkflow(ctx context.Context, workflowID, target, userID string) (*domain.DeployResult, error)
	ExportWorkflow(ctx context.Context, workflowID, format string) ([]byte, error)
}
