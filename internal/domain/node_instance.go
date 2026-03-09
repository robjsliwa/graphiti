package domain

// NodeInstance represents a placed node on the workflow canvas.
type NodeInstance struct {
	ID              string
	DefinitionID    string
	Label           string
	X               float64
	Y               float64
	AttributeValues map[string]any
	Definition      *NodeDefinition // resolved reference
}
