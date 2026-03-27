const SVG_NS = 'http://www.w3.org/2000/svg';

export function renderRoundedRect(width: number, height: number): SVGRectElement {
  const rect = document.createElementNS(SVG_NS, 'rect');
  rect.setAttribute('width', String(width));
  rect.setAttribute('height', String(height));
  rect.setAttribute('rx', '8');
  rect.setAttribute('ry', '8');
  rect.setAttribute('class', 'node-bg');
  rect.setAttribute('fill', 'var(--gc-surface)');
  rect.setAttribute('stroke', 'var(--gc-border)');
  rect.setAttribute('stroke-width', '1.5');
  return rect;
}

export function renderPill(width: number, height: number): SVGRectElement {
  const rect = document.createElementNS(SVG_NS, 'rect');
  rect.setAttribute('width', String(width));
  rect.setAttribute('height', String(height));
  rect.setAttribute('rx', String(height / 2));
  rect.setAttribute('ry', String(height / 2));
  rect.setAttribute('class', 'node-bg');
  rect.setAttribute('fill', 'var(--gc-surface)');
  rect.setAttribute('stroke', 'var(--gc-border)');
  rect.setAttribute('stroke-width', '1.5');
  return rect;
}

export function renderDiamond(width: number, height: number): SVGPolygonElement {
  const poly = document.createElementNS(SVG_NS, 'polygon');
  const cx = width / 2;
  const cy = height / 2;
  poly.setAttribute('points', `${cx},0 ${width},${cy} ${cx},${height} 0,${cy}`);
  poly.setAttribute('class', 'node-bg');
  poly.setAttribute('fill', 'var(--gc-surface)');
  poly.setAttribute('stroke', 'var(--gc-border)');
  poly.setAttribute('stroke-width', '1.5');
  return poly;
}

export function renderHexagon(width: number, height: number): SVGPolygonElement {
  const poly = document.createElementNS(SVG_NS, 'polygon');
  const offset = 20;
  poly.setAttribute(
    'points',
    `${offset},0 ${width - offset},0 ${width},${height / 2} ${width - offset},${height} ${offset},${height} 0,${height / 2}`,
  );
  poly.setAttribute('class', 'node-bg');
  poly.setAttribute('fill', 'var(--gc-surface)');
  poly.setAttribute('stroke', 'var(--gc-border)');
  poly.setAttribute('stroke-width', '1.5');
  return poly;
}

export function renderSubWorkflow(width: number, height: number): SVGGElement {
  const g = document.createElementNS(SVG_NS, 'g');
  const outer = renderRoundedRect(width, height);
  g.appendChild(outer);
  const inner = document.createElementNS(SVG_NS, 'rect');
  inner.setAttribute('x', '3');
  inner.setAttribute('y', '3');
  inner.setAttribute('width', String(width - 6));
  inner.setAttribute('height', String(height - 6));
  inner.setAttribute('rx', '6');
  inner.setAttribute('ry', '6');
  inner.setAttribute('fill', 'none');
  inner.setAttribute('stroke', 'var(--gc-border)');
  inner.setAttribute('stroke-width', '1');
  g.appendChild(inner);
  return g;
}

export type ShapeType = 'rounded-rect' | 'pill' | 'diamond' | 'hexagon' | 'sub-workflow' | 'custom';

export function renderShape(
  type: ShapeType | string,
  width: number,
  height: number,
): SVGElement {
  switch (type) {
    case 'diamond':
      return renderDiamond(width, height);
    case 'hexagon':
      return renderHexagon(width, height);
    case 'pill':
      return renderPill(width, height);
    case 'sub-workflow':
      return renderSubWorkflow(width, height);
    case 'rounded-rect':
    case 'custom':
    default:
      return renderRoundedRect(width, height);
  }
}
