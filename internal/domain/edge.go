package domain

// Edge represents a connection between two ports on different nodes.
type Edge struct {
	ID           string
	SourceNodeID string
	SourcePortID string
	TargetNodeID string
	TargetPortID string
}
