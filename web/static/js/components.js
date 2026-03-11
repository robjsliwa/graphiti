// Alpine.js components for deploy and execution mode
// Registered via alpine:init event before Alpine starts

document.addEventListener('alpine:init', () => {

  // Deploy manager: handles deploy button state, deploy API calls, and export
  Alpine.data('deployManager', () => ({
    menuOpen: false,
    isDeploying: false,
    deployStatus: null, // 'success' | 'error' | null

    get workflowId() {
      return document.getElementById('graphiti-data')?.dataset.workflowId;
    },

    async deploy(target) {
      if (this.isDeploying || !this.workflowId) return;
      this.isDeploying = true;
      this.deployStatus = null;

      try {
        const resp = await fetch(`/api/workflows/${this.workflowId}/deploy`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ target }),
        });
        const result = await resp.json();

        if (result.success) {
          window.toast?.success(`Deployed to ${target} successfully`);
          this.deployStatus = 'success';
          // Update version badge
          const badge = document.querySelector('.version-badge');
          if (badge) {
            const current = parseInt(badge.textContent.replace('v', ''));
            badge.textContent = 'v' + (current + 1);
          }
          setTimeout(() => {
            this.deployStatus = null;
            this.isDeploying = false;
          }, 2000);
        } else {
          const msg = result.message || 'Deploy failed';
          window.toast?.error(msg);
          if (result.validationErrors?.length) {
            result.validationErrors.forEach(ve => {
              window.toast?.warning(
                `${ve.message}${ve.nodeId ? ` (node: ${ve.nodeId.substring(0, 8)})` : ''}`
              );
              if (ve.nodeId) {
                const el = document.querySelector(`[data-node-id="${ve.nodeId}"]`);
                if (el) el.classList.add('node-error');
              }
            });
          }
          this.deployStatus = 'error';
          setTimeout(() => {
            this.deployStatus = null;
            this.isDeploying = false;
          }, 2000);
        }
      } catch (err) {
        console.error('Deploy error:', err);
        window.toast?.error('Network error during deploy');
        this.isDeploying = false;
      }
    },

    exportWf(format) {
      if (!this.workflowId) return;
      const url = `/api/workflows/${this.workflowId}/export?format=${format}`;
      const a = document.createElement('a');
      a.href = url;
      a.download = `workflow.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.toast?.info(`Exporting as ${format.toUpperCase()}`);
    },
  }));

  // Execution mode: mode tabs, WebSocket connection, status overlays
  Alpine.data('executionMode', () => ({
    mode: 'builder',
    ws: null,

    get workflowId() {
      return document.getElementById('graphiti-data')?.dataset.workflowId;
    },

    setMode(newMode) {
      this.mode = newMode;
      // Canvas class toggle
      const canvas = document.getElementById('workflow-canvas');
      if (canvas) canvas.classList.toggle('execution-mode', newMode === 'execution');
      // Undo/redo visibility
      const ur = document.getElementById('undo-redo-controls');
      if (ur) ur.style.display = newMode === 'builder' ? '' : 'none';
      // WebSocket lifecycle
      if (newMode === 'execution') {
        this.connectWebSocket();
      } else {
        this.disconnectWebSocket();
        this._clearOverlays();
      }
    },

    connectWebSocket() {
      if (this.ws || !this.workflowId) return;
      const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
      const url = `${proto}//${location.host}/api/ws/workflows/${this.workflowId}`;
      try {
        this.ws = new WebSocket(url);
        this.ws.onmessage = (e) => this._onMessage(JSON.parse(e.data));
        this.ws.onclose = () => {
          this.ws = null;
          if (this.mode === 'execution') {
            setTimeout(() => this.connectWebSocket(), 3000);
          }
        };
        this.ws.onerror = () => this.ws.close();
      } catch (err) {
        console.warn('WebSocket connection failed:', err);
      }
    },

    disconnectWebSocket() {
      if (this.ws) {
        this.ws.close();
        this.ws = null;
      }
    },

    _onMessage(msg) {
      if (msg.type !== 'node_status') return;
      const nodeEl = document.querySelector(`.node[data-node-id="${msg.nodeID}"]`);
      if (!nodeEl) return;

      const bg = nodeEl.querySelector('.node-bg');
      if (bg) {
        bg.classList.remove('exec-pending', 'exec-running', 'exec-completed', 'exec-failed', 'exec-skipped');
        bg.classList.add(`exec-${msg.status}`);
      }
      nodeEl.classList.toggle('node-pulsing', msg.status === 'running');

      if (msg.status === 'completed') {
        document.querySelectorAll(`.edge[data-source-node="${msg.nodeID}"]`).forEach(edge => {
          edge.style.stroke = 'var(--success)';
        });
      }

      if (msg.startedAt && msg.completedAt) {
        this._showDurationBadge(nodeEl, msg.startedAt, msg.completedAt);
      }

      if (this.mode === 'execution') {
        const runList = document.getElementById('execution-run-list');
        if (runList) htmx.trigger(runList, 'load');
      }
    },

    _showDurationBadge(nodeEl, startStr, endStr) {
      const ms = new Date(endStr) - new Date(startStr);
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
    },

    _clearOverlays() {
      document.querySelectorAll('.node-bg').forEach(bg => {
        bg.classList.remove('exec-pending', 'exec-running', 'exec-completed', 'exec-failed', 'exec-skipped');
      });
      document.querySelectorAll('.node-pulsing').forEach(n => n.classList.remove('node-pulsing'));
      document.querySelectorAll('.exec-duration').forEach(b => b.remove());
      document.querySelectorAll('.edge').forEach(e => e.style.stroke = '');
    },
  }));

  // nodeUpdated listener for config panel SVG updates
  document.body.addEventListener('nodeUpdated', (e) => {
    const { nodeId, label, attributes } = e.detail;
    const svg = document.getElementById('workflow-canvas');
    if (!svg) return;
    const nodeEl = svg.querySelector(`[data-node-id="${nodeId}"]`);
    if (!nodeEl) return;
    const titleEl = nodeEl.querySelector('.node-title');
    if (titleEl) titleEl.textContent = label;
    const attrEls = nodeEl.querySelectorAll('.node-attr-label');
    (attributes || []).forEach((attr, i) => {
      if (attrEls[i]) attrEls[i].textContent = `${attr.label}: ${attr.value}`;
    });
  });
});
