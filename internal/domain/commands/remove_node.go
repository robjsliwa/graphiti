package commands

import (
	"encoding/json"
	"graphiti/internal/domain"
)

// RemoveNodeCommand removes a node and all its connected edges from the workflow.
type RemoveNodeCommand struct {
	NodeID       string              `json:"nodeId"`
	RestoreNode  domain.NodeInstance `json:"restoreNode"`
	RestoreEdges []domain.Edge       `json:"restoreEdges"`
}

func (c *RemoveNodeCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
	// If we have restore data, this is an undo of a remove: restore the node and edges
	if c.RestoreNode.ID != "" && wf.FindNode(c.RestoreNode.ID) == nil {
		wf.RestoreNode(c.RestoreNode)
		for _, e := range c.RestoreEdges {
			wf.RestoreEdge(e)
		}
		// Inverse: remove the node again
		return &RemoveNodeCommand{
			NodeID: c.RestoreNode.ID,
		}, nil
	}

	// Normal remove
	removed, removedEdges, err := wf.RemoveNode(c.NodeID)
	if err != nil {
		return nil, err
	}

	// Inverse: an undo that restores the node and edges
	return &RemoveNodeCommand{
		NodeID:       removed.ID,
		RestoreNode:  *removed,
		RestoreEdges: removedEdges,
	}, nil
}

func (c *RemoveNodeCommand) Type() string             { return "remove_node" }
func (c *RemoveNodeCommand) Serialize() ([]byte, error) { return json.Marshal(c) }

func init() {
	domain.RegisterCommand("remove_node", func(data []byte) (domain.Command, error) {
		var cmd RemoveNodeCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return &cmd, nil
	})
}
