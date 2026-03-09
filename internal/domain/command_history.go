package domain

import "errors"

var (
	ErrNothingToUndo = errors.New("nothing to undo")
	ErrNothingToRedo = errors.New("nothing to redo")
)

// CommandHistory manages undo/redo stacks for workflow mutations.
type CommandHistory struct {
	undoStack []Command
	redoStack []Command
	maxSize   int
}

// NewCommandHistory creates a new history with the given max undo depth.
func NewCommandHistory(maxSize int) *CommandHistory {
	return &CommandHistory{
		undoStack: make([]Command, 0),
		redoStack: make([]Command, 0),
		maxSize:   maxSize,
	}
}

// Execute runs a command on the workflow and pushes its inverse onto the undo stack.
func (h *CommandHistory) Execute(wf *Workflow, cmd Command) error {
	inverse, err := cmd.Execute(wf)
	if err != nil {
		return err
	}
	h.undoStack = append(h.undoStack, inverse)
	h.redoStack = nil // new action clears redo
	if len(h.undoStack) > h.maxSize {
		h.undoStack = h.undoStack[1:]
	}
	return nil
}

// Undo reverses the last command.
func (h *CommandHistory) Undo(wf *Workflow) error {
	if len(h.undoStack) == 0 {
		return ErrNothingToUndo
	}
	cmd := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	inverse, err := cmd.Execute(wf)
	if err != nil {
		return err
	}
	h.redoStack = append(h.redoStack, inverse)
	return nil
}

// Redo reapplies the last undone command.
func (h *CommandHistory) Redo(wf *Workflow) error {
	if len(h.redoStack) == 0 {
		return ErrNothingToRedo
	}
	cmd := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]
	inverse, err := cmd.Execute(wf)
	if err != nil {
		return err
	}
	h.undoStack = append(h.undoStack, inverse)
	return nil
}

// CanUndo returns true if there are commands to undo.
func (h *CommandHistory) CanUndo() bool { return len(h.undoStack) > 0 }

// CanRedo returns true if there are commands to redo.
func (h *CommandHistory) CanRedo() bool { return len(h.redoStack) > 0 }

// UndoCount returns the number of commands on the undo stack.
func (h *CommandHistory) UndoCount() int { return len(h.undoStack) }

// RedoCount returns the number of commands on the redo stack.
func (h *CommandHistory) RedoCount() int { return len(h.redoStack) }
