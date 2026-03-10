// Execution mode: WebSocket connection, status overlays, mode switching

export class ExecutionManager {
  constructor(workflowId) {
    this.workflowId = workflowId;
    this.ws = null;
    this.mode = 'builder'; // 'builder' or 'execution'
    this._bindModeTabs();
  }

  _bindModeTabs() {
    const builderTab = document.getElementById('mode-builder');
    const execTab = document.getElementById('mode-execution');
    if (!builderTab || !execTab) return;

    builderTab.addEventListener('click', () => this.setMode('builder'));
    execTab.addEventListener('click', () => this.setMode('execution'));
  }

  setMode(mode) {
    this.mode = mode;
    const builderTab = document.getElementById('mode-builder');
    const execTab = document.getElementById('mode-execution');
    const builderPanel = document.getElementById('panel-left-builder');
    const execPanel = document.getElementById('panel-left-execution');
    const undoRedo = document.getElementById('undo-redo-controls');

    if (builderTab) builderTab.classList.toggle('active', mode === 'builder');
    if (execTab) execTab.classList.toggle('active', mode === 'execution');
    if (builderPanel) builderPanel.style.display = mode === 'builder' ? '' : 'none';
    if (execPanel) execPanel.style.display = mode === 'execution' ? '' : 'none';
    if (undoRedo) undoRedo.style.display = mode === 'builder' ? '' : 'none';

    // Toggle canvas interactivity
    const canvas = document.getElementById('workflow-canvas');
    if (canvas) {
      canvas.classList.toggle('execution-mode', mode === 'execution');
    }

    if (mode === 'execution') {
      this.connectWebSocket();
    } else {
      this.disconnectWebSocket();
      this._clearOverlays();
    }
  }

  connectWebSocket() {
    if (this.ws) return;

    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${proto}//${location.host}/api/ws/workflows/${this.workflowId}`;

    try {
      this.ws = new WebSocket(url);
      this.ws.onmessage = (e) => this._onMessage(JSON.parse(e.data));
      this.ws.onclose = () => {
        this.ws = null;
        // Reconnect after 3s if still in execution mode
        if (this.mode === 'execution') {
          setTimeout(() => this.connectWebSocket(), 3000);
        }
      };
      this.ws.onerror = () => this.ws.close();
    } catch (err) {
      console.warn('WebSocket connection failed:', err);
    }
  }

  disconnectWebSocket() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  _onMessage(msg) {
    if (msg.type !== 'node_status') return;

    const nodeEl = document.querySelector(`.node[data-node-id="${msg.nodeID}"]`);
    if (!nodeEl) return;

    // Update node border color based on status
    const bg = nodeEl.querySelector('.node-bg');
    if (bg) {
      bg.classList.remove('exec-pending', 'exec-running', 'exec-completed', 'exec-failed', 'exec-skipped');
      bg.classList.add(`exec-${msg.status}`);
    }

    // Add/remove pulse animation for running nodes
    nodeEl.classList.toggle('node-pulsing', msg.status === 'running');

    // Update edge colors (edges from completed nodes turn green)
    if (msg.status === 'completed') {
      this._colorEdgesFromNode(msg.nodeID, 'var(--success)');
    }

    // Show duration badge
    if (msg.startedAt && msg.completedAt) {
      this._showDurationBadge(nodeEl, msg.startedAt, msg.completedAt);
    }

    // Refresh run list
    if (this.mode === 'execution') {
      const runList = document.getElementById('execution-run-list');
      if (runList) {
        htmx.trigger(runList, 'load');
      }
    }
  }

  _colorEdgesFromNode(nodeId, color) {
    document.querySelectorAll(`.edge[data-source-node="${nodeId}"]`).forEach(edge => {
      edge.style.stroke = color;
    });
  }

  _showDurationBadge(nodeEl, startStr, endStr) {
    const start = new Date(startStr);
    const end = new Date(endStr);
    const ms = end - start;
    const label = ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`;

    let badge = nodeEl.querySelector('.exec-duration');
    if (!badge) {
      badge = document.createElementNS('http://www.w3.org/2000/svg', 'text');
      badge.classList.add('exec-duration');
      badge.setAttribute('x', '4');
      badge.setAttribute('y', '-4');
      badge.setAttribute('font-size', '10');
      badge.setAttribute('fill', 'var(--text-muted)');
      nodeEl.appendChild(badge);
    }
    badge.textContent = label;
  }

  _clearOverlays() {
    document.querySelectorAll('.node-bg').forEach(bg => {
      bg.classList.remove('exec-pending', 'exec-running', 'exec-completed', 'exec-failed', 'exec-skipped');
    });
    document.querySelectorAll('.node-pulsing').forEach(n => n.classList.remove('node-pulsing'));
    document.querySelectorAll('.exec-duration').forEach(b => b.remove());
    document.querySelectorAll('.edge').forEach(e => e.style.stroke = '');
  }
}
