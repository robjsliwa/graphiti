package filesystem

import (
	"context"
	"graphiti/internal/domain"
	"os"
	"path/filepath"
	"testing"
)

func writeYAML(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating directory %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("writing file %s: %v", filename, err)
	}
}

const validTwilioYAML = `apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: source-twilio
  name: Twilio
  description: "Receives call recording callbacks"
  version: "1.0.0"
  icon: "T"
category:
  group: Sources
  order: 20
shape:
  type: rounded-rect
  width: 200
  minWidth: 160
  maxWidth: 320
  headerColor: "var(--cyan)"
  headerBackground: "var(--cyan-dim)"
ports:
  inputs: []
  outputs:
    - id: out-main
      label: "Output"
      type: data
      position: right-center
      maxConnections: -1
attributes:
  - id: endpoint
    label: "Endpoint"
    type: string
    default: "/ingest/twilio"
    required: true
    display: node-body
    group: "Connection"
validation:
  connectionRules:
    - outputPort: out-main
      allowedTargetCategories: ["Processing", "Destinations"]
      allowedTargetPorts: ["data"]
`

const validTranscribeYAML = `apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: proc-transcribe
  name: Transcribe
  description: "Converts audio to text using speech-to-text"
  version: "1.0.0"
  icon: "Tr"
category:
  group: Processing
  order: 10
shape:
  type: rounded-rect
  width: 200
  minWidth: 160
  maxWidth: 320
  headerColor: "var(--green)"
  headerBackground: "var(--green-dim)"
ports:
  inputs:
    - id: in-main
      label: "Input"
      type: data
      position: left-center
      maxConnections: 1
  outputs:
    - id: out-main
      label: "Output"
      type: data
      position: right-center
      maxConnections: -1
attributes:
  - id: language
    label: "Language"
    type: enum
    default: "en-US"
    required: true
    display: config-panel
    group: "Settings"
    options: ["en-US", "es-ES", "fr-FR"]
validation:
  connectionRules: []
`

func TestLoadAll_ValidFiles(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, filepath.Join(dir, "sources"), "twilio.yaml", validTwilioYAML)
	writeYAML(t, filepath.Join(dir, "processing"), "transcribe.yaml", validTranscribeYAML)

	loader := NewNodeLoader(dir, nil)
	ctx := context.Background()

	defs, err := loader.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}

	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}
}

func TestLoadAll_InvalidAPIVersion(t *testing.T) {
	dir := t.TempDir()
	badYAML := `apiVersion: graphiti/v2
kind: NodeDefinition
metadata:
  id: bad-version
  name: Bad
  description: "Invalid version"
category:
  group: Sources
  order: 1
shape:
  type: rounded-rect
`
	writeYAML(t, dir, "bad.yaml", badYAML)

	loader := NewNodeLoader(dir, nil)
	defs, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}
	if len(defs) != 0 {
		t.Fatalf("expected 0 definitions (invalid file skipped), got %d", len(defs))
	}
}

func TestLoadAll_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "missing metadata.id",
			yaml: `apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  name: NoID
category:
  group: Sources
shape:
  type: rounded-rect
`,
		},
		{
			name: "missing metadata.name",
			yaml: `apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: no-name
category:
  group: Sources
shape:
  type: rounded-rect
`,
		},
		{
			name: "missing category.group",
			yaml: `apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: no-cat
  name: NoCat
category:
  order: 1
shape:
  type: rounded-rect
`,
		},
		{
			name: "missing shape.type",
			yaml: `apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: no-shape
  name: NoShape
category:
  group: Sources
shape:
  width: 200
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeYAML(t, dir, "bad.yaml", tt.yaml)

			loader := NewNodeLoader(dir, nil)
			defs, err := loader.LoadAll(context.Background())
			if err != nil {
				t.Fatalf("LoadAll returned error: %v", err)
			}
			if len(defs) != 0 {
				t.Fatalf("expected 0 definitions for %s, got %d", tt.name, len(defs))
			}
		})
	}
}

func TestLoadAll_InvalidShapeType(t *testing.T) {
	dir := t.TempDir()
	badYAML := `apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: bad-shape
  name: BadShape
category:
  group: Sources
shape:
  type: triangle
`
	writeYAML(t, dir, "bad.yaml", badYAML)

	loader := NewNodeLoader(dir, nil)
	defs, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}
	if len(defs) != 0 {
		t.Fatalf("expected 0 definitions, got %d", len(defs))
	}
}

func TestLoadAll_InvalidPortType(t *testing.T) {
	dir := t.TempDir()
	badYAML := `apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: bad-port
  name: BadPort
category:
  group: Sources
shape:
  type: rounded-rect
ports:
  inputs: []
  outputs:
    - id: out
      label: Out
      type: invalid-port
      position: right-center
`
	writeYAML(t, dir, "bad.yaml", badYAML)

	loader := NewNodeLoader(dir, nil)
	defs, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}
	if len(defs) != 0 {
		t.Fatalf("expected 0 definitions, got %d", len(defs))
	}
}

func TestLoadAll_InvalidAttributeType(t *testing.T) {
	dir := t.TempDir()
	badYAML := `apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: bad-attr
  name: BadAttr
category:
  group: Sources
shape:
  type: rounded-rect
attributes:
  - id: foo
    label: Foo
    type: widget
`
	writeYAML(t, dir, "bad.yaml", badYAML)

	loader := NewNodeLoader(dir, nil)
	defs, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}
	if len(defs) != 0 {
		t.Fatalf("expected 0 definitions, got %d", len(defs))
	}
}

func TestLoadAll_DuplicateIDs(t *testing.T) {
	dir := t.TempDir()
	yaml1 := `apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: dupe-id
  name: First
category:
  group: Sources
shape:
  type: rounded-rect
`
	yaml2 := `apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: dupe-id
  name: Second
category:
  group: Processing
shape:
  type: pill
`
	writeYAML(t, dir, "first.yaml", yaml1)
	writeYAML(t, dir, "second.yaml", yaml2)

	loader := NewNodeLoader(dir, nil)
	defs, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition (duplicate rejected), got %d", len(defs))
	}
}

func TestGetByID(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "twilio.yaml", validTwilioYAML)

	loader := NewNodeLoader(dir, nil)
	ctx := context.Background()

	if _, err := loader.LoadAll(ctx); err != nil {
		t.Fatalf("LoadAll error: %v", err)
	}

	def, err := loader.GetByID(ctx, "source-twilio")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if def.Name != "Twilio" {
		t.Errorf("expected name Twilio, got %s", def.Name)
	}

	_, err = loader.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

func TestGetByCategory(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, filepath.Join(dir, "sources"), "twilio.yaml", validTwilioYAML)
	writeYAML(t, filepath.Join(dir, "processing"), "transcribe.yaml", validTranscribeYAML)

	loader := NewNodeLoader(dir, nil)
	ctx := context.Background()

	if _, err := loader.LoadAll(ctx); err != nil {
		t.Fatalf("LoadAll error: %v", err)
	}

	sources, err := loader.GetByCategory(ctx, "Sources")
	if err != nil {
		t.Fatalf("GetByCategory error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].ID != "source-twilio" {
		t.Errorf("expected source-twilio, got %s", sources[0].ID)
	}

	// Case-insensitive match
	processing, err := loader.GetByCategory(ctx, "processing")
	if err != nil {
		t.Fatalf("GetByCategory error: %v", err)
	}
	if len(processing) != 1 {
		t.Fatalf("expected 1 processing node, got %d", len(processing))
	}

	empty, err := loader.GetByCategory(ctx, "Nonexistent")
	if err != nil {
		t.Fatalf("GetByCategory error: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected 0 for nonexistent category, got %d", len(empty))
	}
}

func TestSearch(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, filepath.Join(dir, "sources"), "twilio.yaml", validTwilioYAML)
	writeYAML(t, filepath.Join(dir, "processing"), "transcribe.yaml", validTranscribeYAML)

	loader := NewNodeLoader(dir, nil)
	ctx := context.Background()

	if _, err := loader.LoadAll(ctx); err != nil {
		t.Fatalf("LoadAll error: %v", err)
	}

	tests := []struct {
		query string
		count int
	}{
		{"twilio", 1},
		{"TWILIO", 1},         // case-insensitive name
		{"audio", 1},          // matches transcribe description
		{"sources", 1},        // matches category
		{"recording", 1},      // matches twilio description
		{"nonexistent", 0},
		{"transcribe", 1},     // matches name
		{"processing", 1},     // matches category
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			results, err := loader.Search(ctx, tt.query)
			if err != nil {
				t.Fatalf("Search error: %v", err)
			}
			if len(results) != tt.count {
				t.Errorf("search %q: expected %d results, got %d", tt.query, tt.count, len(results))
			}
		})
	}
}

func TestLoadAll_YmlExtension(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "twilio.yml", validTwilioYAML)

	loader := NewNodeLoader(dir, nil)
	defs, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition from .yml file, got %d", len(defs))
	}
}

func TestLoadAll_NonYAMLFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "readme.txt", "not a yaml file")
	writeYAML(t, dir, "twilio.yaml", validTwilioYAML)

	loader := NewNodeLoader(dir, nil)
	defs, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition (txt ignored), got %d", len(defs))
	}
}

func TestLoadAll_InvalidKind(t *testing.T) {
	dir := t.TempDir()
	badYAML := `apiVersion: graphiti/v1
kind: SomethingElse
metadata:
  id: bad-kind
  name: BadKind
category:
  group: Sources
shape:
  type: rounded-rect
`
	writeYAML(t, dir, "bad.yaml", badYAML)

	loader := NewNodeLoader(dir, nil)
	defs, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}
	if len(defs) != 0 {
		t.Fatalf("expected 0 definitions, got %d", len(defs))
	}
}

func TestLoadAll_ComboboxAttribute(t *testing.T) {
	dir := t.TempDir()
	comboYAML := `apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: combo-node
  name: ComboNode
  description: "Node with combobox attribute"
  version: "1.0.0"
  icon: "C"
category:
  group: Destinations
  order: 1
shape:
  type: rounded-rect
  width: 200
attributes:
  - id: status
    label: "Status Code"
    type: combobox
    options: ["200 OK", "404 Not Found", "500 Internal Server Error"]
    default: "200 OK"
    display: node-body
`
	writeYAML(t, dir, "combo.yaml", comboYAML)

	loader := NewNodeLoader(dir, nil)
	defs, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	attr := defs[0].Attributes[0]
	if attr.Type != domain.AttrTypeCombobox {
		t.Errorf("expected type combobox, got %s", attr.Type)
	}
	if len(attr.Options) != 3 {
		t.Errorf("expected 3 options, got %d", len(attr.Options))
	}
	if attr.Options[0] != "200 OK" {
		t.Errorf("expected first option '200 OK', got %q", attr.Options[0])
	}
}

func TestLoadAll_FieldParsing(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "twilio.yaml", validTwilioYAML)

	loader := NewNodeLoader(dir, nil)
	ctx := context.Background()

	defs, err := loader.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1, got %d", len(defs))
	}

	def := defs[0]
	if def.APIVersion != "graphiti/v1" {
		t.Errorf("APIVersion: got %q", def.APIVersion)
	}
	if def.Kind != "NodeDefinition" {
		t.Errorf("Kind: got %q", def.Kind)
	}
	if def.ID != "source-twilio" {
		t.Errorf("ID: got %q", def.ID)
	}
	if def.Name != "Twilio" {
		t.Errorf("Name: got %q", def.Name)
	}
	if def.Description != "Receives call recording callbacks" {
		t.Errorf("Description: got %q", def.Description)
	}
	if def.Version != "1.0.0" {
		t.Errorf("Version: got %q", def.Version)
	}
	if def.Icon != "T" {
		t.Errorf("Icon: got %q", def.Icon)
	}
	if def.Category.Group != "Sources" {
		t.Errorf("Category.Group: got %q", def.Category.Group)
	}
	if def.Category.Order != 20 {
		t.Errorf("Category.Order: got %d", def.Category.Order)
	}
	if def.Shape.Width != 200 {
		t.Errorf("Shape.Width: got %d", def.Shape.Width)
	}
	if len(def.Inputs) != 0 {
		t.Errorf("expected 0 inputs, got %d", len(def.Inputs))
	}
	if len(def.Outputs) != 1 {
		t.Errorf("expected 1 output, got %d", len(def.Outputs))
	}
	if len(def.Attributes) != 1 {
		t.Errorf("expected 1 attribute, got %d", len(def.Attributes))
	}
	if len(def.Validation.ConnectionRules) != 1 {
		t.Errorf("expected 1 connection rule, got %d", len(def.Validation.ConnectionRules))
	}
}
