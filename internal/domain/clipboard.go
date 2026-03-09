package domain

// ClipboardPayload holds the serialized data for copy/paste operations.
type ClipboardPayload struct {
	Version int             `json:"version"`
	Source  string          `json:"source"`
	Nodes   []ClipboardNode `json:"nodes"`
	Edges   []ClipboardEdge `json:"edges"`
}

// ClipboardNode is a portable representation of a node for clipboard operations.
type ClipboardNode struct {
	OriginalID   string         `json:"originalId"`
	DefinitionID string         `json:"definitionId"`
	Label        string         `json:"label"`
	RelativeX    float64        `json:"relativeX"`
	RelativeY    float64        `json:"relativeY"`
	Attributes   map[string]any `json:"attributes"`
}

// ClipboardEdge is a portable representation of an edge for clipboard operations.
type ClipboardEdge struct {
	SourceOriginalID string `json:"sourceOriginalId"`
	SourcePortID     string `json:"sourcePortId"`
	TargetOriginalID string `json:"targetOriginalId"`
	TargetPortID     string `json:"targetPortId"`
}

// BuildClipboardPayload creates a ClipboardPayload from selected nodes in a workflow.
// Only edges where both endpoints are in the selection are included.
func BuildClipboardPayload(wf *Workflow, nodeIDs []string, source string) *ClipboardPayload {
	selected := make(map[string]bool, len(nodeIDs))
	for _, id := range nodeIDs {
		selected[id] = true
	}

	// Find bounding box top-left
	var minX, minY float64
	first := true
	var nodes []ClipboardNode
	for _, n := range wf.Nodes {
		if !selected[n.ID] {
			continue
		}
		if first {
			minX, minY = n.X, n.Y
			first = true
		}
		if first || n.X < minX {
			minX = n.X
		}
		if first || n.Y < minY {
			minY = n.Y
		}
		first = false
	}

	for _, n := range wf.Nodes {
		if !selected[n.ID] {
			continue
		}
		nodes = append(nodes, ClipboardNode{
			OriginalID:   n.ID,
			DefinitionID: n.DefinitionID,
			Label:        n.Label,
			RelativeX:    n.X - minX,
			RelativeY:    n.Y - minY,
			Attributes:   n.AttributeValues,
		})
	}

	var edges []ClipboardEdge
	for _, e := range wf.Edges {
		if selected[e.SourceNodeID] && selected[e.TargetNodeID] {
			edges = append(edges, ClipboardEdge{
				SourceOriginalID: e.SourceNodeID,
				SourcePortID:     e.SourcePortID,
				TargetOriginalID: e.TargetNodeID,
				TargetPortID:     e.TargetPortID,
			})
		}
	}

	return &ClipboardPayload{
		Version: 1,
		Source:  source,
		Nodes:   nodes,
		Edges:   edges,
	}
}
