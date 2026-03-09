// Clipboard: copy, cut, paste, duplicate
export class ClipboardManager {
  constructor(dispatcher) {
    this.dispatcher = dispatcher;
    this.payload = null; // stored clipboard payload
  }

  async copy() {
    const sel = window.selectionManager;
    if (!sel || sel.selectedNodes.size === 0) return;
    const wfId = window.GRAPHITI?.workflowID;
    if (!wfId) return;

    const resp = await fetch(`/api/workflows/${wfId}/clipboard/copy`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nodeIds: sel.getSelectedNodeIds() }),
    });
    const result = await resp.json();
    if (result.ok) this.payload = result.payload;
  }

  async cut() {
    await this.copy();
    if (this.payload) {
      const sel = window.selectionManager;
      const ids = sel ? sel.getSelectedNodeIds() : [];
      sel?.clearSelection();
      for (const id of ids) await this.dispatcher.dispatch('remove_node', { nodeId: id });
    }
  }

  async paste() {
    if (!this.payload) return;
    const wfId = window.GRAPHITI?.workflowID;
    if (!wfId) return;

    const resp = await fetch(`/api/workflows/${wfId}/clipboard/paste`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ payload: this.payload, x: 100, y: 100 }),
    });
    const result = await resp.json();
    if (result.ok && result.workflow) {
      window.canvasEngine?.syncCanvas(result.workflow);
    }
  }

  async duplicate() {
    await this.copy();
    if (this.payload) {
      // Offset pasted nodes by 24px diagonally
      const wfId = window.GRAPHITI?.workflowID;
      if (!wfId) return;
      const resp = await fetch(`/api/workflows/${wfId}/clipboard/paste`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ payload: this.payload, x: 24, y: 24, relative: true }),
      });
      const result = await resp.json();
      if (result.ok && result.workflow) {
        window.canvasEngine?.syncCanvas(result.workflow);
      }
    }
  }
}

// No auto-init: initialized by app.js
