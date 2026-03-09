package commands

import (
	"encoding/json"
	"graphiti/internal/domain"
	"testing"
)

// fakeResolver is a test helper that resolves node definitions from a map.
type fakeResolver struct {
	defs map[string]*domain.NodeDefinition
}

func (r *fakeResolver) GetByID(id string) (*domain.NodeDefinition, error) {
	def, ok := r.defs[id]
	if !ok {
		return nil, domain.ErrNodeNotFound
	}
	return def, nil
}

func testDef(id, category string) *domain.NodeDefinition {
	return &domain.NodeDefinition{
		ID:   id,
		Name: id,
		Category: domain.Category{
			Group: category,
			Order: 1,
		},
		Shape:  domain.Shape{Type: domain.ShapeRoundedRect},
		Inputs: []domain.PortDefinition{{ID: "in", Type: domain.PortTypeData, MaxConnections: 1}},
		Outputs: []domain.PortDefinition{{ID: "out", Type: domain.PortTypeData, MaxConnections: -1}},
	}
}

func testWorkflow() *domain.Workflow {
	return &domain.Workflow{
		ID:     "wf-1",
		Name:   "Test",
		Status: domain.WorkflowStatusDraft,
	}
}

func testResolver() *fakeResolver {
	return &fakeResolver{
		defs: map[string]*domain.NodeDefinition{
			"src": testDef("src", "Sources"),
			"proc": testDef("proc", "Processing"),
		},
	}
}

func TestAddNodeCommand_ExecuteAndInverse(t *testing.T) {
	wf := testWorkflow()
	resolver := testResolver()

	cmd := NewAddNodeCommand("src", "node-1", "", 100, 200, resolver)
	inverse, err := cmd.Execute(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wf.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(wf.Nodes))
	}
	if wf.Nodes[0].X != 100 || wf.Nodes[0].Y != 200 {
		t.Errorf("wrong position: (%v, %v)", wf.Nodes[0].X, wf.Nodes[0].Y)
	}

	// Undo
	_, err = inverse.Execute(wf)
	if err != nil {
		t.Fatalf("undo error: %v", err)
	}
	if len(wf.Nodes) != 0 {
		t.Errorf("expected 0 nodes after undo, got %d", len(wf.Nodes))
	}
}

func TestRemoveNodeCommand_ExecuteAndInverse(t *testing.T) {
	wf := testWorkflow()
	resolver := testResolver()
	addCmd := NewAddNodeCommand("src", "node-1", "", 50, 50, resolver)
	addCmd.Execute(wf)

	cmd := &RemoveNodeCommand{NodeID: "node-1"}
	inverse, err := cmd.Execute(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wf.Nodes) != 0 {
		t.Fatalf("expected 0 nodes, got %d", len(wf.Nodes))
	}

	// Undo (restore)
	_, err = inverse.Execute(wf)
	if err != nil {
		t.Fatalf("undo error: %v", err)
	}
	if len(wf.Nodes) != 1 {
		t.Errorf("expected 1 node after undo, got %d", len(wf.Nodes))
	}
	if wf.Nodes[0].ID != "node-1" {
		t.Errorf("expected node-1, got %s", wf.Nodes[0].ID)
	}
}

func TestMoveNodeCommand_ExecuteAndInverse(t *testing.T) {
	wf := testWorkflow()
	resolver := testResolver()
	NewAddNodeCommand("src", "n1", "", 10, 20, resolver).Execute(wf)

	cmd := &MoveNodeCommand{NodeID: "n1", FromX: 10, FromY: 20, ToX: 100, ToY: 200}
	inverse, err := cmd.Execute(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	node := wf.FindNode("n1")
	if node.X != 100 || node.Y != 200 {
		t.Errorf("expected (100, 200), got (%v, %v)", node.X, node.Y)
	}

	// Undo
	_, err = inverse.Execute(wf)
	if err != nil {
		t.Fatalf("undo error: %v", err)
	}
	node = wf.FindNode("n1")
	if node.X != 10 || node.Y != 20 {
		t.Errorf("expected (10, 20) after undo, got (%v, %v)", node.X, node.Y)
	}
}

func TestMoveNodeCommand_NotFound(t *testing.T) {
	wf := testWorkflow()
	cmd := &MoveNodeCommand{NodeID: "nonexistent", ToX: 100, ToY: 200}
	_, err := cmd.Execute(wf)
	if err != domain.ErrNodeNotFound {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestMoveNodesCommand_ExecuteAndInverse(t *testing.T) {
	wf := testWorkflow()
	resolver := testResolver()
	NewAddNodeCommand("src", "n1", "", 10, 20, resolver).Execute(wf)
	NewAddNodeCommand("proc", "n2", "", 30, 40, resolver).Execute(wf)

	cmd := &MoveNodesCommand{
		From: []NodePosition{
			{NodeID: "n1", X: 10, Y: 20},
			{NodeID: "n2", X: 30, Y: 40},
		},
		To: []NodePosition{
			{NodeID: "n1", X: 100, Y: 200},
			{NodeID: "n2", X: 300, Y: 400},
		},
	}

	inverse, err := cmd.Execute(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	n1 := wf.FindNode("n1")
	n2 := wf.FindNode("n2")
	if n1.X != 100 || n1.Y != 200 {
		t.Errorf("n1: expected (100, 200), got (%v, %v)", n1.X, n1.Y)
	}
	if n2.X != 300 || n2.Y != 400 {
		t.Errorf("n2: expected (300, 400), got (%v, %v)", n2.X, n2.Y)
	}

	// Undo
	inverse.Execute(wf)
	n1 = wf.FindNode("n1")
	n2 = wf.FindNode("n2")
	if n1.X != 10 || n1.Y != 20 {
		t.Errorf("n1 after undo: expected (10, 20), got (%v, %v)", n1.X, n1.Y)
	}
	if n2.X != 30 || n2.Y != 40 {
		t.Errorf("n2 after undo: expected (30, 40), got (%v, %v)", n2.X, n2.Y)
	}
}

func TestAddEdgeCommand_ExecuteAndInverse(t *testing.T) {
	wf := testWorkflow()
	resolver := testResolver()
	NewAddNodeCommand("src", "n1", "", 0, 0, resolver).Execute(wf)
	NewAddNodeCommand("proc", "n2", "", 100, 0, resolver).Execute(wf)

	cmd := &AddEdgeCommand{
		EdgeID:       "e1",
		SourceNodeID: "n1",
		SourcePortID: "out",
		TargetNodeID: "n2",
		TargetPortID: "in",
	}

	inverse, err := cmd.Execute(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wf.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(wf.Edges))
	}

	// Undo
	_, err = inverse.Execute(wf)
	if err != nil {
		t.Fatalf("undo error: %v", err)
	}
	if len(wf.Edges) != 0 {
		t.Errorf("expected 0 edges after undo, got %d", len(wf.Edges))
	}
}

func TestRemoveEdgeCommand_ExecuteAndInverse(t *testing.T) {
	wf := testWorkflow()
	resolver := testResolver()
	NewAddNodeCommand("src", "n1", "", 0, 0, resolver).Execute(wf)
	NewAddNodeCommand("proc", "n2", "", 100, 0, resolver).Execute(wf)
	wf.AddEdge("n1", "out", "n2", "in", "e1")

	cmd := &RemoveEdgeCommand{EdgeID: "e1"}
	inverse, err := cmd.Execute(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wf.Edges) != 0 {
		t.Fatalf("expected 0 edges, got %d", len(wf.Edges))
	}

	// Undo (restore)
	_, err = inverse.Execute(wf)
	if err != nil {
		t.Fatalf("undo error: %v", err)
	}
	if len(wf.Edges) != 1 {
		t.Errorf("expected 1 edge after undo, got %d", len(wf.Edges))
	}
}

func TestUpdateAttributeCommand_ExecuteAndInverse(t *testing.T) {
	wf := testWorkflow()
	resolver := testResolver()
	NewAddNodeCommand("src", "n1", "", 0, 0, resolver).Execute(wf)

	cmd := &UpdateAttributeCommand{
		NodeID:   "n1",
		AttrID:   "endpoint",
		OldValue: nil,
		NewValue: "/api/test",
	}

	inverse, err := cmd.Execute(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	node := wf.FindNode("n1")
	if node.AttributeValues["endpoint"] != "/api/test" {
		t.Errorf("expected /api/test, got %v", node.AttributeValues["endpoint"])
	}

	// Undo
	inverse.Execute(wf)
	node = wf.FindNode("n1")
	if node.AttributeValues["endpoint"] != nil {
		t.Errorf("expected nil after undo, got %v", node.AttributeValues["endpoint"])
	}
}

func TestRenameNodeCommand_ExecuteAndInverse(t *testing.T) {
	wf := testWorkflow()
	resolver := testResolver()
	NewAddNodeCommand("src", "n1", "Old Name", 0, 0, resolver).Execute(wf)

	cmd := &RenameNodeCommand{NodeID: "n1", OldLabel: "Old Name", NewLabel: "New Name"}
	inverse, err := cmd.Execute(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wf.FindNode("n1").Label != "New Name" {
		t.Errorf("expected New Name, got %s", wf.FindNode("n1").Label)
	}

	// Undo
	inverse.Execute(wf)
	if wf.FindNode("n1").Label != "Old Name" {
		t.Errorf("expected Old Name after undo, got %s", wf.FindNode("n1").Label)
	}
}

func TestPasteNodesCommand_ExecuteAndInverse(t *testing.T) {
	wf := testWorkflow()

	nodes := []domain.NodeInstance{
		{ID: "pasted-1", DefinitionID: "src", Label: "Pasted 1", X: 200, Y: 200},
		{ID: "pasted-2", DefinitionID: "proc", Label: "Pasted 2", X: 300, Y: 200},
	}
	edges := []domain.Edge{
		{ID: "pasted-e1", SourceNodeID: "pasted-1", SourcePortID: "out", TargetNodeID: "pasted-2", TargetPortID: "in"},
	}

	cmd := &PasteNodesCommand{Nodes: nodes, Edges: edges}
	inverse, err := cmd.Execute(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wf.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(wf.Nodes))
	}
	if len(wf.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(wf.Edges))
	}

	// Undo
	_, err = inverse.Execute(wf)
	if err != nil {
		t.Fatalf("undo error: %v", err)
	}
	if len(wf.Nodes) != 0 {
		t.Errorf("expected 0 nodes after undo, got %d", len(wf.Nodes))
	}
	if len(wf.Edges) != 0 {
		t.Errorf("expected 0 edges after undo, got %d", len(wf.Edges))
	}
}

func TestCommand_Serialization_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		cmd  domain.Command
	}{
		{"move_node", &MoveNodeCommand{NodeID: "n1", FromX: 10, FromY: 20, ToX: 100, ToY: 200}},
		{"add_edge", &AddEdgeCommand{EdgeID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "in"}},
		{"update_attribute", &UpdateAttributeCommand{NodeID: "n1", AttrID: "endpoint", NewValue: "/api/test"}},
		{"rename_node", &RenameNodeCommand{NodeID: "n1", OldLabel: "Old", NewLabel: "New"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.cmd.Serialize()
			if err != nil {
				t.Fatalf("serialize error: %v", err)
			}

			var raw json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			restored, err := domain.DeserializeCommand(tt.cmd.Type(), data)
			if err != nil {
				t.Fatalf("deserialize error: %v", err)
			}
			if restored.Type() != tt.cmd.Type() {
				t.Errorf("type mismatch: got %s, want %s", restored.Type(), tt.cmd.Type())
			}
		})
	}
}
