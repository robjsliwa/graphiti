package domain

import (
	"testing"
)

// fakeCommand is a test helper that tracks execution.
type fakeCommand struct {
	applied bool
	inverse *fakeCommand
}

func newFakeCommandPair() (*fakeCommand, *fakeCommand) {
	do := &fakeCommand{}
	undo := &fakeCommand{}
	do.inverse = undo
	undo.inverse = do
	return do, undo
}

func (c *fakeCommand) Execute(wf *Workflow) (Command, error) {
	c.applied = true
	inv := &fakeCommand{inverse: c}
	return inv, nil
}

func (c *fakeCommand) Type() string             { return "fake" }
func (c *fakeCommand) Serialize() ([]byte, error) { return []byte(`{}`), nil }

func TestCommandHistory_Execute(t *testing.T) {
	h := NewCommandHistory(100)
	wf := &Workflow{ID: "wf-1"}
	cmd, _ := newFakeCommandPair()

	if err := h.Execute(wf, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !h.CanUndo() {
		t.Error("expected CanUndo to be true")
	}
	if h.CanRedo() {
		t.Error("expected CanRedo to be false")
	}
}

func TestCommandHistory_Execute_ClearsRedoStack(t *testing.T) {
	h := NewCommandHistory(100)
	wf := &Workflow{ID: "wf-1"}
	cmd1, _ := newFakeCommandPair()
	cmd2, _ := newFakeCommandPair()
	cmd3, _ := newFakeCommandPair()

	h.Execute(wf, cmd1)
	h.Execute(wf, cmd2)
	h.Undo(wf)

	if !h.CanRedo() {
		t.Fatal("expected CanRedo before new command")
	}

	h.Execute(wf, cmd3)
	if h.CanRedo() {
		t.Error("redo stack should be cleared after new command")
	}
}

func TestCommandHistory_Undo(t *testing.T) {
	h := NewCommandHistory(100)
	wf := &Workflow{ID: "wf-1"}
	cmd, _ := newFakeCommandPair()

	h.Execute(wf, cmd)
	if err := h.Undo(wf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.CanUndo() {
		t.Error("expected CanUndo to be false after undoing only command")
	}
	if !h.CanRedo() {
		t.Error("expected CanRedo to be true after undo")
	}
}

func TestCommandHistory_Undo_EmptyStack(t *testing.T) {
	h := NewCommandHistory(100)
	wf := &Workflow{ID: "wf-1"}
	if err := h.Undo(wf); err != ErrNothingToUndo {
		t.Errorf("expected ErrNothingToUndo, got %v", err)
	}
}

func TestCommandHistory_Redo(t *testing.T) {
	h := NewCommandHistory(100)
	wf := &Workflow{ID: "wf-1"}
	cmd, _ := newFakeCommandPair()

	h.Execute(wf, cmd)
	h.Undo(wf)
	if err := h.Redo(wf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !h.CanUndo() {
		t.Error("expected CanUndo after redo")
	}
	if h.CanRedo() {
		t.Error("expected CanRedo to be false after redo")
	}
}

func TestCommandHistory_Redo_EmptyStack(t *testing.T) {
	h := NewCommandHistory(100)
	wf := &Workflow{ID: "wf-1"}
	if err := h.Redo(wf); err != ErrNothingToRedo {
		t.Errorf("expected ErrNothingToRedo, got %v", err)
	}
}

func TestCommandHistory_MaxSize(t *testing.T) {
	h := NewCommandHistory(3)
	wf := &Workflow{ID: "wf-1"}

	for i := 0; i < 5; i++ {
		cmd, _ := newFakeCommandPair()
		h.Execute(wf, cmd)
	}

	if h.UndoCount() != 3 {
		t.Errorf("expected undo stack size 3, got %d", h.UndoCount())
	}
}

func TestCommandHistory_Integration_AddUndoRedoAdd(t *testing.T) {
	// add 3 nodes → undo all 3 → redo 2 → add new → redo stack cleared
	h := NewCommandHistory(100)
	wf := &Workflow{ID: "wf-1"}

	for i := 0; i < 3; i++ {
		cmd, _ := newFakeCommandPair()
		h.Execute(wf, cmd)
	}

	// Undo all 3
	for i := 0; i < 3; i++ {
		if err := h.Undo(wf); err != nil {
			t.Fatalf("undo %d: %v", i, err)
		}
	}
	if h.UndoCount() != 0 {
		t.Fatalf("expected 0 undo after undoing all, got %d", h.UndoCount())
	}
	if h.RedoCount() != 3 {
		t.Fatalf("expected 3 redo, got %d", h.RedoCount())
	}

	// Redo 2
	h.Redo(wf)
	h.Redo(wf)
	if h.UndoCount() != 2 {
		t.Fatalf("expected 2 undo after 2 redo, got %d", h.UndoCount())
	}

	// Add new command → redo stack cleared
	cmd, _ := newFakeCommandPair()
	h.Execute(wf, cmd)
	if h.RedoCount() != 0 {
		t.Errorf("expected 0 redo after new command, got %d", h.RedoCount())
	}
	if h.UndoCount() != 3 {
		t.Errorf("expected 3 undo, got %d", h.UndoCount())
	}
}
