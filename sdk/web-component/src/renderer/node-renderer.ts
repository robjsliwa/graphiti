import type { NodeState, NodeDefJSON } from '@graphiti/client';
import { renderShape } from './shapes.js';
import { calcNodeWidth, calcNodeHeight, portY, getBodyAttrs } from './layout.js';

const SVG_NS = 'http://www.w3.org/2000/svg';

function svgEl(
  tag: string,
  attrs?: Record<string, string>,
  text?: string,
): SVGElement {
  const e = document.createElementNS(SVG_NS, tag);
  if (attrs) {
    for (const [k, v] of Object.entries(attrs)) {
      if (k.startsWith('data-')) {
        (e as SVGElement & { dataset: DOMStringMap }).dataset[
          k.slice(5).replace(/-([a-z])/g, (_, c: string) => c.toUpperCase())
        ] = v;
      } else {
        e.setAttribute(k, v);
      }
    }
  }
  if (text) e.textContent = text;
  return e;
}

export function renderNode(node: NodeState): SVGGElement {
  const def = node.definition;
  const w = calcNodeWidth(def);
  const h = calcNodeHeight(def);
  const hdrBg = def?.shape?.headerBackground || 'var(--gc-surface-2)';
  const hdrColor = def?.shape?.headerColor || 'var(--gc-text)';

  const g = svgEl('g', {
    class: 'node',
    'data-node-id': node.id,
    'data-definition-id': node.definitionId,
    transform: `translate(${node.x}, ${node.y})`,
  }) as SVGGElement;

  // Background shape
  const shapeType = def?.shape?.type || 'rounded-rect';
  const shape = renderShape(shapeType, w, h);
  g.appendChild(shape);

  // Header
  g.appendChild(
    svgEl('rect', {
      width: String(w),
      height: '32',
      rx: '6',
      ry: '6',
      class: 'node-header',
      fill: hdrBg,
    }),
  );
  g.appendChild(
    svgEl('rect', { x: '0', y: '16', width: String(w), height: '16', fill: hdrBg }),
  );
  g.appendChild(
    svgEl('text', { x: '12', y: '22', class: 'node-icon', fill: hdrColor }, def?.icon || '?'),
  );
  g.appendChild(
    svgEl('text', { x: '32', y: '22', class: 'node-title', fill: hdrColor }, node.label),
  );

  // Body attributes
  if (def) {
    const attrs = getBodyAttrs(def, node.attributes);
    for (const attr of attrs) {
      g.appendChild(
        svgEl(
          'text',
          { x: String(attr.x), y: String(attr.y), class: 'node-attr-label' },
          `${attr.label}: ${attr.value}`,
        ),
      );
    }
  }

  // Ports
  if (def) {
    const inputs = def.inputs ?? [];
    const outputs = def.outputs ?? [];

    const addPort = (
      port: { id: string; label: string; type: string },
      isInput: boolean,
      idx: number,
      count: number,
    ) => {
      const px = isInput ? 0 : w;
      const py = portY(idx, count, h);
      const circle = svgEl('circle', {
        cx: String(px),
        cy: String(py),
        r: '6',
        class: `port port-${port.type} port-${isInput ? 'input' : 'output'}`,
        'data-port-id': port.id,
        'data-port-type': port.type,
        'data-is-input': String(isInput),
      });
      const title = svgEl('title', {}, port.label || port.id);
      circle.appendChild(title);
      g.appendChild(circle);

      if (port.label) {
        const lx = isInput ? 12 : w - 12;
        const anchor = isInput ? 'start' : 'end';
        g.appendChild(
          svgEl(
            'text',
            {
              x: String(lx),
              y: String(py + 4),
              class: `port-label ${isInput ? 'port-label-input' : 'port-label-output'}`,
              'text-anchor': anchor,
            },
            port.label,
          ),
        );
      }
    };

    inputs.forEach((p, i) => addPort(p, true, i, inputs.length));
    outputs.forEach((p, i) => addPort(p, false, i, outputs.length));
  }

  return g;
}
