// Alpine.js components for deploy and execution mode
// Registered via alpine:init event before Alpine starts

document.addEventListener('alpine:init', () => {

  // Deploy manager: handles deploy button state, deploy API calls, validation, and export
  Alpine.data('deployManager', () => ({
    menuOpen: false,
    isDeploying: false,
    isValidating: false,
    deployStatus: null, // 'success' | 'error' | null
    errorCount: 0,

    get workflowId() {
      return document.getElementById('graphiti-data')?.dataset.workflowId;
    },

    clearValidationHighlights() {
      document.querySelectorAll('.node-error, .node-warning').forEach(el => {
        el.classList.remove('node-error', 'node-warning');
      });
      document.querySelectorAll('.edge-error').forEach(el => {
        el.classList.remove('edge-error');
      });
      document.querySelectorAll('.validation-badge, .validation-badge-bg').forEach(el => {
        el.remove();
      });
      this.errorCount = 0;
    },

    applyValidationResults(results) {
      this.clearValidationHighlights();
      if (!results?.length) return;

      const byNode = {};
      let errors = 0;

      results.forEach(r => {
        if (r.severity === 'error') errors++;
        if (r.nodeId) {
          if (!byNode[r.nodeId]) byNode[r.nodeId] = { errors: 0, warnings: 0 };
          if (r.severity === 'error') byNode[r.nodeId].errors++;
          else if (r.severity === 'warning') byNode[r.nodeId].warnings++;
        }
        if (r.edgeId) {
          const edgeEl = document.querySelector(`[data-edge-id="${r.edgeId}"]`);
          if (edgeEl && r.severity === 'error') edgeEl.classList.add('edge-error');
        }
      });

      Object.entries(byNode).forEach(([nodeId, counts]) => {
        const el = document.querySelector(`[data-node-id="${nodeId}"]`);
        if (!el) return;
        if (counts.errors > 0) el.classList.add('node-error');
        else if (counts.warnings > 0) el.classList.add('node-warning');
      });

      this.errorCount = errors;
    },

    showValidationPanel(results) {
      let panel = document.getElementById('validation-panel');
      if (!panel) {
        panel = document.createElement('div');
        panel.id = 'validation-panel';
        panel.className = 'validation-panel';
        document.body.appendChild(panel);
      }

      if (!results || results.length === 0) {
        panel.remove();
        return;
      }

      // Clear previous content safely
      panel.textContent = '';

      // Header
      const header = document.createElement('div');
      header.className = 'validation-panel-header';
      const headerText = document.createElement('span');
      headerText.textContent = 'Validation Results';
      header.appendChild(headerText);
      const closeBtn = document.createElement('button');
      closeBtn.className = 'validation-panel-close';
      closeBtn.textContent = '\u00D7';
      closeBtn.addEventListener('click', () => panel.remove());
      header.appendChild(closeBtn);
      panel.appendChild(header);

      const body = document.createElement('div');
      body.className = 'validation-panel-body';

      // Group by node, workflow-level first
      const workflowLevel = results.filter(r => !r.nodeId);
      const byNode = {};
      results.filter(r => r.nodeId).forEach(r => {
        if (!byNode[r.nodeId]) byNode[r.nodeId] = [];
        byNode[r.nodeId].push(r);
      });

      const severityIcon = (sev) => sev === 'error' ? '\u2716' : sev === 'warning' ? '\u26A0' : '\u2139';

      const createGroup = (title, items, targetNodeId) => {
        const group = document.createElement('div');
        group.className = 'validation-group';
        const groupHeader = document.createElement('div');
        groupHeader.className = 'validation-group-header';
        groupHeader.textContent = title;
        group.appendChild(groupHeader);
        items.forEach(r => {
          const item = document.createElement('div');
          item.className = `validation-item validation-${r.severity}`;
          item.textContent = `${severityIcon(r.severity)} ${r.message}`;
          if (targetNodeId || r.nodeId) {
            item.style.cursor = 'pointer';
            item.addEventListener('click', () => {
              const nid = targetNodeId || r.nodeId;
              const nodeEl = document.querySelector(`[data-node-id="${nid}"]`);
              if (!nodeEl) return;
              if (window.selectionManager) window.selectionManager.selectNode(nid);
              nodeEl.scrollIntoView({ behavior: 'smooth', block: 'center', inline: 'center' });
            });
          }
          group.appendChild(item);
        });
        return group;
      };

      if (workflowLevel.length > 0) {
        body.appendChild(createGroup('Workflow', workflowLevel));
      }
      Object.entries(byNode).forEach(([nodeId, items]) => {
        const nodeEl = document.querySelector(`[data-node-id="${nodeId}"]`);
        const nodeName = nodeEl?.querySelector('.node-title')?.textContent || nodeId.substring(0, 8);
        body.appendChild(createGroup(nodeName, items, nodeId));
      });

      panel.appendChild(body);
    },

    async validate() {
      if (this.isValidating || !this.workflowId) return;
      this.isValidating = true;

      try {
        const resp = await fetch(`/api/workflows/${this.workflowId}/validate`, {
          method: 'POST',
        });
        const result = await resp.json();
        this.applyValidationResults(result.results);

        if (result.valid) {
          window.toast?.success('Workflow is valid');
          this.showValidationPanel([]);
        } else {
          const s = result.summary;
          window.toast?.error(
            `Validation: ${s.errors} error${s.errors !== 1 ? 's' : ''}` +
            (s.warnings ? `, ${s.warnings} warning${s.warnings !== 1 ? 's' : ''}` : '')
          );
          this.showValidationPanel(result.results);
        }
      } catch (err) {
        console.error('Validate error:', err);
        window.toast?.error('Network error during validation');
      } finally {
        this.isValidating = false;
      }
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
          this.clearValidationHighlights();
          this.showValidationPanel([]);
          window.toast?.success(`Deployed to ${target} successfully`);
          this.deployStatus = 'success';
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
          // Apply structured validation results if available
          if (result.validationResults?.length) {
            this.applyValidationResults(result.validationResults);
            this.showValidationPanel(result.validationResults);
          } else if (result.validationErrors?.length) {
            // Legacy fallback
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
        document.querySelectorAll(`.edge[data-source-node="${msg.nodeID}"] .edge-line`).forEach(line => {
          line.style.stroke = 'var(--success)';
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
      document.querySelectorAll('.edge-line').forEach(e => e.style.stroke = '');
    },
  }));

  // Canvas context menu for edge actions
  Alpine.data('canvasContextMenu', () => ({
    open: false,
    x: 0,
    y: 0,
    edgeId: null,

    init() {
      this._handler = (e) => {
        this.edgeId = e.detail.edgeId;
        this.x = e.detail.x;
        this.y = e.detail.y;
        this.open = true;
      };
      document.addEventListener('canvas:contextmenu', this._handler);
    },

    destroy() {
      document.removeEventListener('canvas:contextmenu', this._handler);
    },

    deleteEdge() {
      if (this.edgeId && window.commandDispatcher) {
        window.commandDispatcher.dispatch('remove_edge', { edgeId: this.edgeId });
      }
      this.open = false;
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
