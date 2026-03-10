package app

import (
	"context"
	"testing"

	"graphiti/internal/adapters/driven/memory"
	"graphiti/internal/domain"
)

func TestWorkflowService_DeployWorkflow_Success(t *testing.T) {
	svc, _ := setupWorkflowService()
	ctx := context.Background()

	wf, _ := svc.CreateWorkflow(ctx, "Deploy Test", "desc", "user-1")

	// Set up a fake deploy target
	fakeTarget := &fakeDeployTarget{success: true, runID: "run-123"}
	svc.SetDeployTarget(fakeTarget)

	result, err := svc.DeployWorkflow(ctx, wf.ID, "production", "user-1")
	if err != nil {
		t.Fatalf("deploy failed: %v", err)
	}
	if !result.Success {
		t.Error("expected deploy success")
	}
	if result.RunID != "run-123" {
		t.Errorf("expected runID 'run-123', got %q", result.RunID)
	}

	// Verify workflow version was incremented
	updated, _ := svc.GetWorkflow(ctx, wf.ID)
	if updated.Version != 2 {
		t.Errorf("expected version 2, got %d", updated.Version)
	}
	if updated.Status != domain.WorkflowStatusDeployed {
		t.Errorf("expected status 'deployed', got %q", updated.Status)
	}
}

func TestWorkflowService_DeployWorkflow_CreatesVersionRecord(t *testing.T) {
	repo := memory.NewWorkflowRepo()
	defRepo := &fakeNodeDefRepo{defs: []*domain.NodeDefinition{
		{
			ID: "src", Name: "Source",
			Category: domain.Category{Group: "Sources"},
			Shape:    domain.Shape{Type: domain.ShapeRoundedRect},
			Outputs:  []domain.PortDefinition{{ID: "out", Type: domain.PortTypeData, MaxConnections: -1}},
		},
	}}
	nodeRegistry := NewNodeRegistry(defRepo)
	nodeRegistry.Load(context.Background())
	svc := NewWorkflowService(repo, nodeRegistry, 100)

	ctx := context.Background()
	wf, _ := svc.CreateWorkflow(ctx, "Version Test", "", "user-1")

	fakeTarget := &fakeDeployTarget{success: true}
	svc.SetDeployTarget(fakeTarget)

	// Deploy
	svc.DeployWorkflow(ctx, wf.ID, "staging", "user-1")

	// Check version history
	versions, err := repo.GetVersionHistory(ctx, wf.ID)
	if err != nil {
		t.Fatalf("get versions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}
	if versions[0].Version != 1 {
		t.Errorf("expected version 1 in history, got %d", versions[0].Version)
	}
	if versions[0].DeployedBy != "user-1" {
		t.Errorf("expected deployed_by 'user-1', got %q", versions[0].DeployedBy)
	}
}

func TestWorkflowService_DeployWorkflow_ValidationFails(t *testing.T) {
	svc, registry := setupWorkflowService()
	ctx := context.Background()

	wf, _ := svc.CreateWorkflow(ctx, "Invalid WF", "", "user-1")

	// Add a node with a required attribute that's missing
	def, _ := registry.GetByID(ctx, "src")
	// Modify the definition to have a required attribute
	defWithRequired := *def
	defWithRequired.Attributes = []domain.AttributeDefinition{
		{ID: "url", Label: "URL", Required: true},
	}
	addCmd := &testAddNodeCmdWithDef{def: &defWithRequired, id: "n1", x: 100, y: 100}
	svc.ExecuteCommand(ctx, wf.ID, addCmd)

	fakeTarget := &fakeDeployTarget{success: true}
	svc.SetDeployTarget(fakeTarget)

	result, err := svc.DeployWorkflow(ctx, wf.ID, "production", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected deploy to fail validation")
	}
	if fakeTarget.called {
		t.Error("deploy target should not have been called when validation fails")
	}
}

func TestWorkflowService_DeployWorkflow_NoTarget(t *testing.T) {
	svc, _ := setupWorkflowService()
	ctx := context.Background()
	wf, _ := svc.CreateWorkflow(ctx, "No Target", "", "user-1")

	_, err := svc.DeployWorkflow(ctx, wf.ID, "production", "user-1")
	if err == nil {
		t.Error("expected error when no deploy target is configured")
	}
}

func TestWorkflowService_ExportWorkflow_JSON(t *testing.T) {
	svc, _ := setupWorkflowService()
	ctx := context.Background()
	wf, _ := svc.CreateWorkflow(ctx, "Export Test", "desc", "user-1")

	data, err := svc.ExportWorkflow(ctx, wf.ID, "json")
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty export data")
	}
}

func TestWorkflowService_ExportWorkflow_YAML(t *testing.T) {
	svc, _ := setupWorkflowService()
	ctx := context.Background()
	wf, _ := svc.CreateWorkflow(ctx, "Export YAML Test", "desc", "user-1")

	data, err := svc.ExportWorkflow(ctx, wf.ID, "yaml")
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty export data")
	}
	// YAML should start with meaningful content (not JSON braces)
	if data[0] == '{' {
		t.Error("YAML export should not start with '{'")
	}
}

func TestWorkflowService_GetVersionHistory(t *testing.T) {
	repo := memory.NewWorkflowRepo()
	defRepo := &fakeNodeDefRepo{defs: []*domain.NodeDefinition{}}
	nodeRegistry := NewNodeRegistry(defRepo)
	nodeRegistry.Load(context.Background())
	svc := NewWorkflowService(repo, nodeRegistry, 100)

	ctx := context.Background()
	wf, _ := svc.CreateWorkflow(ctx, "History Test", "", "user-1")

	fakeTarget := &fakeDeployTarget{success: true}
	svc.SetDeployTarget(fakeTarget)

	// Deploy twice
	svc.DeployWorkflow(ctx, wf.ID, "prod", "user-1")
	svc.DeployWorkflow(ctx, wf.ID, "prod", "user-1")

	versions, err := svc.GetVersionHistory(ctx, wf.ID)
	if err != nil {
		t.Fatalf("get version history: %v", err)
	}
	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(versions))
	}
}

// fakeDeployTarget is a test double for the deploy target port.
type fakeDeployTarget struct {
	success bool
	runID   string
	called  bool
}

func (f *fakeDeployTarget) Deploy(_ context.Context, _ domain.DeployPayload) (*domain.DeployResult, error) {
	f.called = true
	return &domain.DeployResult{Success: f.success, RunID: f.runID, Message: "ok"}, nil
}

// testAddNodeCmdWithDef adds a node with a specific definition.
type testAddNodeCmdWithDef struct {
	def  *domain.NodeDefinition
	id   string
	x, y float64
}

func (c *testAddNodeCmdWithDef) Execute(wf *domain.Workflow) (domain.Command, error) {
	node := wf.AddNode(c.def, c.x, c.y, c.id)
	return &testRemoveNodeCmd{node: *node}, nil
}
func (c *testAddNodeCmdWithDef) Type() string               { return "test_add_def" }
func (c *testAddNodeCmdWithDef) Serialize() ([]byte, error) { return nil, nil }
