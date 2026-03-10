package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

const execMigrationSQL = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    email TEXT,
    avatar_url TEXT,
    auth_provider TEXT NOT NULL,
    auth_provider_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_login_at DATETIME,
    UNIQUE(auth_provider, auth_provider_id)
);
CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    definition JSON NOT NULL,
    status TEXT DEFAULT 'draft',
    version INTEGER DEFAULT 1,
    created_by TEXT REFERENCES users(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS execution_runs (
    id TEXT PRIMARY KEY,
    workflow_id TEXT REFERENCES workflows(id),
    workflow_version INTEGER NOT NULL,
    status TEXT DEFAULT 'pending',
    started_at DATETIME,
    completed_at DATETIME,
    trigger_type TEXT,
    trigger_data JSON
);
CREATE TABLE IF NOT EXISTS execution_node_statuses (
    id TEXT PRIMARY KEY,
    run_id TEXT REFERENCES execution_runs(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    started_at DATETIME,
    completed_at DATETIME,
    input_data JSON,
    output_data JSON,
    error_message TEXT,
    logs JSON
);
`

func setupExecTestDB(t *testing.T) *ExecutionRepository {
	t.Helper()
	db, err := NewDBWithMigration(":memory:", execMigrationSQL)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Insert a test workflow (needed for FK)
	_, err = db.Exec(`INSERT INTO users (id, username, auth_provider, auth_provider_id) VALUES ('u1', 'test', 'fake', 'f1')`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	_, err = db.Exec(`INSERT INTO workflows (id, name, definition, created_by) VALUES ('wf-1', 'Test', '{}', 'u1')`)
	if err != nil {
		t.Fatalf("insert workflow: %v", err)
	}

	return NewExecutionRepository(db)
}

func TestExecutionRepository_ImplementsInterface(t *testing.T) {
	var _ driven.ExecutionRepository = (*ExecutionRepository)(nil)
}

func TestExecutionRepository_CreateAndGetByID(t *testing.T) {
	repo := setupExecTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	run := &domain.ExecutionRun{
		ID:              "run-1",
		WorkflowID:      "wf-1",
		WorkflowVersion: 1,
		Status:          domain.ExecStatusRunning,
		StartedAt:       now,
		TriggerType:     "webhook",
		TriggerData:     map[string]any{"source": "test"},
		NodeStatuses:    map[string]*domain.NodeExecutionStatus{},
	}

	if err := repo.Create(ctx, run); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	got, err := repo.GetByID(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if got.ID != "run-1" {
		t.Errorf("ID = %q, want %q", got.ID, "run-1")
	}
	if got.WorkflowID != "wf-1" {
		t.Errorf("WorkflowID = %q", got.WorkflowID)
	}
	if got.Status != domain.ExecStatusRunning {
		t.Errorf("Status = %q", got.Status)
	}
	if got.TriggerType != "webhook" {
		t.Errorf("TriggerType = %q", got.TriggerType)
	}
}

func TestExecutionRepository_GetByID_NotFound(t *testing.T) {
	repo := setupExecTestDB(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent run")
	}
}

func TestExecutionRepository_ListByWorkflow(t *testing.T) {
	repo := setupExecTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	for i, status := range []domain.ExecutionStatus{domain.ExecStatusCompleted, domain.ExecStatusRunning, domain.ExecStatusFailed} {
		run := &domain.ExecutionRun{
			ID:              fmt.Sprintf("run-%d", i+1),
			WorkflowID:      "wf-1",
			WorkflowVersion: 1,
			Status:          status,
			StartedAt:       now.Add(time.Duration(i) * time.Minute),
			NodeStatuses:    map[string]*domain.NodeExecutionStatus{},
		}
		if err := repo.Create(ctx, run); err != nil {
			t.Fatalf("Create run-%d error: %v", i+1, err)
		}
	}

	runs, err := repo.ListByWorkflow(ctx, "wf-1", driven.ExecutionFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListByWorkflow error: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(runs))
	}
	// Should be newest first
	if runs[0].ID != "run-3" {
		t.Errorf("expected newest first, got %s", runs[0].ID)
	}
}

func TestExecutionRepository_ListByWorkflow_WithStatusFilter(t *testing.T) {
	repo := setupExecTestDB(t)
	ctx := context.Background()

	for _, status := range []domain.ExecutionStatus{domain.ExecStatusCompleted, domain.ExecStatusRunning} {
		run := &domain.ExecutionRun{
			ID:              "run-" + string(status),
			WorkflowID:      "wf-1",
			WorkflowVersion: 1,
			Status:          status,
			NodeStatuses:    map[string]*domain.NodeExecutionStatus{},
		}
		repo.Create(ctx, run)
	}

	runs, err := repo.ListByWorkflow(ctx, "wf-1", driven.ExecutionFilter{Status: domain.ExecStatusRunning, Limit: 10})
	if err != nil {
		t.Fatalf("ListByWorkflow error: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 running run, got %d", len(runs))
	}
}

func TestExecutionRepository_UpdateNodeStatus(t *testing.T) {
	repo := setupExecTestDB(t)
	ctx := context.Background()

	run := &domain.ExecutionRun{
		ID:              "run-1",
		WorkflowID:      "wf-1",
		WorkflowVersion: 1,
		Status:          domain.ExecStatusRunning,
		NodeStatuses:    map[string]*domain.NodeExecutionStatus{},
	}
	repo.Create(ctx, run)

	now := time.Now().UTC().Truncate(time.Second)
	status := domain.NodeExecutionStatus{
		NodeID:    "node-1",
		Status:    domain.NodeExecRunning,
		StartedAt: now,
	}
	if err := repo.UpdateNodeStatus(ctx, "run-1", "node-1", status); err != nil {
		t.Fatalf("UpdateNodeStatus error: %v", err)
	}

	// Retrieve and check
	got, err := repo.GetByID(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	ns, ok := got.NodeStatuses["node-1"]
	if !ok {
		t.Fatal("node-1 status not found")
	}
	if ns.Status != domain.NodeExecRunning {
		t.Errorf("node status = %q, want %q", ns.Status, domain.NodeExecRunning)
	}

	// Update again (completed)
	status2 := domain.NodeExecutionStatus{
		NodeID:      "node-1",
		Status:      domain.NodeExecCompleted,
		StartedAt:   now,
		CompletedAt: now.Add(2 * time.Second),
		OutputData:  map[string]any{"result": "ok"},
	}
	if err := repo.UpdateNodeStatus(ctx, "run-1", "node-1", status2); err != nil {
		t.Fatalf("UpdateNodeStatus (completed) error: %v", err)
	}

	got, _ = repo.GetByID(ctx, "run-1")
	ns = got.NodeStatuses["node-1"]
	if ns.Status != domain.NodeExecCompleted {
		t.Errorf("node status = %q after update", ns.Status)
	}
}

func TestExecutionRepository_AppendNodeLog(t *testing.T) {
	repo := setupExecTestDB(t)
	ctx := context.Background()

	run := &domain.ExecutionRun{
		ID: "run-1", WorkflowID: "wf-1", WorkflowVersion: 1,
		Status: domain.ExecStatusRunning, NodeStatuses: map[string]*domain.NodeExecutionStatus{},
	}
	repo.Create(ctx, run)

	// Create node status first
	repo.UpdateNodeStatus(ctx, "run-1", "node-1", domain.NodeExecutionStatus{
		NodeID: "node-1", Status: domain.NodeExecRunning,
	})

	entry := domain.LogEntry{
		Timestamp: time.Now().UTC().Truncate(time.Second),
		Level:     "info",
		Message:   "processing started",
	}
	if err := repo.AppendNodeLog(ctx, "run-1", "node-1", entry); err != nil {
		t.Fatalf("AppendNodeLog error: %v", err)
	}

	got, _ := repo.GetByID(ctx, "run-1")
	ns := got.NodeStatuses["node-1"]
	if len(ns.Logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(ns.Logs))
	}
	if ns.Logs[0].Message != "processing started" {
		t.Errorf("log message = %q", ns.Logs[0].Message)
	}
}

func TestExecutionRepository_UpdateRunStatus(t *testing.T) {
	repo := setupExecTestDB(t)
	ctx := context.Background()

	run := &domain.ExecutionRun{
		ID: "run-1", WorkflowID: "wf-1", WorkflowVersion: 1,
		Status: domain.ExecStatusRunning, NodeStatuses: map[string]*domain.NodeExecutionStatus{},
	}
	repo.Create(ctx, run)

	if err := repo.UpdateRunStatus(ctx, "run-1", domain.ExecStatusCompleted); err != nil {
		t.Fatalf("UpdateRunStatus error: %v", err)
	}

	got, _ := repo.GetByID(ctx, "run-1")
	if got.Status != domain.ExecStatusCompleted {
		t.Errorf("status = %q, want %q", got.Status, domain.ExecStatusCompleted)
	}
}
