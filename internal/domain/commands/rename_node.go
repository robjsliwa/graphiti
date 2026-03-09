package commands

import (
	"encoding/json"
	"graphiti/internal/domain"
)

// RenameNodeCommand changes the label of a node.
type RenameNodeCommand struct {
	NodeID   string `json:"nodeId"`
	OldLabel string `json:"oldLabel"`
	NewLabel string `json:"newLabel"`
}

func (c *RenameNodeCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
	node := wf.FindNode(c.NodeID)
	if node == nil {
		return nil, domain.ErrNodeNotFound
	}

	node.Label = c.NewLabel

	return &RenameNodeCommand{
		NodeID:   c.NodeID,
		OldLabel: c.NewLabel,
		NewLabel: c.OldLabel,
	}, nil
}

func (c *RenameNodeCommand) Type() string             { return "rename_node" }
func (c *RenameNodeCommand) Serialize() ([]byte, error) { return json.Marshal(c) }

func init() {
	domain.RegisterCommand("rename_node", func(data []byte) (domain.Command, error) {
		var cmd RenameNodeCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return &cmd, nil
	})
}
