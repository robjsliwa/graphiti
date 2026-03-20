package domain

import (
	"testing"
)

func TestValidateGraphStructure(t *testing.T) {
	tests := []struct {
		name         string
		nodes        []NodeInstance
		edges        []Edge
		wantCodes    []string // expected validation codes
		wantSeverity map[string]Severity
	}{
		{
			name:      "empty workflow",
			nodes:     nil,
			edges:     nil,
			wantCodes: []string{"EMPTY_WORKFLOW"},
			wantSeverity: map[string]Severity{
				"EMPTY_WORKFLOW": SeverityWarning,
			},
		},
		{
			name: "single node no edges (valid starting point)",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "src", Definition: makeSourceDef()},
			},
			edges:     nil,
			wantCodes: nil, // no errors or warnings for a single node
		},
		{
			name: "two nodes one edge (valid)",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "src", Definition: makeSourceDef()},
				{ID: "b", DefinitionID: "proc", Definition: makeProcDef()},
			},
			edges: []Edge{
				{ID: "e1", SourceNodeID: "a", TargetNodeID: "b"},
			},
			wantCodes: nil,
		},
		{
			name: "two nodes no edge (disconnected)",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "src", Definition: makeSourceDef()},
				{ID: "b", DefinitionID: "proc", Definition: makeProcDef()},
			},
			edges:     nil,
			wantCodes: []string{"DISCONNECTED_SUBGRAPH"},
			wantSeverity: map[string]Severity{
				"DISCONNECTED_SUBGRAPH": SeverityError,
			},
		},
		{
			name: "two separate pairs (disconnected)",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "src", Definition: makeSourceDef()},
				{ID: "b", DefinitionID: "proc", Definition: makeProcDef()},
				{ID: "c", DefinitionID: "src", Definition: makeSourceDef()},
				{ID: "d", DefinitionID: "proc", Definition: makeProcDef()},
			},
			edges: []Edge{
				{ID: "e1", SourceNodeID: "a", TargetNodeID: "b"},
				{ID: "e2", SourceNodeID: "c", TargetNodeID: "d"},
			},
			wantCodes: []string{"DISCONNECTED_SUBGRAPH"},
			wantSeverity: map[string]Severity{
				"DISCONNECTED_SUBGRAPH": SeverityError,
			},
		},
		{
			name: "diamond pattern (no cycle, valid)",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "src", Definition: makeSourceDef()},
				{ID: "b", DefinitionID: "proc", Definition: makeProcDef()},
				{ID: "c", DefinitionID: "proc", Definition: makeProcDef()},
				{ID: "d", DefinitionID: "dest", Definition: makeDestDef()},
			},
			edges: []Edge{
				{ID: "e1", SourceNodeID: "a", TargetNodeID: "b"},
				{ID: "e2", SourceNodeID: "a", TargetNodeID: "c"},
				{ID: "e3", SourceNodeID: "b", TargetNodeID: "d"},
				{ID: "e4", SourceNodeID: "c", TargetNodeID: "d"},
			},
			wantCodes: nil,
		},
		{
			name: "simple cycle A->B->A",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "proc", Definition: makeProcDef()},
				{ID: "b", DefinitionID: "proc", Definition: makeProcDef()},
			},
			edges: []Edge{
				{ID: "e1", SourceNodeID: "a", TargetNodeID: "b"},
				{ID: "e2", SourceNodeID: "b", TargetNodeID: "a"},
			},
			wantCodes: []string{"CYCLE_DETECTED", "NO_SOURCE_NODE"},
			wantSeverity: map[string]Severity{
				"CYCLE_DETECTED":  SeverityError,
				"NO_SOURCE_NODE": SeverityError,
			},
		},
		{
			name: "longer cycle A->B->C->A",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "proc", Definition: makeProcDef()},
				{ID: "b", DefinitionID: "proc", Definition: makeProcDef()},
				{ID: "c", DefinitionID: "proc", Definition: makeProcDef()},
			},
			edges: []Edge{
				{ID: "e1", SourceNodeID: "a", TargetNodeID: "b"},
				{ID: "e2", SourceNodeID: "b", TargetNodeID: "c"},
				{ID: "e3", SourceNodeID: "c", TargetNodeID: "a"},
			},
			wantCodes: []string{"CYCLE_DETECTED", "NO_SOURCE_NODE"},
			wantSeverity: map[string]Severity{
				"CYCLE_DETECTED":  SeverityError,
				"NO_SOURCE_NODE": SeverityError,
			},
		},
		{
			name: "self-loop",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "proc", Definition: makeProcDef()},
			},
			edges: []Edge{
				{ID: "e1", SourceNodeID: "a", TargetNodeID: "a"},
			},
			wantCodes: []string{"CYCLE_DETECTED"},
			wantSeverity: map[string]Severity{
				"CYCLE_DETECTED": SeverityError,
			},
		},
		{
			name: "no source node (all have incoming edges)",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "proc", Definition: makeProcDef()},
				{ID: "b", DefinitionID: "proc", Definition: makeProcDef()},
			},
			edges: []Edge{
				{ID: "e1", SourceNodeID: "a", TargetNodeID: "b"},
				{ID: "e2", SourceNodeID: "b", TargetNodeID: "a"},
			},
			wantCodes: []string{"CYCLE_DETECTED", "NO_SOURCE_NODE"},
			wantSeverity: map[string]Severity{
				"NO_SOURCE_NODE": SeverityError,
			},
		},
		{
			name: "no terminal node warning",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "src", Definition: makeSourceDef()},
				{ID: "b", DefinitionID: "proc", Definition: makeProcDefWithOutput()},
			},
			edges: []Edge{
				{ID: "e1", SourceNodeID: "a", TargetNodeID: "b"},
				// b has outputs defined but none connected - it IS a terminal
			},
			// b's output has no edges, so it IS a terminal. No warning.
			wantCodes: nil,
		},
		{
			name: "linear chain of 5 (valid)",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "src", Definition: makeSourceDef()},
				{ID: "b", DefinitionID: "proc", Definition: makeProcDef()},
				{ID: "c", DefinitionID: "proc", Definition: makeProcDef()},
				{ID: "d", DefinitionID: "proc", Definition: makeProcDef()},
				{ID: "e", DefinitionID: "dest", Definition: makeDestDef()},
			},
			edges: []Edge{
				{ID: "e1", SourceNodeID: "a", TargetNodeID: "b"},
				{ID: "e2", SourceNodeID: "b", TargetNodeID: "c"},
				{ID: "e3", SourceNodeID: "c", TargetNodeID: "d"},
				{ID: "e4", SourceNodeID: "d", TargetNodeID: "e"},
			},
			wantCodes: nil,
		},
		{
			name: "fan-out from source (valid)",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "src", Definition: makeSourceDef()},
				{ID: "b", DefinitionID: "proc", Definition: makeProcDef()},
				{ID: "c", DefinitionID: "proc", Definition: makeProcDef()},
				{ID: "d", DefinitionID: "proc", Definition: makeProcDef()},
			},
			edges: []Edge{
				{ID: "e1", SourceNodeID: "a", TargetNodeID: "b"},
				{ID: "e2", SourceNodeID: "a", TargetNodeID: "c"},
				{ID: "e3", SourceNodeID: "a", TargetNodeID: "d"},
			},
			wantCodes: nil,
		},
		{
			name: "fan-in to terminal (valid)",
			nodes: []NodeInstance{
				{ID: "a", DefinitionID: "src", Definition: makeSourceDef()},
				{ID: "b", DefinitionID: "src", Definition: makeSourceDef()},
				{ID: "c", DefinitionID: "src", Definition: makeSourceDef()},
				{ID: "d", DefinitionID: "dest", Definition: makeDestDef()},
			},
			edges: []Edge{
				{ID: "e1", SourceNodeID: "a", TargetNodeID: "d"},
				{ID: "e2", SourceNodeID: "b", TargetNodeID: "d"},
				{ID: "e3", SourceNodeID: "c", TargetNodeID: "d"},
			},
			wantCodes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := &Workflow{
				ID:    "wf-test",
				Nodes: tt.nodes,
				Edges: tt.edges,
			}

			results := wf.validateGraphStructure()

			// Check expected codes are present
			gotCodes := make(map[string]bool)
			for _, r := range results {
				gotCodes[r.Code] = true

				// Verify severity if specified
				if expectedSev, ok := tt.wantSeverity[r.Code]; ok {
					if r.Severity != expectedSev {
						t.Errorf("code %s: got severity %s, want %s", r.Code, r.Severity, expectedSev)
					}
				}

				// Verify category is always "graph"
				if r.Category != CategoryGraph {
					t.Errorf("code %s: got category %s, want %s", r.Code, r.Category, CategoryGraph)
				}
			}

			for _, wantCode := range tt.wantCodes {
				if !gotCodes[wantCode] {
					t.Errorf("expected validation code %s, got codes: %v", wantCode, codeList(results))
				}
			}

			// If no codes expected, ensure no results (or only expected ones)
			if tt.wantCodes == nil && len(results) > 0 {
				t.Errorf("expected no validation results, got: %v", codeList(results))
			}
		})
	}
}

func TestFindConnectedComponents(t *testing.T) {
	tests := []struct {
		name           string
		nodes          []NodeInstance
		edges          []Edge
		wantComponents int
	}{
		{"empty", nil, nil, 0},
		{"single node", []NodeInstance{{ID: "a"}}, nil, 1},
		{"two connected", []NodeInstance{{ID: "a"}, {ID: "b"}}, []Edge{{SourceNodeID: "a", TargetNodeID: "b"}}, 1},
		{"two disconnected", []NodeInstance{{ID: "a"}, {ID: "b"}}, nil, 2},
		{"three components", []NodeInstance{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}, {ID: "e"}},
			[]Edge{{SourceNodeID: "a", TargetNodeID: "b"}}, 4}, // a-b, c, d, e
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := &Workflow{Nodes: tt.nodes, Edges: tt.edges}
			components := wf.findConnectedComponents()
			if len(components) != tt.wantComponents {
				t.Errorf("got %d components, want %d", len(components), tt.wantComponents)
			}
		})
	}
}

func TestFindSourceNodes(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []NodeInstance
		edges     []Edge
		wantCount int
	}{
		{"empty", nil, nil, 0},
		{"single node (is source)", []NodeInstance{{ID: "a"}}, nil, 1},
		{"A->B: A is source", []NodeInstance{{ID: "a"}, {ID: "b"}},
			[]Edge{{SourceNodeID: "a", TargetNodeID: "b"}}, 1},
		{"A->B, C->B: A and C are sources", []NodeInstance{{ID: "a"}, {ID: "b"}, {ID: "c"}},
			[]Edge{{SourceNodeID: "a", TargetNodeID: "b"}, {SourceNodeID: "c", TargetNodeID: "b"}}, 2},
		{"cycle: no sources", []NodeInstance{{ID: "a"}, {ID: "b"}},
			[]Edge{{SourceNodeID: "a", TargetNodeID: "b"}, {SourceNodeID: "b", TargetNodeID: "a"}}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := &Workflow{Nodes: tt.nodes, Edges: tt.edges}
			sources := wf.findSourceNodes()
			if len(sources) != tt.wantCount {
				t.Errorf("got %d sources, want %d", len(sources), tt.wantCount)
			}
		})
	}
}

func TestFindTerminalNodes(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []NodeInstance
		edges     []Edge
		wantCount int
	}{
		{"empty", nil, nil, 0},
		{"single node (is terminal)", []NodeInstance{{ID: "a"}}, nil, 1},
		{"A->B: B is terminal", []NodeInstance{{ID: "a"}, {ID: "b"}},
			[]Edge{{SourceNodeID: "a", TargetNodeID: "b"}}, 1},
		{"A->B, A->C: B and C are terminals", []NodeInstance{{ID: "a"}, {ID: "b"}, {ID: "c"}},
			[]Edge{{SourceNodeID: "a", TargetNodeID: "b"}, {SourceNodeID: "a", TargetNodeID: "c"}}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := &Workflow{Nodes: tt.nodes, Edges: tt.edges}
			terminals := wf.findTerminalNodes()
			if len(terminals) != tt.wantCount {
				t.Errorf("got %d terminals, want %d", len(terminals), tt.wantCount)
			}
		})
	}
}

func TestDetectCycle(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []NodeInstance
		edges     []Edge
		wantCycle bool
	}{
		{"empty", nil, nil, false},
		{"single node", []NodeInstance{{ID: "a"}}, nil, false},
		{"linear A->B->C", []NodeInstance{{ID: "a"}, {ID: "b"}, {ID: "c"}},
			[]Edge{{SourceNodeID: "a", TargetNodeID: "b"}, {SourceNodeID: "b", TargetNodeID: "c"}}, false},
		{"simple cycle A->B->A", []NodeInstance{{ID: "a"}, {ID: "b"}},
			[]Edge{{SourceNodeID: "a", TargetNodeID: "b"}, {SourceNodeID: "b", TargetNodeID: "a"}}, true},
		{"self-loop", []NodeInstance{{ID: "a"}},
			[]Edge{{SourceNodeID: "a", TargetNodeID: "a"}}, true},
		{"diamond no cycle", []NodeInstance{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
			[]Edge{
				{SourceNodeID: "a", TargetNodeID: "b"},
				{SourceNodeID: "a", TargetNodeID: "c"},
				{SourceNodeID: "b", TargetNodeID: "d"},
				{SourceNodeID: "c", TargetNodeID: "d"},
			}, false},
		{"long cycle A->B->C->A", []NodeInstance{{ID: "a"}, {ID: "b"}, {ID: "c"}},
			[]Edge{
				{SourceNodeID: "a", TargetNodeID: "b"},
				{SourceNodeID: "b", TargetNodeID: "c"},
				{SourceNodeID: "c", TargetNodeID: "a"},
			}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := &Workflow{Nodes: tt.nodes, Edges: tt.edges}
			cycle := wf.detectCycle()
			if tt.wantCycle && cycle == nil {
				t.Error("expected cycle to be detected")
			}
			if !tt.wantCycle && cycle != nil {
				t.Errorf("unexpected cycle detected: %v", cycle)
			}
		})
	}
}

// Test helpers

func makeSourceDef() *NodeDefinition {
	return &NodeDefinition{
		ID:       "src",
		Name:     "Source",
		Category: Category{Group: "Sources"},
		Shape:    Shape{Type: ShapeRoundedRect},
		Outputs:  []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
	}
}

func makeProcDef() *NodeDefinition {
	return &NodeDefinition{
		ID:       "proc",
		Name:     "Processor",
		Category: Category{Group: "Processing"},
		Shape:    Shape{Type: ShapeRoundedRect},
		Inputs:   []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}},
	}
}

func makeProcDefWithOutput() *NodeDefinition {
	return &NodeDefinition{
		ID:       "proc",
		Name:     "Processor",
		Category: Category{Group: "Processing"},
		Shape:    Shape{Type: ShapeRoundedRect},
		Inputs:   []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: 1}},
		Outputs:  []PortDefinition{{ID: "out", Type: PortTypeData, MaxConnections: -1}},
	}
}

func makeDestDef() *NodeDefinition {
	return &NodeDefinition{
		ID:       "dest",
		Name:     "Destination",
		Category: Category{Group: "Destinations"},
		Shape:    Shape{Type: ShapeRoundedRect},
		Inputs:   []PortDefinition{{ID: "in", Type: PortTypeData, MaxConnections: -1}},
	}
}

func codeList(results []ValidationResult) []string {
	codes := make([]string, len(results))
	for i, r := range results {
		codes[i] = r.Code
	}
	return codes
}
