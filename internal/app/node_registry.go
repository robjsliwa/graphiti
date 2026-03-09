package app

import (
	"context"
	"strings"
	"sync"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

// NodeRegistry is the application service for node definitions.
// It loads definitions from a repository and provides search/lookup.
type NodeRegistry struct {
	repo driven.NodeDefinitionRepository
	mu   sync.RWMutex
	defs []*domain.NodeDefinition
	byID map[string]*domain.NodeDefinition
}

// NewNodeRegistry creates a new NodeRegistry backed by the given repository.
func NewNodeRegistry(repo driven.NodeDefinitionRepository) *NodeRegistry {
	return &NodeRegistry{
		repo: repo,
		byID: make(map[string]*domain.NodeDefinition),
	}
}

// Load fetches all definitions from the repository into memory.
func (r *NodeRegistry) Load(ctx context.Context) error {
	defs, err := r.repo.LoadAll(ctx)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defs = defs
	r.byID = make(map[string]*domain.NodeDefinition, len(defs))
	for _, d := range defs {
		r.byID[d.ID] = d
	}
	return nil
}

// GetAllDefinitions returns all loaded node definitions.
func (r *NodeRegistry) GetAllDefinitions(_ context.Context) ([]*domain.NodeDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*domain.NodeDefinition, len(r.defs))
	copy(result, r.defs)
	return result, nil
}

// GetByCategory returns definitions matching the given category group.
func (r *NodeRegistry) GetByCategory(_ context.Context, category string) ([]*domain.NodeDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*domain.NodeDefinition
	for _, d := range r.defs {
		if d.Category.Group == category {
			result = append(result, d)
		}
	}
	return result, nil
}

// GetByID returns a single definition by its ID.
func (r *NodeRegistry) GetByID(_ context.Context, id string) (*domain.NodeDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrNodeNotFound
	}
	return def, nil
}

// Search finds definitions matching the query against name, description, and category.
func (r *NodeRegistry) Search(_ context.Context, query string) ([]*domain.NodeDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	q := strings.ToLower(query)
	var result []*domain.NodeDefinition
	for _, d := range r.defs {
		if strings.Contains(strings.ToLower(d.Name), q) ||
			strings.Contains(strings.ToLower(d.Description), q) ||
			strings.Contains(strings.ToLower(d.Category.Group), q) {
			result = append(result, d)
		}
	}
	return result, nil
}
