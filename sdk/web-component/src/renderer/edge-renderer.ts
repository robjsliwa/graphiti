import type { EdgeState, NodeState } from '@graphiti/client';
import { calcNodeWidth, findPortY } from './layout.js';

const SVG_NS = 'http://www.w3.org/2000/svg';

export function calcEdgePath(edge: EdgeState, nodes: NodeState[]): string {
  const src = nodes.find((n) => n.id === edge.sourceNodeId);
  const tgt = nodes.find((n) => n.id === edge.targetNodeId);
  if (!src || !tgt) return '';

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

export function renderEdge(edge: EdgeState, nodes: NodeState[]): SVGGElement {
  const g = document.createElementNS(SVG_NS, 'g');
  g.setAttribute('class', 'edge');
  g.dataset.edgeId = edge.id;
  g.dataset.sourceNode = edge.sourceNodeId;
  g.dataset.targetNode = edge.targetNodeId;
  g.dataset.sourcePort = edge.sourcePortId;
  g.dataset.targetPort = edge.targetPortId;

  const d = calcEdgePath(edge, nodes);

  const hit = document.createElementNS(SVG_NS, 'path');
  hit.setAttribute('class', 'edge-hit');
  hit.setAttribute('d', d);
  hit.setAttribute('fill', 'none');
  hit.setAttribute('stroke', 'transparent');
  hit.setAttribute('stroke-width', '16');
  hit.setAttribute('pointer-events', 'stroke');

  const line = document.createElementNS(SVG_NS, 'path');
  line.setAttribute('class', 'edge-line');
  line.setAttribute('d', d);
  line.setAttribute('fill', 'none');
  line.setAttribute('stroke', 'var(--gc-edge-color, #94a3b8)');
  line.setAttribute('stroke-width', '2');
  line.setAttribute('marker-end', 'url(#arrowhead)');
  line.setAttribute('pointer-events', 'none');

  g.appendChild(hit);
  g.appendChild(line);
  return g;
}
