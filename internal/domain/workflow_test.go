package domain

import (
	"testing"
)

func makeTestDef(id, category string, inputs []PortDefinition, outputs []PortDefinition) *NodeDefinition {
	return &NodeDefinition{
		ID:   id,
		Name: id,
		Category: Category{
			Group: category,
			Order: 1,
		},
		Shape:   Shape{Type: ShapeRoundedRect},
		Inputs:  inputs,
		Outputs: outputs,
	}
}

func dataInput(id string) PortDefinition {
	return PortDefinition{ID: id, Type: PortTypeData, MaxConnections: 1}
}

func dataOutput(id string) PortDefinition {
	return PortDefinition{ID: id, Type: PortTypeData, MaxConnections: -1}
}

func controlInput(id string) PortDefinition {
	return PortDefinition{ID: id, Type: PortTypeControl, MaxConnections: 1}
}

func controlOutput(id string) PortDefinition {
	return PortDefinition{ID: id, Type: PortTypeControl, MaxConnections: -1}
}

func newTestWorkflow() *Workflow {
	return &Workflow{
		ID:     "wf-1",
		Name:   "Test Workflow",
		Status: WorkflowStatusDraft,
	}
}

func TestWorkflow_AddNode(t *testing.T) {
	wf := newTestWorkflow()
	def := makeTestDef("source-twilio", "Sources", nil, []PortDefinition{dataOutput("out-main")})
	def.Attributes = []AttributeDefinition{
		{ID: "endpoint", Default: "/ingest/twilio"},
	}

	node := wf.AddNode(def, 100, 200, "node-1")

	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.ID != "node-1" {
		t.Errorf("got ID %q, want %q", node.ID, "node-1")
	}
	if node.X != 100 || node.Y != 200 {
		t.Errorf("got position (%v, %v), want (100, 200)", node.X, node.Y)
	}
	if node.DefinitionID != "source-twilio" {
		t.Errorf("got definition ID %q, want %q", node.DefinitionID, "source-twilio")
	}
	if node.Label != "source-twilio" {
		t.Errorf("got label %q, want %q", node.Label, "source-twilio")
	}
	if v, ok := node.AttributeValues["endpoint"]; !ok || v != "/ingest/twilio" {
		t.Errorf("expected default attribute value, got %v", v)
	}
	if len(wf.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(wf.Nodes))
	}
}

func TestWorkflow_RemoveNode(t *testing.T) {
	wf := newTestWorkflow()
	srcDef := makeTestDef("src", "Sources", nil, []PortDefinition{dataOutput("out")})
	tgtDef := makeTestDef("tgt", "Processing", []PortDefinition{dataInput("in")}, nil)

	wf.AddNode(srcDef, 0, 0, "n1")
	wf.AddNode(tgtDef, 100, 0, "n2")
	_ = wf.AddEdge("n1", "out", "n2", "in", "e1")

	removed, removedEdges, err := wf.RemoveNode("n1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed.ID != "n1" {
		t.Errorf("expected removed node n1, got %s", removed.ID)
	}
	if len(removedEdges) != 1 || removedEdges[0].ID != "e1" {
		t.Errorf("expected 1 removed edge e1, got %v", removedEdges)
	}
	if len(wf.Nodes) != 1 {
		t.Errorf("expected 1 remaining node, got %d", len(wf.Nodes))
	}
	if len(wf.Edges) != 0 {
		t.Errorf("expected 0 remaining edges, got %d", len(wf.Edges))
	}
}

func TestWorkflow_RemoveNode_NotFound(t *testing.T) {
	wf := newTestWorkflow()
	_, _, err := wf.RemoveNode("nonexistent")
	if err != ErrNodeNotFound {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestWorkflow_AddEdge(t *testing.T) {
	tests := []struct {
		name    string
		srcPort PortDefinition
		tgtPort PortDefinition
		wantErr bool
		errIs   error
	}{
		{"data to data", dataOutput("out"), dataInput("in"), false, nil},
		{"data to control type mismatch", dataOutput("out"), controlInput("in"), true, ErrPortTypeMismatch},
		{"control to control", controlOutput("out"), controlInput("in"), false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := newTestWorkflow()
			srcDef := makeTestDef("src", "Sources", nil, []PortDefinition{tt.srcPort})
			tgtDef := makeTestDef("tgt", "Processing", []PortDefinition{tt.tgtPort}, nil)
			wf.AddNode(srcDef, 0, 0, "n1")
			wf.AddNode(tgtDef, 100, 0, "n2")

			err := wf.AddEdge("n1", "out", "n2", "in", "e1")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errIs != nil && !errorContains(err, tt.errIs) {
					t.Errorf("expected error containing %v, got %v", tt.errIs, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestWorkflow_AddEdge_SelfReference(t *testing.T) {
	wf := newTestWorkflow()
	def := makeTestDef("node", "Sources",
		[]PortDefinition{dataInput("in")},
		[]PortDefinition{dataOutput("out")})
	wf.AddNode(def, 0, 0, "n1")

	err := wf.AddEdge("n1", "out", "n1", "in", "e1")
	if err != ErrSelfReference {
		t.Errorf("expected ErrSelfReference, got %v", err)
	}
}

func TestWorkflow_AddEdge_MaxConnections(t *testing.T) {
	wf := newTestWorkflow()
	srcDef := makeTestDef("src", "Sources", nil, []PortDefinition{dataOutput("out")})
	tgtDef := makeTestDef("tgt", "Processing",
		[]PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}}, nil)

	wf.AddNode(srcDef, 0, 0, "n1")
	wf.AddNode(srcDef, 0, 100, "n1b")
	wf.AddNode(tgtDef, 100, 0, "n2")

	if err := wf.AddEdge("n1", "out", "n2", "in", "e1"); err != nil {
		t.Fatalf("first edge should succeed: %v", err)
	}
	err := wf.AddEdge("n1b", "out", "n2", "in", "e2")
	if !errorContains(err, ErrMaxConnections) {
		t.Errorf("expected ErrMaxConnections, got %v", err)
	}
}

func TestWorkflow_AddEdge_DuplicateEdge(t *testing.T) {
	wf := newTestWorkflow()
	srcDef := makeTestDef("src", "Sources", nil, []PortDefinition{dataOutput("out")})
	tgtDef := makeTestDef("tgt", "Processing",
		[]PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: -1}}, nil)
	wf.AddNode(srcDef, 0, 0, "n1")
	wf.AddNode(tgtDef, 100, 0, "n2")

	_ = wf.AddEdge("n1", "out", "n2", "in", "e1")
	err := wf.AddEdge("n1", "out", "n2", "in", "e2")
	if err != ErrDuplicateEdge {
		t.Errorf("expected ErrDuplicateEdge, got %v", err)
	}
}

func TestWorkflow_AddEdge_CycleDetection(t *testing.T) {
	wf := newTestWorkflow()
	def := makeTestDef("node", "Processing",
		[]PortDefinition{dataInput("in")},
		[]PortDefinition{dataOutput("out")})

	wf.AddNode(def, 0, 0, "a")
	wf.AddNode(def, 100, 0, "b")
	wf.AddNode(def, 200, 0, "c")

	// a -> b -> c
	if err := wf.AddEdge("a", "out", "b", "in", "e1"); err != nil {
		t.Fatal(err)
	}
	if err := wf.AddEdge("b", "out", "c", "in", "e2"); err != nil {
		t.Fatal(err)
	}

	// c -> a would create a cycle
	err := wf.AddEdge("c", "out", "a", "in", "e3")
	if !errorContains(err, ErrCycleDetected) {
		t.Errorf("expected ErrCycleDetected, got %v", err)
	}

	// Verify the cycle edge was NOT added
	if len(wf.Edges) != 2 {
		t.Errorf("expected 2 edges after rejected cycle, got %d", len(wf.Edges))
	}
}

func TestWorkflow_AddEdge_ConnectionRules(t *testing.T) {
	wf := newTestWorkflow()
	srcDef := makeTestDef("src", "Sources", nil, []PortDefinition{dataOutput("out")})
	srcDef.Validation = ValidationRules{
		ConnectionRules: []ConnectionRule{
			{
				OutputPort:              "out",
				AllowedTargetCategories: []string{"Processing", "Destinations"},
				AllowedTargetPorts:      []string{"data"},
			},
		},
	}
	tgtDef := makeTestDef("ctrl", "Control", []PortDefinition{dataInput("in")}, nil)

	wf.AddNode(srcDef, 0, 0, "n1")
	wf.AddNode(tgtDef, 100, 0, "n2")

	err := wf.AddEdge("n1", "out", "n2", "in", "e1")
	if !errorContains(err, ErrConnectionRule) {
		t.Errorf("expected ErrConnectionRule, got %v", err)
	}
}

func TestWorkflow_RemoveEdge(t *testing.T) {
	wf := newTestWorkflow()
	srcDef := makeTestDef("src", "Sources", nil, []PortDefinition{dataOutput("out")})
	tgtDef := makeTestDef("tgt", "Processing", []PortDefinition{dataInput("in")}, nil)
	wf.AddNode(srcDef, 0, 0, "n1")
	wf.AddNode(tgtDef, 100, 0, "n2")
	_ = wf.AddEdge("n1", "out", "n2", "in", "e1")

	removed, err := wf.RemoveEdge("e1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed.ID != "e1" {
		t.Errorf("expected removed edge e1, got %s", removed.ID)
	}
	if len(wf.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(wf.Edges))
	}
}

func TestWorkflow_RemoveEdge_NotFound(t *testing.T) {
	wf := newTestWorkflow()
	_, err := wf.RemoveEdge("nonexistent")
	if err != ErrEdgeNotFound {
		t.Errorf("expected ErrEdgeNotFound, got %v", err)
	}
}

func TestWorkflow_FindNode(t *testing.T) {
	wf := newTestWorkflow()
	def := makeTestDef("src", "Sources", nil, nil)
	wf.AddNode(def, 0, 0, "n1")

	if n := wf.FindNode("n1"); n == nil {
		t.Error("expected to find node n1")
	}
	if n := wf.FindNode("nonexistent"); n != nil {
		t.Error("expected nil for nonexistent node")
	}
}

func TestWorkflow_Validate_DuplicateNodeIDs(t *testing.T) {
	wf := newTestWorkflow()
	wf.Nodes = []NodeInstance{
		{ID: "n1", DefinitionID: "src"},
		{ID: "n1", DefinitionID: "src"},
	}

	errs := wf.Validate()
	if len(errs) == 0 {
		t.Error("expected validation errors for duplicate IDs")
	}
	found := false
	for _, e := range errs {
		if e.Message == "duplicate node ID" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'duplicate node ID' error")
	}
}

func TestWorkflow_Validate_MissingRequiredAttributes(t *testing.T) {
	wf := newTestWorkflow()
	def := makeTestDef("src", "Sources", nil, nil)
	def.Attributes = []AttributeDefinition{
		{ID: "endpoint", Label: "Endpoint", Required: true},
	}
	wf.AddNode(def, 0, 0, "n1")

	errs := wf.Validate()
	if len(errs) == 0 {
		t.Error("expected validation error for missing required attribute")
	}
}

func TestWorkflow_Validate_EmptyWorkflow(t *testing.T) {
	wf := newTestWorkflow()
	errs := wf.Validate()
	if len(errs) != 0 {
		t.Errorf("empty workflow should have no errors, got %v", errs)
	}
}

func TestWorkflow_Validate_CycleDetection(t *testing.T) {
	wf := newTestWorkflow()
	// Manually create a cycle by adding edges directly
	wf.Nodes = []NodeInstance{
		{ID: "a", DefinitionID: "node"},
		{ID: "b", DefinitionID: "node"},
	}
	wf.Edges = []Edge{
		{ID: "e1", SourceNodeID: "a", TargetNodeID: "b"},
		{ID: "e2", SourceNodeID: "b", TargetNodeID: "a"},
	}

	errs := wf.Validate()
	found := false
	for _, e := range errs {
		if e.Message == "workflow contains a cycle" {
			found = true
		}
	}
	if !found {
		t.Error("expected cycle detection error")
	}
}

// Compile-time check: all DeployVerification constants must be distinct.
// If two constants have the same value, this map literal will fail to compile.
var _ = map[DeployVerification]struct{}{
	DeployVerificationUnknown:  {},
	DeployVerificationVerified: {},
	DeployVerificationMissing:  {},
	DeployVerificationError:    {},
}

func TestDeployVerification_StringValues(t *testing.T) {
	// Verify the string representation matches the expected wire format,
	// since these values are serialized to JSON for the API.
	tests := []struct {
		v    DeployVerification
		want string
	}{
		{DeployVerificationUnknown, "unknown"},
		{DeployVerificationVerified, "verified"},
		{DeployVerificationMissing, "missing"},
		{DeployVerificationError, "error"},
	}
	for _, tt := range tests {
		if string(tt.v) != tt.want {
			t.Errorf("DeployVerification %q serializes as %q, want %q", tt.v, string(tt.v), tt.want)
		}
	}
}

func errorContains(err, target error) bool {
	if err == nil {
		return false
	}
	return err.Error() == target.Error() || contains(err.Error(), target.Error())
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
