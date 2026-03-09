// Drag from palette to canvas
export class DragManager {
  constructor(canvas, dispatcher) {
    this.canvas = canvas;
    this.dispatcher = dispatcher;
    this.ghost = null;
    this.defId = null;
    this._bound = { move: this._onMove.bind(this), up: this._onUp.bind(this) };
    this._bindPalette();
  }

  _bindPalette() {
    document.querySelectorAll('.palette-item').forEach(item => {
      item.addEventListener('mousedown', (e) => {
        if (e.button !== 0) return;
        this.defId = item.dataset.definitionId;
        this._createGhost(item.dataset.nodeName, e.clientX, e.clientY);
        window.addEventListener('mousemove', this._bound.move);
        window.addEventListener('mouseup', this._bound.up);
        e.preventDefault();
      });
    });
  }

  _createGhost(name, x, y) {
    this.ghost = document.createElement('div');
    this.ghost.className = 'drag-ghost';
    this.ghost.textContent = name;
    this.ghost.style.left = x + 'px';
    this.ghost.style.top = y + 'px';
    document.body.appendChild(this.ghost);
  }

  _onMove(e) {
    if (this.ghost) {
      this.ghost.style.left = e.clientX + 'px';
      this.ghost.style.top = e.clientY + 'px';
    }
  }

  _onUp(e) {
    window.removeEventListener('mousemove', this._bound.move);
    window.removeEventListener('mouseup', this._bound.up);
    if (this.ghost) { this.ghost.remove(); this.ghost = null; }

    const container = document.getElementById('canvas-container');
    if (!container) return;
    const rect = container.getBoundingClientRect();
    if (e.clientX < rect.left || e.clientX > rect.right ||
        e.clientY < rect.top || e.clientY > rect.bottom) return;

    const pos = this.canvas.screenToCanvas(e.clientX, e.clientY);
    const snapped = this.canvas.snapToGrid(pos.x, pos.y);
    this.dispatcher.dispatch('add_node', { definitionId: this.defId, x: snapped.x, y: snapped.y });
    this.defId = null;
  }
}

// No auto-init: initialized by app.js
