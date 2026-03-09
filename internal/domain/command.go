package domain

import (
	"encoding/json"
	"fmt"
)

// Command represents a reversible mutation to a workflow.
type Command interface {
	// Execute applies the mutation and returns the inverse command for undo.
	Execute(wf *Workflow) (Command, error)

	// Type returns a string identifier for serialization.
	Type() string

	// Serialize converts the command to JSON.
	Serialize() ([]byte, error)
}

// commandRegistry maps type strings to factory functions for deserialization.
var commandRegistry = map[string]func([]byte) (Command, error){}

// RegisterCommand adds a command type to the deserialization registry.
func RegisterCommand(cmdType string, factory func([]byte) (Command, error)) {
	commandRegistry[cmdType] = factory
}

// DeserializeCommand reconstructs a Command from its type string and JSON bytes.
func DeserializeCommand(cmdType string, data []byte) (Command, error) {
	factory, ok := commandRegistry[cmdType]
	if !ok {
		return nil, fmt.Errorf("unknown command type: %s", cmdType)
	}
	return factory(data)
}

// SerializedCommand is an envelope for JSON serialization of any command.
type SerializedCommand struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
