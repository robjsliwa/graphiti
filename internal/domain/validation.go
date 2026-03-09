package domain

// ValidationError represents a single validation issue in a workflow.
type ValidationError struct {
	NodeID  string // empty for workflow-level errors
	Field   string
	Message string
}
