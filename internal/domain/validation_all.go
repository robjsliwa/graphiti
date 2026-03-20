package domain

// ValidateAll runs all workflow validators and returns combined results.
// This is the single entry point for full workflow validation.
func (w *Workflow) ValidateAll(registry NodeDefinitionRegistry) []ValidationResult {
	var results []ValidationResult
	results = append(results, w.validateGraphStructure()...)
	results = append(results, w.validateEdges(registry)...)
	results = append(results, w.validateAttributes(registry)...)
	results = append(results, w.ValidateSubworkflows(registry)...)
	return results
}
