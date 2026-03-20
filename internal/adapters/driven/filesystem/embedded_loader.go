package filesystem

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"sync"

	"graphiti/internal/domain"

	"gopkg.in/yaml.v3"
)

// EmbeddedNodeLoader loads node definitions from an fs.FS (typically an embed.FS).
// It implements the NodeDefinitionRepository interface.
type EmbeddedNodeLoader struct {
	mu          sync.RWMutex
	definitions map[string]*domain.NodeDefinition
	allDefs     []*domain.NodeDefinition
	fsys        fs.FS
	root        string
	logger      *slog.Logger
}

// NewEmbeddedNodeLoader creates a new node loader that reads from the given
// filesystem (typically an embed.FS). The root parameter is the directory
// prefix within the FS to walk (e.g., "config/nodes").
func NewEmbeddedNodeLoader(fsys fs.FS, root string, logger *slog.Logger) *EmbeddedNodeLoader {
	if logger == nil {
		logger = slog.Default()
	}
	return &EmbeddedNodeLoader{
		definitions: make(map[string]*domain.NodeDefinition),
		fsys:        fsys,
		root:        root,
		logger:      logger,
	}
}

// LoadAll walks the embedded filesystem, parses all .yaml/.yml files,
// validates them, and stores valid definitions in memory.
func (el *EmbeddedNodeLoader) LoadAll(_ context.Context) ([]*domain.NodeDefinition, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	el.definitions = make(map[string]*domain.NodeDefinition)
	el.allDefs = nil

	err := fs.WalkDir(el.fsys, el.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			el.logger.Warn("error accessing embedded path", "path", path, "error", err)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		dotIdx := strings.LastIndex(path, ".")
		if dotIdx < 0 {
			return nil
		}
		ext := strings.ToLower(path[dotIdx:])
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		def, loadErr := el.loadFile(path)
		if loadErr != nil {
			el.logger.Warn("skipping invalid embedded node definition", "path", path, "error", loadErr)
			return nil
		}

		if _, exists := el.definitions[def.ID]; exists {
			el.logger.Warn("skipping duplicate embedded node definition ID", "id", def.ID, "path", path)
			return nil
		}

		el.definitions[def.ID] = def
		el.allDefs = append(el.allDefs, def)
		return nil
	})

	if err != nil {
		// If the root doesn't exist in the FS, return empty (not an error)
		el.logger.Warn("walking embedded filesystem", "root", el.root, "error", err)
		return nil, nil
	}

	el.logger.Info("loaded embedded node definitions", "count", len(el.allDefs))
	result := make([]*domain.NodeDefinition, len(el.allDefs))
	copy(result, el.allDefs)
	return result, nil
}

// GetByID returns the node definition with the given ID.
func (el *EmbeddedNodeLoader) GetByID(_ context.Context, id string) (*domain.NodeDefinition, error) {
	el.mu.RLock()
	defer el.mu.RUnlock()

	def, ok := el.definitions[id]
	if !ok {
		return nil, fmt.Errorf("node definition not found: %s", id)
	}
	return def, nil
}

// GetByCategory returns all node definitions in the given category group.
func (el *EmbeddedNodeLoader) GetByCategory(_ context.Context, category string) ([]*domain.NodeDefinition, error) {
	el.mu.RLock()
	defer el.mu.RUnlock()

	lc := strings.ToLower(category)
	var result []*domain.NodeDefinition
	for _, def := range el.allDefs {
		if strings.ToLower(def.Category.Group) == lc {
			result = append(result, def)
		}
	}
	return result, nil
}

// Search returns all node definitions whose name, description, or category
// contain the query string (case-insensitive).
func (el *EmbeddedNodeLoader) Search(_ context.Context, query string) ([]*domain.NodeDefinition, error) {
	el.mu.RLock()
	defer el.mu.RUnlock()

	q := strings.ToLower(query)
	var result []*domain.NodeDefinition
	for _, def := range el.allDefs {
		if strings.Contains(strings.ToLower(def.Name), q) ||
			strings.Contains(strings.ToLower(def.Description), q) ||
			strings.Contains(strings.ToLower(def.Category.Group), q) {
			result = append(result, def)
		}
	}
	return result, nil
}

func (el *EmbeddedNodeLoader) loadFile(path string) (*domain.NodeDefinition, error) {
	data, err := fs.ReadFile(el.fsys, path)
	if err != nil {
		return nil, fmt.Errorf("reading embedded file: %w", err)
	}

	var raw yamlNodeDefinition
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	// Reuse the same validation logic from the filesystem loader
	nl := &NodeLoader{logger: el.logger}
	if err := nl.validate(&raw); err != nil {
		return nil, err
	}

	return nl.toDomain(&raw), nil
}
