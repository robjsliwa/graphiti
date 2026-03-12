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

// Severity indicates how critical a validation result is.
type Severity string

const (
	SeverityError   Severity = "error"   // Blocks deploy
	SeverityWarning Severity = "warning" // Surfaced but doesn't block
	SeverityInfo    Severity = "info"    // Suggestion, never blocks
)

// ValidationCategory groups validation results by domain.
type ValidationCategory string

const (
	CategoryGraph       ValidationCategory = "graph"
	CategoryPort        ValidationCategory = "port"
	CategoryEdge        ValidationCategory = "edge"
	CategoryAttribute   ValidationCategory = "attribute"
	CategorySubworkflow ValidationCategory = "subworkflow"
	CategoryDefinition  ValidationCategory = "definition"
)

// ValidationResult is a structured validation finding with severity and category.
type ValidationResult struct {
	Severity Severity           `json:"severity"`
	Category ValidationCategory `json:"category"`
	NodeID   string             `json:"nodeId,omitempty"`
	EdgeID   string             `json:"edgeId,omitempty"`
	Field    string             `json:"field,omitempty"`
	Code     string             `json:"code"`
	Message  string             `json:"message"`
}

// HasErrors returns true if any result has SeverityError.
func HasErrors(results []ValidationResult) bool {
	for _, r := range results {
		if r.Severity == SeverityError {
			return true
		}
	}
	return false
}

// ErrorsOnly filters to just errors.
func ErrorsOnly(results []ValidationResult) []ValidationResult {
	var out []ValidationResult
	for _, r := range results {
		if r.Severity == SeverityError {
			out = append(out, r)
		}
	}
	return out
}

// WarningsOnly filters to just warnings.
func WarningsOnly(results []ValidationResult) []ValidationResult {
	var out []ValidationResult
	for _, r := range results {
		if r.Severity == SeverityWarning {
			out = append(out, r)
		}
	}
	return out
}

// ByNode groups results by NodeID for UI rendering.
func ByNode(results []ValidationResult) map[string][]ValidationResult {
	m := make(map[string][]ValidationResult)
	for _, r := range results {
		m[r.NodeID] = append(m[r.NodeID], r)
	}
	return m
}

// WorkflowResolver can look up workflows by ID for cross-workflow validation.
type WorkflowResolver interface {
	ResolveWorkflow(id string) (*Workflow, error)
}
