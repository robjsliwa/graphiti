package app

import (
	"context"
	"time"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
	"graphiti/internal/ports/driving"
)

// ExecutionCallback represents the payload from an external engine's status callback.
type ExecutionCallback struct {
	APIVersion    string         `json:"apiVersion"`
	Event         string         `json:"event"`
	RunID         string         `json:"runID"`
	WorkflowID    string         `json:"workflowID"`
	NodeID        string         `json:"nodeID"`
	Status        string         `json:"status"`
	StartedAt     time.Time      `json:"startedAt,omitempty"`
	CompletedAt   time.Time      `json:"completedAt,omitempty"`
	ErrorMessage  string         `json:"errorMessage,omitempty"`
	OutputSummary map[string]any `json:"outputSummary,omitempty"`
}

// StatusUpdateNotifier is called when an execution status changes, for WebSocket broadcasting.
type StatusUpdateNotifier func(workflowID string, callback ExecutionCallback)

// ExecutionService implements the driving.ExecutionService port.
type ExecutionService struct {
	repo     driven.ExecutionRepository
	notifier StatusUpdateNotifier
}

// NewExecutionService creates a new execution application service.
func NewExecutionService(repo driven.ExecutionRepository) *ExecutionService {
	return &ExecutionService{repo: repo}
}

// SetNotifier sets the function called when execution status changes.
func (s *ExecutionService) SetNotifier(fn StatusUpdateNotifier) {
	s.notifier = fn
}

// compile-time interface check
var _ driving.ExecutionService = (*ExecutionService)(nil)

// GetRun returns a full execution run by ID.
func (s *ExecutionService) GetRun(ctx context.Context, runID string) (*domain.ExecutionRun, error) {
	return s.repo.GetByID(ctx, runID)
}

// ListRuns returns execution run summaries for a workflow.
func (s *ExecutionService) ListRuns(ctx context.Context, workflowID string) ([]*domain.ExecutionRunSummary, error) {
	return s.repo.ListByWorkflow(ctx, workflowID, driven.ExecutionFilter{Limit: 50})
}

// UpdateNodeStatus updates a node's execution status within a run.
func (s *ExecutionService) UpdateNodeStatus(ctx context.Context, runID string, status domain.NodeExecutionStatus) error {
	return s.repo.UpdateNodeStatus(ctx, runID, status.NodeID, status)
}

// ProcessCallback handles an execution status callback from the external engine.
// If the run doesn't exist, it creates one. Then it updates the node status.
func (s *ExecutionService) ProcessCallback(ctx context.Context, cb ExecutionCallback) error {
	// Check if run exists, create if not
	_, err := s.repo.GetByID(ctx, cb.RunID)
	if err != nil {
		run := &domain.ExecutionRun{
			ID:              cb.RunID,
			WorkflowID:      cb.WorkflowID,
			WorkflowVersion: 1,
			Status:          domain.ExecStatusRunning,
			StartedAt:       time.Now(),
			TriggerType:     "callback",
			NodeStatuses:    make(map[string]*domain.NodeExecutionStatus),
		}
		if err := s.repo.Create(ctx, run); err != nil {
			return err
		}
	}

	// Update node status
	nodeStatus := domain.NodeExecutionStatus{
		NodeID:       cb.NodeID,
		Status:       domain.NodeExecStatus(cb.Status),
		StartedAt:    cb.StartedAt,
		CompletedAt:  cb.CompletedAt,
		ErrorMessage: cb.ErrorMessage,
		OutputData:   cb.OutputSummary,
	}
	if err := s.repo.UpdateNodeStatus(ctx, cb.RunID, cb.NodeID, nodeStatus); err != nil {
		return err
	}

	// Notify WebSocket listeners
	if s.notifier != nil {
		s.notifier(cb.WorkflowID, cb)
	}

	return nil
}
