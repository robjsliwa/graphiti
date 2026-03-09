package commands

import (
	"encoding/json"
	"errors"
	"graphiti/internal/domain"
)

// AddNodeCommand adds a new node instance to the workflow.
type AddNodeCommand struct {
	DefinitionID string  `json:"definitionId"`
	InstanceID   string  `json:"instanceId"`
	Label        string  `json:"label"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`

	// nodeRegistry is injected for resolving definition IDs to definitions.
	nodeRegistry NodeDefinitionResolver
}

// NodeDefinitionResolver resolves a definition ID to a NodeDefinition.
type NodeDefinitionResolver interface {
	GetByID(id string) (*domain.NodeDefinition, error)
}

// NewAddNodeCommand creates an AddNodeCommand with its required resolver.
func NewAddNodeCommand(defID, instanceID, label string, x, y float64, resolver NodeDefinitionResolver) *AddNodeCommand {
	return &AddNodeCommand{
		DefinitionID: defID,
		InstanceID:   instanceID,
		Label:        label,
		X:            x,
		Y:            y,
		nodeRegistry: resolver,
	}
}

func (c *AddNodeCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
	var def *domain.NodeDefinition
	if c.nodeRegistry != nil {
		var err error
		def, err = c.nodeRegistry.GetByID(c.DefinitionID)
		if err != nil {
			return nil, err
		}
	}
	if def == nil {
		return nil, errors.New("node definition not found: " + c.DefinitionID)
	}

	node := wf.AddNode(def, c.X, c.Y, c.InstanceID)
	if c.Label != "" {
		node.Label = c.Label
	}

	return &RemoveNodeCommand{
		NodeID:       node.ID,
		RestoreNode:  *node,
		RestoreEdges: nil,
	}, nil
}

func (c *AddNodeCommand) Type() string             { return "add_node" }
func (c *AddNodeCommand) Serialize() ([]byte, error) { return json.Marshal(c) }

func init() {
	domain.RegisterCommand("add_node", func(data []byte) (domain.Command, error) {
		var cmd AddNodeCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return &cmd, nil
	})
}
