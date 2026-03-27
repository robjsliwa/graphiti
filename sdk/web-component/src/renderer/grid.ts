const SVG_NS = 'http://www.w3.org/2000/svg';

export function createGridPattern(): SVGPatternElement {
  const pattern = document.createElementNS(SVG_NS, 'pattern');
  pattern.setAttribute('id', 'gc-dot-grid');
  pattern.setAttribute('width', '24');
  pattern.setAttribute('height', '24');
  pattern.setAttribute('patternUnits', 'userSpaceOnUse');

  const dot = document.createElementNS(SVG_NS, 'circle');
  dot.setAttribute('cx', '12');
  dot.setAttribute('cy', '12');
  dot.setAttribute('r', '1');
  dot.setAttribute('fill', 'var(--gc-grid-dot, #d0d0d0)');
  pattern.appendChild(dot);

  return pattern;
}

export function createArrowheadMarker(): SVGMarkerElement {
  const marker = document.createElementNS(SVG_NS, 'marker');
  marker.setAttribute('id', 'arrowhead');
  marker.setAttribute('markerWidth', '10');
  marker.setAttribute('markerHeight', '7');
  marker.setAttribute('refX', '10');
  marker.setAttribute('refY', '3.5');
  marker.setAttribute('orient', 'auto');

  const poly = document.createElementNS(SVG_NS, 'polygon');
  poly.setAttribute('points', '0 0, 10 3.5, 0 7');
  poly.setAttribute('fill', 'var(--gc-edge-color, #94a3b8)');
  marker.appendChild(poly);

  return marker;
}

export function updateGridForTransform(
  pattern: SVGPatternElement,
  scale: number,
  panX: number,
  panY: number,
): void {
  const size = 24 * scale;
  pattern.setAttribute('width', String(size));
  pattern.setAttribute('height', String(size));
  pattern.setAttribute(
    'patternTransform',
    `translate(${panX % size}, ${panY % size})`,
  );
  const dot = pattern.querySelector('circle');
  if (dot) {
    dot.setAttribute('cx', String(size / 2));
    dot.setAttribute('cy', String(size / 2));
    dot.setAttribute('r', String(Math.max(0.5, scale)));
  }
}
