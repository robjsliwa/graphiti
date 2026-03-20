package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

// NodeRegistry is the application service for node definitions.
// It loads definitions from a repository and provides search/lookup.
type NodeRegistry struct {
	repo                  driven.NodeDefinitionRepository
	mu                    sync.RWMutex
	defs                  []*domain.NodeDefinition
	byID                  map[string]*domain.NodeDefinition
	valResults []domain.ValidationResult
}

// NewNodeRegistry creates a new NodeRegistry backed by the given repository.
func NewNodeRegistry(repo driven.NodeDefinitionRepository) *NodeRegistry {
	return &NodeRegistry{
		repo: repo,
		byID: make(map[string]*domain.NodeDefinition),
	}
}

// Load fetches all definitions from the repository into memory.
// It validates each definition and detects duplicate IDs across files.
func (r *NodeRegistry) Load(ctx context.Context) error {
	defs, err := r.repo.LoadAll(ctx)
	if err != nil {
		return err
	}

	// Validate definitions and detect duplicates
	r.mu.Lock()
	defer r.mu.Unlock()
	r.valResults = nil
	r.defs = make([]*domain.NodeDefinition, 0, len(defs))
	r.byID = make(map[string]*domain.NodeDefinition, len(defs))

	for _, d := range defs {
		// Schema validation
		results := domain.ValidateNodeDefinition(d)
		for _, vr := range results {
			slog.Warn("node definition validation", "id", d.ID, "code", vr.Code, "message", vr.Message)
			r.valResults = append(r.valResults, vr)
		}

		// Duplicate ID detection
		if existing, ok := r.byID[d.ID]; ok {
			vr := domain.ValidationResult{
				Severity: domain.SeverityError,
				Category: domain.CategoryDefinition,
				Code:     "DEFINITION_DUPLICATE_ACROSS_FILES",
				Message:  fmt.Sprintf("Node definition '%s' (%s) has the same ID as '%s'. Each definition must have a unique metadata.id.", d.ID, d.Name, existing.Name),
			}
			slog.Warn("duplicate node definition", "id", d.ID, "name1", existing.Name, "name2", d.Name)
			r.valResults = append(r.valResults, vr)
			continue // skip duplicate
		}

		r.defs = append(r.defs, d)
		r.byID[d.ID] = d
	}
	return nil
}

// LoadValidationResults returns any validation issues found during the last Load.
func (r *NodeRegistry) LoadValidationResults() []domain.ValidationResult {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.ValidationResult, len(r.valResults))
	copy(out, r.valResults)
	return out
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
