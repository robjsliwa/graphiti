package domain

import "testing"

func TestValidateAll_EmptyWorkflow(t *testing.T) {
	wf := &Workflow{ID: "wf-1"}
	reg := newTestRegistry()

	results := wf.ValidateAll(reg)
	if !hasCode(results, "EMPTY_WORKFLOW") {
		t.Errorf("expected EMPTY_WORKFLOW for empty workflow, got: %v", codeList(results))
	}
}

func TestValidateAll_ValidSimpleWorkflow(t *testing.T) {
	srcDef := &NodeDefinition{
		ID: "src", Name: "Source", Category: Category{Group: "Sources"},
		Shape:   Shape{Type: ShapeRoundedRect},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
	}
	tgtDef := &NodeDefinition{
		ID: "tgt", Name: "Target", Category: Category{Group: "Processing"},
		Shape:  Shape{Type: ShapeRoundedRect},
		Inputs: []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}},
	}
	reg := newTestRegistry(srcDef, tgtDef)

	wf := &Workflow{
		ID: "wf-1",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "src", Label: "Source", Definition: srcDef},
			{ID: "n2", DefinitionID: "tgt", Label: "Target", Definition: tgtDef},
		},
		Edges: []Edge{
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "in"},
		},
	}

	results := wf.ValidateAll(reg)
	errors := ErrorsOnly(results)
	if len(errors) > 0 {
		t.Errorf("expected no errors for valid workflow, got: %v", codeList(errors))
	}
}

func TestValidateAll_CombinesAllValidators(t *testing.T) {
	// A workflow with multiple issues: disconnected node, missing required attr, bad edge
	srcDef := &NodeDefinition{
		ID: "src", Name: "Source", Category: Category{Group: "Sources"},
		Shape:   Shape{Type: ShapeRoundedRect},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
		Attributes: []AttributeDefinition{
			{ID: "url", Label: "URL", Type: AttrTypeString, Required: true},
		},
	}
	tgtDef := &NodeDefinition{
		ID: "tgt", Name: "Target", Category: Category{Group: "Processing"},
		Shape:  Shape{Type: ShapeRoundedRect},
		Inputs: []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}},
	}
	reg := newTestRegistry(srcDef, tgtDef)

	wf := &Workflow{
		ID: "wf-1",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "src", Label: "Source", Definition: srcDef, AttributeValues: map[string]any{}},
			{ID: "n2", DefinitionID: "tgt", Label: "Target", Definition: tgtDef},
			{ID: "n3", DefinitionID: "tgt", Label: "Orphan", Definition: tgtDef}, // disconnected
		},
		Edges: []Edge{
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "in"},
		},
	}

	results := wf.ValidateAll(reg)

	// Should have graph error (disconnected), attribute error (missing required)
	if !hasCode(results, "DISCONNECTED_SUBGRAPH") {
		t.Error("expected DISCONNECTED_SUBGRAPH")
	}
	if !hasCode(results, "ATTR_REQUIRED_MISSING") {
		t.Error("expected ATTR_REQUIRED_MISSING")
	}
}

func TestValidateAll_HasErrors(t *testing.T) {
	results := []ValidationResult{
		{Severity: SeverityWarning, Code: "EMPTY_WORKFLOW"},
	}
	if HasErrors(results) {
		t.Error("warnings should not count as errors")
	}

	results = append(results, ValidationResult{Severity: SeverityError, Code: "CYCLE_DETECTED"})
	if !HasErrors(results) {
		t.Error("should have errors")
	}
}

func TestValidateAll_ByNode(t *testing.T) {
	results := []ValidationResult{
		{NodeID: "n1", Code: "A"},
		{NodeID: "n1", Code: "B"},
		{NodeID: "n2", Code: "C"},
		{NodeID: "", Code: "D"}, // workflow-level
	}

	byNode := ByNode(results)
	if len(byNode["n1"]) != 2 {
		t.Errorf("expected 2 results for n1, got %d", len(byNode["n1"]))
	}
	if len(byNode["n2"]) != 1 {
		t.Errorf("expected 1 result for n2, got %d", len(byNode["n2"]))
	}
	if len(byNode[""]) != 1 {
		t.Errorf("expected 1 workflow-level result, got %d", len(byNode[""]))
	}
}

func TestValidateAll_SubworkflowIntegration(t *testing.T) {
	subDef := &NodeDefinition{
		ID: "sub", Name: "Sub-Workflow", Category: Category{Group: "Control"},
		Shape:  Shape{Type: ShapeSubWorkflow},
		Inputs: []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}},
	}
	srcDef := &NodeDefinition{
		ID: "src", Name: "Source", Category: Category{Group: "Sources"},
		Shape:   Shape{Type: ShapeRoundedRect},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
	}
	reg := newTestRegistry(subDef, srcDef)

	wf := &Workflow{
		ID: "wf-1",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "src", Label: "Source", Definition: srcDef},
			{ID: "n2", DefinitionID: "sub", Label: "Sub", Definition: subDef,
				AttributeValues: map[string]any{"workflow_ref": ""}}, // missing ref
		},
		Edges: []Edge{
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "in"},
		},
	}

	results := wf.ValidateAll(reg)
	if !hasCode(results, "SUBWORKFLOW_NO_REFERENCE") {
		t.Errorf("expected SUBWORKFLOW_NO_REFERENCE, got: %v", codeList(results))
	}
}
