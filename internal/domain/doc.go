// Package domain contains the core business logic for Graphiti.
//
// This package has zero external dependencies. It imports only from the
// Go standard library. All workflow operations, validation rules,
// command pattern implementations, and clipboard logic live here.
//
// Domain types are pure value objects and aggregates. They know nothing
// about HTTP, databases, or the UI layer.
//
// # Key Types
//
//   - [Workflow]: the aggregate root representing a complete workflow graph
//   - [NodeDefinition]: a YAML-driven declaration of a node type
//   - [NodeInstance]: a placed node on a canvas with configured attributes
//   - [Edge]: a connection between two ports
//   - [Command]: the interface for all canvas mutations (undo/redo)
//   - [CommandHistory]: the undo/redo stack
//   - [ClipboardPayload]: serialized node/edge selection for copy/paste
//   - [ExecutionRun]: tracks a single workflow execution
//   - [ValidationResult]: structured output from workflow validation
package domain
