package app

import (
	"context"
	"testing"

	"graphiti/internal/adapters/driven/memory"
	"graphiti/internal/domain"
)

func setupWorkflowService() (*WorkflowService, *NodeRegistry) {
	repo := &fakeNodeDefRepo{defs: []*domain.NodeDefinition{
		{
			ID:   "src",
			Name: "Source",
			Category: domain.Category{Group: "Sources"},
			Shape:    domain.Shape{Type: domain.ShapeRoundedRect},
			Outputs:  []domain.PortDefinition{{ID: "out", Type: domain.PortTypeData, MaxConnections: -1}},
		},
		{
			ID:   "proc",
			Name: "Processor",
			Category: domain.Category{Group: "Processing"},
			Shape:    domain.Shape{Type: domain.ShapeRoundedRect},
			Inputs:   []domain.PortDefinition{{ID: "in", Type: domain.PortTypeData, MaxConnections: 1}},
			Outputs:  []domain.PortDefinition{{ID: "out", Type: domain.PortTypeData, MaxConnections: -1}},
		},
	}}
	nodeRegistry := NewNodeRegistry(repo)
	nodeRegistry.Load(context.Background())

	wfRepo := memory.NewWorkflowRepo()
	svc := NewWorkflowService(wfRepo, nodeRegistry, 100)
	return svc, nodeRegistry
}

func TestWorkflowService_CreateAndGet(t *testing.T) {
	svc, _ := setupWorkflowService()
	ctx := context.Background()

	wf, err := svc.CreateWorkflow(ctx, "Test Workflow", "A test", "user-1")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if wf.Name != "Test Workflow" {
		t.Errorf("expected name 'Test Workflow', got %q", wf.Name)
	}

	retrieved, err := svc.GetWorkflow(ctx, wf.ID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if retrieved.ID != wf.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, wf.ID)
	}
}

func TestWorkflowService_ListWorkflows(t *testing.T) {
	svc, _ := setupWorkflowService()
	ctx := context.Background()

	svc.CreateWorkflow(ctx, "WF1", "", "user-1")
	svc.CreateWorkflow(ctx, "WF2", "", "user-1")
	svc.CreateWorkflow(ctx, "WF3", "", "user-2")

	list, err := svc.ListWorkflows(ctx, "user-1")
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 workflows for user-1, got %d", len(list))
	}
}

func TestWorkflowService_DeleteWorkflow(t *testing.T) {
	svc, _ := setupWorkflowService()
	ctx := context.Background()

	wf, _ := svc.CreateWorkflow(ctx, "To Delete", "", "user-1")
	if err := svc.DeleteWorkflow(ctx, wf.ID); err != nil {
		t.Fatalf("delete error: %v", err)
	}

	_, err := svc.GetWorkflow(ctx, wf.ID)
	if err == nil {
		t.Error("expected error getting deleted workflow")
	}
}

func TestWorkflowService_UndoRedo(t *testing.T) {
	svc, registry := setupWorkflowService()
	ctx := context.Background()

	wf, _ := svc.CreateWorkflow(ctx, "Test", "", "user-1")

	// Add a node via command
	addCmd := &testAddNodeCmd{
		defID: "src",
		id:    "node-1",
		x:     100,
		y:     200,
		registry: registry,
	}

	result, err := svc.ExecuteCommand(ctx, wf.ID, addCmd)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if !result.CanUndo {
		t.Error("expected CanUndo after command")
	}

	// Undo
	result, err = svc.Undo(ctx, wf.ID)
	if err != nil {
		t.Fatalf("undo error: %v", err)
	}
	if !result.CanRedo {
		t.Error("expected CanRedo after undo")
	}

	// Redo
	result, err = svc.Redo(ctx, wf.ID)
	if err != nil {
		t.Fatalf("redo error: %v", err)
	}
	if !result.CanUndo {
		t.Error("expected CanUndo after redo")
	}
}

// testAddNodeCmd is a simple command for testing that adds a node.
type testAddNodeCmd struct {
	defID    string
	id       string
	x, y     float64
	registry *NodeRegistry
}

func (c *testAddNodeCmd) Execute(wf *domain.Workflow) (domain.Command, error) {
	def, err := c.registry.GetByID(context.Background(), c.defID)
	if err != nil {
		return nil, err
	}
	node := wf.AddNode(def, c.x, c.y, c.id)
	return &testRemoveNodeCmd{node: *node}, nil
}
func (c *testAddNodeCmd) Type() string             { return "test_add" }
func (c *testAddNodeCmd) Serialize() ([]byte, error) { return nil, nil }

type testRemoveNodeCmd struct {
	node domain.NodeInstance
}

func (c *testRemoveNodeCmd) Execute(wf *domain.Workflow) (domain.Command, error) {
	removed, _, err := wf.RemoveNode(c.node.ID)
	if err != nil {
		return nil, err
	}
	return &testAddNodeCmd2{node: *removed}, nil
}
func (c *testRemoveNodeCmd) Type() string             { return "test_remove" }
func (c *testRemoveNodeCmd) Serialize() ([]byte, error) { return nil, nil }

type testAddNodeCmd2 struct {
	node domain.NodeInstance
}

func (c *testAddNodeCmd2) Execute(wf *domain.Workflow) (domain.Command, error) {
	wf.RestoreNode(c.node)
	return &testRemoveNodeCmd{node: c.node}, nil
}
func (c *testAddNodeCmd2) Type() string             { return "test_add2" }
func (c *testAddNodeCmd2) Serialize() ([]byte, error) { return nil, nil }
