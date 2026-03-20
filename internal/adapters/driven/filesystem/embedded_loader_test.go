package filesystem

import (
	"context"
	"embed"
	"testing"
	"testing/fstest"
)

func TestEmbeddedNodeLoader_LoadAll(t *testing.T) {
	// Create a test filesystem with a valid YAML node definition
	fsys := fstest.MapFS{
		"nodes/sources/test.yaml": &fstest.MapFile{
			Data: []byte(`apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: source-test
  name: Test Source
  description: A test source node
  version: "1.0.0"
  icon: "T"
category:
  group: Sources
  order: 10
shape:
  type: rounded-rect
  width: 200
  headerColor: "var(--cyan)"
  headerBackground: "var(--cyan-dim)"
ports:
  inputs: []
  outputs:
    - id: out-main
      label: Output
      type: data
      position: right-center
      maxConnections: -1
attributes: []
`),
		},
	}

	loader := NewEmbeddedNodeLoader(fsys, "nodes", nil)
	defs, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("got %d definitions, want 1", len(defs))
	}
	if defs[0].ID != "source-test" {
		t.Errorf("got ID %q, want %q", defs[0].ID, "source-test")
	}
	if defs[0].Name != "Test Source" {
		t.Errorf("got Name %q, want %q", defs[0].Name, "Test Source")
	}
}

func TestEmbeddedNodeLoader_SkipsInvalidFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"nodes/bad.yaml": &fstest.MapFile{
			Data: []byte(`not valid yaml: [[[`),
		},
		"nodes/good.yaml": &fstest.MapFile{
			Data: []byte(`apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: source-good
  name: Good Source
  version: "1.0.0"
  icon: "G"
category:
  group: Sources
  order: 10
shape:
  type: rounded-rect
  width: 200
ports:
  inputs: []
  outputs: []
attributes: []
`),
		},
	}

	loader := NewEmbeddedNodeLoader(fsys, "nodes", nil)
	defs, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("got %d definitions, want 1", len(defs))
	}
}

func TestEmbeddedNodeLoader_EmptyFS(t *testing.T) {
	var fsys embed.FS
	loader := NewEmbeddedNodeLoader(fsys, "nonexistent", nil)
	defs, err := loader.LoadAll(context.Background())
	// Should not error, just return empty
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 0 {
		t.Fatalf("got %d definitions, want 0", len(defs))
	}
}

func TestEmbeddedNodeLoader_GetByID(t *testing.T) {
	fsys := fstest.MapFS{
		"nodes/test.yaml": &fstest.MapFile{
			Data: []byte(`apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: source-lookup
  name: Lookup Test
  version: "1.0.0"
  icon: "L"
category:
  group: Sources
  order: 10
shape:
  type: rounded-rect
  width: 200
ports:
  inputs: []
  outputs: []
attributes: []
`),
		},
	}

	loader := NewEmbeddedNodeLoader(fsys, "nodes", nil)
	_, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	def, err := loader.GetByID(context.Background(), "source-lookup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.Name != "Lookup Test" {
		t.Errorf("got Name %q, want %q", def.Name, "Lookup Test")
	}

	_, err = loader.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}
