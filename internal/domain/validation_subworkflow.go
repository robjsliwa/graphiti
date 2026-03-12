package domain

import "fmt"

const maxSubworkflowNesting = 5

// ValidateSubworkflows checks sub-workflow nodes within this workflow for
// missing references and self-references. Cross-workflow circular reference
// detection requires a WorkflowResolver and is handled by ValidateSubworkflowChain.
func (w *Workflow) ValidateSubworkflows() []ValidationResult {
	var results []ValidationResult

	for _, node := range w.Nodes {
		if node.Definition == nil || node.Definition.Shape.Type != ShapeSubWorkflow {
			continue
		}

		workflowRef, _ := node.AttributeValues["workflow_ref"].(string)

		if workflowRef == "" {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategorySubworkflow,
				NodeID:   node.ID,
				Code:     "SUBWORKFLOW_NO_REFERENCE",
				Message:  fmt.Sprintf("Sub-workflow node '%s' has no referenced workflow. Select a workflow in the configuration panel.", node.Label),
			})
			continue
		}

		if workflowRef == w.ID {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategorySubworkflow,
				NodeID:   node.ID,
				Code:     "SUBWORKFLOW_SELF_REFERENCE",
				Message:  fmt.Sprintf("Sub-workflow node '%s' references itself. A workflow cannot contain itself as a sub-workflow.", node.Label),
			})
			continue
		}
	}

	return results
}

// ValidateSubworkflowChain checks for circular references across workflows.
// Called at deploy time with access to the full workflow repository.
// visited tracks workflow IDs already seen in this chain to detect cycles.
// depth tracks the current nesting level to warn about excessive depth.
func ValidateSubworkflowChain(workflowID string, resolver WorkflowResolver, visited map[string]bool, depth int) []ValidationResult {
	if visited[workflowID] {
		return []ValidationResult{{
			Severity: SeverityError,
			Category: CategorySubworkflow,
			Code:     "SUBWORKFLOW_CIRCULAR_REFERENCE",
			Message:  fmt.Sprintf("Circular sub-workflow reference detected. Workflow '%s' appears in its own sub-workflow chain.", workflowID),
		}}
	}

	if depth > maxSubworkflowNesting {
		return []ValidationResult{{
			Severity: SeverityWarning,
			Category: CategorySubworkflow,
			Code:     "SUBWORKFLOW_NESTING_DEPTH",
			Message:  fmt.Sprintf("Sub-workflow nesting exceeds %d levels. Consider simplifying the workflow hierarchy.", maxSubworkflowNesting),
		}}
	}

	visited[workflowID] = true
	defer func() { delete(visited, workflowID) }()

	wf, err := resolver.ResolveWorkflow(workflowID)
	if err != nil {
		return []ValidationResult{{
			Severity: SeverityError,
			Category: CategorySubworkflow,
			Code:     "SUBWORKFLOW_NOT_FOUND",
			Message:  fmt.Sprintf("Referenced workflow '%s' does not exist.", workflowID),
		}}
	}

	var results []ValidationResult
	for _, node := range wf.Nodes {
		if node.Definition == nil || node.Definition.Shape.Type != ShapeSubWorkflow {
			continue
		}
		if ref, ok := node.AttributeValues["workflow_ref"].(string); ok && ref != "" {
			results = append(results, ValidateSubworkflowChain(ref, resolver, visited, depth+1)...)
		}
	}
	return results
}
