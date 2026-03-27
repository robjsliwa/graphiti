import type { WorkflowState, NodeState, EdgeState } from '@graphiti/client';
import { renderNode } from './node-renderer.js';
import { renderEdge, calcEdgePath } from './edge-renderer.js';
import { createGridPattern, createArrowheadMarker, updateGridForTransform } from './grid.js';
import { Viewport } from './viewport.js';
import { calcNodeWidth, calcNodeHeight, findPortY } from './layout.js';

const SVG_NS = 'http://www.w3.org/2000/svg';

export interface RendererOptions {
  gridSize?: number;
}

export class SVGRenderer {
  readonly svg: SVGSVGElement;
  readonly viewport: Viewport;

  private contentGroup!: SVGGElement;
  private nodeLayer!: SVGGElement;
  private edgeLayer!: SVGGElement;
  private interactionLayer!: SVGGElement;
  private gridPattern!: SVGPatternElement;
  private isPanning = false;
  private lastX = 0;
  private lastY = 0;
  private cullPending = false;
  private nodeSizeCache = new Map<string, { w: number; h: number }>();
  private currentNodes: NodeState[] = [];
  private readonly boundMouseMove: (e: MouseEvent) => void;
  private readonly boundMouseUp: () => void;
  private readonly boundWheel: (e: WheelEvent) => void;
  private readonly boundMouseDown: (e: MouseEvent) => void;

  constructor(options?: RendererOptions) {
    this.viewport = new Viewport(options?.gridSize);

    this.svg = document.createElementNS(SVG_NS, 'svg');
    this.svg.setAttribute('class', 'workflow-canvas');
    this.svg.setAttribute('xmlns', SVG_NS);
    this.svg.setAttribute('width', '100%');
    this.svg.setAttribute('height', '100%');

    this.buildSVGStructure();

    // Bind all event handlers once in constructor
    this.boundMouseMove = this.onMouseMove.bind(this);
    this.boundMouseUp = this.onMouseUp.bind(this);
    this.boundWheel = this.onWheel.bind(this);
    this.boundMouseDown = this.onMouseDown.bind(this);
    this.bindEvents();
  }

  private buildSVGStructure(): void {
    // Defs: grid pattern + arrowhead marker
    const defs = document.createElementNS(SVG_NS, 'defs');
    this.gridPattern = createGridPattern();
    defs.appendChild(this.gridPattern);
    defs.appendChild(createArrowheadMarker());
    this.svg.appendChild(defs);

    // Grid background
    const gridRect = document.createElementNS(SVG_NS, 'rect');
    gridRect.setAttribute('width', '100%');
    gridRect.setAttribute('height', '100%');
    gridRect.setAttribute('fill', 'url(#gc-dot-grid)');
    gridRect.setAttribute('class', 'canvas-grid');
    this.svg.appendChild(gridRect);

    // Content group
    this.contentGroup = document.createElementNS(SVG_NS, 'g');
    this.contentGroup.setAttribute('class', 'canvas-content');
    this.contentGroup.setAttribute('transform', 'translate(0, 0) scale(1)');

    this.edgeLayer = document.createElementNS(SVG_NS, 'g');
    this.edgeLayer.setAttribute('class', 'edge-layer');

    this.nodeLayer = document.createElementNS(SVG_NS, 'g');
    this.nodeLayer.setAttribute('class', 'node-layer');

    this.interactionLayer = document.createElementNS(SVG_NS, 'g');
    this.interactionLayer.setAttribute('class', 'interaction-layer');

    this.contentGroup.appendChild(this.edgeLayer);
    this.contentGroup.appendChild(this.nodeLayer);
    this.contentGroup.appendChild(this.interactionLayer);
    this.svg.appendChild(this.contentGroup);
  }

  /** Render the full workflow state. Replaces all existing nodes/edges. */
  renderState(state: WorkflowState): void {
    this.nodeSizeCache.clear();
    this.currentNodes = state.nodes ?? [];

    const stateNodeIds = new Set(this.currentNodes.map((n) => n.id));
    const stateEdgeIds = new Set((state.edges ?? []).map((e) => e.id));

    // Remove deleted
    this.nodeLayer.querySelectorAll('.node').forEach((el) => {
      if (!stateNodeIds.has((el as SVGElement).dataset.nodeId!)) el.remove();
    });
    this.edgeLayer.querySelectorAll('.edge').forEach((el) => {
      if (!stateEdgeIds.has((el as SVGElement).dataset.edgeId!)) el.remove();
    });

    // Add/update nodes
    for (const node of this.currentNodes) {
      const existing = this.nodeLayer.querySelector(
        `[data-node-id="${node.id}"]`,
      ) as SVGGElement | null;
      if (existing) {
        existing.setAttribute('transform', `translate(${node.x}, ${node.y})`);
        const titleEl = existing.querySelector('.node-title');
        if (titleEl) titleEl.textContent = node.label;
      } else {
        this.nodeLayer.appendChild(renderNode(node));
      }
    }

    // Add/update edges
    for (const edge of state.edges ?? []) {
      const existing = this.edgeLayer.querySelector(
        `[data-edge-id="${edge.id}"]`,
      ) as SVGGElement | null;
      if (existing) {
        const newD = calcEdgePath(edge, this.currentNodes);
        existing.querySelectorAll('path').forEach((p) => p.setAttribute('d', newD));
      } else {
        this.edgeLayer.appendChild(renderEdge(edge, this.currentNodes));
      }
    }
  }

  /** Add a single node to the canvas. */
  addNode(node: NodeState): SVGGElement {
    const g = renderNode(node);
    this.nodeLayer.appendChild(g);
    this.currentNodes.push(node);
    return g;
  }

  /** Remove a node by ID. */
  removeNode(nodeId: string): void {
    this.nodeLayer.querySelector(`[data-node-id="${nodeId}"]`)?.remove();
    this.currentNodes = this.currentNodes.filter((n) => n.id !== nodeId);
    this.nodeSizeCache.delete(nodeId);
  }

  /** Add an edge to the canvas. */
  addEdge(edge: EdgeState): SVGGElement {
    const g = renderEdge(edge, this.currentNodes);
    this.edgeLayer.appendChild(g);
    return g;
  }

  /** Remove an edge by ID. */
  removeEdge(edgeId: string): void {
    this.edgeLayer.querySelector(`[data-edge-id="${edgeId}"]`)?.remove();
  }

  /** Update a node's position without re-rendering. */
  updateNodePosition(nodeId: string, x: number, y: number): void {
    const el = this.nodeLayer.querySelector(`[data-node-id="${nodeId}"]`);
    if (el) {
      el.setAttribute('transform', `translate(${x}, ${y})`);
      this.updateEdgesForNode(nodeId);
    }
  }

  /** Update edges connected to a given node (during drag). */
  updateEdgesForNode(nodeId: string): void {
    this.edgeLayer.querySelectorAll('.edge').forEach((el) => {
      const g = el as SVGElement;
      if (g.dataset.sourceNode !== nodeId && g.dataset.targetNode !== nodeId) return;

      const srcEl = this.nodeLayer.querySelector(
        `[data-node-id="${g.dataset.sourceNode}"]`,
      ) as SVGGElement | null;
      const tgtEl = this.nodeLayer.querySelector(
        `[data-node-id="${g.dataset.targetNode}"]`,
      ) as SVGGElement | null;
      if (!srcEl || !tgtEl) return;

      const sp = this.getNodePosition(srcEl);
      const tp = this.getNodePosition(tgtEl);
      const sw = parseInt(
        srcEl.querySelector('.node-header')?.getAttribute('width') || '200',
      );
      const srcPortY = this.domPortY(srcEl, g.dataset.sourcePort!);
      const tgtPortY = this.domPortY(tgtEl, g.dataset.targetPort!);
      const x1 = sp.x + sw;
      const y1 = sp.y + srcPortY;
      const x2 = tp.x;
      const y2 = tp.y + tgtPortY;
      const cp = Math.max(50, (x2 - x1) * 0.5);
      const d = `M ${x1} ${y1} C ${x1 + cp} ${y1}, ${x2 - cp} ${y2}, ${x2} ${y2}`;
      el.querySelectorAll('path').forEach((p) => p.setAttribute('d', d));
    });
  }

  /** Viewport controls. */
  setViewport(x: number, y: number, zoom: number): void {
    this.viewport.setViewport(x, y, zoom);
    this.applyTransform();
  }

  /** Zoom to fit all nodes in view. */
  zoomToFit(): void {
    const nodes = this.nodeLayer.querySelectorAll('.node');
    if (nodes.length === 0) return;

    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;

    nodes.forEach((n) => {
      const pos = this.getNodePosition(n as SVGGElement);
      minX = Math.min(minX, pos.x);
      minY = Math.min(minY, pos.y);
      const nodeW = parseInt(
        n.querySelector('.node-header')?.getAttribute('width') || '200',
      );
      const nodeBg = n.querySelector('.node-bg');
      const nodeH = parseInt(nodeBg?.getAttribute('height') || '80');
      maxX = Math.max(maxX, pos.x + nodeW);
      maxY = Math.max(maxY, pos.y + nodeH);
    });

    if (!isFinite(minX)) return;

    const pad = 48;
    const rect = this.svg.getBoundingClientRect();
    const cw = maxX - minX + pad * 2;
    const ch = maxY - minY + pad * 2;
    this.viewport.scale = Math.min(rect.width / cw, rect.height / ch, 1.0);
    this.viewport.panX =
      (rect.width - cw * this.viewport.scale) / 2 -
      minX * this.viewport.scale +
      pad * this.viewport.scale;
    this.viewport.panY =
      (rect.height - ch * this.viewport.scale) / 2 -
      minY * this.viewport.scale +
      pad * this.viewport.scale;
    this.applyTransform();
  }

  /** Get the node layer (for interaction handlers to query). */
  getNodeLayer(): SVGGElement {
    return this.nodeLayer;
  }

  /** Get the edge layer. */
  getEdgeLayer(): SVGGElement {
    return this.edgeLayer;
  }

  /** Get the interaction layer (for previews/selection rect). */
  getInteractionLayer(): SVGGElement {
    return this.interactionLayer;
  }

  /** Parse a node's position from its transform attribute. */
  getNodePosition(el: SVGElement | Element): { x: number; y: number } {
    const t = el.getAttribute('transform');
    const m = t && t.match(/translate\(([^,]+),\s*([^)]+)\)/);
    return m ? { x: parseFloat(m[1]), y: parseFloat(m[2]) } : { x: 0, y: 0 };
  }

  /** Start a pan gesture (called by interaction handlers). */
  startPan(e: MouseEvent): void {
    this.isPanning = true;
    this.lastX = e.clientX;
    this.lastY = e.clientY;
    this.svg.classList.add('panning');
    // Attach document-level listeners only while panning to avoid leaks
    document.addEventListener('mousemove', this.boundMouseMove);
    document.addEventListener('mouseup', this.boundMouseUp);
  }

  /** Tear down event listeners. */
  destroy(): void {
    this.svg.removeEventListener('wheel', this.boundWheel);
    this.svg.removeEventListener('mousedown', this.boundMouseDown);
    // Clean up any lingering pan listeners
    document.removeEventListener('mousemove', this.boundMouseMove);
    document.removeEventListener('mouseup', this.boundMouseUp);
  }

  // --- Private helpers ---

  private domPortY(nodeEl: SVGElement | Element, portId: string): number {
    const port = nodeEl.querySelector(`[data-port-id="${portId}"]`);
    return port ? parseFloat(port.getAttribute('cy') || '36') : 36;
  }

  private applyTransform(): void {
    this.contentGroup.setAttribute('transform', this.viewport.getTransform());
    updateGridForTransform(
      this.gridPattern,
      this.viewport.scale,
      this.viewport.panX,
      this.viewport.panY,
    );
    this.applyCulling();
  }

  private applyCulling(): void {
    if (this.cullPending) return;
    this.cullPending = true;
    requestAnimationFrame(() => {
      this.cullPending = false;
      const svgRect = this.svg.getBoundingClientRect();
      if (svgRect.width === 0) return; // not mounted
      const topLeft = this.viewport.screenToCanvas(svgRect.left, svgRect.top, svgRect);
      const bottomRight = this.viewport.screenToCanvas(
        svgRect.right,
        svgRect.bottom,
        svgRect,
      );
      const vp = { x1: topLeft.x, y1: topLeft.y, x2: bottomRight.x, y2: bottomRight.y };
      const lowDetail = this.viewport.scale < 0.5;
      const hidePorts = this.viewport.scale < 0.3;

      this.nodeLayer.querySelectorAll('.node').forEach((el) => {
        const svgEl = el as SVGElement;
        const pos = this.getNodePosition(svgEl);
        const id = svgEl.dataset.nodeId!;
        let size = this.nodeSizeCache.get(id);
        if (!size) {
          const w = parseInt(
            el.querySelector('.node-header')?.getAttribute('width') || '200',
          );
          const bg = el.querySelector('.node-bg');
          const h = parseInt(bg?.getAttribute('height') || '80');
          size = { w, h };
          this.nodeSizeCache.set(id, size);
        }

        const pad = 50;
        const inView =
          pos.x + size.w + pad >= vp.x1 &&
          pos.x - pad <= vp.x2 &&
          pos.y + size.h + pad >= vp.y1 &&
          pos.y - pad <= vp.y2;

        el.classList.toggle('culled', !inView);
        if (inView) {
          el.classList.toggle('lod-low', lowDetail);
          el.classList.toggle('lod-no-ports', hidePorts);
        }
      });
    });
  }

  private bindEvents(): void {
    this.svg.addEventListener('wheel', this.boundWheel, { passive: false });
    this.svg.addEventListener('mousedown', this.boundMouseDown);
    // mousemove/mouseup are attached on-demand in startPan() to avoid document-level leaks
  }

  private onWheel(e: WheelEvent): void {
    e.preventDefault();
    const rect = this.svg.getBoundingClientRect();
    this.viewport.zoom(-e.deltaY, e.clientX, e.clientY, rect);
    this.applyTransform();
  }

  private onMouseDown(e: MouseEvent): void {
    const spaceHeld = typeof e.getModifierState === 'function' && e.getModifierState('Space');
    if (e.button === 1 || (e.button === 0 && spaceHeld)) {
      this.startPan(e);
      e.preventDefault();
    }
  }

  private onMouseMove(e: MouseEvent): void {
    if (!this.isPanning) return;
    this.viewport.pan(e.clientX - this.lastX, e.clientY - this.lastY);
    this.lastX = e.clientX;
    this.lastY = e.clientY;
    this.applyTransform();
  }

  private onMouseUp(): void {
    if (this.isPanning) {
      this.isPanning = false;
      this.svg.classList.remove('panning');
      document.removeEventListener('mousemove', this.boundMouseMove);
      document.removeEventListener('mouseup', this.boundMouseUp);
    }
  }
}
