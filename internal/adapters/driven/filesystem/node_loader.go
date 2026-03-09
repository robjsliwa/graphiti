package filesystem

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"graphiti/internal/domain"

	"gopkg.in/yaml.v3"
)

// yamlNodeDefinition mirrors the YAML structure for parsing.
type yamlNodeDefinition struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   yamlMetadata `yaml:"metadata"`
	Category   yamlCategory `yaml:"category"`
	Shape      yamlShape    `yaml:"shape"`
	Ports      yamlPorts    `yaml:"ports"`
	Attributes []yamlAttr   `yaml:"attributes"`
	Validation yamlValidation `yaml:"validation"`
}

type yamlMetadata struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
	Icon        string `yaml:"icon"`
	IconSvg     string `yaml:"iconSvg"`
}

type yamlCategory struct {
	Group string `yaml:"group"`
	Order int    `yaml:"order"`
}

type yamlShape struct {
	Type             string `yaml:"type"`
	Width            int    `yaml:"width"`
	MinWidth         int    `yaml:"minWidth"`
	MaxWidth         int    `yaml:"maxWidth"`
	HeaderColor      string `yaml:"headerColor"`
	HeaderBackground string `yaml:"headerBackground"`
	CustomSvg        string `yaml:"customSvg"`
}

type yamlPort struct {
	ID             string `yaml:"id"`
	Label          string `yaml:"label"`
	Type           string `yaml:"type"`
	Position       string `yaml:"position"`
	MaxConnections int    `yaml:"maxConnections"`
}

type yamlPorts struct {
	Inputs  []yamlPort `yaml:"inputs"`
	Outputs []yamlPort `yaml:"outputs"`
}

type yamlAttr struct {
	ID       string   `yaml:"id"`
	Label    string   `yaml:"label"`
	Type     string   `yaml:"type"`
	Default  any      `yaml:"default"`
	Required bool     `yaml:"required"`
	Display  string   `yaml:"display"`
	Group    string   `yaml:"group"`
	Options  []string `yaml:"options"`
	Min      *float64 `yaml:"min"`
	Max      *float64 `yaml:"max"`
	Hint     string   `yaml:"hint"`
}

type yamlValidation struct {
	ConnectionRules []yamlConnectionRule `yaml:"connectionRules"`
	AttributeRules  []yamlAttributeRule  `yaml:"attributeRules"`
}

type yamlConnectionRule struct {
	OutputPort              string   `yaml:"outputPort"`
	AllowedTargetCategories []string `yaml:"allowedTargetCategories"`
	AllowedTargetPorts      []string `yaml:"allowedTargetPorts"`
}

type yamlAttributeRule struct {
	Expression string `yaml:"expression"`
	Message    string `yaml:"message"`
}

// NodeLoader loads node definitions from YAML files in a directory tree
// and stores them in memory. It implements the NodeDefinitionRepository interface.
type NodeLoader struct {
	mu          sync.RWMutex
	definitions map[string]*domain.NodeDefinition
	allDefs     []*domain.NodeDefinition
	rootDir     string
	logger      *slog.Logger
}

// NewNodeLoader creates a new NodeLoader that reads from the given root directory.
func NewNodeLoader(rootDir string, logger *slog.Logger) *NodeLoader {
	if logger == nil {
		logger = slog.Default()
	}
	return &NodeLoader{
		definitions: make(map[string]*domain.NodeDefinition),
		rootDir:     rootDir,
		logger:      logger,
	}
}

// LoadAll walks the root directory recursively, parses all .yaml/.yml files,
// validates them, and stores valid definitions in memory.
func (nl *NodeLoader) LoadAll(ctx context.Context) ([]*domain.NodeDefinition, error) {
	nl.mu.Lock()
	defer nl.mu.Unlock()

	// Reset state
	nl.definitions = make(map[string]*domain.NodeDefinition)
	nl.allDefs = nil

	err := filepath.Walk(nl.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			nl.logger.Warn("error accessing path", "path", path, "error", err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		def, loadErr := nl.loadFile(path)
		if loadErr != nil {
			nl.logger.Warn("skipping invalid node definition", "path", path, "error", loadErr)
			return nil
		}

		if _, exists := nl.definitions[def.ID]; exists {
			nl.logger.Warn("skipping duplicate node definition ID", "id", def.ID, "path", path)
			return nil
		}

		nl.definitions[def.ID] = def
		nl.allDefs = append(nl.allDefs, def)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", nl.rootDir, err)
	}

	nl.logger.Info("loaded node definitions", "count", len(nl.allDefs), "directory", nl.rootDir)

	result := make([]*domain.NodeDefinition, len(nl.allDefs))
	copy(result, nl.allDefs)
	return result, nil
}

// GetByID returns the node definition with the given ID.
func (nl *NodeLoader) GetByID(_ context.Context, id string) (*domain.NodeDefinition, error) {
	nl.mu.RLock()
	defer nl.mu.RUnlock()

	def, ok := nl.definitions[id]
	if !ok {
		return nil, fmt.Errorf("node definition not found: %s", id)
	}
	return def, nil
}

// GetByCategory returns all node definitions in the given category group.
func (nl *NodeLoader) GetByCategory(_ context.Context, category string) ([]*domain.NodeDefinition, error) {
	nl.mu.RLock()
	defer nl.mu.RUnlock()

	lowerCategory := strings.ToLower(category)
	var result []*domain.NodeDefinition
	for _, def := range nl.allDefs {
		if strings.ToLower(def.Category.Group) == lowerCategory {
			result = append(result, def)
		}
	}
	return result, nil
}

// Search returns all node definitions whose name, description, or category
// contain the query string (case-insensitive).
func (nl *NodeLoader) Search(_ context.Context, query string) ([]*domain.NodeDefinition, error) {
	nl.mu.RLock()
	defer nl.mu.RUnlock()

	lowerQuery := strings.ToLower(query)
	var result []*domain.NodeDefinition
	for _, def := range nl.allDefs {
		if strings.Contains(strings.ToLower(def.Name), lowerQuery) ||
			strings.Contains(strings.ToLower(def.Description), lowerQuery) ||
			strings.Contains(strings.ToLower(def.Category.Group), lowerQuery) {
			result = append(result, def)
		}
	}
	return result, nil
}

// loadFile reads and parses a single YAML file into a domain.NodeDefinition.
func (nl *NodeLoader) loadFile(path string) (*domain.NodeDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var raw yamlNodeDefinition
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if err := nl.validate(&raw); err != nil {
		return nil, err
	}

	return nl.toDomain(&raw), nil
}

// validate checks that a parsed YAML definition meets all requirements.
func (nl *NodeLoader) validate(raw *yamlNodeDefinition) error {
	if raw.APIVersion != "graphiti/v1" {
		return fmt.Errorf("invalid apiVersion: %q (expected \"graphiti/v1\")", raw.APIVersion)
	}
	if raw.Kind != "NodeDefinition" {
		return fmt.Errorf("invalid kind: %q (expected \"NodeDefinition\")", raw.Kind)
	}
	if raw.Metadata.ID == "" {
		return fmt.Errorf("metadata.id is required")
	}
	if raw.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if raw.Category.Group == "" {
		return fmt.Errorf("category.group is required")
	}
	if raw.Shape.Type == "" {
		return fmt.Errorf("shape.type is required")
	}

	shapeType := domain.ShapeType(raw.Shape.Type)
	if !domain.ValidShapeTypes()[shapeType] {
		return fmt.Errorf("invalid shape.type: %q", raw.Shape.Type)
	}

	validPortTypes := domain.ValidPortTypes()
	for _, p := range raw.Ports.Inputs {
		if !validPortTypes[domain.PortType(p.Type)] {
			return fmt.Errorf("invalid input port type: %q (port %q)", p.Type, p.ID)
		}
	}
	for _, p := range raw.Ports.Outputs {
		if !validPortTypes[domain.PortType(p.Type)] {
			return fmt.Errorf("invalid output port type: %q (port %q)", p.Type, p.ID)
		}
	}

	validAttrTypes := domain.ValidAttributeTypes()
	for _, a := range raw.Attributes {
		if !validAttrTypes[domain.AttributeType(a.Type)] {
			return fmt.Errorf("invalid attribute type: %q (attribute %q)", a.Type, a.ID)
		}
	}

	return nil
}

// toDomain converts a parsed YAML structure to a domain.NodeDefinition.
func (nl *NodeLoader) toDomain(raw *yamlNodeDefinition) *domain.NodeDefinition {
	def := &domain.NodeDefinition{
		APIVersion:  raw.APIVersion,
		Kind:        raw.Kind,
		ID:          raw.Metadata.ID,
		Name:        raw.Metadata.Name,
		Description: raw.Metadata.Description,
		Version:     raw.Metadata.Version,
		Icon:        raw.Metadata.Icon,
		IconSvg:     raw.Metadata.IconSvg,
		Category: domain.Category{
			Group: raw.Category.Group,
			Order: raw.Category.Order,
		},
		Shape: domain.Shape{
			Type:             domain.ShapeType(raw.Shape.Type),
			Width:            raw.Shape.Width,
			MinWidth:         raw.Shape.MinWidth,
			MaxWidth:         raw.Shape.MaxWidth,
			HeaderColor:      raw.Shape.HeaderColor,
			HeaderBackground: raw.Shape.HeaderBackground,
			CustomSvg:        raw.Shape.CustomSvg,
		},
	}

	for _, p := range raw.Ports.Inputs {
		def.Inputs = append(def.Inputs, domain.PortDefinition{
			ID:             p.ID,
			Label:          p.Label,
			Type:           domain.PortType(p.Type),
			Position:       domain.PortPosition(p.Position),
			MaxConnections: p.MaxConnections,
		})
	}

	for _, p := range raw.Ports.Outputs {
		def.Outputs = append(def.Outputs, domain.PortDefinition{
			ID:             p.ID,
			Label:          p.Label,
			Type:           domain.PortType(p.Type),
			Position:       domain.PortPosition(p.Position),
			MaxConnections: p.MaxConnections,
		})
	}

	for _, a := range raw.Attributes {
		def.Attributes = append(def.Attributes, domain.AttributeDefinition{
			ID:       a.ID,
			Label:    a.Label,
			Type:     domain.AttributeType(a.Type),
			Default:  a.Default,
			Required: a.Required,
			Display:  domain.DisplayLocation(a.Display),
			Group:    a.Group,
			Options:  a.Options,
			Min:      a.Min,
			Max:      a.Max,
			Hint:     a.Hint,
		})
	}

	for _, cr := range raw.Validation.ConnectionRules {
		def.Validation.ConnectionRules = append(def.Validation.ConnectionRules, domain.ConnectionRule{
			OutputPort:              cr.OutputPort,
			AllowedTargetCategories: cr.AllowedTargetCategories,
			AllowedTargetPorts:      cr.AllowedTargetPorts,
		})
	}

	for _, ar := range raw.Validation.AttributeRules {
		def.Validation.AttributeRules = append(def.Validation.AttributeRules, domain.AttributeRule{
			Expression: ar.Expression,
			Message:    ar.Message,
		})
	}

	return def
}
