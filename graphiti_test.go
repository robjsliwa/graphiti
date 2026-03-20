package graphiti

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"graphiti/internal/adapters/driven/auth"
	"graphiti/internal/adapters/driven/memory"
	"graphiti/internal/domain"
)

func TestNew_MinimalConfig(t *testing.T) {
	app, err := New(Config{}, Deps{
		WorkflowRepo: memory.NewWorkflowRepo(),
		UserRepo:     memory.NewUserRepo(),
		AuthProvider: auth.NewFakeAuth(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app == nil {
		t.Fatal("app is nil")
	}
	if app.Handler() == nil {
		t.Fatal("handler is nil")
	}
}

func TestNew_MissingRequiredDeps(t *testing.T) {
	tests := []struct {
		name string
		deps Deps
	}{
		{"no workflow repo", Deps{UserRepo: memory.NewUserRepo(), AuthProvider: auth.NewFakeAuth()}},
		{"no user repo", Deps{WorkflowRepo: memory.NewWorkflowRepo(), AuthProvider: auth.NewFakeAuth()}},
		{"no auth provider", Deps{WorkflowRepo: memory.NewWorkflowRepo(), UserRepo: memory.NewUserRepo()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(Config{}, tt.deps)
			if err == nil {
				t.Fatal("expected error for missing dependency")
			}
		})
	}
}

func TestNew_DefaultConfig(t *testing.T) {
	app, err := New(Config{}, Deps{
		WorkflowRepo: memory.NewWorkflowRepo(),
		UserRepo:     memory.NewUserRepo(),
		AuthProvider: auth.NewFakeAuth(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default BasePath should be "/"
	if app.config.BasePath != "/" {
		t.Errorf("default BasePath = %q, want %q", app.config.BasePath, "/")
	}
	// Default CommandHistoryMaxDepth should be 100
	if app.config.CommandHistoryMaxDepth != 100 {
		t.Errorf("default CommandHistoryMaxDepth = %d, want 100", app.config.CommandHistoryMaxDepth)
	}
}

func TestApp_Handler_ServesStaticAssets(t *testing.T) {
	app, err := New(Config{}, Deps{
		WorkflowRepo: memory.NewWorkflowRepo(),
		UserRepo:     memory.NewUserRepo(),
		AuthProvider: auth.NewFakeAuth(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Static CSS should be served from embedded FS
	req := httptest.NewRequest("GET", "/static/css/base.css", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /static/css/base.css status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestApp_Handler_AuthRedirect(t *testing.T) {
	app, err := New(Config{}, Deps{
		WorkflowRepo: memory.NewWorkflowRepo(),
		UserRepo:     memory.NewUserRepo(),
		AuthProvider: auth.NewFakeAuth(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unauthenticated request to / should redirect to login
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("GET / unauthenticated status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != "/auth/login" {
		t.Errorf("redirect location = %q, want %q", loc, "/auth/login")
	}
}

func TestApp_Services(t *testing.T) {
	app, err := New(Config{}, Deps{
		WorkflowRepo:  memory.NewWorkflowRepo(),
		UserRepo:      memory.NewUserRepo(),
		AuthProvider:  auth.NewFakeAuth(),
		ExecutionRepo: memory.NewExecutionRepository(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc := app.Services()
	if svc == nil {
		t.Fatal("services is nil")
	}
	if svc.Workflows == nil {
		t.Fatal("workflows service is nil")
	}
	if svc.NodeRegistry == nil {
		t.Fatal("node registry service is nil")
	}
	if svc.Executions == nil {
		t.Fatal("execution service is nil")
	}
}

func TestApp_Services_CreateWorkflow(t *testing.T) {
	app, err := New(Config{}, Deps{
		WorkflowRepo: memory.NewWorkflowRepo(),
		UserRepo:     memory.NewUserRepo(),
		AuthProvider: auth.NewFakeAuth(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	wf, err := app.Services().Workflows.CreateWorkflow(ctx, "Test Pipeline", "A test", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wf.Name != "Test Pipeline" {
		t.Errorf("got name %q, want %q", wf.Name, "Test Pipeline")
	}
}

func TestApp_WebSocketHub(t *testing.T) {
	app, err := New(Config{}, Deps{
		WorkflowRepo:  memory.NewWorkflowRepo(),
		UserRepo:      memory.NewUserRepo(),
		AuthProvider:  auth.NewFakeAuth(),
		ExecutionRepo: memory.NewExecutionRepository(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hub := app.WebSocketHub()
	if hub == nil {
		t.Fatal("websocket hub is nil")
	}
}

func TestApp_ReportNodeStatus(t *testing.T) {
	execRepo := memory.NewExecutionRepository()
	// Pre-create a run so UpdateNodeStatus doesn't fail
	execRepo.Create(context.Background(), &domain.ExecutionRun{
		ID:           "run-001",
		WorkflowID:   "wf-001",
		Status:       domain.ExecStatusRunning,
		StartedAt:    time.Now(),
		NodeStatuses: make(map[string]*domain.NodeExecutionStatus),
	})

	app, err := New(Config{}, Deps{
		WorkflowRepo:  memory.NewWorkflowRepo(),
		UserRepo:      memory.NewUserRepo(),
		AuthProvider:  auth.NewFakeAuth(),
		ExecutionRepo: execRepo,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now()
	err = app.ReportNodeStatus(context.Background(), ExecutionUpdate{
		RunID:      "run-001",
		WorkflowID: "wf-001",
		NodeID:     "node-001",
		Status:     "completed",
		CompletedAt: &now,
		OutputSummary: map[string]any{"rows": 42},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the status was persisted
	run, err := execRepo.GetByID(context.Background(), "run-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns, ok := run.NodeStatuses["node-001"]
	if !ok {
		t.Fatal("node status not found")
	}
	if ns.Status != domain.NodeExecCompleted {
		t.Errorf("got status %q, want %q", ns.Status, domain.NodeExecCompleted)
	}
}

func TestApp_ReportNodeStatus_NilExecutionRepo(t *testing.T) {
	app, err := New(Config{}, Deps{
		WorkflowRepo: memory.NewWorkflowRepo(),
		UserRepo:     memory.NewUserRepo(),
		AuthProvider: auth.NewFakeAuth(),
		// ExecutionRepo intentionally nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not error — just broadcasts without persisting
	err = app.ReportNodeStatus(context.Background(), ExecutionUpdate{
		RunID:      "run-001",
		WorkflowID: "wf-001",
		NodeID:     "node-001",
		Status:     "running",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApp_ReportNodeStatus_InvalidStatus(t *testing.T) {
	app, err := New(Config{}, Deps{
		WorkflowRepo: memory.NewWorkflowRepo(),
		UserRepo:     memory.NewUserRepo(),
		AuthProvider: auth.NewFakeAuth(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = app.ReportNodeStatus(context.Background(), ExecutionUpdate{
		RunID:      "run-001",
		WorkflowID: "wf-001",
		NodeID:     "node-001",
		Status:     "bogus",
	})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	if !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("expected error to mention 'invalid status', got: %v", err)
	}
}

func TestNew_LoadsEmbeddedNodeDefinitions(t *testing.T) {
	app, err := New(Config{}, Deps{
		WorkflowRepo: memory.NewWorkflowRepo(),
		UserRepo:     memory.NewUserRepo(),
		AuthProvider: auth.NewFakeAuth(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	defs, err := app.Services().NodeRegistry.GetAllDefinitions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have loaded the embedded default node definitions
	if len(defs) == 0 {
		t.Fatal("expected embedded node definitions to be loaded, got 0")
	}

	// Check for a known definition
	found := false
	for _, d := range defs {
		if d.ID == "source-twilio" {
			found = true
			break
		}
	}
	if !found {
		names := make([]string, len(defs))
		for i, d := range defs {
			names[i] = d.ID
		}
		t.Errorf("expected to find source-twilio in definitions, got: %v", names)
	}
}

func TestNew_CustomNodeDefinitionsPath(t *testing.T) {
	app, err := New(Config{
		NodeDefinitionsPath: "config/nodes",
	}, Deps{
		WorkflowRepo: memory.NewWorkflowRepo(),
		UserRepo:     memory.NewUserRepo(),
		AuthProvider: auth.NewFakeAuth(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	defs, err := app.Services().NodeRegistry.GetAllDefinitions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) == 0 {
		t.Fatal("expected node definitions from custom path")
	}
}

func TestUserContextKey_Type(t *testing.T) {
	// UserContextKey should be usable as a context key
	if fmt.Sprintf("%v", UserContextKey) == "" {
		t.Fatal("UserContextKey should have a string representation")
	}
}
