package domain

import "time"

// ExecutionStatus represents the overall status of an execution run.
type ExecutionStatus string

const (
	ExecStatusPending   ExecutionStatus = "pending"
	ExecStatusRunning   ExecutionStatus = "running"
	ExecStatusCompleted ExecutionStatus = "completed"
	ExecStatusFailed    ExecutionStatus = "failed"
	ExecStatusCancelled ExecutionStatus = "cancelled"
)

// NodeExecStatus represents the execution status of a single node.
type NodeExecStatus string

const (
	NodeExecPending   NodeExecStatus = "pending"
	NodeExecRunning   NodeExecStatus = "running"
	NodeExecCompleted NodeExecStatus = "completed"
	NodeExecFailed    NodeExecStatus = "failed"
	NodeExecSkipped   NodeExecStatus = "skipped"
)

// ExecutionRun tracks a single workflow execution.
type ExecutionRun struct {
	ID              string
	WorkflowID      string
	WorkflowVersion int
	Status          ExecutionStatus
	StartedAt       time.Time
	CompletedAt     time.Time
	TriggerType     string
	TriggerData     map[string]any
	NodeStatuses    map[string]*NodeExecutionStatus
}

// NodeExecutionStatus tracks the execution state of one node in a run.
type NodeExecutionStatus struct {
	NodeID       string
	Status       NodeExecStatus
	StartedAt    time.Time
	CompletedAt  time.Time
	InputData    map[string]any
	OutputData   map[string]any
	ErrorMessage string
	Logs         []LogEntry
}

// LogEntry is a single log line from a node execution.
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// ExecutionRunSummary is a lightweight projection for listing runs.
type ExecutionRunSummary struct {
	ID          string
	WorkflowID  string
	Status      ExecutionStatus
	StartedAt   time.Time
	CompletedAt time.Time
	TriggerType string
}
