/** Viewport manages pan/zoom transforms and coordinate conversion. */
export class Viewport {
  scale = 1.0;
  panX = 0;
  panY = 0;
  readonly minScale = 0.25;
  readonly maxScale = 2.0;
  readonly gridSize: number;

  constructor(gridSize?: number) {
    this.gridSize = gridSize ?? 24;
  }

  screenToCanvas(
    screenX: number,
    screenY: number,
    svgRect: DOMRect,
  ): { x: number; y: number } {
    return {
      x: (screenX - svgRect.left - this.panX) / this.scale,
      y: (screenY - svgRect.top - this.panY) / this.scale,
    };
  }

  canvasToScreen(
    canvasX: number,
    canvasY: number,
    svgRect: DOMRect,
  ): { x: number; y: number } {
    return {
      x: canvasX * this.scale + this.panX + svgRect.left,
      y: canvasY * this.scale + this.panY + svgRect.top,
    };
  }

  snapToGrid(x: number, y: number): { x: number; y: number } {
    return {
      x: Math.round(x / this.gridSize) * this.gridSize,
      y: Math.round(y / this.gridSize) * this.gridSize,
    };
  }

  zoom(delta: number, centerX: number, centerY: number, svgRect: DOMRect): void {
    const oldScale = this.scale;
    this.scale *= delta > 0 ? 1.1 : 0.9;
    this.scale = Math.max(this.minScale, Math.min(this.maxScale, this.scale));
    const cx = centerX - svgRect.left;
    const cy = centerY - svgRect.top;
    this.panX = cx - (cx - this.panX) * (this.scale / oldScale);
    this.panY = cy - (cy - this.panY) * (this.scale / oldScale);
  }

  pan(dx: number, dy: number): void {
    this.panX += dx;
    this.panY += dy;
  }

  setViewport(x: number, y: number, zoom: number): void {
    this.panX = x;
    this.panY = y;
    this.scale = Math.max(this.minScale, Math.min(this.maxScale, zoom));
  }

  getTransform(): string {
    return `translate(${this.panX}, ${this.panY}) scale(${this.scale})`;
  }
}
