package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// validateAttributes checks all node attribute values against their definitions.
func (w *Workflow) validateAttributes(registry NodeDefinitionRegistry) []ValidationResult {
	var results []ValidationResult

	for _, node := range w.Nodes {
		def := node.Definition
		if def == nil && registry != nil {
			def = registry.GetByID(node.DefinitionID)
		}
		if def == nil {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryDefinition,
				NodeID:   node.ID,
				Code:     "UNKNOWN_DEFINITION",
				Message:  fmt.Sprintf("Node '%s' references definition '%s' which doesn't exist in the registry. The node type may have been removed.", node.Label, node.DefinitionID),
			})
			continue
		}

		// Validate each defined attribute
		definedAttrs := make(map[string]bool)
		for _, attrDef := range def.Attributes {
			definedAttrs[attrDef.ID] = true
			value, exists := node.AttributeValues[attrDef.ID]
			results = append(results, validateSingleAttribute(node, attrDef, value, exists)...)
		}

		// Check for unexpected attributes (stale data)
		for attrID := range node.AttributeValues {
			if !definedAttrs[attrID] {
				results = append(results, ValidationResult{
					Severity: SeverityInfo,
					Category: CategoryAttribute,
					NodeID:   node.ID,
					Field:    attrID,
					Code:     "ATTR_UNEXPECTED",
					Message:  fmt.Sprintf("Node '%s' has attribute '%s' which is not defined in its node type. This may be stale data from a previous version.", node.Label, attrID),
				})
			}
		}
	}

	return results
}

// validateSingleAttribute checks one attribute value against its definition.
func validateSingleAttribute(node NodeInstance, attrDef AttributeDefinition, value any, exists bool) []ValidationResult {
	var results []ValidationResult

	// Required check
	if attrDef.Required {
		if !exists || value == nil || value == "" {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryAttribute,
				NodeID:   node.ID,
				Field:    attrDef.ID,
				Code:     "ATTR_REQUIRED_MISSING",
				Message:  fmt.Sprintf("Required attribute '%s' on node '%s' is missing or empty.", attrDef.Label, node.Label),
			})
			return results // Don't validate further if required field is missing
		}
	}

	// Skip further checks if value is not set
	if !exists || value == nil || value == "" {
		return results
	}

	// Type-specific validation
	switch attrDef.Type {
	case AttrTypeNumber:
		results = append(results, validateNumberAttribute(node, attrDef, value)...)
	case AttrTypeBoolean:
		results = append(results, validateBooleanAttribute(node, attrDef, value)...)
	case AttrTypeEnum:
		results = append(results, validateEnumAttribute(node, attrDef, value)...)
	case AttrTypeSecret:
		results = append(results, validateSecretAttribute(node, attrDef, value)...)
	case AttrTypeJSON:
		results = append(results, validateJSONAttribute(node, attrDef, value)...)
	}

	return results
}

func validateNumberAttribute(node NodeInstance, attrDef AttributeDefinition, value any) []ValidationResult {
	var results []ValidationResult

	numVal, ok := toFloat64(value)
	if !ok {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryAttribute,
			NodeID:   node.ID,
			Field:    attrDef.ID,
			Code:     "ATTR_TYPE_MISMATCH",
			Message:  fmt.Sprintf("Attribute '%s' on node '%s' expects a number but got '%v'.", attrDef.Label, node.Label, value),
		})
		return results
	}

	if attrDef.Min != nil && numVal < *attrDef.Min {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryAttribute,
			NodeID:   node.ID,
			Field:    attrDef.ID,
			Code:     "ATTR_NUMBER_BELOW_MIN",
			Message:  fmt.Sprintf("Attribute '%s' on node '%s' has value %v which is below the minimum %v.", attrDef.Label, node.Label, numVal, *attrDef.Min),
		})
	}

	if attrDef.Max != nil && numVal > *attrDef.Max {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryAttribute,
			NodeID:   node.ID,
			Field:    attrDef.ID,
			Code:     "ATTR_NUMBER_ABOVE_MAX",
			Message:  fmt.Sprintf("Attribute '%s' on node '%s' has value %v which is above the maximum %v.", attrDef.Label, node.Label, numVal, *attrDef.Max),
		})
	}

	return results
}

func validateBooleanAttribute(node NodeInstance, attrDef AttributeDefinition, value any) []ValidationResult {
	if _, ok := value.(bool); !ok {
		return []ValidationResult{{
			Severity: SeverityError,
			Category: CategoryAttribute,
			NodeID:   node.ID,
			Field:    attrDef.ID,
			Code:     "ATTR_TYPE_MISMATCH",
			Message:  fmt.Sprintf("Attribute '%s' on node '%s' expects a boolean but got '%v'.", attrDef.Label, node.Label, value),
		}}
	}
	return nil
}

func validateEnumAttribute(node NodeInstance, attrDef AttributeDefinition, value any) []ValidationResult {
	strVal, ok := value.(string)
	if !ok {
		return []ValidationResult{{
			Severity: SeverityError,
			Category: CategoryAttribute,
			NodeID:   node.ID,
			Field:    attrDef.ID,
			Code:     "ATTR_TYPE_MISMATCH",
			Message:  fmt.Sprintf("Attribute '%s' on node '%s' expects a string enum value but got '%v'.", attrDef.Label, node.Label, value),
		}}
	}

	for _, opt := range attrDef.Options {
		if opt == strVal {
			return nil
		}
	}

	return []ValidationResult{{
		Severity: SeverityError,
		Category: CategoryAttribute,
		NodeID:   node.ID,
		Field:    attrDef.ID,
		Code:     "ATTR_ENUM_INVALID",
		Message:  fmt.Sprintf("Attribute '%s' on node '%s' has value '%s' which is not in the allowed options: %v.", attrDef.Label, node.Label, strVal, attrDef.Options),
	}}
}

func validateSecretAttribute(node NodeInstance, attrDef AttributeDefinition, value any) []ValidationResult {
	strVal, ok := value.(string)
	if !ok {
		return nil
	}

	// Check for env var syntax: ${...}
	if !strings.HasPrefix(strVal, "${") || !strings.HasSuffix(strVal, "}") {
		return []ValidationResult{{
			Severity: SeverityWarning,
			Category: CategoryAttribute,
			NodeID:   node.ID,
			Field:    attrDef.ID,
			Code:     "ATTR_SECRET_ENV_SYNTAX",
			Message:  fmt.Sprintf("Attribute '%s' on node '%s' appears to contain a plaintext secret. Use environment variable syntax like ${MY_SECRET} instead.", attrDef.Label, node.Label),
		}}
	}

	return nil
}

func validateJSONAttribute(node NodeInstance, attrDef AttributeDefinition, value any) []ValidationResult {
	strVal, ok := value.(string)
	if !ok {
		return nil // Non-string values might be pre-parsed JSON, which is fine
	}

	if !json.Valid([]byte(strVal)) {
		return []ValidationResult{{
			Severity: SeverityError,
			Category: CategoryAttribute,
			NodeID:   node.ID,
			Field:    attrDef.ID,
			Code:     "ATTR_JSON_INVALID",
			Message:  fmt.Sprintf("Attribute '%s' on node '%s' contains invalid JSON.", attrDef.Label, node.Label),
		}}
	}

	return nil
}

// toFloat64 attempts to convert a value to float64.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	default:
		return 0, false
	}
}
