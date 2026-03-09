package commands

import (
	"encoding/json"
	"fmt"
	"graphiti/internal/domain"
)

// NodePosition captures a node ID and its x/y coordinates.
type NodePosition struct {
	NodeID string  `json:"nodeId"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

// MoveNodesCommand moves multiple nodes as a single undoable operation.
type MoveNodesCommand struct {
	From []NodePosition `json:"from"`
	To   []NodePosition `json:"to"`
}

func (c *MoveNodesCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
	if len(c.From) != len(c.To) {
		return nil, fmt.Errorf("from/to position count mismatch: %d vs %d", len(c.From), len(c.To))
	}

	// Validate all nodes exist before mutating
	for _, pos := range c.To {
		if wf.FindNode(pos.NodeID) == nil {
			return nil, fmt.Errorf("%w: %s", domain.ErrNodeNotFound, pos.NodeID)
		}
	}

	// Apply moves
	for _, pos := range c.To {
		node := wf.FindNode(pos.NodeID)
		node.X = pos.X
		node.Y = pos.Y
	}

	// Inverse: swap from/to
	return &MoveNodesCommand{
		From: c.To,
		To:   c.From,
	}, nil
}

func (c *MoveNodesCommand) Type() string             { return "move_nodes" }
func (c *MoveNodesCommand) Serialize() ([]byte, error) { return json.Marshal(c) }

func init() {
	domain.RegisterCommand("move_nodes", func(data []byte) (domain.Command, error) {
		var cmd MoveNodesCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return &cmd, nil
	})
}
