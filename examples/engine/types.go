package main

import "time"

// DeployPayload matches the webhook payload Graphiti sends on deploy.
type DeployPayload struct {
	APIVersion      string         `json:"apiVersion"`
	Event           string         `json:"event"`
	Timestamp       time.Time      `json:"timestamp"`
	Deployment      DeploymentInfo `json:"deployment"`
	Workflow        WorkflowInfo   `json:"workflow"`
	PreviousVersion int            `json:"previousVersion"`
	Checksum        string         `json:"checksum"`
}

// DeploymentInfo describes the deploy action.
type DeploymentInfo struct {
	ID          string       `json:"id"`
	Target      string       `json:"target"`
	TriggeredBy TriggeredBy  `json:"triggeredBy"`
}

// TriggeredBy identifies who triggered the deploy.
type TriggeredBy struct {
	UserID   string `json:"userID"`
	Username string `json:"username"`
}

// WorkflowInfo is the workflow data in the deploy payload.
type WorkflowInfo struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Version    int                `json:"version"`
	Definition WorkflowDefinition `json:"definition"`
}

// WorkflowDefinition contains the nodes and edges of a workflow.
type WorkflowDefinition struct {
	Nodes []NodeDef `json:"nodes"`
	Edges []EdgeDef `json:"edges"`
}

// NodeDef is a node instance in the workflow definition.
type NodeDef struct {
	ID              string         `json:"id"`
	DefinitionID    string         `json:"definitionId"`
	Label           string         `json:"label"`
	X               float64        `json:"x"`
	Y               float64        `json:"y"`
	AttributeValues map[string]any `json:"attributeValues"`
}

// EdgeDef is an edge connecting two node ports.
type EdgeDef struct {
	ID           string `json:"id"`
	SourceNodeID string `json:"sourceNodeId"`
	SourcePortID string `json:"sourcePortId"`
	TargetNodeID string `json:"targetNodeId"`
	TargetPortID string `json:"targetPortId"`
}

// StatusCallback is the payload sent back to Graphiti for execution status updates.
type StatusCallback struct {
	APIVersion    string         `json:"apiVersion"`
	Event         string         `json:"event"`
	RunID         string         `json:"runID"`
	WorkflowID    string         `json:"workflowID"`
	NodeID        string         `json:"nodeID"`
	Status        string         `json:"status"`
	StartedAt     time.Time      `json:"startedAt,omitempty"`
	CompletedAt   time.Time      `json:"completedAt,omitempty"`
	ErrorMessage  string         `json:"errorMessage,omitempty"`
	OutputSummary map[string]any `json:"outputSummary,omitempty"`
}
