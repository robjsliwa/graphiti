import type { SVGRenderer } from '../renderer/svg-renderer.js';

export interface SelectionCallbacks {
  onSelect: (nodeIds: string[]) => void;
  onDeselect: () => void;
  onDeleteNodes: (nodeIds: string[]) => void;
  onDeleteEdges: (edgeIds: string[]) => void;
  onDragEnd: (nodeId: string, x: number, y: number) => void;
  onMultiDragEnd: (moves: Array<{ nodeId: string; x: number; y: number }>) => void;
}

export class SelectionManager {
  private renderer: SVGRenderer;
  private callbacks: SelectionCallbacks;
  private selectedNodes = new Set<string>();
  private selectedEdges = new Set<string>();
  private dragging = false;
  private dragStartX = 0;
  private dragStartY = 0;
  private nodeOrigPositions = new Map<string, { x: number; y: number }>();
  private selRect: SVGRectElement | null = null;
  private selStartX = 0;
  private selStartY = 0;

  private boundMouseDown: (e: MouseEvent) => void;
  private boundMouseMove: (e: MouseEvent) => void;
  private boundMouseUp: (e: MouseEvent) => void;
  private boundKeyDown: (e: KeyboardEvent) => void;

  constructor(renderer: SVGRenderer, callbacks: SelectionCallbacks) {
    this.renderer = renderer;
    this.callbacks = callbacks;

    this.boundMouseDown = this.onMouseDown.bind(this);
    this.boundMouseMove = this.onMouseMove.bind(this);
    this.boundMouseUp = this.onMouseUp.bind(this);
    this.boundKeyDown = this.onKeyDown.bind(this);
  }

  get selectedIds(): string[] {
    return Array.from(this.selectedNodes);
  }

  get selectedEdgeIds(): string[] {
    return Array.from(this.selectedEdges);
  }

  attach(root: ShadowRoot | HTMLElement): void {
    this.renderer.svg.addEventListener('mousedown', this.boundMouseDown);
    // These must be on the shadow root to capture events outside the SVG
    root.addEventListener('mousemove', this.boundMouseMove as EventListener);
    root.addEventListener('mouseup', this.boundMouseUp as EventListener);
    root.addEventListener('keydown', this.boundKeyDown as EventListener);
  }

  detach(root: ShadowRoot | HTMLElement): void {
    this.renderer.svg.removeEventListener('mousedown', this.boundMouseDown);
    root.removeEventListener('mousemove', this.boundMouseMove as EventListener);
    root.removeEventListener('mouseup', this.boundMouseUp as EventListener);
    root.removeEventListener('keydown', this.boundKeyDown as EventListener);
  }

  select(ids: string[]): void {
    this.clearSelection();
    for (const id of ids) this.selectNode(id);
  }

  clearSelection(): void {
    this.selectedNodes.clear();
    this.selectedEdges.clear();
    this.renderer.svg
      .querySelectorAll('.selected')
      .forEach((el) => el.classList.remove('selected'));
    this.callbacks.onDeselect();
  }

  private selectNode(id: string): void {
    this.selectedNodes.add(id);
    this.renderer.svg
      .querySelector(`[data-node-id="${id}"]`)
      ?.classList.add('selected');
    this.callbacks.onSelect(this.selectedIds);
  }

  private selectEdge(id: string): void {
    this.selectedEdges.add(id);
    this.renderer.svg
      .querySelector(`[data-edge-id="${id}"]`)
      ?.classList.add('selected');
  }

  private toggleNode(id: string): void {
    if (this.selectedNodes.has(id)) {
      this.selectedNodes.delete(id);
      this.renderer.svg
        .querySelector(`[data-node-id="${id}"]`)
        ?.classList.remove('selected');
      this.callbacks.onSelect(this.selectedIds);
    } else {
      this.selectNode(id);
    }
  }

  private onMouseDown(e: MouseEvent): void {
    const target = e.target as Element;
    if (target.closest('.port')) return; // ports are for ConnectionManager
    if (e.button !== 0 || e.getModifierState('Space')) return;

    const node = target.closest('.node') as SVGElement | null;
    const edge = target.closest('.edge') as SVGElement | null;

    if (node) {
      const id = node.dataset.nodeId!;
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
      this.selectEdge(edge.dataset.edgeId!);
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

  private onMouseMove(e: MouseEvent): void {
    if (this.dragging) this.onDrag(e);
    if (this.selRect) this.onRubberBand(e);
  }

  private onMouseUp(e: MouseEvent): void {
    if (this.dragging) this.endDrag(e);
    if (this.selRect) this.endRubberBand(e);
  }

  private onKeyDown(e: KeyboardEvent): void {
    const tag = (e.target as HTMLElement).tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;

    if (e.key === 'Delete' || e.key === 'Backspace') {
      e.preventDefault();
      this.deleteSelection();
    } else if (e.key === 'Escape') {
      this.clearSelection();
    } else if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
      e.preventDefault();
      this.selectAll();
    }
  }

  // --- Drag ---
  private startDrag(e: MouseEvent): void {
    this.dragging = true;
    this.dragStartX = e.clientX;
    this.dragStartY = e.clientY;
    this.nodeOrigPositions.clear();

    const clickedNode = (e.target as Element).closest('.node') as SVGElement | null;
    const clickedId = clickedNode?.dataset.nodeId;

    const ids =
      this.selectedNodes.size > 0 && clickedId && this.selectedNodes.has(clickedId)
        ? this.selectedNodes
        : new Set([clickedId!]);

    for (const id of ids) {
      const el = this.renderer.svg.querySelector(`[data-node-id="${id}"]`);
      if (el) {
        this.nodeOrigPositions.set(id, this.renderer.getNodePosition(el));
      }
    }
  }

  private onDrag(e: MouseEvent): void {
    const dx = (e.clientX - this.dragStartX) / this.renderer.viewport.scale;
    const dy = (e.clientY - this.dragStartY) / this.renderer.viewport.scale;
    for (const [id, orig] of this.nodeOrigPositions) {
      const el = this.renderer.svg.querySelector(`[data-node-id="${id}"]`);
      if (!el) continue;
      el.setAttribute('transform', `translate(${orig.x + dx}, ${orig.y + dy})`);
      this.renderer.updateEdgesForNode(id);
    }
  }

  private endDrag(e: MouseEvent): void {
    this.dragging = false;
    const dx = (e.clientX - this.dragStartX) / this.renderer.viewport.scale;
    const dy = (e.clientY - this.dragStartY) / this.renderer.viewport.scale;
    if (Math.abs(dx) < 2 && Math.abs(dy) < 2) return; // was a click

    if (this.nodeOrigPositions.size > 1) {
      const moves: Array<{ nodeId: string; x: number; y: number }> = [];
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
  private startRubberBand(e: MouseEvent): void {
    const rect = this.renderer.svg.getBoundingClientRect();
    const pos = this.renderer.viewport.screenToCanvas(e.clientX, e.clientY, rect);
    this.selStartX = pos.x;
    this.selStartY = pos.y;
    this.selRect = document.createElementNS(
      'http://www.w3.org/2000/svg',
      'rect',
    ) as SVGRectElement;
    this.selRect.setAttribute('class', 'selection-rect');
    this.selRect.setAttribute('fill', 'var(--gc-accent)');
    this.selRect.setAttribute('fill-opacity', '0.1');
    this.selRect.setAttribute('stroke', 'var(--gc-accent)');
    this.selRect.setAttribute('stroke-width', '1');
    this.selRect.setAttribute('stroke-dasharray', '4 2');
    this.renderer.getInteractionLayer().appendChild(this.selRect);
  }

  private onRubberBand(e: MouseEvent): void {
    if (!this.selRect) return;
    const rect = this.renderer.svg.getBoundingClientRect();
    const pos = this.renderer.viewport.screenToCanvas(e.clientX, e.clientY, rect);
    const x = Math.min(this.selStartX, pos.x);
    const y = Math.min(this.selStartY, pos.y);
    this.selRect.setAttribute('x', String(x));
    this.selRect.setAttribute('y', String(y));
    this.selRect.setAttribute('width', String(Math.abs(pos.x - this.selStartX)));
    this.selRect.setAttribute('height', String(Math.abs(pos.y - this.selStartY)));
  }

  private endRubberBand(e: MouseEvent): void {
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

    this.renderer.svg.querySelectorAll('.node').forEach((el) => {
      const p = this.renderer.getNodePosition(el);
      if (p.x >= x1 && p.y >= y1 && p.x <= x2 && p.y <= y2) {
        this.selectNode((el as SVGElement).dataset.nodeId!);
      }
    });
  }

  private deleteSelection(): void {
    if (this.selectedNodes.size === 0 && this.selectedEdges.size === 0) return;
    const edgeIds = [...this.selectedEdges];
    const nodeIds = [...this.selectedNodes];
    this.clearSelection();
    if (edgeIds.length > 0) this.callbacks.onDeleteEdges(edgeIds);
    if (nodeIds.length > 0) this.callbacks.onDeleteNodes(nodeIds);
  }

  private selectAll(): void {
    this.renderer.svg.querySelectorAll('.node').forEach((el) => {
      this.selectedNodes.add((el as SVGElement).dataset.nodeId!);
      el.classList.add('selected');
    });
    this.callbacks.onSelect(this.selectedIds);
  }
}
