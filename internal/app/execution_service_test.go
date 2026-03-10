package app

import (
	"context"
	"testing"
	"time"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

// fakeExecRepo is an in-memory execution repository for tests.
type fakeExecRepo struct {
	runs map[string]*domain.ExecutionRun
}

func newFakeExecRepo() *fakeExecRepo {
	return &fakeExecRepo{runs: make(map[string]*domain.ExecutionRun)}
}

func (r *fakeExecRepo) Create(_ context.Context, run *domain.ExecutionRun) error {
	r.runs[run.ID] = run
	return nil
}

func (r *fakeExecRepo) GetByID(_ context.Context, id string) (*domain.ExecutionRun, error) {
	run, ok := r.runs[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return run, nil
}

func (r *fakeExecRepo) ListByWorkflow(_ context.Context, workflowID string, filter driven.ExecutionFilter) ([]*domain.ExecutionRunSummary, error) {
	var result []*domain.ExecutionRunSummary
	for _, run := range r.runs {
		if run.WorkflowID != workflowID {
			continue
		}
		if filter.Status != "" && run.Status != filter.Status {
			continue
		}
		result = append(result, &domain.ExecutionRunSummary{
			ID:          run.ID,
			WorkflowID:  run.WorkflowID,
			Status:      run.Status,
			StartedAt:   run.StartedAt,
			CompletedAt: run.CompletedAt,
			TriggerType: run.TriggerType,
		})
	}
	return result, nil
}

func (r *fakeExecRepo) UpdateNodeStatus(_ context.Context, runID, nodeID string, status domain.NodeExecutionStatus) error {
	run, ok := r.runs[runID]
	if !ok {
		return domain.ErrNotFound
	}
	if run.NodeStatuses == nil {
		run.NodeStatuses = make(map[string]*domain.NodeExecutionStatus)
	}
	run.NodeStatuses[nodeID] = &status
	return nil
}

func (r *fakeExecRepo) AppendNodeLog(_ context.Context, runID, nodeID string, entry domain.LogEntry) error {
	run, ok := r.runs[runID]
	if !ok {
		return domain.ErrNotFound
	}
	ns, ok := run.NodeStatuses[nodeID]
	if !ok {
		return domain.ErrNotFound
	}
	ns.Logs = append(ns.Logs, entry)
	return nil
}

func (r *fakeExecRepo) UpdateRunStatus(_ context.Context, runID string, status domain.ExecutionStatus) error {
	run, ok := r.runs[runID]
	if !ok {
		return domain.ErrNotFound
	}
	run.Status = status
	if status == domain.ExecStatusCompleted || status == domain.ExecStatusFailed {
		run.CompletedAt = time.Now()
	}
	return nil
}

func TestExecutionService_GetRun(t *testing.T) {
	repo := newFakeExecRepo()
	svc := NewExecutionService(repo)
	ctx := context.Background()

	run := &domain.ExecutionRun{
		ID:         "run-1",
		WorkflowID: "wf-1",
		Status:     domain.ExecStatusRunning,
	}
	repo.Create(ctx, run)

	got, err := svc.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetRun error: %v", err)
	}
	if got.ID != "run-1" {
		t.Errorf("ID = %q", got.ID)
	}
}

func TestExecutionService_GetRun_NotFound(t *testing.T) {
	svc := NewExecutionService(newFakeExecRepo())
	_, err := svc.GetRun(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExecutionService_ListRuns(t *testing.T) {
	repo := newFakeExecRepo()
	svc := NewExecutionService(repo)
	ctx := context.Background()

	repo.Create(ctx, &domain.ExecutionRun{ID: "run-1", WorkflowID: "wf-1", Status: domain.ExecStatusCompleted})
	repo.Create(ctx, &domain.ExecutionRun{ID: "run-2", WorkflowID: "wf-1", Status: domain.ExecStatusRunning})
	repo.Create(ctx, &domain.ExecutionRun{ID: "run-3", WorkflowID: "wf-2", Status: domain.ExecStatusRunning})

	runs, err := svc.ListRuns(ctx, "wf-1")
	if err != nil {
		t.Fatalf("ListRuns error: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
}

func TestExecutionService_UpdateNodeStatus(t *testing.T) {
	repo := newFakeExecRepo()
	svc := NewExecutionService(repo)
	ctx := context.Background()

	// Create a run
	repo.Create(ctx, &domain.ExecutionRun{
		ID: "run-1", WorkflowID: "wf-1", Status: domain.ExecStatusRunning,
		NodeStatuses: map[string]*domain.NodeExecutionStatus{},
	})

	status := domain.NodeExecutionStatus{
		NodeID: "node-1",
		Status: domain.NodeExecCompleted,
	}
	err := svc.UpdateNodeStatus(ctx, "run-1", status)
	if err != nil {
		t.Fatalf("UpdateNodeStatus error: %v", err)
	}

	run, _ := repo.GetByID(ctx, "run-1")
	ns, ok := run.NodeStatuses["node-1"]
	if !ok {
		t.Fatal("node status not found")
	}
	if ns.Status != domain.NodeExecCompleted {
		t.Errorf("status = %q", ns.Status)
	}
}

func TestExecutionService_ProcessCallback_CreatesRunIfMissing(t *testing.T) {
	repo := newFakeExecRepo()
	svc := NewExecutionService(repo)
	ctx := context.Background()

	callback := ExecutionCallback{
		APIVersion: "graphiti/v1",
		Event:      "execution.node_status",
		RunID:      "new-run",
		WorkflowID: "wf-1",
		NodeID:     "node-1",
		Status:     "running",
		StartedAt:  time.Now(),
	}

	err := svc.ProcessCallback(ctx, callback)
	if err != nil {
		t.Fatalf("ProcessCallback error: %v", err)
	}

	run, err := repo.GetByID(ctx, "new-run")
	if err != nil {
		t.Fatalf("run should have been created: %v", err)
	}
	if run.Status != domain.ExecStatusRunning {
		t.Errorf("status = %q, want running", run.Status)
	}
}

func TestExecutionService_ProcessCallback_UpdatesExistingRun(t *testing.T) {
	repo := newFakeExecRepo()
	svc := NewExecutionService(repo)
	ctx := context.Background()

	repo.Create(ctx, &domain.ExecutionRun{
		ID: "run-1", WorkflowID: "wf-1", Status: domain.ExecStatusRunning,
		NodeStatuses: map[string]*domain.NodeExecutionStatus{},
	})

	callback := ExecutionCallback{
		APIVersion: "graphiti/v1",
		Event:      "execution.node_status",
		RunID:      "run-1",
		WorkflowID: "wf-1",
		NodeID:     "node-1",
		Status:     "completed",
	}

	err := svc.ProcessCallback(ctx, callback)
	if err != nil {
		t.Fatalf("ProcessCallback error: %v", err)
	}

	run, _ := repo.GetByID(ctx, "run-1")
	ns := run.NodeStatuses["node-1"]
	if ns.Status != domain.NodeExecCompleted {
		t.Errorf("node status = %q", ns.Status)
	}
}
