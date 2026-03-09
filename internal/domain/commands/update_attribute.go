package commands

import (
	"encoding/json"
	"graphiti/internal/domain"
)

// UpdateAttributeCommand changes a single attribute value on a node.
type UpdateAttributeCommand struct {
	NodeID   string `json:"nodeId"`
	AttrID   string `json:"attrId"`
	OldValue any    `json:"oldValue"`
	NewValue any    `json:"newValue"`
}

func (c *UpdateAttributeCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
	node := wf.FindNode(c.NodeID)
	if node == nil {
		return nil, domain.ErrNodeNotFound
	}

	if node.AttributeValues == nil {
		node.AttributeValues = make(map[string]any)
	}

	old := node.AttributeValues[c.AttrID]
	node.AttributeValues[c.AttrID] = c.NewValue

	return &UpdateAttributeCommand{
		NodeID:   c.NodeID,
		AttrID:   c.AttrID,
		OldValue: c.NewValue,
		NewValue: old,
	}, nil
}

func (c *UpdateAttributeCommand) Type() string             { return "update_attribute" }
func (c *UpdateAttributeCommand) Serialize() ([]byte, error) { return json.Marshal(c) }

func init() {
	domain.RegisterCommand("update_attribute", func(data []byte) (domain.Command, error) {
		var cmd UpdateAttributeCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return &cmd, nil
	})
}
