package app

import (
	"context"
	"testing"

	"graphiti/internal/domain"
)

type fakeNodeDefRepo struct {
	defs []*domain.NodeDefinition
}

func (r *fakeNodeDefRepo) LoadAll(_ context.Context) ([]*domain.NodeDefinition, error) {
	return r.defs, nil
}

func (r *fakeNodeDefRepo) GetByID(_ context.Context, id string) (*domain.NodeDefinition, error) {
	for _, d := range r.defs {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, domain.ErrNodeNotFound
}

func (r *fakeNodeDefRepo) GetByCategory(_ context.Context, category string) ([]*domain.NodeDefinition, error) {
	var result []*domain.NodeDefinition
	for _, d := range r.defs {
		if d.Category.Group == category {
			result = append(result, d)
		}
	}
	return result, nil
}

func (r *fakeNodeDefRepo) Search(_ context.Context, query string) ([]*domain.NodeDefinition, error) {
	return r.defs, nil
}

func testDefs() []*domain.NodeDefinition {
	return []*domain.NodeDefinition{
		{ID: "source-twilio", Name: "Twilio", Description: "Receives calls", Category: domain.Category{Group: "Sources", Order: 1}},
		{ID: "proc-transcribe", Name: "Transcribe", Description: "Speech to text", Category: domain.Category{Group: "Processing", Order: 1}},
		{ID: "dest-s3", Name: "S3 Storage", Description: "Store to S3", Category: domain.Category{Group: "Destinations", Order: 1}},
	}
}

func TestNodeRegistry_Load(t *testing.T) {
	repo := &fakeNodeDefRepo{defs: testDefs()}
	registry := NewNodeRegistry(repo)

	if err := registry.Load(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defs, _ := registry.GetAllDefinitions(context.Background())
	if len(defs) != 3 {
		t.Errorf("expected 3 definitions, got %d", len(defs))
	}
}

func TestNodeRegistry_GetByID(t *testing.T) {
	repo := &fakeNodeDefRepo{defs: testDefs()}
	registry := NewNodeRegistry(repo)
	registry.Load(context.Background())

	def, err := registry.GetByID(context.Background(), "source-twilio")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.Name != "Twilio" {
		t.Errorf("expected Twilio, got %s", def.Name)
	}

	_, err = registry.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent ID")
	}
}

func TestNodeRegistry_GetByCategory(t *testing.T) {
	repo := &fakeNodeDefRepo{defs: testDefs()}
	registry := NewNodeRegistry(repo)
	registry.Load(context.Background())

	defs, _ := registry.GetByCategory(context.Background(), "Sources")
	if len(defs) != 1 {
		t.Errorf("expected 1 source, got %d", len(defs))
	}
}

func TestNodeRegistry_Search(t *testing.T) {
	repo := &fakeNodeDefRepo{defs: testDefs()}
	registry := NewNodeRegistry(repo)
	registry.Load(context.Background())

	results, _ := registry.Search(context.Background(), "twilio")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'twilio', got %d", len(results))
	}

	results, _ = registry.Search(context.Background(), "STORAGE")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'STORAGE', got %d", len(results))
	}

	results, _ = registry.Search(context.Background(), "Processing")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'Processing', got %d", len(results))
	}
}
