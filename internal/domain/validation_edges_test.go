package domain

import (
	"fmt"
	"testing"
)

// testRegistry implements NodeDefinitionRegistry for tests.
type testRegistry struct {
	defs map[string]*NodeDefinition
}

func (r *testRegistry) GetByID(id string) *NodeDefinition {
	return r.defs[id]
}

func newTestRegistry(defs ...*NodeDefinition) *testRegistry {
	m := make(map[string]*NodeDefinition)
	for _, d := range defs {
		m[d.ID] = d
	}
	return &testRegistry{defs: m}
}

func TestValidateEdges_PortTypeMismatch(t *testing.T) {
	srcDef := &NodeDefinition{
		ID:       "src",
		Name:     "Source",
		Category: Category{Group: "Sources"},
		Outputs:  []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
	}
	tgtDef := &NodeDefinition{
		ID:       "tgt",
		Name:     "Target",
		Category: Category{Group: "Processing"},
		Inputs:   []PortDefinition{{ID: "in", Type: PortTypeControl, MaxConnections: 1}},
	}
	registry := newTestRegistry(srcDef, tgtDef)

	tests := []struct {
		name       string
		srcPortType PortType
		tgtPortType PortType
		wantCode   string
	}{
		{"data to data", PortTypeData, PortTypeData, ""},
		{"data to control", PortTypeData, PortTypeControl, "PORT_TYPE_MISMATCH"},
		{"data to error", PortTypeData, PortTypeError, "PORT_TYPE_MISMATCH"},
		{"error to error", PortTypeError, PortTypeError, ""},
		{"control to control", PortTypeControl, PortTypeControl, ""},
		{"control to data", PortTypeControl, PortTypeData, "PORT_TYPE_MISMATCH"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &NodeDefinition{
				ID: "src", Name: "Source", Category: Category{Group: "Sources"},
				Outputs: []PortDefinition{{ID: "out", Type: tt.srcPortType, MaxConnections: -1}},
			}
			tgt := &NodeDefinition{
				ID: "tgt", Name: "Target", Category: Category{Group: "Processing"},
				Inputs: []PortDefinition{{ID: "in", Type: tt.tgtPortType, MaxConnections: 1}},
			}
			reg := newTestRegistry(src, tgt)

			wf := &Workflow{
				ID: "wf-test",
				Nodes: []NodeInstance{
					{ID: "n1", DefinitionID: "src", Definition: src},
					{ID: "n2", DefinitionID: "tgt", Definition: tgt},
				},
				Edges: []Edge{
					{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "in"},
				},
			}

			results := wf.validateEdges(reg)
			if tt.wantCode == "" {
				if hasCode(results, "PORT_TYPE_MISMATCH") {
					t.Errorf("unexpected PORT_TYPE_MISMATCH for %s -> %s", tt.srcPortType, tt.tgtPortType)
				}
			} else {
				if !hasCode(results, tt.wantCode) {
					t.Errorf("expected %s for %s -> %s, got: %v", tt.wantCode, tt.srcPortType, tt.tgtPortType, codeList(results))
				}
			}
		})
	}

	_ = registry // prevent unused
}

func TestValidateEdges_MissingSourceNode(t *testing.T) {
	tgtDef := &NodeDefinition{
		ID: "tgt", Name: "Target", Category: Category{Group: "Processing"},
		Inputs: []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}},
	}
	reg := newTestRegistry(tgtDef)

	wf := &Workflow{
		ID: "wf-test",
		Nodes: []NodeInstance{
			{ID: "n2", DefinitionID: "tgt", Definition: tgtDef},
		},
		Edges: []Edge{
			{ID: "e1", SourceNodeID: "nonexistent", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "in"},
		},
	}

	results := wf.validateEdges(reg)
	if !hasCode(results, "EDGE_MISSING_SOURCE") {
		t.Errorf("expected EDGE_MISSING_SOURCE, got: %v", codeList(results))
	}
}

func TestValidateEdges_MissingTargetNode(t *testing.T) {
	srcDef := &NodeDefinition{
		ID: "src", Name: "Source", Category: Category{Group: "Sources"},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
	}
	reg := newTestRegistry(srcDef)

	wf := &Workflow{
		ID: "wf-test",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "src", Definition: srcDef},
		},
		Edges: []Edge{
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "nonexistent", TargetPortID: "in"},
		},
	}

	results := wf.validateEdges(reg)
	if !hasCode(results, "EDGE_MISSING_TARGET") {
		t.Errorf("expected EDGE_MISSING_TARGET, got: %v", codeList(results))
	}
}

func TestValidateEdges_MissingPorts(t *testing.T) {
	srcDef := &NodeDefinition{
		ID: "src", Name: "Source", Category: Category{Group: "Sources"},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
	}
	tgtDef := &NodeDefinition{
		ID: "tgt", Name: "Target", Category: Category{Group: "Processing"},
		Inputs: []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}},
	}
	reg := newTestRegistry(srcDef, tgtDef)

	t.Run("missing source port", func(t *testing.T) {
		wf := &Workflow{
			ID: "wf-test",
			Nodes: []NodeInstance{
				{ID: "n1", DefinitionID: "src", Definition: srcDef},
				{ID: "n2", DefinitionID: "tgt", Definition: tgtDef},
			},
			Edges: []Edge{
				{ID: "e1", SourceNodeID: "n1", SourcePortID: "nonexistent", TargetNodeID: "n2", TargetPortID: "in"},
			},
		}
		results := wf.validateEdges(reg)
		if !hasCode(results, "EDGE_MISSING_SOURCE_PORT") {
			t.Errorf("expected EDGE_MISSING_SOURCE_PORT, got: %v", codeList(results))
		}
	})

	t.Run("missing target port", func(t *testing.T) {
		wf := &Workflow{
			ID: "wf-test",
			Nodes: []NodeInstance{
				{ID: "n1", DefinitionID: "src", Definition: srcDef},
				{ID: "n2", DefinitionID: "tgt", Definition: tgtDef},
			},
			Edges: []Edge{
				{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "nonexistent"},
			},
		}
		results := wf.validateEdges(reg)
		if !hasCode(results, "EDGE_MISSING_TARGET_PORT") {
			t.Errorf("expected EDGE_MISSING_TARGET_PORT, got: %v", codeList(results))
		}
	})
}

func TestValidateEdges_WrongDirection(t *testing.T) {
	// Source port is actually an input, target port is actually an output
	def := &NodeDefinition{
		ID: "node", Name: "Node", Category: Category{Group: "Processing"},
		Inputs:  []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
	}
	reg := newTestRegistry(def)

	wf := &Workflow{
		ID: "wf-test",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "node", Definition: def},
			{ID: "n2", DefinitionID: "node", Definition: def},
		},
		Edges: []Edge{
			// Source uses an input port, target uses an output port - wrong direction
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "in", TargetNodeID: "n2", TargetPortID: "out"},
		},
	}

	results := wf.validateEdges(reg)
	if !hasCode(results, "EDGE_WRONG_DIRECTION") {
		t.Errorf("expected EDGE_WRONG_DIRECTION, got: %v", codeList(results))
	}
}

func TestValidateEdges_DuplicateEdge(t *testing.T) {
	srcDef := &NodeDefinition{
		ID: "src", Name: "Source", Category: Category{Group: "Sources"},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
	}
	tgtDef := &NodeDefinition{
		ID: "tgt", Name: "Target", Category: Category{Group: "Processing"},
		Inputs: []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: -1}},
	}
	reg := newTestRegistry(srcDef, tgtDef)

	wf := &Workflow{
		ID: "wf-test",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "src", Definition: srcDef},
			{ID: "n2", DefinitionID: "tgt", Definition: tgtDef},
		},
		Edges: []Edge{
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "in"},
			{ID: "e2", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "in"},
		},
	}

	results := wf.validateEdges(reg)
	if !hasCode(results, "EDGE_DUPLICATE") {
		t.Errorf("expected EDGE_DUPLICATE, got: %v", codeList(results))
	}
}

func TestValidateEdges_SelfReference(t *testing.T) {
	def := &NodeDefinition{
		ID: "node", Name: "Node", Category: Category{Group: "Processing"},
		Inputs:  []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
	}
	reg := newTestRegistry(def)

	wf := &Workflow{
		ID: "wf-test",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "node", Definition: def},
		},
		Edges: []Edge{
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n1", TargetPortID: "in"},
		},
	}

	results := wf.validateEdges(reg)
	if !hasCode(results, "EDGE_SELF_REFERENCE") {
		t.Errorf("expected EDGE_SELF_REFERENCE, got: %v", codeList(results))
	}
}

func TestValidateEdges_MaxConnections(t *testing.T) {
	tests := []struct {
		name      string
		maxConn   int
		edgeCount int
		wantCode  string
	}{
		{"first connection to max-1 input", 1, 1, ""},
		{"second connection to max-1 input", 1, 2, "PORT_MAX_CONNECTIONS"},
		{"first connection to max-3 input", 3, 1, ""},
		{"fourth connection to max-3 input", 3, 4, "PORT_MAX_CONNECTIONS"},
		{"unlimited output (50 edges)", -1, 50, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcDef := &NodeDefinition{
				ID: "src", Name: "Source", Category: Category{Group: "Sources"},
				Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
			}
			tgtDef := &NodeDefinition{
				ID: "tgt", Name: "Target", Category: Category{Group: "Processing"},
				Inputs: []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: tt.maxConn}},
			}
			reg := newTestRegistry(srcDef, tgtDef)

			nodes := []NodeInstance{
				{ID: "tgt", DefinitionID: "tgt", Definition: tgtDef},
			}
			var edges []Edge
			for i := range tt.edgeCount {
				srcID := fmt.Sprintf("src-%d", i)
				nodes = append(nodes, NodeInstance{ID: srcID, DefinitionID: "src", Definition: srcDef})
				edges = append(edges, Edge{
					ID: fmt.Sprintf("e-%d", i), SourceNodeID: srcID, SourcePortID: "out",
					TargetNodeID: "tgt", TargetPortID: "in",
				})
			}

			wf := &Workflow{ID: "wf-test", Nodes: nodes, Edges: edges}
			results := wf.validateConnectionLimits(reg)

			if tt.wantCode == "" {
				if hasCode(results, "PORT_MAX_CONNECTIONS") {
					t.Errorf("unexpected PORT_MAX_CONNECTIONS with %d edges and max %d", tt.edgeCount, tt.maxConn)
				}
			} else {
				if !hasCode(results, tt.wantCode) {
					t.Errorf("expected %s, got: %v", tt.wantCode, codeList(results))
				}
			}
		})
	}
}

func TestValidateEdges_ConnectionCategoryDenied(t *testing.T) {
	srcDef := &NodeDefinition{
		ID: "src", Name: "Source", Category: Category{Group: "Sources"},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
		Validation: ValidationRules{
			ConnectionRules: []ConnectionRule{
				{OutputPort: "out", AllowedTargetCategories: []string{"Processing", "Destinations"}},
			},
		},
	}
	ctrlDef := &NodeDefinition{
		ID: "ctrl", Name: "Control", Category: Category{Group: "Control"},
		Inputs: []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}},
	}
	reg := newTestRegistry(srcDef, ctrlDef)

	wf := &Workflow{
		ID: "wf-test",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "src", Definition: srcDef},
			{ID: "n2", DefinitionID: "ctrl", Definition: ctrlDef},
		},
		Edges: []Edge{
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "in"},
		},
	}

	results := wf.validateEdges(reg)
	if !hasCode(results, "CONNECTION_CATEGORY_DENIED") {
		t.Errorf("expected CONNECTION_CATEGORY_DENIED, got: %v", codeList(results))
	}
}

func TestValidateEdges_ConnectionPortTypeDenied(t *testing.T) {
	srcDef := &NodeDefinition{
		ID: "src", Name: "Source", Category: Category{Group: "Sources"},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
		Validation: ValidationRules{
			ConnectionRules: []ConnectionRule{
				{OutputPort: "out", AllowedTargetPorts: []string{"control"}},
			},
		},
	}
	tgtDef := &NodeDefinition{
		ID: "tgt", Name: "Target", Category: Category{Group: "Processing"},
		Inputs: []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}},
	}
	reg := newTestRegistry(srcDef, tgtDef)

	wf := &Workflow{
		ID: "wf-test",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "src", Definition: srcDef},
			{ID: "n2", DefinitionID: "tgt", Definition: tgtDef},
		},
		Edges: []Edge{
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "in"},
		},
	}

	results := wf.validateEdges(reg)
	if !hasCode(results, "CONNECTION_PORT_TYPE_DENIED") {
		t.Errorf("expected CONNECTION_PORT_TYPE_DENIED, got: %v", codeList(results))
	}
}

func TestValidateEdges_ValidWorkflow(t *testing.T) {
	srcDef := &NodeDefinition{
		ID: "src", Name: "Source", Category: Category{Group: "Sources"},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
	}
	tgtDef := &NodeDefinition{
		ID: "tgt", Name: "Target", Category: Category{Group: "Processing"},
		Inputs: []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}},
	}
	reg := newTestRegistry(srcDef, tgtDef)

	wf := &Workflow{
		ID: "wf-test",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "src", Definition: srcDef},
			{ID: "n2", DefinitionID: "tgt", Definition: tgtDef},
		},
		Edges: []Edge{
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "in"},
		},
	}

	results := wf.validateEdges(reg)
	if len(results) > 0 {
		t.Errorf("expected no validation results for valid workflow, got: %v", codeList(results))
	}
}

func TestValidateEdges_AllSeveritiesAreError(t *testing.T) {
	// All edge validation issues should be errors (block deploy)
	srcDef := &NodeDefinition{
		ID: "src", Name: "Source", Category: Category{Group: "Sources"},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
	}
	reg := newTestRegistry(srcDef)

	wf := &Workflow{
		ID: "wf-test",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "src", Definition: srcDef},
		},
		Edges: []Edge{
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "missing", TargetPortID: "in"},
		},
	}

	results := wf.validateEdges(reg)
	for _, r := range results {
		if r.Severity != SeverityError {
			t.Errorf("edge validation code %s has severity %s, expected error", r.Code, r.Severity)
		}
	}
}

// hasCode checks if any result has the given code.
func hasCode(results []ValidationResult, code string) bool {
	for _, r := range results {
		if r.Code == code {
			return true
		}
	}
	return false
}

