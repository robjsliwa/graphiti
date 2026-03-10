package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"graphiti/internal/app"
	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

// fakeExecRepo is a minimal in-memory execution repository for handler tests.
type fakeExecRepo struct {
	runs map[string]*domain.ExecutionRun
}

func newHandlerFakeExecRepo() *fakeExecRepo {
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
func (r *fakeExecRepo) ListByWorkflow(_ context.Context, workflowID string, _ driven.ExecutionFilter) ([]*domain.ExecutionRunSummary, error) {
	var result []*domain.ExecutionRunSummary
	for _, run := range r.runs {
		if run.WorkflowID == workflowID {
			result = append(result, &domain.ExecutionRunSummary{
				ID: run.ID, WorkflowID: run.WorkflowID, Status: run.Status,
				StartedAt: run.StartedAt, CompletedAt: run.CompletedAt,
			})
		}
	}
	return result, nil
}
func (r *fakeExecRepo) UpdateNodeStatus(_ context.Context, runID, nodeID string, status domain.NodeExecutionStatus) error {
	run := r.runs[runID]
	if run == nil {
		return domain.ErrNotFound
	}
	if run.NodeStatuses == nil {
		run.NodeStatuses = make(map[string]*domain.NodeExecutionStatus)
	}
	run.NodeStatuses[nodeID] = &status
	return nil
}
func (r *fakeExecRepo) AppendNodeLog(_ context.Context, _, _ string, _ domain.LogEntry) error {
	return nil
}
func (r *fakeExecRepo) UpdateRunStatus(_ context.Context, runID string, status domain.ExecutionStatus) error {
	run := r.runs[runID]
	if run == nil {
		return domain.ErrNotFound
	}
	run.Status = status
	return nil
}

func TestHandleExecutionCallback_Success(t *testing.T) {
	repo := newHandlerFakeExecRepo()
	execSvc := app.NewExecutionService(repo)
	hub := NewWebSocketHub()
	handler := handleExecutionCallback(execSvc, hub, "")

	callback := app.ExecutionCallback{
		APIVersion: "graphiti/v1",
		Event:      "execution.node_status",
		RunID:      "run-1",
		WorkflowID: "wf-1",
		NodeID:     "node-1",
		Status:     "running",
		StartedAt:  time.Now(),
	}
	body, _ := json.Marshal(callback)

	req := httptest.NewRequest("POST", "/api/callbacks/execution", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}

	// Give the async goroutine time to process
	time.Sleep(50 * time.Millisecond)

	// Verify run was created
	run, err := repo.GetByID(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("run should exist: %v", err)
	}
	if run.Status != domain.ExecStatusRunning {
		t.Errorf("run status = %q", run.Status)
	}
}

func TestHandleExecutionCallback_InvalidJSON(t *testing.T) {
	execSvc := app.NewExecutionService(newHandlerFakeExecRepo())
	hub := NewWebSocketHub()
	handler := handleExecutionCallback(execSvc, hub, "")

	req := httptest.NewRequest("POST", "/api/callbacks/execution", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleExecutionCallback_MissingFields(t *testing.T) {
	execSvc := app.NewExecutionService(newHandlerFakeExecRepo())
	hub := NewWebSocketHub()
	handler := handleExecutionCallback(execSvc, hub, "")

	callback := app.ExecutionCallback{
		APIVersion: "graphiti/v1",
		// Missing RunID, WorkflowID, NodeID
	}
	body, _ := json.Marshal(callback)

	req := httptest.NewRequest("POST", "/api/callbacks/execution", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleListRuns(t *testing.T) {
	repo := newHandlerFakeExecRepo()
	execSvc := app.NewExecutionService(repo)
	ctx := context.Background()

	repo.Create(ctx, &domain.ExecutionRun{
		ID: "run-1", WorkflowID: "wf-1", Status: domain.ExecStatusCompleted,
		StartedAt: time.Now(), NodeStatuses: map[string]*domain.NodeExecutionStatus{},
	})

	handler := handleListRuns(execSvc)
	req := httptest.NewRequest("GET", "/api/workflows/wf-1/runs", nil)
	req.SetPathValue("id", "wf-1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleGetRun(t *testing.T) {
	repo := newHandlerFakeExecRepo()
	execSvc := app.NewExecutionService(repo)
	ctx := context.Background()

	repo.Create(ctx, &domain.ExecutionRun{
		ID: "run-1", WorkflowID: "wf-1", Status: domain.ExecStatusRunning,
		NodeStatuses: map[string]*domain.NodeExecutionStatus{},
	})

	handler := handleGetRun(execSvc)
	req := httptest.NewRequest("GET", "/api/runs/run-1", nil)
	req.SetPathValue("runId", "run-1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleGetRun_NotFound(t *testing.T) {
	execSvc := app.NewExecutionService(newHandlerFakeExecRepo())

	handler := handleGetRun(execSvc)
	req := httptest.NewRequest("GET", "/api/runs/nope", nil)
	req.SetPathValue("runId", "nope")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}
