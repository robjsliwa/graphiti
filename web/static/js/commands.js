// Command dispatcher: sends commands to server, handles undo/redo
export class CommandDispatcher {
  constructor(workflowId) {
    this.workflowId = workflowId;
    this._bindShortcuts();
  }

  async dispatch(type, payload) {
    const resp = await fetch(`/api/workflows/${this.workflowId}/commands`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ type, ...payload }),
    });
    const result = await resp.json();
    this._updateButtons(result.canUndo, result.canRedo);
    return result;
  }

  async undo() {
    const resp = await fetch(`/api/workflows/${this.workflowId}/undo`, { method: 'POST' });
    const result = await resp.json();
    this._updateButtons(result.canUndo, result.canRedo);
    return result;
  }

  async redo() {
    const resp = await fetch(`/api/workflows/${this.workflowId}/redo`, { method: 'POST' });
    const result = await resp.json();
    this._updateButtons(result.canUndo, result.canRedo);
    return result;
  }

  _updateButtons(canUndo, canRedo) {
    const undoBtn = document.getElementById('undo-btn');
    const redoBtn = document.getElementById('redo-btn');
    if (undoBtn) undoBtn.disabled = !canUndo;
    if (redoBtn) redoBtn.disabled = !canRedo;
  }

  _bindShortcuts() {
    document.addEventListener('keydown', (e) => {
      // Skip if typing in input
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') return;

      const mod = e.metaKey || e.ctrlKey;
      if (mod && e.key === 'z' && !e.shiftKey) {
        e.preventDefault();
        this.undo();
      } else if (mod && e.key === 'z' && e.shiftKey) {
        e.preventDefault();
        this.redo();
      } else if (mod && e.key === 'y') {
        e.preventDefault();
        this.redo();
      }
    });

    document.getElementById('undo-btn')?.addEventListener('click', () => this.undo());
    document.getElementById('redo-btn')?.addEventListener('click', () => this.redo());
  }
}

// Auto-init
if (window.GRAPHITI?.workflowID) {
  window.commandDispatcher = new CommandDispatcher(window.GRAPHITI.workflowID);
}
