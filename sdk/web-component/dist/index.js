// src/graphiti-canvas.ts
import { GraphitiClient } from "@graphiti/client";

// src/renderer/shapes.ts
var SVG_NS = "http://www.w3.org/2000/svg";
function renderRoundedRect(width, height) {
  const rect = document.createElementNS(SVG_NS, "rect");
  rect.setAttribute("width", String(width));
  rect.setAttribute("height", String(height));
  rect.setAttribute("rx", "8");
  rect.setAttribute("ry", "8");
  rect.setAttribute("class", "node-bg");
  rect.setAttribute("fill", "var(--gc-surface)");
  rect.setAttribute("stroke", "var(--gc-border)");
  rect.setAttribute("stroke-width", "1.5");
  return rect;
}
function renderPill(width, height) {
  const rect = document.createElementNS(SVG_NS, "rect");
  rect.setAttribute("width", String(width));
  rect.setAttribute("height", String(height));
  rect.setAttribute("rx", String(height / 2));
  rect.setAttribute("ry", String(height / 2));
  rect.setAttribute("class", "node-bg");
  rect.setAttribute("fill", "var(--gc-surface)");
  rect.setAttribute("stroke", "var(--gc-border)");
  rect.setAttribute("stroke-width", "1.5");
  return rect;
}
function renderDiamond(width, height) {
  const poly = document.createElementNS(SVG_NS, "polygon");
  const cx = width / 2;
  const cy = height / 2;
  poly.setAttribute("points", `${cx},0 ${width},${cy} ${cx},${height} 0,${cy}`);
  poly.setAttribute("class", "node-bg");
  poly.setAttribute("fill", "var(--gc-surface)");
  poly.setAttribute("stroke", "var(--gc-border)");
  poly.setAttribute("stroke-width", "1.5");
  return poly;
}
function renderHexagon(width, height) {
  const poly = document.createElementNS(SVG_NS, "polygon");
  const offset = 20;
  poly.setAttribute(
    "points",
    `${offset},0 ${width - offset},0 ${width},${height / 2} ${width - offset},${height} ${offset},${height} 0,${height / 2}`
  );
  poly.setAttribute("class", "node-bg");
  poly.setAttribute("fill", "var(--gc-surface)");
  poly.setAttribute("stroke", "var(--gc-border)");
  poly.setAttribute("stroke-width", "1.5");
  return poly;
}
function renderSubWorkflow(width, height) {
  const g = document.createElementNS(SVG_NS, "g");
  const outer = renderRoundedRect(width, height);
  g.appendChild(outer);
  const inner = document.createElementNS(SVG_NS, "rect");
  inner.setAttribute("x", "3");
  inner.setAttribute("y", "3");
  inner.setAttribute("width", String(width - 6));
  inner.setAttribute("height", String(height - 6));
  inner.setAttribute("rx", "6");
  inner.setAttribute("ry", "6");
  inner.setAttribute("fill", "none");
  inner.setAttribute("stroke", "var(--gc-border)");
  inner.setAttribute("stroke-width", "1");
  g.appendChild(inner);
  return g;
}
function renderShape(type, width, height) {
  switch (type) {
    case "diamond":
      return renderDiamond(width, height);
    case "hexagon":
      return renderHexagon(width, height);
    case "pill":
      return renderPill(width, height);
    case "sub-workflow":
      return renderSubWorkflow(width, height);
    case "rounded-rect":
    case "custom":
    default:
      return renderRoundedRect(width, height);
  }
}

// src/renderer/layout.ts
var INPUT_LABEL_COL_WIDTH = 56;
var OUTPUT_LABEL_COL_WIDTH = 56;
var COL_GAP = 8;
var MIN_CENTER_WIDTH = 100;
var HEADER_HEIGHT = 36;
var BOTTOM_PADDING = 12;
var PORT_SPACING = 24;
function hasInputLabels(def) {
  return (def.inputs ?? []).some((p) => p.label);
}
function hasOutputLabels(def) {
  return (def.outputs ?? []).some((p) => p.label);
}
function bodyAttrCount(def) {
  return (def.attributes ?? []).filter(
    (a) => a.display === "node-body" || a.display === "both"
  ).length;
}
function hasBodyAttrs(def) {
  return bodyAttrCount(def) > 0;
}
function calcNodeWidth(def) {
  const baseWidth = def?.shape?.width || 200;
  if (!def) return baseWidth;
  const hasIn = hasInputLabels(def);
  const hasOut = hasOutputLabels(def);
  const hasBody = hasBodyAttrs(def);
  if (hasBody && (hasIn || hasOut)) {
    let needed = 12 + MIN_CENTER_WIDTH + 12;
    if (hasIn) needed += INPUT_LABEL_COL_WIDTH + COL_GAP;
    if (hasOut) needed += OUTPUT_LABEL_COL_WIDTH + COL_GAP;
    return Math.max(baseWidth, needed);
  }
  return baseWidth;
}
function calcNodeHeight(def) {
  if (!def) return 80;
  const inputs = (def.inputs ?? []).length;
  const outputs = (def.outputs ?? []).length;
  const maxPorts = Math.max(inputs, outputs);
  const bac = bodyAttrCount(def);
  const bodyAttrHeight = bac * 20;
  const portsHeight = maxPorts * PORT_SPACING;
  const contentHeight = Math.max(bodyAttrHeight, portsHeight);
  return Math.max(60, HEADER_HEIGHT + contentHeight + BOTTOM_PADDING);
}
function bodyAttrX(def) {
  if (hasInputLabels(def) && hasBodyAttrs(def)) {
    return 12 + INPUT_LABEL_COL_WIDTH + COL_GAP;
  }
  return 12;
}
function portY(index, count, nodeHeight) {
  if (count === 0) return HEADER_HEIGHT;
  const contentHeight = nodeHeight - HEADER_HEIGHT - BOTTOM_PADDING;
  const startY = HEADER_HEIGHT + (contentHeight - (count - 1) * PORT_SPACING) / 2;
  return startY + index * PORT_SPACING;
}
function findPortY(def, portId, isOutput) {
  if (!def) return HEADER_HEIGHT;
  const ports = isOutput ? def.outputs ?? [] : def.inputs ?? [];
  const h = calcNodeHeight(def);
  const idx = ports.findIndex((p) => p.id === portId);
  if (idx < 0) return h / 2;
  return portY(idx, ports.length, h);
}
function getBodyAttrs(def, attributes) {
  const result = [];
  const x = bodyAttrX(def);
  let idx = 0;
  for (const attr of def.attributes ?? []) {
    if (attr.display === "node-body" || attr.display === "both") {
      const val = attributes?.[attr.id];
      result.push({
        label: attr.label,
        value: val != null ? String(val) : "",
        x,
        y: 52 + idx * 20
      });
      idx++;
    }
  }
  return result;
}

// src/renderer/node-renderer.ts
var SVG_NS2 = "http://www.w3.org/2000/svg";
function svgEl(tag, attrs, text) {
  const e = document.createElementNS(SVG_NS2, tag);
  if (attrs) {
    for (const [k, v] of Object.entries(attrs)) {
      if (k.startsWith("data-")) {
        e.dataset[k.slice(5).replace(/-([a-z])/g, (_, c) => c.toUpperCase())] = v;
      } else {
        e.setAttribute(k, v);
      }
    }
  }
  if (text) e.textContent = text;
  return e;
}
function renderNode(node) {
  const def = node.definition;
  const w = calcNodeWidth(def);
  const h = calcNodeHeight(def);
  const hdrBg = def?.shape?.headerBackground || "var(--gc-surface-2)";
  const hdrColor = def?.shape?.headerColor || "var(--gc-text)";
  const g = svgEl("g", {
    class: "node",
    "data-node-id": node.id,
    "data-definition-id": node.definitionId,
    transform: `translate(${node.x}, ${node.y})`
  });
  const shapeType = def?.shape?.type || "rounded-rect";
  const shape = renderShape(shapeType, w, h);
  g.appendChild(shape);
  g.appendChild(
    svgEl("rect", {
      width: String(w),
      height: "32",
      rx: "6",
      ry: "6",
      class: "node-header",
      fill: hdrBg
    })
  );
  g.appendChild(
    svgEl("rect", { x: "0", y: "16", width: String(w), height: "16", fill: hdrBg })
  );
  g.appendChild(
    svgEl("text", { x: "12", y: "22", class: "node-icon", fill: hdrColor }, def?.icon || "?")
  );
  g.appendChild(
    svgEl("text", { x: "32", y: "22", class: "node-title", fill: hdrColor }, node.label)
  );
  if (def) {
    const attrs = getBodyAttrs(def, node.attributes);
    for (const attr of attrs) {
      g.appendChild(
        svgEl(
          "text",
          { x: String(attr.x), y: String(attr.y), class: "node-attr-label" },
          `${attr.label}: ${attr.value}`
        )
      );
    }
  }
  if (def) {
    const inputs = def.inputs ?? [];
    const outputs = def.outputs ?? [];
    const addPort = (port, isInput, idx, count) => {
      const px = isInput ? 0 : w;
      const py = portY(idx, count, h);
      const circle = svgEl("circle", {
        cx: String(px),
        cy: String(py),
        r: "6",
        class: `port port-${port.type} port-${isInput ? "input" : "output"}`,
        "data-port-id": port.id,
        "data-port-type": port.type,
        "data-is-input": String(isInput)
      });
      const title = svgEl("title", {}, port.label || port.id);
      circle.appendChild(title);
      g.appendChild(circle);
      if (port.label) {
        const lx = isInput ? 12 : w - 12;
        const anchor = isInput ? "start" : "end";
        g.appendChild(
          svgEl(
            "text",
            {
              x: String(lx),
              y: String(py + 4),
              class: `port-label ${isInput ? "port-label-input" : "port-label-output"}`,
              "text-anchor": anchor
            },
            port.label
          )
        );
      }
    };
    inputs.forEach((p, i) => addPort(p, true, i, inputs.length));
    outputs.forEach((p, i) => addPort(p, false, i, outputs.length));
  }
  return g;
}

// src/renderer/edge-renderer.ts
var SVG_NS3 = "http://www.w3.org/2000/svg";
function calcEdgePath(edge, nodes) {
  const src = nodes.find((n) => n.id === edge.sourceNodeId);
  const tgt = nodes.find((n) => n.id === edge.targetNodeId);
  if (!src || !tgt) return "";
  const sw = calcNodeWidth(src.definition);
  const srcPortY = findPortY(src.definition, edge.sourcePortId, true);
  const tgtPortY = findPortY(tgt.definition, edge.targetPortId, false);
  const x1 = src.x + sw;
  const y1 = src.y + srcPortY;
  const x2 = tgt.x;
  const y2 = tgt.y + tgtPortY;
  const cp = Math.max(50, (x2 - x1) * 0.5);
  return `M ${x1} ${y1} C ${x1 + cp} ${y1}, ${x2 - cp} ${y2}, ${x2} ${y2}`;
}
function renderEdge(edge, nodes) {
  const g = document.createElementNS(SVG_NS3, "g");
  g.setAttribute("class", "edge");
  g.dataset.edgeId = edge.id;
  g.dataset.sourceNode = edge.sourceNodeId;
  g.dataset.targetNode = edge.targetNodeId;
  g.dataset.sourcePort = edge.sourcePortId;
  g.dataset.targetPort = edge.targetPortId;
  const d = calcEdgePath(edge, nodes);
  const hit = document.createElementNS(SVG_NS3, "path");
  hit.setAttribute("class", "edge-hit");
  hit.setAttribute("d", d);
  hit.setAttribute("fill", "none");
  hit.setAttribute("stroke", "transparent");
  hit.setAttribute("stroke-width", "16");
  hit.setAttribute("pointer-events", "stroke");
  const line = document.createElementNS(SVG_NS3, "path");
  line.setAttribute("class", "edge-line");
  line.setAttribute("d", d);
  line.setAttribute("fill", "none");
  line.setAttribute("stroke", "var(--gc-edge-color, #94a3b8)");
  line.setAttribute("stroke-width", "2");
  line.setAttribute("marker-end", "url(#arrowhead)");
  line.setAttribute("pointer-events", "none");
  g.appendChild(hit);
  g.appendChild(line);
  return g;
}

// src/renderer/grid.ts
var SVG_NS4 = "http://www.w3.org/2000/svg";
function createGridPattern() {
  const pattern = document.createElementNS(SVG_NS4, "pattern");
  pattern.setAttribute("id", "gc-dot-grid");
  pattern.setAttribute("width", "24");
  pattern.setAttribute("height", "24");
  pattern.setAttribute("patternUnits", "userSpaceOnUse");
  const dot = document.createElementNS(SVG_NS4, "circle");
  dot.setAttribute("cx", "12");
  dot.setAttribute("cy", "12");
  dot.setAttribute("r", "1");
  dot.setAttribute("fill", "var(--gc-grid-dot, #d0d0d0)");
  pattern.appendChild(dot);
  return pattern;
}
function createArrowheadMarker() {
  const marker = document.createElementNS(SVG_NS4, "marker");
  marker.setAttribute("id", "arrowhead");
  marker.setAttribute("markerWidth", "10");
  marker.setAttribute("markerHeight", "7");
  marker.setAttribute("refX", "10");
  marker.setAttribute("refY", "3.5");
  marker.setAttribute("orient", "auto");
  const poly = document.createElementNS(SVG_NS4, "polygon");
  poly.setAttribute("points", "0 0, 10 3.5, 0 7");
  poly.setAttribute("fill", "var(--gc-edge-color, #94a3b8)");
  marker.appendChild(poly);
  return marker;
}
function updateGridForTransform(pattern, scale, panX, panY) {
  const size = 24 * scale;
  pattern.setAttribute("width", String(size));
  pattern.setAttribute("height", String(size));
  pattern.setAttribute(
    "patternTransform",
    `translate(${panX % size}, ${panY % size})`
  );
  const dot = pattern.querySelector("circle");
  if (dot) {
    dot.setAttribute("cx", String(size / 2));
    dot.setAttribute("cy", String(size / 2));
    dot.setAttribute("r", String(Math.max(0.5, scale)));
  }
}

// src/renderer/viewport.ts
var Viewport = class {
  scale = 1;
  panX = 0;
  panY = 0;
  minScale = 0.25;
  maxScale = 2;
  gridSize;
  constructor(gridSize) {
    this.gridSize = gridSize ?? 24;
  }
  screenToCanvas(screenX, screenY, svgRect) {
    return {
      x: (screenX - svgRect.left - this.panX) / this.scale,
      y: (screenY - svgRect.top - this.panY) / this.scale
    };
  }
  canvasToScreen(canvasX, canvasY, svgRect) {
    return {
      x: canvasX * this.scale + this.panX + svgRect.left,
      y: canvasY * this.scale + this.panY + svgRect.top
    };
  }
  snapToGrid(x, y) {
    return {
      x: Math.round(x / this.gridSize) * this.gridSize,
      y: Math.round(y / this.gridSize) * this.gridSize
    };
  }
  zoom(delta, centerX, centerY, svgRect) {
    const oldScale = this.scale;
    this.scale *= delta > 0 ? 1.1 : 0.9;
    this.scale = Math.max(this.minScale, Math.min(this.maxScale, this.scale));
    const cx = centerX - svgRect.left;
    const cy = centerY - svgRect.top;
    this.panX = cx - (cx - this.panX) * (this.scale / oldScale);
    this.panY = cy - (cy - this.panY) * (this.scale / oldScale);
  }
  pan(dx, dy) {
    this.panX += dx;
    this.panY += dy;
  }
  setViewport(x, y, zoom) {
    this.panX = x;
    this.panY = y;
    this.scale = Math.max(this.minScale, Math.min(this.maxScale, zoom));
  }
  getTransform() {
    return `translate(${this.panX}, ${this.panY}) scale(${this.scale})`;
  }
};

// src/renderer/svg-renderer.ts
var SVG_NS5 = "http://www.w3.org/2000/svg";
var SVGRenderer = class {
  svg;
  viewport;
  contentGroup;
  nodeLayer;
  edgeLayer;
  interactionLayer;
  gridPattern;
  isPanning = false;
  lastX = 0;
  lastY = 0;
  cullPending = false;
  nodeSizeCache = /* @__PURE__ */ new Map();
  currentNodes = [];
  boundMouseMove;
  boundMouseUp;
  boundWheel;
  boundMouseDown;
  constructor(options) {
    this.viewport = new Viewport(options?.gridSize);
    this.svg = document.createElementNS(SVG_NS5, "svg");
    this.svg.setAttribute("class", "workflow-canvas");
    this.svg.setAttribute("xmlns", SVG_NS5);
    this.svg.setAttribute("width", "100%");
    this.svg.setAttribute("height", "100%");
    this.buildSVGStructure();
    this.boundMouseMove = this.onMouseMove.bind(this);
    this.boundMouseUp = this.onMouseUp.bind(this);
    this.boundWheel = this.onWheel.bind(this);
    this.boundMouseDown = this.onMouseDown.bind(this);
    this.bindEvents();
  }
  buildSVGStructure() {
    const defs = document.createElementNS(SVG_NS5, "defs");
    this.gridPattern = createGridPattern();
    defs.appendChild(this.gridPattern);
    defs.appendChild(createArrowheadMarker());
    this.svg.appendChild(defs);
    const gridRect = document.createElementNS(SVG_NS5, "rect");
    gridRect.setAttribute("width", "100%");
    gridRect.setAttribute("height", "100%");
    gridRect.setAttribute("fill", "url(#gc-dot-grid)");
    gridRect.setAttribute("class", "canvas-grid");
    this.svg.appendChild(gridRect);
    this.contentGroup = document.createElementNS(SVG_NS5, "g");
    this.contentGroup.setAttribute("class", "canvas-content");
    this.contentGroup.setAttribute("transform", "translate(0, 0) scale(1)");
    this.edgeLayer = document.createElementNS(SVG_NS5, "g");
    this.edgeLayer.setAttribute("class", "edge-layer");
    this.nodeLayer = document.createElementNS(SVG_NS5, "g");
    this.nodeLayer.setAttribute("class", "node-layer");
    this.interactionLayer = document.createElementNS(SVG_NS5, "g");
    this.interactionLayer.setAttribute("class", "interaction-layer");
    this.contentGroup.appendChild(this.edgeLayer);
    this.contentGroup.appendChild(this.nodeLayer);
    this.contentGroup.appendChild(this.interactionLayer);
    this.svg.appendChild(this.contentGroup);
  }
  /** Render the full workflow state. Replaces all existing nodes/edges. */
  renderState(state) {
    this.nodeSizeCache.clear();
    this.currentNodes = state.nodes ?? [];
    const stateNodeIds = new Set(this.currentNodes.map((n) => n.id));
    const stateEdgeIds = new Set((state.edges ?? []).map((e) => e.id));
    this.nodeLayer.querySelectorAll(".node").forEach((el) => {
      if (!stateNodeIds.has(el.dataset.nodeId)) el.remove();
    });
    this.edgeLayer.querySelectorAll(".edge").forEach((el) => {
      if (!stateEdgeIds.has(el.dataset.edgeId)) el.remove();
    });
    for (const node of this.currentNodes) {
      const existing = this.nodeLayer.querySelector(
        `[data-node-id="${node.id}"]`
      );
      if (existing) {
        existing.setAttribute("transform", `translate(${node.x}, ${node.y})`);
        const titleEl = existing.querySelector(".node-title");
        if (titleEl) titleEl.textContent = node.label;
      } else {
        this.nodeLayer.appendChild(renderNode(node));
      }
    }
    for (const edge of state.edges ?? []) {
      const existing = this.edgeLayer.querySelector(
        `[data-edge-id="${edge.id}"]`
      );
      if (existing) {
        const newD = calcEdgePath(edge, this.currentNodes);
        existing.querySelectorAll("path").forEach((p) => p.setAttribute("d", newD));
      } else {
        this.edgeLayer.appendChild(renderEdge(edge, this.currentNodes));
      }
    }
  }
  /** Add a single node to the canvas. */
  addNode(node) {
    const g = renderNode(node);
    this.nodeLayer.appendChild(g);
    this.currentNodes.push(node);
    return g;
  }
  /** Remove a node by ID. */
  removeNode(nodeId) {
    this.nodeLayer.querySelector(`[data-node-id="${nodeId}"]`)?.remove();
    this.currentNodes = this.currentNodes.filter((n) => n.id !== nodeId);
    this.nodeSizeCache.delete(nodeId);
  }
  /** Add an edge to the canvas. */
  addEdge(edge) {
    const g = renderEdge(edge, this.currentNodes);
    this.edgeLayer.appendChild(g);
    return g;
  }
  /** Remove an edge by ID. */
  removeEdge(edgeId) {
    this.edgeLayer.querySelector(`[data-edge-id="${edgeId}"]`)?.remove();
  }
  /** Update a node's position without re-rendering. */
  updateNodePosition(nodeId, x, y) {
    const el = this.nodeLayer.querySelector(`[data-node-id="${nodeId}"]`);
    if (el) {
      el.setAttribute("transform", `translate(${x}, ${y})`);
      this.updateEdgesForNode(nodeId);
    }
  }
  /** Update edges connected to a given node (during drag). */
  updateEdgesForNode(nodeId) {
    this.edgeLayer.querySelectorAll(".edge").forEach((el) => {
      const g = el;
      if (g.dataset.sourceNode !== nodeId && g.dataset.targetNode !== nodeId) return;
      const srcEl = this.nodeLayer.querySelector(
        `[data-node-id="${g.dataset.sourceNode}"]`
      );
      const tgtEl = this.nodeLayer.querySelector(
        `[data-node-id="${g.dataset.targetNode}"]`
      );
      if (!srcEl || !tgtEl) return;
      const sp = this.getNodePosition(srcEl);
      const tp = this.getNodePosition(tgtEl);
      const sw = parseInt(
        srcEl.querySelector(".node-header")?.getAttribute("width") || "200"
      );
      const srcPortY = this.domPortY(srcEl, g.dataset.sourcePort);
      const tgtPortY = this.domPortY(tgtEl, g.dataset.targetPort);
      const x1 = sp.x + sw;
      const y1 = sp.y + srcPortY;
      const x2 = tp.x;
      const y2 = tp.y + tgtPortY;
      const cp = Math.max(50, (x2 - x1) * 0.5);
      const d = `M ${x1} ${y1} C ${x1 + cp} ${y1}, ${x2 - cp} ${y2}, ${x2} ${y2}`;
      el.querySelectorAll("path").forEach((p) => p.setAttribute("d", d));
    });
  }
  /** Viewport controls. */
  setViewport(x, y, zoom) {
    this.viewport.setViewport(x, y, zoom);
    this.applyTransform();
  }
  /** Zoom to fit all nodes in view. */
  zoomToFit() {
    const nodes = this.nodeLayer.querySelectorAll(".node");
    if (nodes.length === 0) return;
    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;
    nodes.forEach((n) => {
      const pos = this.getNodePosition(n);
      minX = Math.min(minX, pos.x);
      minY = Math.min(minY, pos.y);
      const nodeW = parseInt(
        n.querySelector(".node-header")?.getAttribute("width") || "200"
      );
      const nodeBg = n.querySelector(".node-bg");
      const nodeH = parseInt(nodeBg?.getAttribute("height") || "80");
      maxX = Math.max(maxX, pos.x + nodeW);
      maxY = Math.max(maxY, pos.y + nodeH);
    });
    if (!isFinite(minX)) return;
    const pad = 48;
    const rect = this.svg.getBoundingClientRect();
    const cw = maxX - minX + pad * 2;
    const ch = maxY - minY + pad * 2;
    this.viewport.scale = Math.min(rect.width / cw, rect.height / ch, 1);
    this.viewport.panX = (rect.width - cw * this.viewport.scale) / 2 - minX * this.viewport.scale + pad * this.viewport.scale;
    this.viewport.panY = (rect.height - ch * this.viewport.scale) / 2 - minY * this.viewport.scale + pad * this.viewport.scale;
    this.applyTransform();
  }
  /** Get the node layer (for interaction handlers to query). */
  getNodeLayer() {
    return this.nodeLayer;
  }
  /** Get the edge layer. */
  getEdgeLayer() {
    return this.edgeLayer;
  }
  /** Get the interaction layer (for previews/selection rect). */
  getInteractionLayer() {
    return this.interactionLayer;
  }
  /** Parse a node's position from its transform attribute. */
  getNodePosition(el) {
    const t = el.getAttribute("transform");
    const m = t && t.match(/translate\(([^,]+),\s*([^)]+)\)/);
    return m ? { x: parseFloat(m[1]), y: parseFloat(m[2]) } : { x: 0, y: 0 };
  }
  /** Start a pan gesture (called by interaction handlers). */
  startPan(e) {
    this.isPanning = true;
    this.lastX = e.clientX;
    this.lastY = e.clientY;
    this.svg.classList.add("panning");
    document.addEventListener("mousemove", this.boundMouseMove);
    document.addEventListener("mouseup", this.boundMouseUp);
  }
  /** Tear down event listeners. */
  destroy() {
    this.svg.removeEventListener("wheel", this.boundWheel);
    this.svg.removeEventListener("mousedown", this.boundMouseDown);
    document.removeEventListener("mousemove", this.boundMouseMove);
    document.removeEventListener("mouseup", this.boundMouseUp);
  }
  // --- Private helpers ---
  domPortY(nodeEl, portId) {
    const port = nodeEl.querySelector(`[data-port-id="${portId}"]`);
    return port ? parseFloat(port.getAttribute("cy") || "36") : 36;
  }
  applyTransform() {
    this.contentGroup.setAttribute("transform", this.viewport.getTransform());
    updateGridForTransform(
      this.gridPattern,
      this.viewport.scale,
      this.viewport.panX,
      this.viewport.panY
    );
    this.applyCulling();
  }
  applyCulling() {
    if (this.cullPending) return;
    this.cullPending = true;
    requestAnimationFrame(() => {
      this.cullPending = false;
      const svgRect = this.svg.getBoundingClientRect();
      if (svgRect.width === 0) return;
      const topLeft = this.viewport.screenToCanvas(svgRect.left, svgRect.top, svgRect);
      const bottomRight = this.viewport.screenToCanvas(
        svgRect.right,
        svgRect.bottom,
        svgRect
      );
      const vp = { x1: topLeft.x, y1: topLeft.y, x2: bottomRight.x, y2: bottomRight.y };
      const lowDetail = this.viewport.scale < 0.5;
      const hidePorts = this.viewport.scale < 0.3;
      this.nodeLayer.querySelectorAll(".node").forEach((el) => {
        const svgEl2 = el;
        const pos = this.getNodePosition(svgEl2);
        const id = svgEl2.dataset.nodeId;
        let size = this.nodeSizeCache.get(id);
        if (!size) {
          const w = parseInt(
            el.querySelector(".node-header")?.getAttribute("width") || "200"
          );
          const bg = el.querySelector(".node-bg");
          const h = parseInt(bg?.getAttribute("height") || "80");
          size = { w, h };
          this.nodeSizeCache.set(id, size);
        }
        const pad = 50;
        const inView = pos.x + size.w + pad >= vp.x1 && pos.x - pad <= vp.x2 && pos.y + size.h + pad >= vp.y1 && pos.y - pad <= vp.y2;
        el.classList.toggle("culled", !inView);
        if (inView) {
          el.classList.toggle("lod-low", lowDetail);
          el.classList.toggle("lod-no-ports", hidePorts);
        }
      });
    });
  }
  bindEvents() {
    this.svg.addEventListener("wheel", this.boundWheel, { passive: false });
    this.svg.addEventListener("mousedown", this.boundMouseDown);
  }
  onWheel(e) {
    e.preventDefault();
    const rect = this.svg.getBoundingClientRect();
    this.viewport.zoom(-e.deltaY, e.clientX, e.clientY, rect);
    this.applyTransform();
  }
  onMouseDown(e) {
    const spaceHeld = typeof e.getModifierState === "function" && e.getModifierState("Space");
    if (e.button === 1 || e.button === 0 && spaceHeld) {
      this.startPan(e);
      e.preventDefault();
    }
  }
  onMouseMove(e) {
    if (!this.isPanning) return;
    this.viewport.pan(e.clientX - this.lastX, e.clientY - this.lastY);
    this.lastX = e.clientX;
    this.lastY = e.clientY;
    this.applyTransform();
  }
  onMouseUp() {
    if (this.isPanning) {
      this.isPanning = false;
      this.svg.classList.remove("panning");
      document.removeEventListener("mousemove", this.boundMouseMove);
      document.removeEventListener("mouseup", this.boundMouseUp);
    }
  }
};

// src/interactions/selection.ts
var SelectionManager = class {
  renderer;
  callbacks;
  selectedNodes = /* @__PURE__ */ new Set();
  selectedEdges = /* @__PURE__ */ new Set();
  dragging = false;
  dragStartX = 0;
  dragStartY = 0;
  nodeOrigPositions = /* @__PURE__ */ new Map();
  selRect = null;
  selStartX = 0;
  selStartY = 0;
  boundMouseDown;
  boundMouseMove;
  boundMouseUp;
  boundKeyDown;
  constructor(renderer, callbacks) {
    this.renderer = renderer;
    this.callbacks = callbacks;
    this.boundMouseDown = this.onMouseDown.bind(this);
    this.boundMouseMove = this.onMouseMove.bind(this);
    this.boundMouseUp = this.onMouseUp.bind(this);
    this.boundKeyDown = this.onKeyDown.bind(this);
  }
  get selectedIds() {
    return Array.from(this.selectedNodes);
  }
  get selectedEdgeIds() {
    return Array.from(this.selectedEdges);
  }
  attach(root) {
    this.renderer.svg.addEventListener("mousedown", this.boundMouseDown);
    root.addEventListener("mousemove", this.boundMouseMove);
    root.addEventListener("mouseup", this.boundMouseUp);
    root.addEventListener("keydown", this.boundKeyDown);
  }
  detach(root) {
    this.renderer.svg.removeEventListener("mousedown", this.boundMouseDown);
    root.removeEventListener("mousemove", this.boundMouseMove);
    root.removeEventListener("mouseup", this.boundMouseUp);
    root.removeEventListener("keydown", this.boundKeyDown);
  }
  select(ids) {
    this.clearSelection();
    for (const id of ids) this.selectNode(id);
  }
  clearSelection() {
    this.selectedNodes.clear();
    this.selectedEdges.clear();
    this.renderer.svg.querySelectorAll(".selected").forEach((el) => el.classList.remove("selected"));
    this.callbacks.onDeselect();
  }
  selectNode(id) {
    this.selectedNodes.add(id);
    this.renderer.svg.querySelector(`[data-node-id="${id}"]`)?.classList.add("selected");
    this.callbacks.onSelect(this.selectedIds);
  }
  selectEdge(id) {
    this.selectedEdges.add(id);
    this.renderer.svg.querySelector(`[data-edge-id="${id}"]`)?.classList.add("selected");
  }
  toggleNode(id) {
    if (this.selectedNodes.has(id)) {
      this.selectedNodes.delete(id);
      this.renderer.svg.querySelector(`[data-node-id="${id}"]`)?.classList.remove("selected");
      this.callbacks.onSelect(this.selectedIds);
    } else {
      this.selectNode(id);
    }
  }
  onMouseDown(e) {
    const target = e.target;
    if (target.closest(".port")) return;
    if (e.button !== 0 || e.getModifierState("Space")) return;
    const node = target.closest(".node");
    const edge = target.closest(".edge");
    if (node) {
      const id = node.dataset.nodeId;
      if (e.ctrlKey || e.metaKey) {
        this.toggleNode(id);
      } else if (!this.selectedNodes.has(id)) {
        this.clearSelection();
        this.selectNode(id);
      }
      this.startDrag(e);
      e.stopPropagation();
    } else if (edge) {
      this.clearSelection();
      this.selectEdge(edge.dataset.edgeId);
      e.stopPropagation();
    } else {
      if (e.shiftKey) {
        if (!e.ctrlKey && !e.metaKey) this.clearSelection();
        this.startRubberBand(e);
      } else {
        this.clearSelection();
        this.renderer.startPan(e);
      }
    }
  }
  onMouseMove(e) {
    if (this.dragging) this.onDrag(e);
    if (this.selRect) this.onRubberBand(e);
  }
  onMouseUp(e) {
    if (this.dragging) this.endDrag(e);
    if (this.selRect) this.endRubberBand(e);
  }
  onKeyDown(e) {
    const tag = e.target.tagName;
    if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
    if (e.key === "Delete" || e.key === "Backspace") {
      e.preventDefault();
      this.deleteSelection();
    } else if (e.key === "Escape") {
      this.clearSelection();
    } else if ((e.ctrlKey || e.metaKey) && e.key === "a") {
      e.preventDefault();
      this.selectAll();
    }
  }
  // --- Drag ---
  startDrag(e) {
    this.dragging = true;
    this.dragStartX = e.clientX;
    this.dragStartY = e.clientY;
    this.nodeOrigPositions.clear();
    const clickedNode = e.target.closest(".node");
    const clickedId = clickedNode?.dataset.nodeId;
    const ids = this.selectedNodes.size > 0 && clickedId && this.selectedNodes.has(clickedId) ? this.selectedNodes : /* @__PURE__ */ new Set([clickedId]);
    for (const id of ids) {
      const el = this.renderer.svg.querySelector(`[data-node-id="${id}"]`);
      if (el) {
        this.nodeOrigPositions.set(id, this.renderer.getNodePosition(el));
      }
    }
  }
  onDrag(e) {
    const dx = (e.clientX - this.dragStartX) / this.renderer.viewport.scale;
    const dy = (e.clientY - this.dragStartY) / this.renderer.viewport.scale;
    for (const [id, orig] of this.nodeOrigPositions) {
      const el = this.renderer.svg.querySelector(`[data-node-id="${id}"]`);
      if (!el) continue;
      el.setAttribute("transform", `translate(${orig.x + dx}, ${orig.y + dy})`);
      this.renderer.updateEdgesForNode(id);
    }
  }
  endDrag(e) {
    this.dragging = false;
    const dx = (e.clientX - this.dragStartX) / this.renderer.viewport.scale;
    const dy = (e.clientY - this.dragStartY) / this.renderer.viewport.scale;
    if (Math.abs(dx) < 2 && Math.abs(dy) < 2) return;
    if (this.nodeOrigPositions.size > 1) {
      const moves = [];
      for (const [id, orig] of this.nodeOrigPositions) {
        const snapped = this.renderer.viewport.snapToGrid(orig.x + dx, orig.y + dy);
        moves.push({ nodeId: id, x: snapped.x, y: snapped.y });
      }
      this.callbacks.onMultiDragEnd(moves);
    } else {
      const [id, orig] = [...this.nodeOrigPositions.entries()][0];
      const snapped = this.renderer.viewport.snapToGrid(orig.x + dx, orig.y + dy);
      this.callbacks.onDragEnd(id, snapped.x, snapped.y);
    }
  }
  // --- Rubber band ---
  startRubberBand(e) {
    const rect = this.renderer.svg.getBoundingClientRect();
    const pos = this.renderer.viewport.screenToCanvas(e.clientX, e.clientY, rect);
    this.selStartX = pos.x;
    this.selStartY = pos.y;
    this.selRect = document.createElementNS(
      "http://www.w3.org/2000/svg",
      "rect"
    );
    this.selRect.setAttribute("class", "selection-rect");
    this.selRect.setAttribute("fill", "var(--gc-accent)");
    this.selRect.setAttribute("fill-opacity", "0.1");
    this.selRect.setAttribute("stroke", "var(--gc-accent)");
    this.selRect.setAttribute("stroke-width", "1");
    this.selRect.setAttribute("stroke-dasharray", "4 2");
    this.renderer.getInteractionLayer().appendChild(this.selRect);
  }
  onRubberBand(e) {
    if (!this.selRect) return;
    const rect = this.renderer.svg.getBoundingClientRect();
    const pos = this.renderer.viewport.screenToCanvas(e.clientX, e.clientY, rect);
    const x = Math.min(this.selStartX, pos.x);
    const y = Math.min(this.selStartY, pos.y);
    this.selRect.setAttribute("x", String(x));
    this.selRect.setAttribute("y", String(y));
    this.selRect.setAttribute("width", String(Math.abs(pos.x - this.selStartX)));
    this.selRect.setAttribute("height", String(Math.abs(pos.y - this.selStartY)));
  }
  endRubberBand(e) {
    const rect = this.renderer.svg.getBoundingClientRect();
    const pos = this.renderer.viewport.screenToCanvas(e.clientX, e.clientY, rect);
    const x1 = Math.min(this.selStartX, pos.x);
    const y1 = Math.min(this.selStartY, pos.y);
    const x2 = Math.max(this.selStartX, pos.x);
    const y2 = Math.max(this.selStartY, pos.y);
    if (this.selRect) {
      this.selRect.remove();
      this.selRect = null;
    }
    if (Math.abs(x2 - x1) < 5 && Math.abs(y2 - y1) < 5) return;
    this.renderer.svg.querySelectorAll(".node").forEach((el) => {
      const p = this.renderer.getNodePosition(el);
      if (p.x >= x1 && p.y >= y1 && p.x <= x2 && p.y <= y2) {
        this.selectNode(el.dataset.nodeId);
      }
    });
  }
  deleteSelection() {
    if (this.selectedNodes.size === 0 && this.selectedEdges.size === 0) return;
    const edgeIds = [...this.selectedEdges];
    const nodeIds = [...this.selectedNodes];
    this.clearSelection();
    if (edgeIds.length > 0) this.callbacks.onDeleteEdges(edgeIds);
    if (nodeIds.length > 0) this.callbacks.onDeleteNodes(nodeIds);
  }
  selectAll() {
    this.renderer.svg.querySelectorAll(".node").forEach((el) => {
      this.selectedNodes.add(el.dataset.nodeId);
      el.classList.add("selected");
    });
    this.callbacks.onSelect(this.selectedIds);
  }
};

// src/interactions/connection.ts
var ConnectionManager = class {
  renderer;
  callbacks;
  drawing = false;
  preview = null;
  srcNodeId = "";
  srcPortId = "";
  srcPortType = "";
  startX = 0;
  startY = 0;
  boundMouseDown;
  boundMouseMove;
  boundMouseUp;
  constructor(renderer, callbacks) {
    this.renderer = renderer;
    this.callbacks = callbacks;
    this.boundMouseDown = this.onMouseDown.bind(this);
    this.boundMouseMove = this.onMouseMove.bind(this);
    this.boundMouseUp = this.onMouseUp.bind(this);
  }
  attach() {
    const svg = this.renderer.svg;
    svg.addEventListener("mousedown", this.boundMouseDown);
    svg.addEventListener("mousemove", this.boundMouseMove);
    svg.addEventListener("mouseup", this.boundMouseUp);
  }
  detach() {
    const svg = this.renderer.svg;
    svg.removeEventListener("mousedown", this.boundMouseDown);
    svg.removeEventListener("mousemove", this.boundMouseMove);
    svg.removeEventListener("mouseup", this.boundMouseUp);
  }
  onMouseDown(e) {
    const port = e.target.closest(".port-output");
    if (!port || e.button !== 0) return;
    const node = port.closest(".node");
    if (!node) return;
    this.drawing = true;
    this.srcNodeId = node.dataset.nodeId;
    this.srcPortId = port.dataset.portId;
    this.srcPortType = port.dataset.portType;
    const nPos = this.renderer.getNodePosition(node);
    this.startX = nPos.x + parseFloat(port.getAttribute("cx") || "0");
    this.startY = nPos.y + parseFloat(port.getAttribute("cy") || "0");
    this.createPreview();
    e.preventDefault();
    e.stopPropagation();
  }
  onMouseMove(e) {
    if (!this.drawing) return;
    const rect = this.renderer.svg.getBoundingClientRect();
    const pos = this.renderer.viewport.screenToCanvas(e.clientX, e.clientY, rect);
    this.updatePreview(pos.x, pos.y);
    this.highlightTargets(e.target);
  }
  onMouseUp(e) {
    if (!this.drawing) return;
    this.drawing = false;
    this.removePreview();
    this.clearHighlights();
    const port = e.target.closest(".port-input");
    if (!port) return;
    const node = port.closest(".node");
    if (!node) return;
    if (port.dataset.portType !== this.srcPortType) return;
    if (node.dataset.nodeId === this.srcNodeId) return;
    this.callbacks.onConnect(
      { nodeId: this.srcNodeId, portId: this.srcPortId, portType: this.srcPortType },
      {
        nodeId: node.dataset.nodeId,
        portId: port.dataset.portId,
        portType: port.dataset.portType
      }
    );
  }
  createPreview() {
    const ns = "http://www.w3.org/2000/svg";
    this.preview = document.createElementNS(ns, "path");
    this.preview.setAttribute("class", "edge-preview");
    this.preview.setAttribute("fill", "none");
    this.preview.setAttribute("stroke", "var(--gc-accent)");
    this.preview.setAttribute("stroke-width", "2");
    this.preview.setAttribute("stroke-dasharray", "6 3");
    this.renderer.getInteractionLayer().appendChild(this.preview);
  }
  updatePreview(x, y) {
    if (!this.preview) return;
    const cp = Math.max(50, Math.abs(x - this.startX) * 0.5);
    this.preview.setAttribute(
      "d",
      `M ${this.startX} ${this.startY} C ${this.startX + cp} ${this.startY}, ${x - cp} ${y}, ${x} ${y}`
    );
  }
  removePreview() {
    if (this.preview) {
      this.preview.remove();
      this.preview = null;
    }
  }
  highlightTargets(target) {
    this.clearHighlights();
    const port = target.closest(".port-input");
    if (port) {
      const valid = port.dataset.portType === this.srcPortType && port.closest(".node")?.getAttribute("data-node-id") !== this.srcNodeId;
      port.classList.add(valid ? "port-valid-target" : "port-invalid-target");
    }
  }
  clearHighlights() {
    this.renderer.svg.querySelectorAll(".port-valid-target, .port-invalid-target").forEach(
      (el) => el.classList.remove("port-valid-target", "port-invalid-target")
    );
  }
};

// src/interactions/keyboard.ts
var KeyboardManager = class {
  callbacks;
  boundKeyDown;
  constructor(callbacks) {
    this.callbacks = callbacks;
    this.boundKeyDown = this.onKeyDown.bind(this);
  }
  attach(root) {
    root.addEventListener("keydown", this.boundKeyDown);
  }
  detach(root) {
    root.removeEventListener("keydown", this.boundKeyDown);
  }
  onKeyDown(e) {
    const tag = e.target.tagName;
    if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
    const mod = e.metaKey || e.ctrlKey;
    if (mod && e.key === "z" && !e.shiftKey) {
      e.preventDefault();
      this.callbacks.onUndo();
    } else if (mod && e.key === "z" && e.shiftKey) {
      e.preventDefault();
      this.callbacks.onRedo();
    } else if (mod && e.key === "y") {
      e.preventDefault();
      this.callbacks.onRedo();
    } else if (mod && e.key === "=") {
      e.preventDefault();
      this.callbacks.onZoomIn();
    } else if (mod && e.key === "-") {
      e.preventDefault();
      this.callbacks.onZoomOut();
    } else if (mod && e.key === "0") {
      e.preventDefault();
      this.callbacks.onZoomToFit();
    }
  }
};

// src/styles/canvas.css.ts
var canvasStyles = `
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

// src/graphiti-canvas.ts
var GraphitiCanvasElement = class extends HTMLElement {
  static observedAttributes = ["api-url", "workflow-id", "token", "theme", "read-only"];
  client = null;
  clientBaseUrl = "";
  clientToken = "";
  ws = null;
  renderer = null;
  selection = null;
  connection = null;
  keyboard = null;
  shadowInitialized = false;
  // --- Reflected properties ---
  get apiUrl() {
    return this.getAttribute("api-url") || "";
  }
  set apiUrl(value) {
    this.setAttribute("api-url", value);
  }
  get workflowId() {
    return this.getAttribute("workflow-id") || "";
  }
  set workflowId(value) {
    this.setAttribute("workflow-id", value);
  }
  get token() {
    return this.getAttribute("token") || "";
  }
  set token(value) {
    this.setAttribute("token", value);
  }
  get theme() {
    return this.getAttribute("theme") || "light";
  }
  set theme(value) {
    this.setAttribute("theme", value);
  }
  get readOnly() {
    return this.hasAttribute("read-only");
  }
  set readOnly(value) {
    if (value) this.setAttribute("read-only", "");
    else this.removeAttribute("read-only");
  }
  // --- Lifecycle ---
  connectedCallback() {
    if (!this.shadowInitialized) {
      const shadow = this.attachShadow({ mode: "open" });
      const style = document.createElement("style");
      style.textContent = canvasStyles;
      shadow.appendChild(style);
      const root = document.createElement("div");
      root.setAttribute("class", "canvas-root");
      root.setAttribute("tabindex", "0");
      shadow.appendChild(root);
      this.renderer = new SVGRenderer();
      root.appendChild(this.renderer.svg);
      this.setupInteractions(shadow);
      this.shadowInitialized = true;
    } else {
      this.reattachInteractions();
    }
    if (this.workflowId && this.apiUrl) {
      this.loadWorkflow();
    }
  }
  disconnectedCallback() {
    this.teardownInteractions();
    if (this.ws) {
      this.ws.disconnect();
      this.ws = null;
    }
    if (this.renderer) {
      this.renderer.destroy();
    }
  }
  attributeChangedCallback(name, oldValue, newValue) {
    if (oldValue === newValue) return;
    if (name === "workflow-id" && this.shadowInitialized && newValue) {
      this.loadWorkflow();
    }
  }
  // --- Public methods ---
  async loadWorkflow() {
    if (!this.apiUrl || !this.workflowId) return;
    this.ensureClient();
    try {
      const state = await this.client.getWorkflowState(this.workflowId);
      this.renderer.renderState(state);
      if (this.renderer.svg.querySelectorAll(".node").length > 0) {
        this.renderer.zoomToFit();
      }
      this.emit("graphiti:workflow-changed", { workflow: state });
    } catch (err) {
      this.emit("graphiti:error", {
        message: err instanceof Error ? err.message : "Failed to load workflow"
      });
    }
  }
  async executeCommand(cmd) {
    if (this.readOnly) return null;
    this.ensureClient();
    try {
      const result = await this.client.executeCommand(this.workflowId, cmd);
      if (result.ok && result.workflow) {
        this.renderer.renderState(result.workflow);
        this.emit("graphiti:workflow-changed", { workflow: result.workflow });
      }
      this.emit("graphiti:command-executed", { type: cmd.type, ok: result.ok });
      return result;
    } catch (err) {
      this.emit("graphiti:error", {
        message: err instanceof Error ? err.message : "Command failed"
      });
      return null;
    }
  }
  async undo() {
    if (this.readOnly) return;
    this.ensureClient();
    try {
      const result = await this.client.undo(this.workflowId);
      if (result.ok && result.workflow) {
        this.renderer.renderState(result.workflow);
        this.selection?.clearSelection();
        this.emit("graphiti:workflow-changed", { workflow: result.workflow });
      }
    } catch (err) {
      this.emit("graphiti:error", {
        message: err instanceof Error ? err.message : "Undo failed"
      });
    }
  }
  async redo() {
    if (this.readOnly) return;
    this.ensureClient();
    try {
      const result = await this.client.redo(this.workflowId);
      if (result.ok && result.workflow) {
        this.renderer.renderState(result.workflow);
        this.selection?.clearSelection();
        this.emit("graphiti:workflow-changed", { workflow: result.workflow });
      }
    } catch (err) {
      this.emit("graphiti:error", {
        message: err instanceof Error ? err.message : "Redo failed"
      });
    }
  }
  async exportJSON() {
    this.ensureClient();
    try {
      return await this.client.getWorkflowState(this.workflowId);
    } catch (err) {
      this.emit("graphiti:error", {
        message: err instanceof Error ? err.message : "Export failed"
      });
      return null;
    }
  }
  zoomToFit() {
    this.renderer?.zoomToFit();
  }
  // --- Private helpers ---
  ensureClient() {
    if (!this.client || this.clientBaseUrl !== this.apiUrl || this.clientToken !== this.token) {
      this.clientBaseUrl = this.apiUrl;
      this.clientToken = this.token;
      this.client = new GraphitiClient({
        baseUrl: this.apiUrl,
        token: this.token
      });
    }
  }
  setupInteractions(shadow) {
    if (!this.renderer) return;
    this.selection = new SelectionManager(this.renderer, {
      onSelect: (nodeIds) => {
        if (nodeIds.length === 1) {
          this.emit("graphiti:node-selected", { nodeId: nodeIds[0] });
        }
      },
      onDeselect: () => {
        this.emit("graphiti:node-deselected", {});
      },
      onDeleteNodes: async (nodeIds) => {
        for (const id of nodeIds) {
          await this.executeCommand({ type: "remove_node", nodeId: id });
        }
      },
      onDeleteEdges: async (edgeIds) => {
        for (const id of edgeIds) {
          await this.executeCommand({ type: "remove_edge", edgeId: id });
        }
      },
      onDragEnd: (nodeId, x, y) => {
        const el = this.renderer.svg.querySelector(`[data-node-id="${nodeId}"]`);
        if (!el) return;
        const orig = this.renderer.getNodePosition(el);
        this.executeCommand({
          type: "move_node",
          nodeId,
          fromX: orig.x,
          fromY: orig.y,
          toX: x,
          toY: y
        });
      },
      onMultiDragEnd: (moves) => {
        const from = moves.map((m) => {
          const el = this.renderer.svg.querySelector(`[data-node-id="${m.nodeId}"]`);
          const pos = el ? this.renderer.getNodePosition(el) : { x: m.x, y: m.y };
          return { nodeId: m.nodeId, x: pos.x, y: pos.y };
        });
        this.executeCommand({
          type: "move_nodes",
          from,
          to: moves
        });
      }
    });
    this.connection = new ConnectionManager(this.renderer, {
      onConnect: (source, target) => {
        this.executeCommand({
          type: "add_edge",
          sourceNodeId: source.nodeId,
          sourcePortId: source.portId,
          targetNodeId: target.nodeId,
          targetPortId: target.portId
        });
      }
    });
    this.keyboard = new KeyboardManager({
      onUndo: () => this.undo(),
      onRedo: () => this.redo(),
      onZoomIn: () => {
        const rect = this.renderer.svg.getBoundingClientRect();
        this.renderer.viewport.zoom(1, rect.left + rect.width / 2, rect.top + rect.height / 2, rect);
      },
      onZoomOut: () => {
        const rect = this.renderer.svg.getBoundingClientRect();
        this.renderer.viewport.zoom(-1, rect.left + rect.width / 2, rect.top + rect.height / 2, rect);
      },
      onZoomToFit: () => this.zoomToFit()
    });
    this.selection.attach(shadow);
    this.connection.attach();
    this.keyboard.attach(shadow);
  }
  reattachInteractions() {
    const shadow = this.shadowRoot;
    if (!shadow || !this.renderer) return;
    this.renderer = new SVGRenderer();
    const root = shadow.querySelector(".canvas-root");
    if (root) {
      const oldSvg = root.querySelector("svg");
      if (oldSvg) oldSvg.remove();
      root.appendChild(this.renderer.svg);
    }
    this.setupInteractions(shadow);
  }
  teardownInteractions() {
    const shadow = this.shadowRoot;
    if (!shadow) return;
    this.selection?.detach(shadow);
    this.connection?.detach();
    this.keyboard?.detach(shadow);
  }
  emit(name, detail) {
    this.dispatchEvent(
      new CustomEvent(name, {
        detail,
        bubbles: true,
        composed: true
      })
    );
  }
};
if (typeof customElements !== "undefined" && !customElements.get("graphiti-canvas")) {
  customElements.define("graphiti-canvas", GraphitiCanvasElement);
}
export {
  ConnectionManager,
  GraphitiCanvasElement,
  KeyboardManager,
  SVGRenderer,
  SelectionManager,
  Viewport,
  calcNodeHeight,
  calcNodeWidth,
  findPortY,
  getBodyAttrs,
  portY,
  renderDiamond,
  renderHexagon,
  renderPill,
  renderRoundedRect,
  renderShape,
  renderSubWorkflow
};
//# sourceMappingURL=index.js.map