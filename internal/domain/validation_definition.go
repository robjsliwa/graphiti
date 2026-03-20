package domain

import "fmt"

// ValidateNodeDefinition validates a node definition's schema (YAML structure).
// Called at startup when loading definitions from disk.
func ValidateNodeDefinition(def *NodeDefinition) []ValidationResult {
	var results []ValidationResult

	// API version check
	if def.APIVersion != "" && def.APIVersion != "graphiti/v1" {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryDefinition,
			Code:     "DEFINITION_INVALID_API_VERSION",
			Message:  fmt.Sprintf("Definition '%s' has apiVersion '%s', expected 'graphiti/v1'.", def.ID, def.APIVersion),
		})
	}

	// Kind check
	if def.Kind != "" && def.Kind != "NodeDefinition" {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryDefinition,
			Code:     "DEFINITION_INVALID_KIND",
			Message:  fmt.Sprintf("Definition '%s' has kind '%s', expected 'NodeDefinition'.", def.ID, def.Kind),
		})
	}

	// Required fields
	if def.ID == "" {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryDefinition,
			Code:     "DEFINITION_MISSING_ID",
			Message:  "Node definition is missing 'metadata.id'.",
		})
	}
	if def.Name == "" {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryDefinition,
			Code:     "DEFINITION_MISSING_NAME",
			Message:  fmt.Sprintf("Node definition '%s' is missing 'metadata.name'.", def.ID),
		})
	}
	if def.Category.Group == "" {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryDefinition,
			Code:     "DEFINITION_MISSING_CATEGORY",
			Message:  fmt.Sprintf("Node definition '%s' is missing 'category.group'.", def.ID),
		})
	}

	// Shape type validation
	if def.Shape.Type != "" && !ValidShapeTypes()[def.Shape.Type] {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryDefinition,
			Code:     "DEFINITION_INVALID_SHAPE",
			Message:  fmt.Sprintf("Node definition '%s' has invalid shape type '%s'.", def.ID, def.Shape.Type),
		})
	}

	// Width range check
	if def.Shape.MinWidth > 0 && def.Shape.MaxWidth > 0 && def.Shape.Width > 0 {
		if def.Shape.Width < def.Shape.MinWidth || def.Shape.Width > def.Shape.MaxWidth {
			results = append(results, ValidationResult{
				Severity: SeverityWarning,
				Category: CategoryDefinition,
				Code:     "DEFINITION_WIDTH_RANGE",
				Message:  fmt.Sprintf("Node definition '%s' has width %d outside min/max range [%d, %d].", def.ID, def.Shape.Width, def.Shape.MinWidth, def.Shape.MaxWidth),
			})
		}
	}

	// Port validation
	portIDs := make(map[string]bool)
	allPorts := make([]PortDefinition, 0, len(def.Inputs)+len(def.Outputs))
	allPorts = append(allPorts, def.Inputs...)
	allPorts = append(allPorts, def.Outputs...)

	validPorts := ValidPortTypes()
	validPositions := validPortPositions()

	for _, port := range allPorts {
		// Duplicate port ID check
		if portIDs[port.ID] {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryDefinition,
				Code:     "DEFINITION_DUPLICATE_PORT_ID",
				Message:  fmt.Sprintf("Node definition '%s' has duplicate port ID '%s'.", def.ID, port.ID),
			})
		}
		portIDs[port.ID] = true

		// Port type check
		if port.Type != "" && !validPorts[port.Type] {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryDefinition,
				Code:     "DEFINITION_INVALID_PORT_TYPE",
				Message:  fmt.Sprintf("Node definition '%s' port '%s' has invalid type '%s'.", def.ID, port.ID, port.Type),
			})
		}

		// Port position check
		if port.Position != "" && !validPositions[port.Position] {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryDefinition,
				Code:     "DEFINITION_INVALID_PORT_POSITION",
				Message:  fmt.Sprintf("Node definition '%s' port '%s' has invalid position '%s'.", def.ID, port.ID, port.Position),
			})
		}
	}

	// Attribute validation
	validAttrTypes := ValidAttributeTypes()
	attrIDs := make(map[string]bool)
	for _, attr := range def.Attributes {
		if attrIDs[attr.ID] {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryDefinition,
				Code:     "DEFINITION_DUPLICATE_ATTR_ID",
				Message:  fmt.Sprintf("Node definition '%s' has duplicate attribute ID '%s'.", def.ID, attr.ID),
			})
		}
		attrIDs[attr.ID] = true

		if attr.Type != "" && !validAttrTypes[attr.Type] {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryDefinition,
				Code:     "DEFINITION_INVALID_ATTR_TYPE",
				Message:  fmt.Sprintf("Node definition '%s' attribute '%s' has invalid type '%s'.", def.ID, attr.ID, attr.Type),
			})
		}

		if attr.Type == AttrTypeEnum && len(attr.Options) == 0 {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryDefinition,
				Code:     "DEFINITION_ENUM_NO_OPTIONS",
				Message:  fmt.Sprintf("Node definition '%s' attribute '%s' is an enum with no options.", def.ID, attr.ID),
			})
		}
	}

	// Connection rules reference valid output ports
	outputPortIDs := make(map[string]bool)
	for _, p := range def.Outputs {
		outputPortIDs[p.ID] = true
	}
	for _, rule := range def.Validation.ConnectionRules {
		if !outputPortIDs[rule.OutputPort] {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryDefinition,
				Code:     "DEFINITION_RULE_BAD_PORT_REF",
				Message:  fmt.Sprintf("Node definition '%s' connection rule references port '%s' which is not an output port.", def.ID, rule.OutputPort),
			})
		}
	}

	return results
}

// validPortPositions returns the set of allowed port positions.
func validPortPositions() map[PortPosition]bool {
	return map[PortPosition]bool{
		PortPositionLeftCenter:   true,
		PortPositionRightCenter:  true,
		PortPositionTopCenter:    true,
		PortPositionBottomCenter: true,
		PortPositionRightTop:     true,
		PortPositionRightBottom:  true,
	}
}
