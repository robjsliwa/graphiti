package filesystem

import (
	"context"
	"os"
	"testing"
)

// TestLoadRealNodeDefinitions verifies that all YAML files in config/nodes/ parse correctly.
func TestLoadRealNodeDefinitions(t *testing.T) {
	// Skip if config/nodes doesn't exist (e.g., running in CI without the full repo)
	configPath := "../../../../config/nodes"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("config/nodes not found, skipping integration test")
	}

	loader := NewNodeLoader(configPath, nil)
	defs, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	// We expect at least 11 node definitions (4 original + 6 Phase 4.5 + 1 Phase 5)
	if len(defs) < 11 {
		t.Errorf("loaded %d definitions, expected at least 11", len(defs))
	}

	// Verify specific nodes exist
	wantIDs := []string{
		"source-api-gateway",
		"control-http-router",
		"processing-validate-payload",
		"destination-postgresql",
		"destination-http-response",
		"processing-json-transform",
		"control-sub-workflow",
	}

	defMap := make(map[string]bool, len(defs))
	for _, d := range defs {
		defMap[d.ID] = true
	}

	for _, id := range wantIDs {
		if !defMap[id] {
			t.Errorf("missing node definition: %s", id)
		}
	}

	// Verify categories
	categories := make(map[string]int)
	for _, d := range defs {
		categories[d.Category.Group]++
	}

	expectedCategories := []string{"Sources", "Processing", "Destinations", "Control"}
	for _, cat := range expectedCategories {
		if categories[cat] == 0 {
			t.Errorf("no nodes in category %q", cat)
		}
	}
}
