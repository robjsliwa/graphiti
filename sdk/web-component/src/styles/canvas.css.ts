/** Shadow DOM styles for <graphiti-canvas>. Exported as a string for injection. */
export const canvasStyles = `
:host {
  display: block;
  width: 100%;
  height: 100%;
  position: relative;
  overflow: hidden;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

:host([theme="dark"]) {
  --gc-bg: #1a1a2e;
  --gc-surface: #16213e;
  --gc-surface-2: #1a1a40;
  --gc-border: #334155;
  --gc-text: #e2e8f0;
  --gc-text-muted: #94a3b8;
  --gc-accent: #6366f1;
  --gc-accent-hover: #818cf8;
  --gc-success: #10b981;
  --gc-warning: #f59e0b;
  --gc-danger: #ef4444;
  --gc-grid-dot: #334155;
  --gc-edge-color: #64748b;
  --gc-node-control: #8b5cf6;
}

:host, :host([theme="light"]) {
  --gc-bg: #f8fafc;
  --gc-surface: #ffffff;
  --gc-surface-2: #f1f5f9;
  --gc-border: #cbd5e1;
  --gc-text: #1e293b;
  --gc-text-muted: #64748b;
  --gc-accent: #6366f1;
  --gc-accent-hover: #4f46e5;
  --gc-success: #10b981;
  --gc-warning: #f59e0b;
  --gc-danger: #ef4444;
  --gc-grid-dot: #d0d0d0;
  --gc-edge-color: #94a3b8;
  --gc-node-control: #8b5cf6;
}

.canvas-root {
  width: 100%;
  height: 100%;
  background: var(--gc-bg);
}

.workflow-canvas {
  width: 100%;
  height: 100%;
  display: block;
  cursor: grab;
  user-select: none;
}
.workflow-canvas.panning {
  cursor: grabbing;
}

.canvas-grid { pointer-events: all; }

/* Node styles */
.node { cursor: pointer; }
.node:hover .node-bg { stroke: var(--gc-accent); }
.node.selected .node-bg { stroke: var(--gc-accent); stroke-width: 2; }
.node-bg { transition: stroke 0.15s; }
.node-header { pointer-events: none; }
.node-icon { font-size: 14px; font-weight: 700; pointer-events: none; }
.node-title { font-size: 12px; font-weight: 500; pointer-events: none; }
.node-attr-label { font-size: 11px; fill: var(--gc-text-muted); pointer-events: none; }

/* Port styles */
.port {
  fill: var(--gc-surface);
  stroke: var(--gc-border);
  stroke-width: 1.5;
  cursor: crosshair;
  transition: stroke 0.15s, stroke-width 0.15s, r 0.15s;
}
.port:hover {
  stroke: var(--gc-accent);
  stroke-width: 2.5;
  r: 8;
}
.port-data { stroke: var(--gc-accent); }
.port-control { stroke: var(--gc-node-control); }
.port-error { stroke: var(--gc-danger); }

.port-label { font-size: 10px; fill: var(--gc-text-muted); pointer-events: none; }
.port-label-input { text-anchor: start; }
.port-label-output { text-anchor: end; }

/* Edge styles */
.edge { cursor: pointer; }
.edge-hit { pointer-events: stroke; fill: none; }
.edge-line { pointer-events: none; transition: stroke 0.15s, stroke-width 0.15s; }
.edge:hover .edge-line { stroke: var(--gc-accent); stroke-width: 3; }
.edge.selected .edge-line {
  stroke: var(--gc-accent);
  stroke-width: 3;
  stroke-dasharray: 8 4;
  animation: edge-march 0.5s linear infinite;
}
@keyframes edge-march {
  to { stroke-dashoffset: -12; }
}

/* Port connection targets */
.port-valid-target {
  stroke: var(--gc-success) !important;
  fill: rgba(16, 185, 129, 0.2);
  stroke-width: 3;
  r: 9;
}
.port-invalid-target {
  stroke: var(--gc-danger) !important;
  fill: rgba(239, 68, 68, 0.2);
}

/* Edge drawing preview */
.edge-preview { pointer-events: none; }

/* Rubber band selection */
.selection-rect { pointer-events: none; }

/* Viewport culling and level-of-detail */
.node.culled { display: none; }
.node.lod-low .node-title,
.node.lod-low .node-attr-label,
.node.lod-low .node-icon,
.node.lod-low .port-label { display: none; }
.node.lod-no-ports .port { display: none; }
.node.lod-no-ports .port-label { display: none; }

/* Execution status overlays */
@keyframes pulse {
  0%, 100% { stroke-opacity: 1; }
  50% { stroke-opacity: 0.4; }
}
.node.running .node-bg { animation: pulse 1.5s ease-in-out infinite; stroke: var(--gc-warning); stroke-width: 2; }
.exec-pending { stroke: var(--gc-text-muted); }
.exec-running { stroke: var(--gc-warning); stroke-width: 2; }
.exec-completed { stroke: var(--gc-success); stroke-width: 2; }
.exec-failed { stroke: var(--gc-danger); stroke-width: 2; }

/* Loading state */
.loading-message {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: var(--gc-text-muted);
  font-size: 14px;
}

/* Error state */
.error-message {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: var(--gc-danger);
  font-size: 14px;
  text-align: center;
}
`;
