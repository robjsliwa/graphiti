// Canvas engine: pan, zoom, viewport transforms, SVG node rendering, canvas sync
export class CanvasEngine {
  constructor(svgElement) {
    this.svg = svgElement;
    this.contentGroup = svgElement.querySelector('.canvas-content');
    this.nodeLayer = svgElement.querySelector('.node-layer');
    this.edgeLayer = svgElement.querySelector('.edge-layer');
    this.interactionLayer = svgElement.querySelector('.interaction-layer');
    this.scale = 1.0;
    this.panX = 0;
    this.panY = 0;
    this.minScale = 0.25;
    this.maxScale = 2.0;
    this.gridSize = 24;
    this._isPanning = false;
    this._bindEvents();
    // Auto-fit view when loading a workflow with existing nodes
    if (this.svg.querySelectorAll('.node').length > 0) {
      this.fitToView();
    }
  }

  screenToCanvas(screenX, screenY) {
    const rect = this.svg.getBoundingClientRect();
    return {
      x: (screenX - rect.left - this.panX) / this.scale,
      y: (screenY - rect.top - this.panY) / this.scale,
    };
  }

  canvasToScreen(canvasX, canvasY) {
    const rect = this.svg.getBoundingClientRect();
    return {
      x: canvasX * this.scale + this.panX + rect.left,
      y: canvasY * this.scale + this.panY + rect.top,
    };
  }

  snapToGrid(x, y) {
    return {
      x: Math.round(x / this.gridSize) * this.gridSize,
      y: Math.round(y / this.gridSize) * this.gridSize,
    };
  }

  zoom(delta, centerX, centerY) {
    const oldScale = this.scale;
    this.scale *= delta > 0 ? 1.1 : 0.9;
    this.scale = Math.max(this.minScale, Math.min(this.maxScale, this.scale));
    const rect = this.svg.getBoundingClientRect();
    const cx = centerX - rect.left;
    const cy = centerY - rect.top;
    this.panX = cx - (cx - this.panX) * (this.scale / oldScale);
    this.panY = cy - (cy - this.panY) * (this.scale / oldScale);
    this._applyTransform();
    this._updateZoomDisplay();
  }

  pan(dx, dy) {
    this.panX += dx;
    this.panY += dy;
    this._applyTransform();
  }

  startPan(e) {
    this._isPanning = true;
    this._lastX = e.clientX;
    this._lastY = e.clientY;
    this.svg.classList.add('panning');
  }

  fitToView() {
    const nodes = this.svg.querySelectorAll('.node');
    if (nodes.length === 0) return;
    let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
    nodes.forEach(n => {
      const pos = this.getNodePosition(n);
      minX = Math.min(minX, pos.x);
      minY = Math.min(minY, pos.y);
      maxX = Math.max(maxX, pos.x + 200);
      maxY = Math.max(maxY, pos.y + 80);
    });
    if (!isFinite(minX)) return;
    const pad = 48;
    const rect = this.svg.getBoundingClientRect();
    const cw = maxX - minX + pad * 2, ch = maxY - minY + pad * 2;
    this.scale = Math.min(rect.width / cw, rect.height / ch, 1.0);
    this.panX = (rect.width - cw * this.scale) / 2 - minX * this.scale + pad * this.scale;
    this.panY = (rect.height - ch * this.scale) / 2 - minY * this.scale + pad * this.scale;
    this._applyTransform();
    this._updateZoomDisplay();
  }

  getNodePosition(el) {
    const t = el.getAttribute('transform');
    const m = t && t.match(/translate\(([^,]+),\s*([^)]+)\)/);
    return m ? { x: parseFloat(m[1]), y: parseFloat(m[2]) } : { x: 0, y: 0 };
  }

  // Sync the canvas SVG DOM from workflow state JSON
  syncCanvas(state) {
    if (!state) return;
    const stateNodeIds = new Set((state.nodes || []).map(n => n.id));
    const stateEdgeIds = new Set((state.edges || []).map(e => e.id));

    // Remove deleted nodes/edges
    this.nodeLayer.querySelectorAll('.node').forEach(el => {
      if (!stateNodeIds.has(el.dataset.nodeId)) el.remove();
    });
    this.edgeLayer.querySelectorAll('.edge').forEach(el => {
      if (!stateEdgeIds.has(el.dataset.edgeId)) el.remove();
    });

    // Add/update nodes
    for (const node of (state.nodes || [])) {
      let el = this.nodeLayer.querySelector(`[data-node-id="${node.id}"]`);
      if (el) {
        el.setAttribute('transform', `translate(${node.x}, ${node.y})`);
        const titleEl = el.querySelector('.node-title');
        if (titleEl) titleEl.textContent = node.label;
      } else {
        this.nodeLayer.appendChild(this.createNodeSVG(node));
      }
    }

    // Add/update edges
    for (const edge of (state.edges || [])) {
      let el = this.edgeLayer.querySelector(`[data-edge-id="${edge.id}"]`);
      if (!el) {
        this.edgeLayer.appendChild(this._createEdgeSVG(edge, state.nodes));
      } else {
        el.setAttribute('d', this._calcEdgePath(edge, state.nodes));
      }
    }
  }

  // Update edges connected to a node during drag
  updateEdgesForNode(nodeId) {
    this.edgeLayer.querySelectorAll('.edge').forEach(el => {
      if (el.dataset.sourceNode !== nodeId && el.dataset.targetNode !== nodeId) return;
      const srcEl = this.nodeLayer.querySelector(`[data-node-id="${el.dataset.sourceNode}"]`);
      const tgtEl = this.nodeLayer.querySelector(`[data-node-id="${el.dataset.targetNode}"]`);
      if (!srcEl || !tgtEl) return;
      const sp = this.getNodePosition(srcEl), tp = this.getNodePosition(tgtEl);
      const sw = parseInt(srcEl.querySelector('.node-header')?.getAttribute('width') || '200');
      const x1 = sp.x + sw, y1 = sp.y + 30, x2 = tp.x, y2 = tp.y + 30;
      const cp = Math.max(50, (x2 - x1) * 0.5);
      el.setAttribute('d', `M ${x1} ${y1} C ${x1+cp} ${y1}, ${x2-cp} ${y2}, ${x2} ${y2}`);
    });
  }

  // Create SVG group for a node from JSON data
  createNodeSVG(node) {
    const ns = 'http://www.w3.org/2000/svg';
    const el = (tag, attrs, text) => {
      const e = document.createElementNS(ns, tag);
      for (const [k, v] of Object.entries(attrs || {})) {
        if (k === 'class') e.setAttribute('class', v);
        else if (k.startsWith('data-')) e.dataset[k.slice(5).replace(/-([a-z])/g, (_, c) => c.toUpperCase())] = v;
        else e.setAttribute(k, v);
      }
      if (text) e.textContent = text;
      return e;
    };

    const def = node.definition || {};
    const shape = def.shape || {};
    const w = shape.width || 200;
    const bodyAttrs = (def.attributes || []).filter(a => a.display === 'node-body' || a.display === 'both');
    const h = Math.max(60, 40 + bodyAttrs.length * 20 + 8);
    const hdrBg = shape.headerBackground || 'var(--surface-2)';
    const hdrColor = shape.headerColor || 'var(--text)';

    const g = el('g', { class: 'node', 'data-node-id': node.id, 'data-definition-id': node.definitionId,
      transform: `translate(${node.x}, ${node.y})` });

    // Background shape
    if (shape.type === 'diamond') {
      g.appendChild(el('polygon', { points: `${w/2} 0, ${w} 40, ${w/2} 80, 0 40`,
        class: 'node-bg', fill: 'var(--surface)', stroke: 'var(--border)', 'stroke-width': '1.5' }));
    } else if (shape.type === 'hexagon') {
      g.appendChild(el('polygon', { points: `20 0, ${w-20} 0, ${w} 40, ${w-20} 80, 20 80, 0 40`,
        class: 'node-bg', fill: 'var(--surface)', stroke: 'var(--border)', 'stroke-width': '1.5' }));
    } else {
      g.appendChild(el('rect', { width: w, height: h, rx: 8, ry: 8,
        class: 'node-bg', fill: 'var(--surface)', stroke: 'var(--border)', 'stroke-width': '1.5' }));
      if (shape.type === 'sub-workflow') {
        g.appendChild(el('rect', { x: 3, y: 3, width: w-6, height: h-6, rx: 6, ry: 6,
          fill: 'none', stroke: 'var(--border)', 'stroke-width': '1' }));
      }
    }

    // Header
    g.appendChild(el('rect', { width: w, height: 32, rx: 6, ry: 6, class: 'node-header', fill: hdrBg }));
    g.appendChild(el('rect', { x: 0, y: 16, width: w, height: 16, fill: hdrBg }));
    g.appendChild(el('text', { x: 12, y: 22, class: 'node-icon', fill: hdrColor }, def.icon || '?'));
    g.appendChild(el('text', { x: 32, y: 22, class: 'node-title', fill: hdrColor }, node.label));

    // Body attributes
    bodyAttrs.forEach((attr, i) => {
      const val = node.attributes?.[attr.id] ?? '';
      g.appendChild(el('text', { x: 12, y: 52 + i * 20, class: 'node-attr-label' }, `${attr.label}: ${val}`));
    });

    // Ports
    const portEl = (port, isInput, idx) => {
      const px = this._portX(port, isInput, w);
      const py = this._portY(port, idx, h);
      return el('circle', { cx: px, cy: py, r: 6,
        class: `port port-${port.type} port-${isInput ? 'input' : 'output'}`,
        'data-port-id': port.id, 'data-port-type': port.type, 'data-is-input': isInput });
    };
    (def.inputs || []).forEach((p, i) => g.appendChild(portEl(p, true, i)));
    (def.outputs || []).forEach((p, i) => g.appendChild(portEl(p, false, i)));

    return g;
  }

  _portX(port, isInput, w) {
    if (port.position === 'left-center') return 0;
    if (['right-center', 'right-top', 'right-bottom'].includes(port.position)) return w;
    return isInput ? 0 : w;
  }

  _portY(port, idx, h) {
    switch (port.position) {
      case 'top-center': return 0;
      case 'bottom-center': return h;
      case 'right-top': return 20 + idx * 20;
      case 'right-bottom': return 50 + idx * 20;
      default: return 30 + idx * 20;
    }
  }

  _createEdgeSVG(edge, nodes) {
    const ns = 'http://www.w3.org/2000/svg';
    const path = document.createElementNS(ns, 'path');
    path.setAttribute('class', 'edge');
    path.dataset.edgeId = edge.id;
    path.dataset.sourceNode = edge.sourceNodeId;
    path.dataset.targetNode = edge.targetNodeId;
    path.setAttribute('d', this._calcEdgePath(edge, nodes));
    path.setAttribute('fill', 'none');
    path.setAttribute('stroke', 'var(--edge-color, #94a3b8)');
    path.setAttribute('stroke-width', '2');
    path.setAttribute('marker-end', 'url(#arrowhead)');
    return path;
  }

  _calcEdgePath(edge, nodes) {
    const src = nodes.find(n => n.id === edge.sourceNodeId);
    const tgt = nodes.find(n => n.id === edge.targetNodeId);
    if (!src || !tgt) return '';
    const sw = src.definition?.shape?.width || 200;
    const x1 = src.x + sw, y1 = src.y + 30, x2 = tgt.x, y2 = tgt.y + 30;
    const cp = Math.max(50, (x2 - x1) * 0.5);
    return `M ${x1} ${y1} C ${x1+cp} ${y1}, ${x2-cp} ${y2}, ${x2} ${y2}`;
  }

  _applyTransform() {
    requestAnimationFrame(() => {
      this.contentGroup.setAttribute('transform',
        `translate(${this.panX}, ${this.panY}) scale(${this.scale})`);
      const pattern = this.svg.querySelector('#dot-grid');
      if (pattern) {
        const size = 24 * this.scale;
        pattern.setAttribute('width', size);
        pattern.setAttribute('height', size);
        pattern.setAttribute('patternTransform', `translate(${this.panX % size}, ${this.panY % size})`);
        const dot = pattern.querySelector('circle');
        if (dot) {
          dot.setAttribute('cx', size / 2);
          dot.setAttribute('cy', size / 2);
          dot.setAttribute('r', Math.max(0.5, this.scale));
        }
      }
    });
  }

  _updateZoomDisplay() {
    const el = document.getElementById('zoom-level');
    if (el) el.textContent = Math.round(this.scale * 100) + '%';
  }

  _bindEvents() {
    this.svg.addEventListener('wheel', (e) => {
      e.preventDefault();
      this.zoom(-e.deltaY, e.clientX, e.clientY);
    }, { passive: false });

    this.svg.addEventListener('mousedown', (e) => {
      if (e.button === 1 || (e.button === 0 && e.getModifierState('Space'))) {
        this.startPan(e);
        e.preventDefault();
      }
    });

    window.addEventListener('mousemove', (e) => {
      if (!this._isPanning) return;
      this.pan(e.clientX - this._lastX, e.clientY - this._lastY);
      this._lastX = e.clientX;
      this._lastY = e.clientY;
    });

    window.addEventListener('mouseup', () => {
      if (this._isPanning) {
        this._isPanning = false;
        this.svg.classList.remove('panning');
      }
    });

    document.getElementById('zoom-in')?.addEventListener('click', () => {
      const r = this.svg.getBoundingClientRect();
      this.zoom(1, r.left + r.width / 2, r.top + r.height / 2);
    });
    document.getElementById('zoom-out')?.addEventListener('click', () => {
      const r = this.svg.getBoundingClientRect();
      this.zoom(-1, r.left + r.width / 2, r.top + r.height / 2);
    });
    document.getElementById('zoom-fit')?.addEventListener('click', () => this.fitToView());
  }
}

// No auto-init: initialized by app.js
