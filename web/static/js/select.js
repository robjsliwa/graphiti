// Selection management: click, multi-select, rubber band, node drag, deletion
export class SelectionManager {
  constructor(canvas, dispatcher) {
    this.canvas = canvas;
    this.dispatcher = dispatcher;
    this.selectedNodes = new Set();
    this.selectedEdges = new Set();
    this._dragging = false;
    this._dragNodeId = null;
    this._dragStartX = 0;
    this._dragStartY = 0;
    this._nodeOrigPositions = new Map();
    this._selRect = null;
    this._selStartX = 0;
    this._selStartY = 0;
    this._bindEvents();
  }

  _bindEvents() {
    const svg = this.canvas.svg;

    // Double-click on sub-workflow node to drill in
    svg.addEventListener('dblclick', (e) => {
      const node = e.target.closest('.node');
      if (!node) return;
      const defId = node.dataset.definitionId;
      if (defId !== 'control-sub-workflow') return;
      e.preventDefault();
      e.stopPropagation();
      this._openSubWorkflow(node.dataset.nodeId);
    });

    svg.addEventListener('mousedown', (e) => {
      if (e.target.closest('.port')) return; // ports are for ConnectManager
      if (e.button !== 0 || e.getModifierState('Space')) return;

      const node = e.target.closest('.node');
      const edge = e.target.closest('.edge');

      if (node) {
        const id = node.dataset.nodeId;
        if (e.ctrlKey || e.metaKey) {
          this._toggle(id);
        } else if (!this.selectedNodes.has(id)) {
          this.clearSelection();
          this._selectNode(id);
        }
        this._startDrag(e);
        e.stopPropagation();
      } else if (edge) {
        this.clearSelection();
        this._selectEdge(edge.dataset.edgeId);
        e.stopPropagation();
      } else {
        if (e.shiftKey) {
          // Shift+drag = rubber band selection
          if (!e.ctrlKey && !e.metaKey) this.clearSelection();
          this._startRubberBand(e);
        } else {
          // Plain drag on empty canvas = pan
          this.clearSelection();
          this.canvas.startPan(e);
        }
      }
    });

    window.addEventListener('mousemove', (e) => {
      if (this._dragging) this._onDrag(e);
      if (this._selRect) this._onRubberBand(e);
    });

    window.addEventListener('mouseup', (e) => {
      if (this._dragging) this._endDrag(e);
      if (this._selRect) this._endRubberBand(e);
    });

    document.addEventListener('keydown', (e) => {
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') return;
      if (e.key === 'Delete' || e.key === 'Backspace') {
        e.preventDefault();
        this._deleteSelection();
      } else if (e.key === 'Escape') {
        this.clearSelection();
      } else if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
        e.preventDefault();
        this._selectAll();
      } else if (e.key === 'Tab') {
        e.preventDefault();
        this._cycleSelection(e.shiftKey);
      }
    });
  }

  _selectNode(id) {
    this.selectedNodes.add(id);
    this.canvas.svg.querySelector(`[data-node-id="${id}"]`)?.classList.add('selected');
    this._loadConfig(id);
  }

  _selectEdge(id) {
    this.selectedEdges.add(id);
    this.canvas.svg.querySelector(`[data-edge-id="${id}"]`)?.classList.add('selected');
  }

  _toggle(id) {
    if (this.selectedNodes.has(id)) {
      this.selectedNodes.delete(id);
      this.canvas.svg.querySelector(`[data-node-id="${id}"]`)?.classList.remove('selected');
    } else {
      this._selectNode(id);
    }
  }

  clearSelection() {
    this.selectedNodes.clear();
    this.selectedEdges.clear();
    this.canvas.svg.querySelectorAll('.selected').forEach(el => el.classList.remove('selected'));
    const content = document.getElementById('config-content');
    if (content) {
      content.textContent = '';
      const empty = document.createElement('div');
      empty.className = 'empty-config';
      const p = document.createElement('p');
      p.textContent = 'Select a node to configure';
      empty.appendChild(p);
      content.appendChild(empty);
    }
  }

  getSelectedNodeIds() { return Array.from(this.selectedNodes); }

  _loadConfig(nodeId) {
    const wfId = window.GRAPHITI?.workflowID;
    if (!wfId || typeof htmx === 'undefined') return;
    htmx.ajax('GET', `/api/workflows/${wfId}/nodes/${nodeId}/config`, '#config-content');
  }

  // Node dragging
  _startDrag(e) {
    this._dragging = true;
    this._dragStartX = e.clientX;
    this._dragStartY = e.clientY;
    this._nodeOrigPositions.clear();
    // Capture original positions of all selected nodes
    const clickedNode = e.target.closest('.node');
    this._dragNodeId = clickedNode?.dataset.nodeId;
    const ids = this.selectedNodes.size > 0 && this.selectedNodes.has(this._dragNodeId)
      ? this.selectedNodes : new Set([this._dragNodeId]);
    ids.forEach(id => {
      const el = this.canvas.svg.querySelector(`[data-node-id="${id}"]`);
      if (el) this._nodeOrigPositions.set(id, this.canvas.getNodePosition(el));
    });
  }

  _onDrag(e) {
    const dx = (e.clientX - this._dragStartX) / this.canvas.scale;
    const dy = (e.clientY - this._dragStartY) / this.canvas.scale;
    this._nodeOrigPositions.forEach((orig, id) => {
      const el = this.canvas.svg.querySelector(`[data-node-id="${id}"]`);
      if (!el) return;
      el.setAttribute('transform', `translate(${orig.x + dx}, ${orig.y + dy})`);
      this.canvas.updateEdgesForNode(id);
    });
  }

  _endDrag(e) {
    this._dragging = false;
    const dx = (e.clientX - this._dragStartX) / this.canvas.scale;
    const dy = (e.clientY - this._dragStartY) / this.canvas.scale;
    if (Math.abs(dx) < 2 && Math.abs(dy) < 2) return; // was a click, not drag

    if (this._nodeOrigPositions.size > 1) {
      const from = [], to = [];
      this._nodeOrigPositions.forEach((orig, id) => {
        const snapped = this.canvas.snapToGrid(orig.x + dx, orig.y + dy);
        from.push({ nodeId: id, x: orig.x, y: orig.y });
        to.push({ nodeId: id, x: snapped.x, y: snapped.y });
      });
      this.dispatcher.dispatch('move_nodes', { from, to });
    } else {
      const [id, orig] = [...this._nodeOrigPositions.entries()][0];
      const snapped = this.canvas.snapToGrid(orig.x + dx, orig.y + dy);
      this.dispatcher.dispatch('move_node', {
        nodeId: id, fromX: orig.x, fromY: orig.y, toX: snapped.x, toY: snapped.y,
      });
    }
  }

  // Rubber band selection
  _startRubberBand(e) {
    const pos = this.canvas.screenToCanvas(e.clientX, e.clientY);
    this._selStartX = pos.x;
    this._selStartY = pos.y;
    const ns = 'http://www.w3.org/2000/svg';
    this._selRect = document.createElementNS(ns, 'rect');
    this._selRect.setAttribute('class', 'selection-rect');
    this._selRect.setAttribute('fill', 'var(--accent)');
    this._selRect.setAttribute('fill-opacity', '0.1');
    this._selRect.setAttribute('stroke', 'var(--accent)');
    this._selRect.setAttribute('stroke-width', '1');
    this._selRect.setAttribute('stroke-dasharray', '4 2');
    this.canvas.interactionLayer.appendChild(this._selRect);
  }

  _onRubberBand(e) {
    const pos = this.canvas.screenToCanvas(e.clientX, e.clientY);
    const x = Math.min(this._selStartX, pos.x);
    const y = Math.min(this._selStartY, pos.y);
    this._selRect.setAttribute('x', x);
    this._selRect.setAttribute('y', y);
    this._selRect.setAttribute('width', Math.abs(pos.x - this._selStartX));
    this._selRect.setAttribute('height', Math.abs(pos.y - this._selStartY));
  }

  _endRubberBand(e) {
    const pos = this.canvas.screenToCanvas(e.clientX, e.clientY);
    const x1 = Math.min(this._selStartX, pos.x), y1 = Math.min(this._selStartY, pos.y);
    const x2 = Math.max(this._selStartX, pos.x), y2 = Math.max(this._selStartY, pos.y);
    if (this._selRect) { this._selRect.remove(); this._selRect = null; }
    // Only select if we actually dragged a rectangle (not just a click)
    if (Math.abs(x2 - x1) < 5 && Math.abs(y2 - y1) < 5) return;
    this.canvas.svg.querySelectorAll('.node').forEach(el => {
      const p = this.canvas.getNodePosition(el);
      if (p.x >= x1 && p.y >= y1 && p.x <= x2 && p.y <= y2) {
        this._selectNode(el.dataset.nodeId);
      }
    });
  }

  _deleteSelection() {
    if (this.selectedNodes.size === 0 && this.selectedEdges.size === 0) return;
    // Delete edges first, then nodes (each is a separate undo step)
    const edgeIds = [...this.selectedEdges];
    const nodeIds = [...this.selectedNodes];
    this.clearSelection();

    const deleteNext = async () => {
      for (const id of edgeIds) await this.dispatcher.dispatch('remove_edge', { edgeId: id });
      for (const id of nodeIds) await this.dispatcher.dispatch('remove_node', { nodeId: id });
    };
    deleteNext();
  }

  _selectAll() {
    this.canvas.svg.querySelectorAll('.node').forEach(el => {
      this.selectedNodes.add(el.dataset.nodeId);
      el.classList.add('selected');
    });
  }

  // Sub-workflow drill-in: fetch the referenced workflow ID and navigate
  async _openSubWorkflow(nodeId) {
    const wfId = window.GRAPHITI?.workflowID;
    if (!wfId) return;
    try {
      const resp = await fetch(`/api/workflows/${wfId}/nodes/${nodeId}/ref`);
      if (!resp.ok) {
        window.toast?.error('Failed to get sub-workflow reference');
        return;
      }
      const data = await resp.json();
      if (!data.workflowRef) {
        window.toast?.warn('No workflow reference configured for this sub-workflow node');
        return;
      }
      window.location.href = `/workflows/${data.workflowRef}?parent=${wfId}&parentNode=${nodeId}`;
    } catch (err) {
      window.toast?.error('Failed to navigate to sub-workflow');
    }
  }

  _cycleSelection(reverse) {
    const nodes = Array.from(this.canvas.svg.querySelectorAll('.node'));
    if (nodes.length === 0) return;
    const currentId = this.selectedNodes.size === 1 ? [...this.selectedNodes][0] : null;
    let idx = currentId ? nodes.findIndex(n => n.dataset.nodeId === currentId) : -1;
    idx = reverse ? (idx <= 0 ? nodes.length - 1 : idx - 1) : (idx + 1) % nodes.length;
    this.clearSelection();
    this._selectNode(nodes[idx].dataset.nodeId);
  }
}

// No auto-init: initialized by app.js
