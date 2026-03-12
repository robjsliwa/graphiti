package domain

import (
	"fmt"
	"testing"
)

func makeSubWorkflowDef() *NodeDefinition {
	return &NodeDefinition{
		ID:   "control-sub-workflow",
		Name: "Sub-Workflow",
		Category: Category{
			Group: "Control Flow",
			Order: 30,
		},
		Shape: Shape{Type: ShapeSubWorkflow},
		Inputs: []PortDefinition{
			{ID: "in-main", Type: PortTypeData, MaxConnections: 1},
		},
		Outputs: []PortDefinition{
			{ID: "out-main", Type: PortTypeData, MaxConnections: 1},
			{ID: "out-error", Type: PortTypeError, MaxConnections: 1},
		},
		Attributes: []AttributeDefinition{
			{ID: "workflow_ref", Label: "Referenced Workflow", Type: AttrTypeWorkflowRef, Required: true},
			{ID: "input_mapping", Label: "Input Mapping", Type: AttrTypeJSON, Default: "{}"},
			{ID: "output_mapping", Label: "Output Mapping", Type: AttrTypeJSON, Default: "{}"},
		},
	}
}

func TestValidateSubworkflows_NoReference(t *testing.T) {
	wf := &Workflow{ID: "wf-1", Name: "Parent"}
	def := makeSubWorkflowDef()
	wf.AddNode(def, 100, 100, "sub-1")

	results := wf.ValidateSubworkflows()
	if len(results) == 0 {
		t.Fatal("expected validation error for missing workflow reference")
	}
	found := false
	for _, r := range results {
		if r.Code == "SUBWORKFLOW_NO_REFERENCE" && r.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Errorf("expected SUBWORKFLOW_NO_REFERENCE error, got %+v", results)
	}
}

func TestValidateSubworkflows_SelfReference(t *testing.T) {
	wf := &Workflow{ID: "wf-1", Name: "Parent"}
	def := makeSubWorkflowDef()
	node := wf.AddNode(def, 100, 100, "sub-1")
	node.AttributeValues["workflow_ref"] = "wf-1" // self-reference

	results := wf.ValidateSubworkflows()
	if len(results) == 0 {
		t.Fatal("expected validation error for self-reference")
	}
	found := false
	for _, r := range results {
		if r.Code == "SUBWORKFLOW_SELF_REFERENCE" && r.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Errorf("expected SUBWORKFLOW_SELF_REFERENCE error, got %+v", results)
	}
}

func TestValidateSubworkflows_ValidReference(t *testing.T) {
	wf := &Workflow{ID: "wf-1", Name: "Parent"}
	def := makeSubWorkflowDef()
	node := wf.AddNode(def, 100, 100, "sub-1")
	node.AttributeValues["workflow_ref"] = "wf-other"

	results := wf.ValidateSubworkflows()
	if len(results) != 0 {
		t.Errorf("expected no validation errors, got %+v", results)
	}
}

func TestValidateSubworkflows_NonSubWorkflowNodesIgnored(t *testing.T) {
	wf := &Workflow{ID: "wf-1", Name: "Test"}
	def := makeTestDef("src", "Sources", nil, []PortDefinition{dataOutput("out")})
	wf.AddNode(def, 100, 100, "node-1")

	results := wf.ValidateSubworkflows()
	if len(results) != 0 {
		t.Errorf("expected no errors for non-sub-workflow nodes, got %+v", results)
	}
}

// WorkflowResolver tests for circular reference detection

type mockResolver struct {
	workflows map[string]*Workflow
}

func (m *mockResolver) ResolveWorkflow(id string) (*Workflow, error) {
	wf, ok := m.workflows[id]
	if !ok {
		return nil, ErrNotFound
	}
	return wf, nil
}

func TestValidateSubworkflowChain_ValidChain(t *testing.T) {
	subDef := makeSubWorkflowDef()

	wfB := &Workflow{ID: "wf-b", Name: "Child"}
	// wfB has no sub-workflows

	wfA := &Workflow{ID: "wf-a", Name: "Parent"}
	nodeA := wfA.AddNode(subDef, 100, 100, "sub-1")
	nodeA.AttributeValues["workflow_ref"] = "wf-b"

	resolver := &mockResolver{workflows: map[string]*Workflow{"wf-b": wfB}}

	results := ValidateSubworkflowChain("wf-b", resolver, map[string]bool{"wf-a": true}, 1)
	if len(results) != 0 {
		t.Errorf("expected no errors for valid chain, got %+v", results)
	}
}

func TestValidateSubworkflowChain_CircularReference(t *testing.T) {
	subDef := makeSubWorkflowDef()

	wfA := &Workflow{ID: "wf-a", Name: "A"}
	nodeA := wfA.AddNode(subDef, 100, 100, "sub-a")
	nodeA.AttributeValues["workflow_ref"] = "wf-b"

	wfB := &Workflow{ID: "wf-b", Name: "B"}
	nodeB := wfB.AddNode(subDef, 100, 100, "sub-b")
	nodeB.AttributeValues["workflow_ref"] = "wf-a"

	resolver := &mockResolver{workflows: map[string]*Workflow{
		"wf-a": wfA,
		"wf-b": wfB,
	}}

	results := ValidateSubworkflowChain("wf-b", resolver, map[string]bool{"wf-a": true}, 1)
	found := false
	for _, r := range results {
		if r.Code == "SUBWORKFLOW_CIRCULAR_REFERENCE" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected SUBWORKFLOW_CIRCULAR_REFERENCE, got %+v", results)
	}
}

func TestValidateSubworkflowChain_ThreeWayCycle(t *testing.T) {
	subDef := makeSubWorkflowDef()

	wfA := &Workflow{ID: "wf-a", Name: "A"}
	nA := wfA.AddNode(subDef, 100, 100, "sub-a")
	nA.AttributeValues["workflow_ref"] = "wf-b"

	wfB := &Workflow{ID: "wf-b", Name: "B"}
	nB := wfB.AddNode(subDef, 100, 100, "sub-b")
	nB.AttributeValues["workflow_ref"] = "wf-c"

	wfC := &Workflow{ID: "wf-c", Name: "C"}
	nC := wfC.AddNode(subDef, 100, 100, "sub-c")
	nC.AttributeValues["workflow_ref"] = "wf-a"

	resolver := &mockResolver{workflows: map[string]*Workflow{
		"wf-a": wfA,
		"wf-b": wfB,
		"wf-c": wfC,
	}}

	results := ValidateSubworkflowChain("wf-b", resolver, map[string]bool{"wf-a": true}, 1)
	found := false
	for _, r := range results {
		if r.Code == "SUBWORKFLOW_CIRCULAR_REFERENCE" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected SUBWORKFLOW_CIRCULAR_REFERENCE, got %+v", results)
	}
}

func TestValidateSubworkflowChain_NotFound(t *testing.T) {
	resolver := &mockResolver{workflows: map[string]*Workflow{}}

	results := ValidateSubworkflowChain("nonexistent", resolver, map[string]bool{"wf-a": true}, 1)
	found := false
	for _, r := range results {
		if r.Code == "SUBWORKFLOW_NOT_FOUND" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected SUBWORKFLOW_NOT_FOUND, got %+v", results)
	}
}

func TestValidateSubworkflowChain_NestingDepthWarning(t *testing.T) {
	subDef := makeSubWorkflowDef()

	// Create a chain of 6 workflows (exceeds 5-level limit)
	resolver := &mockResolver{workflows: map[string]*Workflow{}}
	for i := 1; i <= 6; i++ {
		wf := &Workflow{ID: fmt.Sprintf("wf-%d", i), Name: fmt.Sprintf("WF %d", i)}
		if i < 6 {
			n := wf.AddNode(subDef, 100, 100, fmt.Sprintf("sub-%d", i))
			n.AttributeValues["workflow_ref"] = fmt.Sprintf("wf-%d", i+1)
		}
		resolver.workflows[wf.ID] = wf
	}

	results := ValidateSubworkflowChain("wf-1", resolver, map[string]bool{"wf-0": true}, 1)
	found := false
	for _, r := range results {
		if r.Code == "SUBWORKFLOW_NESTING_DEPTH" && r.Severity == SeverityWarning {
			found = true
		}
	}
	if !found {
		t.Errorf("expected SUBWORKFLOW_NESTING_DEPTH warning, got %+v", results)
	}
}
