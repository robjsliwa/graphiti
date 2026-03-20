// Package app contains the application service implementations that
// bridge between driving ports (use cases) and the domain layer.
//
// Application services coordinate domain operations, manage command
// history for undo/redo, and delegate persistence to driven port
// implementations. They are the primary consumers of domain types
// and the primary implementors of driving port interfaces.
package app
