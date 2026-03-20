package app

import (
	"context"
	"testing"

	"graphiti/internal/adapters/driven/memory"
	"graphiti/internal/domain"
)

func TestWorkflowService_ValidateWorkflow_EmptyWorkflow(t *testing.T) {
	svc, _ := setupWorkflowService()
	ctx := context.Background()
	wf, _ := svc.CreateWorkflow(ctx, "Empty WF", "", "user-1")

	results, err := svc.ValidateWorkflow(ctx, wf.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected validation results for empty workflow")
	}

	found := false
	for _, r := range results {
		if r.Code == "EMPTY_WORKFLOW" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected EMPTY_WORKFLOW code")
	}
}

func TestWorkflowService_ValidateWorkflow_ValidWorkflow(t *testing.T) {
	svc, registry := setupWorkflowService()
	ctx := context.Background()
	wf, _ := svc.CreateWorkflow(ctx, "Valid WF", "", "user-1")

	// Add source and processor connected
	srcDef, _ := registry.GetByID(ctx, "src")
	procDef, _ := registry.GetByID(ctx, "proc")

	addSrc := &testAddNodeCmdWithDef{def: srcDef, id: "n1", x: 100, y: 100}
	svc.ExecuteCommand(ctx, wf.ID, addSrc)
	addProc := &testAddNodeCmdWithDef{def: procDef, id: "n2", x: 300, y: 100}
	svc.ExecuteCommand(ctx, wf.ID, addProc)

	addEdge := &testAddEdgeCmd{srcNode: "n1", srcPort: "out", tgtNode: "n2", tgtPort: "in", id: "e1"}
	svc.ExecuteCommand(ctx, wf.ID, addEdge)

	results, err := svc.ValidateWorkflow(ctx, wf.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errors := domain.ErrorsOnly(results)
	if len(errors) > 0 {
		codes := make([]string, len(errors))
		for i, e := range errors {
			codes[i] = e.Code
		}
		t.Errorf("expected no errors, got: %v", codes)
	}
}

func TestWorkflowService_ValidateWorkflow_NotFound(t *testing.T) {
	svc, _ := setupWorkflowService()
	ctx := context.Background()

	_, err := svc.ValidateWorkflow(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent workflow")
	}
}

func TestWorkflowService_DeployWorkflow_UsesNewValidation(t *testing.T) {
	svc, registry := setupWorkflowService()
	ctx := context.Background()
	wf, _ := svc.CreateWorkflow(ctx, "Deploy Validate", "", "user-1")

	// Add node with required attribute missing
	srcDef, _ := registry.GetByID(ctx, "src")
	defWithRequired := *srcDef
	defWithRequired.Attributes = []domain.AttributeDefinition{
		{ID: "url", Label: "URL", Type: domain.AttrTypeString, Required: true},
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
	if len(result.ValidationResults) == 0 {
		t.Error("expected validation results in deploy response")
	}
	if fakeTarget.called {
		t.Error("deploy target should not be called when validation fails")
	}
}

// testAddEdgeCmd adds an edge to the workflow.
type testAddEdgeCmd struct {
	srcNode, srcPort, tgtNode, tgtPort, id string
}

func (c *testAddEdgeCmd) Execute(wf *domain.Workflow) (domain.Command, error) {
	err := wf.AddEdge(c.srcNode, c.srcPort, c.tgtNode, c.tgtPort, c.id)
	if err != nil {
		return nil, err
	}
	return &testRemoveEdgeCmd{edgeID: c.id}, nil
}
func (c *testAddEdgeCmd) Type() string               { return "test_add_edge" }
func (c *testAddEdgeCmd) Serialize() ([]byte, error)  { return nil, nil }

type testRemoveEdgeCmd struct {
	edgeID string
}

func (c *testRemoveEdgeCmd) Execute(wf *domain.Workflow) (domain.Command, error) {
	edge, err := wf.RemoveEdge(c.edgeID)
	if err != nil {
		return nil, err
	}
	return &testAddEdgeCmd{
		srcNode: edge.SourceNodeID, srcPort: edge.SourcePortID,
		tgtNode: edge.TargetNodeID, tgtPort: edge.TargetPortID,
		id: edge.ID,
	}, nil
}
func (c *testRemoveEdgeCmd) Type() string               { return "test_remove_edge" }
func (c *testRemoveEdgeCmd) Serialize() ([]byte, error)  { return nil, nil }

func TestWorkflowService_ValidateWorkflow_SubworkflowChain(t *testing.T) {
	repo := memory.NewWorkflowRepo()
	subDef := &domain.NodeDefinition{
		ID:   "sub",
		Name: "Sub-Workflow",
		Category: domain.Category{Group: "Control"},
		Shape:    domain.Shape{Type: domain.ShapeSubWorkflow},
		Inputs:   []domain.PortDefinition{{ID: "in", Type: domain.PortTypeData, MaxConnections: 1}},
		Outputs:  []domain.PortDefinition{{ID: "out", Type: domain.PortTypeData, MaxConnections: -1}},
		Attributes: []domain.AttributeDefinition{
			{ID: "workflow_ref", Label: "Ref", Type: domain.AttrTypeWorkflowRef, Required: true},
		},
	}
	srcDef := &domain.NodeDefinition{
		ID:   "src",
		Name: "Source",
		Category: domain.Category{Group: "Sources"},
		Shape:    domain.Shape{Type: domain.ShapeRoundedRect},
		Outputs:  []domain.PortDefinition{{ID: "out", Type: domain.PortTypeData, MaxConnections: -1}},
	}
	defRepo := &fakeNodeDefRepo{defs: []*domain.NodeDefinition{subDef, srcDef}}
	nodeRegistry := NewNodeRegistry(defRepo)
	nodeRegistry.Load(context.Background())
	svc := NewWorkflowService(repo, nodeRegistry, 100)

	ctx := context.Background()

	// Create workflow A with a sub-workflow referencing B
	wfA, _ := svc.CreateWorkflow(ctx, "WF A", "", "user-1")
	addSrc := &testAddNodeCmdWithDef{def: srcDef, id: "n-src", x: 50, y: 50}
	svc.ExecuteCommand(ctx, wfA.ID, addSrc)
	addSub := &testAddNodeCmdWithDef{def: subDef, id: "n-sub", x: 200, y: 50}
	svc.ExecuteCommand(ctx, wfA.ID, addSub)
	addEdge := &testAddEdgeCmd{srcNode: "n-src", srcPort: "out", tgtNode: "n-sub", tgtPort: "in", id: "e1"}
	svc.ExecuteCommand(ctx, wfA.ID, addEdge)

	// Create workflow B with a sub-workflow referencing A (circular!)
	wfB, _ := svc.CreateWorkflow(ctx, "WF B", "", "user-1")
	addSrc2 := &testAddNodeCmdWithDef{def: srcDef, id: "n-src2", x: 50, y: 50}
	svc.ExecuteCommand(ctx, wfB.ID, addSrc2)
	addSub2 := &testAddNodeCmdWithDef{def: subDef, id: "n-sub2", x: 200, y: 50}
	svc.ExecuteCommand(ctx, wfB.ID, addSub2)
	addEdge2 := &testAddEdgeCmd{srcNode: "n-src2", srcPort: "out", tgtNode: "n-sub2", tgtPort: "in", id: "e2"}
	svc.ExecuteCommand(ctx, wfB.ID, addEdge2)

	// Set sub-workflow refs to create A -> B -> A cycle
	updateRefA := &testUpdateAttrCmd{nodeID: "n-sub", attrID: "workflow_ref", value: wfB.ID}
	svc.ExecuteCommand(ctx, wfA.ID, updateRefA)
	updateRefB := &testUpdateAttrCmd{nodeID: "n-sub2", attrID: "workflow_ref", value: wfA.ID}
	svc.ExecuteCommand(ctx, wfB.ID, updateRefB)

	results, err := svc.ValidateWorkflow(ctx, wfA.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, r := range results {
		if r.Code == "SUBWORKFLOW_CIRCULAR_REFERENCE" {
			found = true
		}
	}
	if !found {
		codes := make([]string, len(results))
		for i, r := range results {
			codes[i] = r.Code
		}
		t.Errorf("expected SUBWORKFLOW_CIRCULAR_REFERENCE, got: %v", codes)
	}
}

// testUpdateAttrCmd sets an attribute value on a node.
type testUpdateAttrCmd struct {
	nodeID string
	attrID string
	value  any
}

func (c *testUpdateAttrCmd) Execute(wf *domain.Workflow) (domain.Command, error) {
	node := wf.FindNode(c.nodeID)
	if node == nil {
		return nil, domain.ErrNodeNotFound
	}
	old := node.AttributeValues[c.attrID]
	node.AttributeValues[c.attrID] = c.value
	return &testUpdateAttrCmd{nodeID: c.nodeID, attrID: c.attrID, value: old}, nil
}
func (c *testUpdateAttrCmd) Type() string               { return "test_update_attr" }
func (c *testUpdateAttrCmd) Serialize() ([]byte, error)  { return nil, nil }
