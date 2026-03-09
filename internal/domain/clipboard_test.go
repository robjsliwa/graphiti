package domain

import (
	"testing"
)

func TestBuildClipboardPayload_BasicCopy(t *testing.T) {
	wf := &Workflow{
		ID: "wf-1",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "src", Label: "Node 1", X: 100, Y: 200, AttributeValues: map[string]any{"key": "val"}},
			{ID: "n2", DefinitionID: "proc", Label: "Node 2", X: 200, Y: 200},
			{ID: "n3", DefinitionID: "dest", Label: "Node 3", X: 300, Y: 300},
		},
		Edges: []Edge{
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "out", TargetNodeID: "n2", TargetPortID: "in"},
			{ID: "e2", SourceNodeID: "n2", SourcePortID: "out", TargetNodeID: "n3", TargetPortID: "in"},
		},
	}

	// Copy n1 and n2 only
	payload := BuildClipboardPayload(wf, []string{"n1", "n2"}, "wf-1")

	if payload.Version != 1 {
		t.Errorf("expected version 1, got %d", payload.Version)
	}
	if len(payload.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(payload.Nodes))
	}
	// Only e1 should be included (both endpoints in selection)
	if len(payload.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(payload.Edges))
	}
	if payload.Edges[0].SourceOriginalID != "n1" {
		t.Errorf("expected edge from n1, got %s", payload.Edges[0].SourceOriginalID)
	}
}

func TestBuildClipboardPayload_RelativePositions(t *testing.T) {
	wf := &Workflow{
		ID: "wf-1",
		Nodes: []NodeInstance{
			{ID: "n1", X: 100, Y: 200},
			{ID: "n2", X: 300, Y: 400},
		},
	}

	payload := BuildClipboardPayload(wf, []string{"n1", "n2"}, "wf-1")

	// Min is (100, 200), so n1 should be (0, 0) and n2 should be (200, 200)
	for _, n := range payload.Nodes {
		if n.OriginalID == "n1" {
			if n.RelativeX != 0 || n.RelativeY != 0 {
				t.Errorf("n1: expected (0, 0), got (%v, %v)", n.RelativeX, n.RelativeY)
			}
		}
		if n.OriginalID == "n2" {
			if n.RelativeX != 200 || n.RelativeY != 200 {
				t.Errorf("n2: expected (200, 200), got (%v, %v)", n.RelativeX, n.RelativeY)
			}
		}
	}
}

func TestBuildClipboardPayload_EmptySelection(t *testing.T) {
	wf := &Workflow{ID: "wf-1"}
	payload := BuildClipboardPayload(wf, []string{}, "wf-1")

	if len(payload.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(payload.Nodes))
	}
	if len(payload.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(payload.Edges))
	}
}
