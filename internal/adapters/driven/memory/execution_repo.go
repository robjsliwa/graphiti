package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

// ExecutionRepository is an in-memory implementation of driven.ExecutionRepository.
type ExecutionRepository struct {
	mu   sync.RWMutex
	runs map[string]*domain.ExecutionRun
}

// NewExecutionRepository creates a new in-memory execution repository.
func NewExecutionRepository() *ExecutionRepository {
	return &ExecutionRepository{runs: make(map[string]*domain.ExecutionRun)}
}

var _ driven.ExecutionRepository = (*ExecutionRepository)(nil)

func (r *ExecutionRepository) Create(_ context.Context, run *domain.ExecutionRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs[run.ID] = run
	return nil
}

func (r *ExecutionRepository) GetByID(_ context.Context, id string) (*domain.ExecutionRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run, ok := r.runs[id]
	if !ok {
		return nil, fmt.Errorf("execution run %s: not found", id)
	}
	return run, nil
}

func (r *ExecutionRepository) ListByWorkflow(_ context.Context, workflowID string, filter driven.ExecutionFilter) ([]*domain.ExecutionRunSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.ExecutionRunSummary
	for _, run := range r.runs {
		if run.WorkflowID != workflowID {
			continue
		}
		if filter.Status != "" && run.Status != filter.Status {
			continue
		}
		result = append(result, &domain.ExecutionRunSummary{
			ID: run.ID, WorkflowID: run.WorkflowID, Status: run.Status,
			StartedAt: run.StartedAt, CompletedAt: run.CompletedAt, TriggerType: run.TriggerType,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.After(result[j].StartedAt)
	})

	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

func (r *ExecutionRepository) UpdateNodeStatus(_ context.Context, runID, nodeID string, status domain.NodeExecutionStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[runID]
	if !ok {
		return fmt.Errorf("execution run %s: not found", runID)
	}
	if run.NodeStatuses == nil {
		run.NodeStatuses = make(map[string]*domain.NodeExecutionStatus)
	}
	run.NodeStatuses[nodeID] = &status
	return nil
}

func (r *ExecutionRepository) AppendNodeLog(_ context.Context, runID, nodeID string, entry domain.LogEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[runID]
	if !ok {
		return fmt.Errorf("execution run %s: not found", runID)
	}
	ns, ok := run.NodeStatuses[nodeID]
	if !ok {
		return fmt.Errorf("node %s: not found in run %s", nodeID, runID)
	}
	ns.Logs = append(ns.Logs, entry)
	return nil
}

func (r *ExecutionRepository) UpdateRunStatus(_ context.Context, runID string, status domain.ExecutionStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[runID]
	if !ok {
		return fmt.Errorf("execution run %s: not found", runID)
	}
	run.Status = status
	if status == domain.ExecStatusCompleted || status == domain.ExecStatusFailed || status == domain.ExecStatusCancelled {
		run.CompletedAt = time.Now()
	}
	return nil
}
