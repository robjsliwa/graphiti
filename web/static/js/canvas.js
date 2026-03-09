// Canvas engine: pan, zoom, viewport transforms
export class CanvasEngine {
  constructor(svgElement) {
    this.svg = svgElement;
    this.contentGroup = svgElement.querySelector('.canvas-content');
    this.scale = 1.0;
    this.panX = 0;
    this.panY = 0;
    this.minScale = 0.25;
    this.maxScale = 2.0;
    this.gridSize = 24;
    this._isPanning = false;
    this._bindEvents();
  }

  screenToCanvas(screenX, screenY) {
    const rect = this.svg.getBoundingClientRect();
    return {
      x: (screenX - rect.left - this.panX) / this.scale,
      y: (screenY - rect.top - this.panY) / this.scale,
    };
  }

  canvasToScreen(canvasX, canvasY) {
    const rect = this.svg.getBoundingClientRect();
    return {
      x: canvasX * this.scale + this.panX + rect.left,
      y: canvasY * this.scale + this.panY + rect.top,
    };
  }

  snapToGrid(x, y) {
    return {
      x: Math.round(x / this.gridSize) * this.gridSize,
      y: Math.round(y / this.gridSize) * this.gridSize,
    };
  }

  zoom(delta, centerX, centerY) {
    const oldScale = this.scale;
    this.scale *= delta > 0 ? 1.1 : 0.9;
    this.scale = Math.max(this.minScale, Math.min(this.maxScale, this.scale));

    const rect = this.svg.getBoundingClientRect();
    const cx = centerX - rect.left;
    const cy = centerY - rect.top;
    this.panX = cx - (cx - this.panX) * (this.scale / oldScale);
    this.panY = cy - (cy - this.panY) * (this.scale / oldScale);

    this._applyTransform();
    this._updateZoomDisplay();
  }

  pan(dx, dy) {
    this.panX += dx;
    this.panY += dy;
    this._applyTransform();
  }

  fitToView() {
    const nodes = this.svg.querySelectorAll('.node');
    if (nodes.length === 0) return;

    let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
    nodes.forEach(n => {
      const t = n.getAttribute('transform');
      const m = t && t.match(/translate\(([^,]+),\s*([^)]+)\)/);
      if (m) {
        const x = parseFloat(m[1]), y = parseFloat(m[2]);
        minX = Math.min(minX, x);
        minY = Math.min(minY, y);
        maxX = Math.max(maxX, x + 200);
        maxY = Math.max(maxY, y + 80);
      }
    });

    if (!isFinite(minX)) return;

    const padding = 48;
    const rect = this.svg.getBoundingClientRect();
    const contentW = maxX - minX + padding * 2;
    const contentH = maxY - minY + padding * 2;
    this.scale = Math.min(rect.width / contentW, rect.height / contentH, 1.0);
    this.panX = (rect.width - contentW * this.scale) / 2 - minX * this.scale + padding * this.scale;
    this.panY = (rect.height - contentH * this.scale) / 2 - minY * this.scale + padding * this.scale;

    this._applyTransform();
    this._updateZoomDisplay();
  }

  _applyTransform() {
    requestAnimationFrame(() => {
      this.contentGroup.setAttribute('transform',
        `translate(${this.panX}, ${this.panY}) scale(${this.scale})`);
      // Update grid pattern
      const pattern = this.svg.querySelector('#dot-grid');
      if (pattern) {
        const size = 24 * this.scale;
        pattern.setAttribute('width', size);
        pattern.setAttribute('height', size);
        pattern.setAttribute('patternTransform', `translate(${this.panX % size}, ${this.panY % size})`);
        const dot = pattern.querySelector('circle');
        if (dot) {
          dot.setAttribute('cx', size / 2);
          dot.setAttribute('cy', size / 2);
          dot.setAttribute('r', Math.max(0.5, this.scale));
        }
      }
    });
  }

  _updateZoomDisplay() {
    const el = document.getElementById('zoom-level');
    if (el) el.textContent = Math.round(this.scale * 100) + '%';
  }

  _bindEvents() {
    // Wheel zoom
    this.svg.addEventListener('wheel', (e) => {
      e.preventDefault();
      this.zoom(-e.deltaY, e.clientX, e.clientY);
    }, { passive: false });

    // Middle-click pan
    this.svg.addEventListener('mousedown', (e) => {
      if (e.button === 1 || (e.button === 0 && e.getModifierState('Space'))) {
        this._isPanning = true;
        this._lastX = e.clientX;
        this._lastY = e.clientY;
        e.preventDefault();
      }
    });

    window.addEventListener('mousemove', (e) => {
      if (!this._isPanning) return;
      this.pan(e.clientX - this._lastX, e.clientY - this._lastY);
      this._lastX = e.clientX;
      this._lastY = e.clientY;
    });

    window.addEventListener('mouseup', () => {
      this._isPanning = false;
    });

    // Zoom buttons
    document.getElementById('zoom-in')?.addEventListener('click', () => {
      const rect = this.svg.getBoundingClientRect();
      this.zoom(1, rect.left + rect.width / 2, rect.top + rect.height / 2);
    });
    document.getElementById('zoom-out')?.addEventListener('click', () => {
      const rect = this.svg.getBoundingClientRect();
      this.zoom(-1, rect.left + rect.width / 2, rect.top + rect.height / 2);
    });
    document.getElementById('zoom-fit')?.addEventListener('click', () => this.fitToView());
  }
}

// Auto-init
const svg = document.getElementById('workflow-canvas');
if (svg) {
  window.canvasEngine = new CanvasEngine(svg);
}
