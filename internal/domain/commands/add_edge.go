package commands

import (
	"encoding/json"
	"graphiti/internal/domain"
)

// AddEdgeCommand creates a new edge between two ports.
type AddEdgeCommand struct {
	EdgeID       string `json:"edgeId"`
	SourceNodeID string `json:"sourceNodeId"`
	SourcePortID string `json:"sourcePortId"`
	TargetNodeID string `json:"targetNodeId"`
	TargetPortID string `json:"targetPortId"`
}

func (c *AddEdgeCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
	err := wf.AddEdge(c.SourceNodeID, c.SourcePortID, c.TargetNodeID, c.TargetPortID, c.EdgeID)
	if err != nil {
		return nil, err
	}

	return &RemoveEdgeCommand{
		EdgeID: c.EdgeID,
		Edge: domain.Edge{
			ID:           c.EdgeID,
			SourceNodeID: c.SourceNodeID,
			SourcePortID: c.SourcePortID,
			TargetNodeID: c.TargetNodeID,
			TargetPortID: c.TargetPortID,
		},
	}, nil
}

func (c *AddEdgeCommand) Type() string             { return "add_edge" }
func (c *AddEdgeCommand) Serialize() ([]byte, error) { return json.Marshal(c) }

func init() {
	domain.RegisterCommand("add_edge", func(data []byte) (domain.Command, error) {
		var cmd AddEdgeCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return &cmd, nil
	})
}
