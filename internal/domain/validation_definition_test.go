package domain

import "testing"

func TestValidateNodeDefinition_Valid(t *testing.T) {
	def := &NodeDefinition{
		APIVersion: "graphiti/v1",
		Kind:       "NodeDefinition",
		ID:         "source-twilio",
		Name:       "Twilio",
		Category:   Category{Group: "Sources"},
		Shape:      Shape{Type: ShapeRoundedRect, Width: 200, MinWidth: 160, MaxWidth: 320},
		Outputs:    []PortDefinition{{ID: "out", Label: "Output", Type: PortTypeData, Position: PortPositionRightCenter}},
		Attributes: []AttributeDefinition{
			{ID: "endpoint", Label: "Endpoint", Type: AttrTypeString, Required: true},
			{ID: "format", Label: "Format", Type: AttrTypeEnum, Options: []string{"wav", "mp3"}},
		},
		Validation: ValidationRules{
			ConnectionRules: []ConnectionRule{
				{OutputPort: "out", AllowedTargetCategories: []string{"Processing"}},
			},
		},
	}

	results := ValidateNodeDefinition(def)
	if len(results) > 0 {
		t.Errorf("expected no validation results for valid definition, got: %v", codeList(results))
	}
}

func TestValidateNodeDefinition_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name     string
		def      *NodeDefinition
		wantCode string
	}{
		{
			"missing ID",
			&NodeDefinition{APIVersion: "graphiti/v1", Kind: "NodeDefinition", Name: "X", Category: Category{Group: "Sources"}, Shape: Shape{Type: ShapeRoundedRect}},
			"DEFINITION_MISSING_ID",
		},
		{
			"missing name",
			&NodeDefinition{APIVersion: "graphiti/v1", Kind: "NodeDefinition", ID: "x", Category: Category{Group: "Sources"}, Shape: Shape{Type: ShapeRoundedRect}},
			"DEFINITION_MISSING_NAME",
		},
		{
			"missing category",
			&NodeDefinition{APIVersion: "graphiti/v1", Kind: "NodeDefinition", ID: "x", Name: "X", Shape: Shape{Type: ShapeRoundedRect}},
			"DEFINITION_MISSING_CATEGORY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := ValidateNodeDefinition(tt.def)
			if !hasCode(results, tt.wantCode) {
				t.Errorf("expected %s, got: %v", tt.wantCode, codeList(results))
			}
		})
	}
}

func TestValidateNodeDefinition_InvalidAPIVersion(t *testing.T) {
	def := &NodeDefinition{
		APIVersion: "graphiti/v99",
		Kind:       "NodeDefinition",
		ID:         "x", Name: "X", Category: Category{Group: "Sources"},
		Shape: Shape{Type: ShapeRoundedRect},
	}
	results := ValidateNodeDefinition(def)
	if !hasCode(results, "DEFINITION_INVALID_API_VERSION") {
		t.Errorf("expected DEFINITION_INVALID_API_VERSION, got: %v", codeList(results))
	}
}

func TestValidateNodeDefinition_InvalidKind(t *testing.T) {
	def := &NodeDefinition{
		APIVersion: "graphiti/v1",
		Kind:       "WrongKind",
		ID:         "x", Name: "X", Category: Category{Group: "Sources"},
		Shape: Shape{Type: ShapeRoundedRect},
	}
	results := ValidateNodeDefinition(def)
	if !hasCode(results, "DEFINITION_INVALID_KIND") {
		t.Errorf("expected DEFINITION_INVALID_KIND, got: %v", codeList(results))
	}
}

func TestValidateNodeDefinition_InvalidShapeType(t *testing.T) {
	def := &NodeDefinition{
		APIVersion: "graphiti/v1",
		Kind:       "NodeDefinition",
		ID:         "x", Name: "X", Category: Category{Group: "Sources"},
		Shape: Shape{Type: "pentagon"},
	}
	results := ValidateNodeDefinition(def)
	if !hasCode(results, "DEFINITION_INVALID_SHAPE") {
		t.Errorf("expected DEFINITION_INVALID_SHAPE, got: %v", codeList(results))
	}
}

func TestValidateNodeDefinition_InvalidPortType(t *testing.T) {
	def := &NodeDefinition{
		APIVersion: "graphiti/v1",
		Kind:       "NodeDefinition",
		ID:         "x", Name: "X", Category: Category{Group: "Sources"},
		Shape:   Shape{Type: ShapeRoundedRect},
		Outputs: []PortDefinition{{ID: "out", Type: "stream", Position: PortPositionRightCenter}},
	}
	results := ValidateNodeDefinition(def)
	if !hasCode(results, "DEFINITION_INVALID_PORT_TYPE") {
		t.Errorf("expected DEFINITION_INVALID_PORT_TYPE, got: %v", codeList(results))
	}
}

func TestValidateNodeDefinition_InvalidPortPosition(t *testing.T) {
	def := &NodeDefinition{
		APIVersion: "graphiti/v1",
		Kind:       "NodeDefinition",
		ID:         "x", Name: "X", Category: Category{Group: "Sources"},
		Shape:   Shape{Type: ShapeRoundedRect},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, Position: "invalid-pos"}},
	}
	results := ValidateNodeDefinition(def)
	if !hasCode(results, "DEFINITION_INVALID_PORT_POSITION") {
		t.Errorf("expected DEFINITION_INVALID_PORT_POSITION, got: %v", codeList(results))
	}
}

func TestValidateNodeDefinition_DuplicatePortID(t *testing.T) {
	def := &NodeDefinition{
		APIVersion: "graphiti/v1",
		Kind:       "NodeDefinition",
		ID:         "x", Name: "X", Category: Category{Group: "Sources"},
		Shape: Shape{Type: ShapeRoundedRect},
		Outputs: []PortDefinition{
			{ID: "out-main", Type: PortTypeData, Position: PortPositionRightCenter},
			{ID: "out-main", Type: PortTypeData, Position: PortPositionRightBottom},
		},
	}
	results := ValidateNodeDefinition(def)
	if !hasCode(results, "DEFINITION_DUPLICATE_PORT_ID") {
		t.Errorf("expected DEFINITION_DUPLICATE_PORT_ID, got: %v", codeList(results))
	}
}

func TestValidateNodeDefinition_DuplicateAttrID(t *testing.T) {
	def := &NodeDefinition{
		APIVersion: "graphiti/v1",
		Kind:       "NodeDefinition",
		ID:         "x", Name: "X", Category: Category{Group: "Sources"},
		Shape: Shape{Type: ShapeRoundedRect},
		Attributes: []AttributeDefinition{
			{ID: "name", Label: "Name", Type: AttrTypeString},
			{ID: "name", Label: "Name 2", Type: AttrTypeString},
		},
	}
	results := ValidateNodeDefinition(def)
	if !hasCode(results, "DEFINITION_DUPLICATE_ATTR_ID") {
		t.Errorf("expected DEFINITION_DUPLICATE_ATTR_ID, got: %v", codeList(results))
	}
}

func TestValidateNodeDefinition_InvalidAttrType(t *testing.T) {
	def := &NodeDefinition{
		APIVersion: "graphiti/v1",
		Kind:       "NodeDefinition",
		ID:         "x", Name: "X", Category: Category{Group: "Sources"},
		Shape: Shape{Type: ShapeRoundedRect},
		Attributes: []AttributeDefinition{
			{ID: "attr", Label: "Attr", Type: "invalid-type"},
		},
	}
	results := ValidateNodeDefinition(def)
	if !hasCode(results, "DEFINITION_INVALID_ATTR_TYPE") {
		t.Errorf("expected DEFINITION_INVALID_ATTR_TYPE, got: %v", codeList(results))
	}
}

func TestValidateNodeDefinition_EnumNoOptions(t *testing.T) {
	def := &NodeDefinition{
		APIVersion: "graphiti/v1",
		Kind:       "NodeDefinition",
		ID:         "x", Name: "X", Category: Category{Group: "Sources"},
		Shape: Shape{Type: ShapeRoundedRect},
		Attributes: []AttributeDefinition{
			{ID: "format", Label: "Format", Type: AttrTypeEnum, Options: nil},
		},
	}
	results := ValidateNodeDefinition(def)
	if !hasCode(results, "DEFINITION_ENUM_NO_OPTIONS") {
		t.Errorf("expected DEFINITION_ENUM_NO_OPTIONS, got: %v", codeList(results))
	}
}

func TestValidateNodeDefinition_RuleBadPortRef(t *testing.T) {
	def := &NodeDefinition{
		APIVersion: "graphiti/v1",
		Kind:       "NodeDefinition",
		ID:         "x", Name: "X", Category: Category{Group: "Sources"},
		Shape:   Shape{Type: ShapeRoundedRect},
		Outputs: []PortDefinition{{ID: "out", Type: PortTypeData, Position: PortPositionRightCenter}},
		Validation: ValidationRules{
			ConnectionRules: []ConnectionRule{
				{OutputPort: "nonexistent", AllowedTargetCategories: []string{"Processing"}},
			},
		},
	}
	results := ValidateNodeDefinition(def)
	if !hasCode(results, "DEFINITION_RULE_BAD_PORT_REF") {
		t.Errorf("expected DEFINITION_RULE_BAD_PORT_REF, got: %v", codeList(results))
	}
}

func TestValidateNodeDefinition_WidthRange(t *testing.T) {
	def := &NodeDefinition{
		APIVersion: "graphiti/v1",
		Kind:       "NodeDefinition",
		ID:         "x", Name: "X", Category: Category{Group: "Sources"},
		Shape: Shape{Type: ShapeRoundedRect, Width: 400, MinWidth: 160, MaxWidth: 320},
	}
	results := ValidateNodeDefinition(def)
	if !hasCode(results, "DEFINITION_WIDTH_RANGE") {
		t.Errorf("expected DEFINITION_WIDTH_RANGE, got: %v", codeList(results))
	}
	// Should be a warning, not error
	for _, r := range results {
		if r.Code == "DEFINITION_WIDTH_RANGE" && r.Severity != SeverityWarning {
			t.Errorf("DEFINITION_WIDTH_RANGE should be warning, got %s", r.Severity)
		}
	}
}

func TestValidateNodeDefinition_AllSeveritiesCorrect(t *testing.T) {
	// All definition errors should be SeverityError except DEFINITION_WIDTH_RANGE
	def := &NodeDefinition{} // missing everything
	results := ValidateNodeDefinition(def)
	for _, r := range results {
		if r.Code == "DEFINITION_WIDTH_RANGE" {
			if r.Severity != SeverityWarning {
				t.Errorf("code %s should be warning", r.Code)
			}
		} else {
			if r.Severity != SeverityError {
				t.Errorf("code %s should be error, got %s", r.Code, r.Severity)
			}
		}
	}
}
