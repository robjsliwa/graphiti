package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

// WorkflowRepo is an in-memory implementation of WorkflowRepository for tests.
type WorkflowRepo struct {
	mu        sync.RWMutex
	workflows map[string]*domain.Workflow
	versions  map[string][]*domain.WorkflowVersion
}

// NewWorkflowRepo creates a new in-memory workflow repository.
func NewWorkflowRepo() *WorkflowRepo {
	return &WorkflowRepo{
		workflows: make(map[string]*domain.Workflow),
		versions:  make(map[string][]*domain.WorkflowVersion),
	}
}

func (r *WorkflowRepo) Create(_ context.Context, wf *domain.Workflow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.workflows[wf.ID]; exists {
		return fmt.Errorf("workflow %s already exists", wf.ID)
	}
	r.workflows[wf.ID] = deepCopyWorkflow(wf)
	return nil
}

func (r *WorkflowRepo) GetByID(_ context.Context, id string) (*domain.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wf, ok := r.workflows[id]
	if !ok {
		return nil, fmt.Errorf("workflow %s not found", id)
	}
	return deepCopyWorkflow(wf), nil
}

func (r *WorkflowRepo) List(_ context.Context, filter driven.WorkflowFilter) ([]*domain.WorkflowSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*domain.WorkflowSummary
	for _, wf := range r.workflows {
		if filter.UserID != "" && wf.CreatedBy != filter.UserID {
			continue
		}
		if filter.Status != "" && wf.Status != filter.Status {
			continue
		}
		result = append(result, &domain.WorkflowSummary{
			ID:        wf.ID,
			Name:      wf.Name,
			Status:    wf.Status,
			Version:   wf.Version,
			UpdatedAt: wf.UpdatedAt,
		})
	}
	return result, nil
}

func (r *WorkflowRepo) Update(_ context.Context, wf *domain.Workflow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.workflows[wf.ID]; !exists {
		return fmt.Errorf("workflow %s not found", wf.ID)
	}
	r.workflows[wf.ID] = deepCopyWorkflow(wf)
	return nil
}

func (r *WorkflowRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.workflows, id)
	delete(r.versions, id)
	return nil
}

func (r *WorkflowRepo) GetVersionHistory(_ context.Context, id string) ([]*domain.WorkflowVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.versions[id], nil
}

// deepCopyWorkflow creates a deep copy via JSON round-trip.
func deepCopyWorkflow(wf *domain.Workflow) *domain.Workflow {
	data, _ := json.Marshal(wf)
	var copy domain.Workflow
	json.Unmarshal(data, &copy)
	return &copy
}
