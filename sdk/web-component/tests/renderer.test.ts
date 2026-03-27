import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { SVGRenderer } from '../src/renderer/svg-renderer.js';
import { renderShape } from '../src/renderer/shapes.js';
import { calcNodeWidth, calcNodeHeight, portY, findPortY } from '../src/renderer/layout.js';
import type { WorkflowState, NodeState, EdgeState, NodeDefJSON } from '@graphiti/client';

function makeNodeDef(overrides: Partial<NodeDefJSON> = {}): NodeDefJSON {
  return {
    icon: 'G',
    shape: { type: 'rounded-rect', width: 200, headerColor: '', headerBackground: '' },
    category: { group: 'test' },
    inputs: [{ id: 'in', label: 'Input', type: 'data', position: 'left', maxConnections: 1 }],
    outputs: [{ id: 'out', label: 'Output', type: 'data', position: 'right', maxConnections: 1 }],
    attributes: [],
    ...overrides,
  };
}

function makeNode(id: string, x: number, y: number, def?: NodeDefJSON): NodeState {
  return {
    id,
    definitionId: 'test-def',
    label: `Node ${id}`,
    x,
    y,
    attributes: {},
    definition: def ?? makeNodeDef(),
  };
}

function makeEdge(id: string, srcNodeId: string, tgtNodeId: string): EdgeState {
  return {
    id,
    sourceNodeId: srcNodeId,
    sourcePortId: 'out',
    targetNodeId: tgtNodeId,
    targetPortId: 'in',
  };
}

describe('SVGRenderer', () => {
  let renderer: SVGRenderer;

  beforeEach(() => {
    renderer = new SVGRenderer();
    document.body.appendChild(renderer.svg);
  });

  afterEach(() => {
    renderer.destroy();
    renderer.svg.remove();
  });

  it('creates an SVG element with correct structure', () => {
    expect(renderer.svg.tagName.toLowerCase()).toBe('svg');
    expect(renderer.svg.querySelector('.canvas-content')).not.toBeNull();
    expect(renderer.svg.querySelector('.edge-layer')).not.toBeNull();
    expect(renderer.svg.querySelector('.node-layer')).not.toBeNull();
    expect(renderer.svg.querySelector('.interaction-layer')).not.toBeNull();
  });

  it('includes grid pattern and arrowhead marker in defs', () => {
    const defs = renderer.svg.querySelector('defs');
    expect(defs).not.toBeNull();
    expect(defs!.querySelector('#gc-dot-grid')).not.toBeNull();
    expect(defs!.querySelector('#arrowhead')).not.toBeNull();
  });

  it('renderState creates node groups', () => {
    const state: WorkflowState = {
      nodes: [makeNode('n1', 100, 200), makeNode('n2', 300, 200)],
      edges: [],
    };
    renderer.renderState(state);

    const nodes = renderer.svg.querySelectorAll('.node');
    expect(nodes.length).toBe(2);
    expect((nodes[0] as SVGElement).dataset.nodeId).toBe('n1');
    expect((nodes[1] as SVGElement).dataset.nodeId).toBe('n2');
  });

  it('renderState creates edge paths', () => {
    const state: WorkflowState = {
      nodes: [makeNode('n1', 100, 200), makeNode('n2', 500, 200)],
      edges: [makeEdge('e1', 'n1', 'n2')],
    };
    renderer.renderState(state);

    const edges = renderer.svg.querySelectorAll('.edge');
    expect(edges.length).toBe(1);
    expect((edges[0] as SVGElement).dataset.edgeId).toBe('e1');
    const paths = edges[0].querySelectorAll('path');
    expect(paths.length).toBe(2); // hit + line
  });

  it('renderState removes deleted nodes and edges', () => {
    const state1: WorkflowState = {
      nodes: [makeNode('n1', 100, 200), makeNode('n2', 300, 200)],
      edges: [makeEdge('e1', 'n1', 'n2')],
    };
    renderer.renderState(state1);
    expect(renderer.svg.querySelectorAll('.node').length).toBe(2);
    expect(renderer.svg.querySelectorAll('.edge').length).toBe(1);

    const state2: WorkflowState = {
      nodes: [makeNode('n1', 100, 200)],
      edges: [],
    };
    renderer.renderState(state2);
    expect(renderer.svg.querySelectorAll('.node').length).toBe(1);
    expect(renderer.svg.querySelectorAll('.edge').length).toBe(0);
  });

  it('renderState updates existing node position', () => {
    renderer.renderState({
      nodes: [makeNode('n1', 100, 200)],
      edges: [],
    });
    renderer.renderState({
      nodes: [makeNode('n1', 300, 400)],
      edges: [],
    });

    const el = renderer.svg.querySelector('[data-node-id="n1"]')!;
    expect(el.getAttribute('transform')).toBe('translate(300, 400)');
  });

  it('addNode adds a single node', () => {
    const g = renderer.addNode(makeNode('n1', 50, 50));
    expect(g.dataset.nodeId).toBe('n1');
    expect(renderer.svg.querySelectorAll('.node').length).toBe(1);
  });

  it('removeNode removes a node', () => {
    renderer.addNode(makeNode('n1', 50, 50));
    renderer.removeNode('n1');
    expect(renderer.svg.querySelectorAll('.node').length).toBe(0);
  });

  it('addEdge adds an edge', () => {
    renderer.addNode(makeNode('n1', 0, 0));
    renderer.addNode(makeNode('n2', 300, 0));
    const g = renderer.addEdge(makeEdge('e1', 'n1', 'n2'));
    expect(g.dataset.edgeId).toBe('e1');
  });

  it('removeEdge removes an edge', () => {
    renderer.addNode(makeNode('n1', 0, 0));
    renderer.addNode(makeNode('n2', 300, 0));
    renderer.addEdge(makeEdge('e1', 'n1', 'n2'));
    renderer.removeEdge('e1');
    expect(renderer.svg.querySelectorAll('.edge').length).toBe(0);
  });

  it('updateNodePosition changes transform', () => {
    renderer.addNode(makeNode('n1', 50, 50));
    renderer.updateNodePosition('n1', 200, 300);
    const el = renderer.svg.querySelector('[data-node-id="n1"]')!;
    expect(el.getAttribute('transform')).toBe('translate(200, 300)');
  });

  it('getNodePosition parses transform', () => {
    renderer.addNode(makeNode('n1', 123, 456));
    const el = renderer.svg.querySelector('[data-node-id="n1"]')!;
    const pos = renderer.getNodePosition(el);
    expect(pos.x).toBe(123);
    expect(pos.y).toBe(456);
  });
});

describe('renderShape', () => {
  it('renders rounded-rect as rect with rx/ry=8', () => {
    const el = renderShape('rounded-rect', 200, 80);
    expect(el.tagName.toLowerCase()).toBe('rect');
    expect(el.getAttribute('rx')).toBe('8');
  });

  it('renders diamond as polygon', () => {
    const el = renderShape('diamond', 200, 80);
    expect(el.tagName.toLowerCase()).toBe('polygon');
    expect(el.getAttribute('points')).toContain('100');
  });

  it('renders hexagon as polygon', () => {
    const el = renderShape('hexagon', 200, 80);
    expect(el.tagName.toLowerCase()).toBe('polygon');
    expect(el.getAttribute('points')).toContain('20');
  });

  it('renders pill as rect with large rx', () => {
    const el = renderShape('pill', 200, 40);
    expect(el.tagName.toLowerCase()).toBe('rect');
    expect(el.getAttribute('rx')).toBe('20');
  });

  it('renders sub-workflow as g with two rects', () => {
    const el = renderShape('sub-workflow', 200, 80);
    expect(el.tagName.toLowerCase()).toBe('g');
    expect(el.querySelectorAll('rect').length).toBe(2);
  });

  it('renders custom as rounded-rect fallback', () => {
    const el = renderShape('custom', 200, 80);
    expect(el.tagName.toLowerCase()).toBe('rect');
  });
});

describe('Layout calculations', () => {
  it('calcNodeWidth returns base width for simple node', () => {
    expect(calcNodeWidth(makeNodeDef())).toBe(200);
  });

  it('calcNodeWidth widens for body attrs + port labels', () => {
    const def = makeNodeDef({
      attributes: [{ id: 'a1', label: 'Attr', type: 'string', display: 'node-body' }],
    });
    const w = calcNodeWidth(def);
    expect(w).toBeGreaterThanOrEqual(200);
  });

  it('calcNodeHeight grows with ports', () => {
    const simple = makeNodeDef({ inputs: [], outputs: [] });
    const complex = makeNodeDef({
      inputs: [
        { id: 'i1', label: 'A', type: 'data', position: 'left', maxConnections: 1 },
        { id: 'i2', label: 'B', type: 'data', position: 'left', maxConnections: 1 },
        { id: 'i3', label: 'C', type: 'data', position: 'left', maxConnections: 1 },
      ],
      outputs: [],
    });
    expect(calcNodeHeight(complex)).toBeGreaterThan(calcNodeHeight(simple));
  });

  it('portY distributes ports evenly', () => {
    const h = 120;
    const y0 = portY(0, 3, h);
    const y1 = portY(1, 3, h);
    const y2 = portY(2, 3, h);
    expect(y1 - y0).toBe(y2 - y1); // even spacing
    expect(y1 - y0).toBe(24); // PORT_SPACING
  });

  it('findPortY finds correct port by id', () => {
    const def = makeNodeDef();
    const y = findPortY(def, 'out', true);
    expect(y).toBeGreaterThan(0);
  });
});
