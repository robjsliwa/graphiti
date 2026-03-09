package domain

// NodeDefinition describes a type of node that can be placed on the canvas.
// Loaded from YAML configuration files at startup.
type NodeDefinition struct {
	APIVersion  string
	Kind        string
	ID          string
	Name        string
	Description string
	Version     string
	Icon        string
	IconSvg     string
	Category    Category
	Shape       Shape
	Inputs      []PortDefinition
	Outputs     []PortDefinition
	Attributes  []AttributeDefinition
	Validation  ValidationRules
}

// Category groups node definitions in the palette.
type Category struct {
	Group string
	Order int
}

// ShapeType determines the visual appearance of a node on the canvas.
type ShapeType string

const (
	ShapeRoundedRect ShapeType = "rounded-rect"
	ShapePill        ShapeType = "pill"
	ShapeDiamond     ShapeType = "diamond"
	ShapeHexagon     ShapeType = "hexagon"
	ShapeSubWorkflow ShapeType = "sub-workflow"
	ShapeCustom      ShapeType = "custom"
)

// ValidShapeTypes returns the set of allowed shape types.
func ValidShapeTypes() map[ShapeType]bool {
	return map[ShapeType]bool{
		ShapeRoundedRect: true,
		ShapePill:        true,
		ShapeDiamond:     true,
		ShapeHexagon:     true,
		ShapeSubWorkflow: true,
		ShapeCustom:      true,
	}
}

// Shape defines the visual properties of a node.
type Shape struct {
	Type             ShapeType
	Width            int
	MinWidth         int
	MaxWidth         int
	HeaderColor      string
	HeaderBackground string
	CustomSvg        string
}

// PortType classifies what kind of data a port carries.
type PortType string

const (
	PortTypeData    PortType = "data"
	PortTypeControl PortType = "control"
	PortTypeError   PortType = "error"
)

// ValidPortTypes returns the set of allowed port types.
func ValidPortTypes() map[PortType]bool {
	return map[PortType]bool{
		PortTypeData:    true,
		PortTypeControl: true,
		PortTypeError:   true,
	}
}

// PortPosition determines where a port circle appears on the node.
type PortPosition string

const (
	PortPositionLeftCenter   PortPosition = "left-center"
	PortPositionRightCenter  PortPosition = "right-center"
	PortPositionTopCenter    PortPosition = "top-center"
	PortPositionBottomCenter PortPosition = "bottom-center"
	PortPositionRightTop     PortPosition = "right-top"
	PortPositionRightBottom  PortPosition = "right-bottom"
)

// PortDefinition describes a single input or output port on a node.
type PortDefinition struct {
	ID             string
	Label          string
	Type           PortType
	Position       PortPosition
	MaxConnections int // -1 = unlimited
}

// AttributeType determines the widget used to edit an attribute.
type AttributeType string

const (
	AttrTypeString      AttributeType = "string"
	AttrTypeText        AttributeType = "text"
	AttrTypeNumber      AttributeType = "number"
	AttrTypeBoolean     AttributeType = "boolean"
	AttrTypeEnum        AttributeType = "enum"
	AttrTypeSecret      AttributeType = "secret"
	AttrTypeJSON        AttributeType = "json"
	AttrTypeExpression  AttributeType = "expression"
	AttrTypePortMapping AttributeType = "port-mapping"
	AttrTypeWorkflowRef AttributeType = "workflow-reference"
)

// ValidAttributeTypes returns the set of allowed attribute types.
func ValidAttributeTypes() map[AttributeType]bool {
	return map[AttributeType]bool{
		AttrTypeString:      true,
		AttrTypeText:        true,
		AttrTypeNumber:      true,
		AttrTypeBoolean:     true,
		AttrTypeEnum:        true,
		AttrTypeSecret:      true,
		AttrTypeJSON:        true,
		AttrTypeExpression:  true,
		AttrTypePortMapping: true,
		AttrTypeWorkflowRef: true,
	}
}

// DisplayLocation controls where an attribute is shown.
type DisplayLocation string

const (
	DisplayNodeBody    DisplayLocation = "node-body"
	DisplayConfigPanel DisplayLocation = "config-panel"
	DisplayBoth        DisplayLocation = "both"
)

// AttributeDefinition describes a configurable property of a node.
type AttributeDefinition struct {
	ID       string
	Label    string
	Type     AttributeType
	Default  any
	Required bool
	Display  DisplayLocation
	Group    string
	Options  []string // for enum type
	Min      *float64 // for number type
	Max      *float64 // for number type
	Hint     string
}

// ConnectionRule constrains which node types/ports an output can connect to.
type ConnectionRule struct {
	OutputPort              string
	AllowedTargetCategories []string
	AllowedTargetPorts      []string
}

// AttributeRule defines a validation expression for an attribute.
type AttributeRule struct {
	Expression string
	Message    string
}

// ValidationRules contains all validation constraints for a node definition.
type ValidationRules struct {
	ConnectionRules []ConnectionRule
	AttributeRules  []AttributeRule
}
