package commands

import (
	"encoding/json"
	"graphiti/internal/domain"
)

// RemoveEdgeCommand removes an edge, capturing state for undo.
type RemoveEdgeCommand struct {
	EdgeID string      `json:"edgeId"`
	Edge   domain.Edge `json:"edge"` // populated after execute for undo
}

func (c *RemoveEdgeCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
	// If we have a stored edge and it doesn't exist in the workflow, we're restoring (undo of remove)
	if c.Edge.ID != "" && wf.FindEdge(c.Edge.ID) == nil {
		wf.RestoreEdge(c.Edge)
		return &RemoveEdgeCommand{
			EdgeID: c.Edge.ID,
		}, nil
	}

	removed, err := wf.RemoveEdge(c.EdgeID)
	if err != nil {
		return nil, err
	}

	return &RemoveEdgeCommand{
		EdgeID: removed.ID,
		Edge:   *removed,
	}, nil
}

func (c *RemoveEdgeCommand) Type() string             { return "remove_edge" }
func (c *RemoveEdgeCommand) Serialize() ([]byte, error) { return json.Marshal(c) }

func init() {
	domain.RegisterCommand("remove_edge", func(data []byte) (domain.Command, error) {
		var cmd RemoveEdgeCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return &cmd, nil
	})
}
