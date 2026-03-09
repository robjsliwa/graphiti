package commands

import (
	"encoding/json"
	"graphiti/internal/domain"
)

// MoveNodeCommand changes a node's position on the canvas.
type MoveNodeCommand struct {
	NodeID string  `json:"nodeId"`
	FromX  float64 `json:"fromX"`
	FromY  float64 `json:"fromY"`
	ToX    float64 `json:"toX"`
	ToY    float64 `json:"toY"`
}

func (c *MoveNodeCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
	node := wf.FindNode(c.NodeID)
	if node == nil {
		return nil, domain.ErrNodeNotFound
	}
	node.X = c.ToX
	node.Y = c.ToY
	return &MoveNodeCommand{
		NodeID: c.NodeID,
		FromX:  c.ToX,
		FromY:  c.ToY,
		ToX:    c.FromX,
		ToY:    c.FromY,
	}, nil
}

func (c *MoveNodeCommand) Type() string             { return "move_node" }
func (c *MoveNodeCommand) Serialize() ([]byte, error) { return json.Marshal(c) }

func init() {
	domain.RegisterCommand("move_node", func(data []byte) (domain.Command, error) {
		var cmd MoveNodeCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return &cmd, nil
	})
}
