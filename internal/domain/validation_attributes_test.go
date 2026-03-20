package domain

import (
	"testing"
)

func TestValidateAttributes_RequiredMissing(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		exists   bool
		wantCode string
	}{
		{"nil value", nil, true, "ATTR_REQUIRED_MISSING"},
		{"empty string", "", true, "ATTR_REQUIRED_MISSING"},
		{"not set", nil, false, "ATTR_REQUIRED_MISSING"},
		{"valid string", "hello", true, ""},
		{"zero number", float64(0), true, ""}, // 0 is a valid number, not missing
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &NodeDefinition{
				ID: "src", Name: "Source", Category: Category{Group: "Sources"},
				Attributes: []AttributeDefinition{
					{ID: "endpoint", Label: "Endpoint", Type: AttrTypeString, Required: true},
				},
			}
			reg := newTestRegistry(def)

			attrs := make(map[string]any)
			if tt.exists {
				attrs["endpoint"] = tt.value
			}

			wf := &Workflow{
				ID: "wf-test",
				Nodes: []NodeInstance{
					{ID: "n1", DefinitionID: "src", Label: "Source", Definition: def, AttributeValues: attrs},
				},
			}

			results := wf.validateAttributes(reg)
			if tt.wantCode == "" {
				if hasCode(results, "ATTR_REQUIRED_MISSING") {
					t.Errorf("unexpected ATTR_REQUIRED_MISSING for value %v (exists=%v)", tt.value, tt.exists)
				}
			} else {
				if !hasCode(results, tt.wantCode) {
					t.Errorf("expected %s, got: %v", tt.wantCode, codeList(results))
				}
			}
		})
	}
}

func TestValidateAttributes_NonRequiredEmpty(t *testing.T) {
	def := &NodeDefinition{
		ID: "src", Name: "Source", Category: Category{Group: "Sources"},
		Attributes: []AttributeDefinition{
			{ID: "desc", Label: "Description", Type: AttrTypeString, Required: false},
		},
	}
	reg := newTestRegistry(def)

	wf := &Workflow{
		ID: "wf-test",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "src", Label: "Source", Definition: def, AttributeValues: map[string]any{}},
		},
	}

	results := wf.validateAttributes(reg)
	if hasCode(results, "ATTR_REQUIRED_MISSING") {
		t.Error("non-required empty attribute should not trigger ATTR_REQUIRED_MISSING")
	}
}

func TestValidateAttributes_TypeMismatch(t *testing.T) {
	tests := []struct {
		name     string
		attrType AttributeType
		value    any
		wantCode string
	}{
		{"number with string value", AttrTypeNumber, "hello", "ATTR_TYPE_MISMATCH"},
		{"number with numeric string", AttrTypeNumber, "256", ""},
		{"number with float string", AttrTypeNumber, "3.14", ""},
		{"number with valid int", AttrTypeNumber, float64(42), ""},
		{"number with valid float", AttrTypeNumber, 3.14, ""},
		{"boolean with string", AttrTypeBoolean, "yes", "ATTR_TYPE_MISMATCH"},
		{"boolean with true", AttrTypeBoolean, true, ""},
		{"boolean with false", AttrTypeBoolean, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &NodeDefinition{
				ID: "node", Name: "Node", Category: Category{Group: "Processing"},
				Attributes: []AttributeDefinition{
					{ID: "attr", Label: "Attr", Type: tt.attrType},
				},
			}
			reg := newTestRegistry(def)

			wf := &Workflow{
				ID: "wf-test",
				Nodes: []NodeInstance{
					{ID: "n1", DefinitionID: "node", Label: "Node", Definition: def,
						AttributeValues: map[string]any{"attr": tt.value}},
				},
			}

			results := wf.validateAttributes(reg)
			if tt.wantCode == "" {
				if hasCode(results, "ATTR_TYPE_MISMATCH") {
					t.Errorf("unexpected ATTR_TYPE_MISMATCH for %T(%v)", tt.value, tt.value)
				}
			} else {
				if !hasCode(results, tt.wantCode) {
					t.Errorf("expected %s for %T(%v), got: %v", tt.wantCode, tt.value, tt.value, codeList(results))
				}
			}
		})
	}
}

func TestValidateAttributes_EnumInvalid(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		options  []string
		required bool
		wantCode string
	}{
		{"valid option", "wav", []string{"wav", "mp3", "ogg"}, true, ""},
		{"invalid option", "aac", []string{"wav", "mp3", "ogg"}, true, "ATTR_ENUM_INVALID"},
		{"empty required enum", "", []string{"wav", "mp3", "ogg"}, true, "ATTR_REQUIRED_MISSING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &NodeDefinition{
				ID: "node", Name: "Node", Category: Category{Group: "Processing"},
				Attributes: []AttributeDefinition{
					{ID: "format", Label: "Format", Type: AttrTypeEnum, Options: tt.options, Required: tt.required},
				},
			}
			reg := newTestRegistry(def)

			wf := &Workflow{
				ID: "wf-test",
				Nodes: []NodeInstance{
					{ID: "n1", DefinitionID: "node", Label: "Node", Definition: def,
						AttributeValues: map[string]any{"format": tt.value}},
				},
			}

			results := wf.validateAttributes(reg)
			if tt.wantCode == "" {
				if hasCode(results, "ATTR_ENUM_INVALID") {
					t.Errorf("unexpected ATTR_ENUM_INVALID for %v", tt.value)
				}
			} else {
				if !hasCode(results, tt.wantCode) {
					t.Errorf("expected %s for %v, got: %v", tt.wantCode, tt.value, codeList(results))
				}
			}
		})
	}
}

func TestValidateAttributes_NumberRange(t *testing.T) {
	min := float64(0)
	max := float64(100)

	tests := []struct {
		name     string
		value    any
		min      *float64
		max      *float64
		wantCode string
	}{
		{"within range", float64(42), &min, &max, ""},
		{"below min", float64(-5), &min, &max, "ATTR_NUMBER_BELOW_MIN"},
		{"above max", float64(150), &min, &max, "ATTR_NUMBER_ABOVE_MAX"},
		{"at min boundary", float64(0), &min, &max, ""},
		{"at max boundary", float64(100), &min, &max, ""},
		{"no min constraint", float64(-999), nil, &max, ""},
		{"no max constraint", float64(999), &min, nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &NodeDefinition{
				ID: "node", Name: "Node", Category: Category{Group: "Processing"},
				Attributes: []AttributeDefinition{
					{ID: "count", Label: "Count", Type: AttrTypeNumber, Min: tt.min, Max: tt.max},
				},
			}
			reg := newTestRegistry(def)

			wf := &Workflow{
				ID: "wf-test",
				Nodes: []NodeInstance{
					{ID: "n1", DefinitionID: "node", Label: "Node", Definition: def,
						AttributeValues: map[string]any{"count": tt.value}},
				},
			}

			results := wf.validateAttributes(reg)
			if tt.wantCode == "" {
				belowOrAbove := hasCode(results, "ATTR_NUMBER_BELOW_MIN") || hasCode(results, "ATTR_NUMBER_ABOVE_MAX")
				if belowOrAbove {
					t.Errorf("unexpected range error for %v, got: %v", tt.value, codeList(results))
				}
			} else {
				if !hasCode(results, tt.wantCode) {
					t.Errorf("expected %s for %v, got: %v", tt.wantCode, tt.value, codeList(results))
				}
			}
		})
	}
}

func TestValidateAttributes_SecretEnvSyntax(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		required bool
		wantCode string
	}{
		{"env var syntax", "${TOKEN}", true, ""},
		{"plaintext secret", "my-secret-password", true, "ATTR_SECRET_ENV_SYNTAX"},
		{"empty required", "", true, "ATTR_REQUIRED_MISSING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &NodeDefinition{
				ID: "node", Name: "Node", Category: Category{Group: "Processing"},
				Attributes: []AttributeDefinition{
					{ID: "token", Label: "Token", Type: AttrTypeSecret, Required: tt.required},
				},
			}
			reg := newTestRegistry(def)

			wf := &Workflow{
				ID: "wf-test",
				Nodes: []NodeInstance{
					{ID: "n1", DefinitionID: "node", Label: "Node", Definition: def,
						AttributeValues: map[string]any{"token": tt.value}},
				},
			}

			results := wf.validateAttributes(reg)
			if tt.wantCode == "" {
				if hasCode(results, "ATTR_SECRET_ENV_SYNTAX") {
					t.Errorf("unexpected ATTR_SECRET_ENV_SYNTAX for %v", tt.value)
				}
			} else {
				if !hasCode(results, tt.wantCode) {
					t.Errorf("expected %s for %v, got: %v", tt.wantCode, tt.value, codeList(results))
				}
			}
		})
	}
}

func TestValidateAttributes_JSONInvalid(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		required bool
		wantCode string
	}{
		{"valid object", `{"key": "value"}`, false, ""},
		{"valid array", `[1,2,3]`, false, ""},
		{"valid empty object", `{}`, false, ""},
		{"invalid json", `{broken`, false, "ATTR_JSON_INVALID"},
		{"plain text", `not json at all`, false, "ATTR_JSON_INVALID"},
		{"empty required", "", true, "ATTR_REQUIRED_MISSING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &NodeDefinition{
				ID: "node", Name: "Node", Category: Category{Group: "Processing"},
				Attributes: []AttributeDefinition{
					{ID: "config", Label: "Config", Type: AttrTypeJSON, Required: tt.required},
				},
			}
			reg := newTestRegistry(def)

			wf := &Workflow{
				ID: "wf-test",
				Nodes: []NodeInstance{
					{ID: "n1", DefinitionID: "node", Label: "Node", Definition: def,
						AttributeValues: map[string]any{"config": tt.value}},
				},
			}

			results := wf.validateAttributes(reg)
			if tt.wantCode == "" {
				if hasCode(results, "ATTR_JSON_INVALID") {
					t.Errorf("unexpected ATTR_JSON_INVALID for %v", tt.value)
				}
			} else {
				if !hasCode(results, tt.wantCode) {
					t.Errorf("expected %s for %v, got: %v", tt.wantCode, tt.value, codeList(results))
				}
			}
		})
	}
}

func TestValidateAttributes_UnknownDefinition(t *testing.T) {
	// Empty registry - node references a def that doesn't exist
	reg := newTestRegistry()

	wf := &Workflow{
		ID: "wf-test",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "deleted-def", Label: "Orphan Node"},
		},
	}

	results := wf.validateAttributes(reg)
	if !hasCode(results, "UNKNOWN_DEFINITION") {
		t.Errorf("expected UNKNOWN_DEFINITION, got: %v", codeList(results))
	}
}

func TestValidateAttributes_UnexpectedAttribute(t *testing.T) {
	def := &NodeDefinition{
		ID: "node", Name: "Node", Category: Category{Group: "Processing"},
		Attributes: []AttributeDefinition{
			{ID: "name", Label: "Name", Type: AttrTypeString},
		},
	}
	reg := newTestRegistry(def)

	wf := &Workflow{
		ID: "wf-test",
		Nodes: []NodeInstance{
			{ID: "n1", DefinitionID: "node", Label: "Node", Definition: def,
				AttributeValues: map[string]any{
					"name":         "valid",
					"stale_attr":   "leftover",
				}},
		},
	}

	results := wf.validateAttributes(reg)
	if !hasCode(results, "ATTR_UNEXPECTED") {
		t.Errorf("expected ATTR_UNEXPECTED for stale attribute, got: %v", codeList(results))
	}
	// Check it's info severity
	for _, r := range results {
		if r.Code == "ATTR_UNEXPECTED" && r.Severity != SeverityInfo {
			t.Errorf("ATTR_UNEXPECTED should be info severity, got %s", r.Severity)
		}
	}
}
