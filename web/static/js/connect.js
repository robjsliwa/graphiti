// Edge drawing between ports
export class ConnectManager {
  constructor(canvas, dispatcher) {
    this.canvas = canvas;
    this.dispatcher = dispatcher;
    this.drawing = false;
    this.preview = null;
    this.srcNodeId = null;
    this.srcPortId = null;
    this.srcPortType = null;
    this.startX = 0;
    this.startY = 0;
    this._bindEvents();
  }

  _bindEvents() {
    const svg = this.canvas.svg;

    svg.addEventListener('mousedown', (e) => {
      const port = e.target.closest('.port-output');
      if (!port || e.button !== 0) return;
      const node = port.closest('.node');
      if (!node) return;

      this.drawing = true;
      this.srcNodeId = node.dataset.nodeId;
      this.srcPortId = port.dataset.portId;
      this.srcPortType = port.dataset.portType;

      const nPos = this.canvas.getNodePosition(node);
      this.startX = nPos.x + parseFloat(port.getAttribute('cx'));
      this.startY = nPos.y + parseFloat(port.getAttribute('cy'));
      this._createPreview();
      e.stopPropagation();
    });

    svg.addEventListener('mousemove', (e) => {
      if (!this.drawing) return;
      const pos = this.canvas.screenToCanvas(e.clientX, e.clientY);
      this._updatePreview(pos.x, pos.y);
      this._highlightTargets(e.target);
    });

    svg.addEventListener('mouseup', (e) => {
      if (!this.drawing) return;
      this.drawing = false;
      this._removePreview();
      this._clearHighlights();

      const port = e.target.closest('.port-input');
      if (!port) return;
      const node = port.closest('.node');
      if (!node) return;
      if (port.dataset.portType !== this.srcPortType) return;
      if (node.dataset.nodeId === this.srcNodeId) return;

      this.dispatcher.dispatch('add_edge', {
        sourceNodeId: this.srcNodeId,
        sourcePortId: this.srcPortId,
        targetNodeId: node.dataset.nodeId,
        targetPortId: port.dataset.portId,
      });
    });
  }

  _createPreview() {
    const ns = 'http://www.w3.org/2000/svg';
    this.preview = document.createElementNS(ns, 'path');
    this.preview.setAttribute('class', 'edge-preview');
    this.preview.setAttribute('fill', 'none');
    this.preview.setAttribute('stroke', 'var(--accent)');
    this.preview.setAttribute('stroke-width', '2');
    this.preview.setAttribute('stroke-dasharray', '6 3');
    this.canvas.interactionLayer.appendChild(this.preview);
  }

  _updatePreview(x, y) {
    if (!this.preview) return;
    const cp = Math.max(50, Math.abs(x - this.startX) * 0.5);
    this.preview.setAttribute('d',
      `M ${this.startX} ${this.startY} C ${this.startX+cp} ${this.startY}, ${x-cp} ${y}, ${x} ${y}`);
  }

  _removePreview() {
    if (this.preview) { this.preview.remove(); this.preview = null; }
  }

  _highlightTargets(target) {
    this._clearHighlights();
    const port = target.closest('.port-input');
    if (port) {
      const valid = port.dataset.portType === this.srcPortType &&
                    port.closest('.node')?.dataset.nodeId !== this.srcNodeId;
      port.classList.add(valid ? 'port-valid-target' : 'port-invalid-target');
    }
  }

  _clearHighlights() {
    this.canvas.svg.querySelectorAll('.port-valid-target, .port-invalid-target')
      .forEach(el => el.classList.remove('port-valid-target', 'port-invalid-target'));
  }
}

// No auto-init: initialized by app.js
