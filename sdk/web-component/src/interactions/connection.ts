import type { SVGRenderer } from '../renderer/svg-renderer.js';

export interface PortRef {
  nodeId: string;
  portId: string;
  portType: string;
}

export interface ConnectionCallbacks {
  onConnect: (source: PortRef, target: PortRef) => void;
}

export class ConnectionManager {
  private renderer: SVGRenderer;
  private callbacks: ConnectionCallbacks;
  private drawing = false;
  private preview: SVGPathElement | null = null;
  private srcNodeId = '';
  private srcPortId = '';
  private srcPortType = '';
  private startX = 0;
  private startY = 0;

  private boundMouseDown: (e: MouseEvent) => void;
  private boundMouseMove: (e: MouseEvent) => void;
  private boundMouseUp: (e: MouseEvent) => void;

  constructor(renderer: SVGRenderer, callbacks: ConnectionCallbacks) {
    this.renderer = renderer;
    this.callbacks = callbacks;

    this.boundMouseDown = this.onMouseDown.bind(this);
    this.boundMouseMove = this.onMouseMove.bind(this);
    this.boundMouseUp = this.onMouseUp.bind(this);
  }

  attach(): void {
    const svg = this.renderer.svg;
    svg.addEventListener('mousedown', this.boundMouseDown);
    svg.addEventListener('mousemove', this.boundMouseMove);
    svg.addEventListener('mouseup', this.boundMouseUp);
  }

  detach(): void {
    const svg = this.renderer.svg;
    svg.removeEventListener('mousedown', this.boundMouseDown);
    svg.removeEventListener('mousemove', this.boundMouseMove);
    svg.removeEventListener('mouseup', this.boundMouseUp);
  }

  private onMouseDown(e: MouseEvent): void {
    const port = (e.target as Element).closest('.port-output') as SVGElement | null;
    if (!port || e.button !== 0) return;
    const node = port.closest('.node') as SVGElement | null;
    if (!node) return;

    this.drawing = true;
    this.srcNodeId = node.dataset.nodeId!;
    this.srcPortId = port.dataset.portId!;
    this.srcPortType = port.dataset.portType!;

    const nPos = this.renderer.getNodePosition(node);
    this.startX = nPos.x + parseFloat(port.getAttribute('cx') || '0');
    this.startY = nPos.y + parseFloat(port.getAttribute('cy') || '0');
    this.createPreview();
    e.preventDefault();
    e.stopPropagation();
  }

  private onMouseMove(e: MouseEvent): void {
    if (!this.drawing) return;
    const rect = this.renderer.svg.getBoundingClientRect();
    const pos = this.renderer.viewport.screenToCanvas(e.clientX, e.clientY, rect);
    this.updatePreview(pos.x, pos.y);
    this.highlightTargets(e.target as Element);
  }

  private onMouseUp(e: MouseEvent): void {
    if (!this.drawing) return;
    this.drawing = false;
    this.removePreview();
    this.clearHighlights();

    const port = (e.target as Element).closest('.port-input') as SVGElement | null;
    if (!port) return;
    const node = port.closest('.node') as SVGElement | null;
    if (!node) return;
    if (port.dataset.portType !== this.srcPortType) return;
    if (node.dataset.nodeId === this.srcNodeId) return;

    this.callbacks.onConnect(
      { nodeId: this.srcNodeId, portId: this.srcPortId, portType: this.srcPortType },
      {
        nodeId: node.dataset.nodeId!,
        portId: port.dataset.portId!,
        portType: port.dataset.portType!,
      },
    );
  }

  private createPreview(): void {
    const ns = 'http://www.w3.org/2000/svg';
    this.preview = document.createElementNS(ns, 'path') as SVGPathElement;
    this.preview.setAttribute('class', 'edge-preview');
    this.preview.setAttribute('fill', 'none');
    this.preview.setAttribute('stroke', 'var(--gc-accent)');
    this.preview.setAttribute('stroke-width', '2');
    this.preview.setAttribute('stroke-dasharray', '6 3');
    this.renderer.getInteractionLayer().appendChild(this.preview);
  }

  private updatePreview(x: number, y: number): void {
    if (!this.preview) return;
    const cp = Math.max(50, Math.abs(x - this.startX) * 0.5);
    this.preview.setAttribute(
      'd',
      `M ${this.startX} ${this.startY} C ${this.startX + cp} ${this.startY}, ${x - cp} ${y}, ${x} ${y}`,
    );
  }

  private removePreview(): void {
    if (this.preview) {
      this.preview.remove();
      this.preview = null;
    }
  }

  private highlightTargets(target: Element): void {
    this.clearHighlights();
    const port = target.closest('.port-input') as SVGElement | null;
    if (port) {
      const valid =
        port.dataset.portType === this.srcPortType &&
        port.closest('.node')?.getAttribute('data-node-id') !== this.srcNodeId;
      port.classList.add(valid ? 'port-valid-target' : 'port-invalid-target');
    }
  }

  private clearHighlights(): void {
    this.renderer.svg
      .querySelectorAll('.port-valid-target, .port-invalid-target')
      .forEach((el) =>
        el.classList.remove('port-valid-target', 'port-invalid-target'),
      );
  }
}
