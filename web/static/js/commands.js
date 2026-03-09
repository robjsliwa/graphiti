// Command dispatcher: sends commands to server, handles undo/redo, keyboard shortcuts
import { toast } from './toast.js';

// Human-readable error messages for common command failures
const ERROR_LABELS = {
  add_edge: 'Connection failed',
  add_node: 'Could not add node',
  move_node: 'Could not move node',
  move_nodes: 'Could not move nodes',
  remove_node: 'Could not delete node',
  remove_edge: 'Could not delete connection',
  update_attribute: 'Could not update attribute',
};

export class CommandDispatcher {
  constructor(workflowId) {
    this.workflowId = workflowId;
    this.selection = null; // Set by app init
    this._bindShortcuts();
  }

  async dispatch(type, payload) {
    try {
      const resp = await fetch(`/api/workflows/${this.workflowId}/commands`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type, ...payload }),
      });
      if (!resp.ok) {
        const text = await resp.text();
        console.error(`[dispatch] ${type} HTTP ${resp.status}:`, text);
        toast.error(ERROR_LABELS[type] || 'Command failed');
        return { ok: false };
      }
      const result = await resp.json();
      if (!result.ok) {
        console.error(`[dispatch] ${type} failed:`, result.error);
        const label = ERROR_LABELS[type] || 'Command failed';
        const detail = this._humanizeError(result.error);
        toast.warning(`${label}: ${detail}`);
      } else if (result.workflow) {
        window.canvasEngine?.syncCanvas(result.workflow);
      }
      this._updateButtons(result.canUndo, result.canRedo);
      return result;
    } catch (err) {
      console.error(`[dispatch] ${type} exception:`, err);
      toast.error('Network error — could not reach the server');
      return { ok: false };
    }
  }

  _humanizeError(msg) {
    if (!msg) return 'unknown error';
    // Strip technical prefixes for cleaner display
    return msg
      .replace(/^connection not allowed by node definition rules:\s*/i, '')
      .replace(/^workflow [^:]+:\s*/i, '');
  }

  async undo() {
    try {
      const resp = await fetch(`/api/workflows/${this.workflowId}/undo`, { method: 'POST' });
      const result = await resp.json();
      if (result.ok && result.workflow) {
        window.canvasEngine?.syncCanvas(result.workflow);
        this.selection?.clearSelection();
      } else if (result.error) {
        toast.info('Nothing to undo');
      }
      this._updateButtons(result.canUndo, result.canRedo);
      return result;
    } catch (err) {
      toast.error('Network error — could not reach the server');
      return { ok: false };
    }
  }

  async redo() {
    try {
      const resp = await fetch(`/api/workflows/${this.workflowId}/redo`, { method: 'POST' });
      const result = await resp.json();
      if (result.ok && result.workflow) {
        window.canvasEngine?.syncCanvas(result.workflow);
        this.selection?.clearSelection();
      } else if (result.error) {
        toast.info('Nothing to redo');
      }
      this._updateButtons(result.canUndo, result.canRedo);
      return result;
    } catch (err) {
      toast.error('Network error — could not reach the server');
      return { ok: false };
    }
  }

  _updateButtons(canUndo, canRedo) {
    const undoBtn = document.getElementById('undo-btn');
    const redoBtn = document.getElementById('redo-btn');
    if (undoBtn) undoBtn.disabled = !canUndo;
    if (redoBtn) redoBtn.disabled = !canRedo;
  }

  _bindShortcuts() {
    document.addEventListener('keydown', (e) => {
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') return;
      const mod = e.metaKey || e.ctrlKey;

      if (mod && e.key === 'z' && !e.shiftKey) { e.preventDefault(); this.undo(); }
      else if (mod && e.key === 'z' && e.shiftKey) { e.preventDefault(); this.redo(); }
      else if (mod && e.key === 'y') { e.preventDefault(); this.redo(); }
      else if (mod && e.key === 'c') { e.preventDefault(); window.clipboardManager?.copy(); }
      else if (mod && e.key === 'x') { e.preventDefault(); window.clipboardManager?.cut(); }
      else if (mod && e.key === 'v') { e.preventDefault(); window.clipboardManager?.paste(); }
      else if (mod && e.key === 'd') { e.preventDefault(); window.clipboardManager?.duplicate(); }
      else if (mod && e.key === '=') { e.preventDefault(); document.getElementById('zoom-in')?.click(); }
      else if (mod && e.key === '-') { e.preventDefault(); document.getElementById('zoom-out')?.click(); }
      else if (mod && e.key === '0') { e.preventDefault(); window.canvasEngine?.fitToView(); }
      else if (mod && e.key === 's') { e.preventDefault(); /* auto-saved */ }
      else if (e.key === '?') { e.preventDefault(); this._toggleHelp(); }
    });

    document.getElementById('undo-btn')?.addEventListener('click', () => this.undo());
    document.getElementById('redo-btn')?.addEventListener('click', () => this.redo());
  }

  _toggleHelp() {
    let overlay = document.getElementById('shortcut-overlay');
    if (overlay) { overlay.remove(); return; }

    const mod = navigator.platform.includes('Mac') ? '\u2318' : 'Ctrl';
    const shortcuts = [
      ['Edit', [
        [mod + '+Z', 'Undo'], [mod + '+Shift+Z', 'Redo'],
        [mod + '+C', 'Copy'], [mod + '+X', 'Cut'],
        [mod + '+V', 'Paste'], [mod + '+D', 'Duplicate'],
        [mod + '+A', 'Select All'], ['Del', 'Delete'],
      ]],
      ['View', [
        [mod + '+=', 'Zoom In'], [mod + '+-', 'Zoom Out'],
        [mod + '+0', 'Fit to View'], ['Space+Drag', 'Pan'],
        ['Esc', 'Deselect'], ['?', 'This Help'],
      ]],
    ];

    overlay = document.createElement('div');
    overlay.id = 'shortcut-overlay';
    overlay.className = 'shortcut-overlay';
    overlay.addEventListener('click', (e) => { if (e.target === overlay) overlay.remove(); });

    const modal = document.createElement('div');
    modal.className = 'shortcut-modal';
    const title = document.createElement('h3');
    title.textContent = 'Keyboard Shortcuts';
    modal.appendChild(title);

    const grid = document.createElement('div');
    grid.className = 'shortcut-grid';
    for (const [groupName, items] of shortcuts) {
      const group = document.createElement('div');
      group.className = 'shortcut-group';
      const h4 = document.createElement('h4');
      h4.textContent = groupName;
      group.appendChild(h4);
      for (const [key, desc] of items) {
        const row = document.createElement('div');
        const kbd = document.createElement('kbd');
        kbd.textContent = key;
        row.appendChild(kbd);
        row.appendChild(document.createTextNode(' ' + desc));
        group.appendChild(row);
      }
      grid.appendChild(group);
    }
    modal.appendChild(grid);

    const closeBtn = document.createElement('button');
    closeBtn.className = 'btn';
    closeBtn.textContent = 'Close';
    closeBtn.addEventListener('click', () => overlay.remove());
    modal.appendChild(closeBtn);

    overlay.appendChild(modal);
    document.body.appendChild(overlay);
  }
}

// No auto-init: initialized by app.js
