package commands

import (
	"encoding/json"
	"graphiti/internal/domain"
)

// PasteNodesCommand pastes previously copied nodes onto the workflow.
type PasteNodesCommand struct {
	Nodes     []domain.NodeInstance `json:"nodes"`
	Edges     []domain.Edge         `json:"edges"`
	IDMapping map[string]string     `json:"idMapping"` // old ID -> new ID

	// pastedNodeIDs tracks which IDs were actually added (for undo)
	pastedNodeIDs []string
}

func (c *PasteNodesCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
	var addedNodeIDs []string

	for _, n := range c.Nodes {
		wf.RestoreNode(n)
		addedNodeIDs = append(addedNodeIDs, n.ID)
	}

	for _, e := range c.Edges {
		wf.RestoreEdge(e)
	}

	c.pastedNodeIDs = addedNodeIDs

	// Inverse: remove all pasted nodes (which also removes their edges)
	return &removePastedNodesCommand{
		Nodes: c.Nodes,
		Edges: c.Edges,
	}, nil
}

func (c *PasteNodesCommand) Type() string             { return "paste_nodes" }
func (c *PasteNodesCommand) Serialize() ([]byte, error) { return json.Marshal(c) }

// removePastedNodesCommand is the inverse of paste: removes all pasted nodes/edges.
type removePastedNodesCommand struct {
	Nodes []domain.NodeInstance `json:"nodes"`
	Edges []domain.Edge         `json:"edges"`
}

func (c *removePastedNodesCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
	// Remove edges first
	for _, e := range c.Edges {
		wf.RemoveEdge(e.ID)
	}
	// Remove nodes
	for _, n := range c.Nodes {
		wf.RemoveNode(n.ID)
	}

	// Inverse: paste them all back
	return &PasteNodesCommand{
		Nodes: c.Nodes,
		Edges: c.Edges,
	}, nil
}

func (c *removePastedNodesCommand) Type() string             { return "remove_pasted_nodes" }
func (c *removePastedNodesCommand) Serialize() ([]byte, error) { return json.Marshal(c) }

func init() {
	domain.RegisterCommand("paste_nodes", func(data []byte) (domain.Command, error) {
		var cmd PasteNodesCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return &cmd, nil
	})
	domain.RegisterCommand("remove_pasted_nodes", func(data []byte) (domain.Command, error) {
		var cmd removePastedNodesCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return &cmd, nil
	})
}
