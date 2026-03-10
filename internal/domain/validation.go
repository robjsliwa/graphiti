package domain

import "errors"

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// ValidationError represents a single validation issue in a workflow.
type ValidationError struct {
	NodeID  string // empty for workflow-level errors
	Field   string
	Message string
}
