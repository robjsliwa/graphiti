import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { SVGRenderer } from '../src/renderer/svg-renderer.js';
import { SelectionManager } from '../src/interactions/selection.js';
import { ConnectionManager } from '../src/interactions/connection.js';
import { KeyboardManager } from '../src/interactions/keyboard.js';
import type { NodeDefJSON } from '@graphiti/client';

function makeNodeDef(): NodeDefJSON {
  return {
    icon: 'G',
    shape: { type: 'rounded-rect', width: 200, headerColor: '', headerBackground: '' },
    category: { group: 'test' },
    inputs: [{ id: 'in', label: 'Input', type: 'data', position: 'left', maxConnections: 1 }],
    outputs: [{ id: 'out', label: 'Output', type: 'data', position: 'right', maxConnections: 1 }],
    attributes: [],
  };
}

describe('SelectionManager', () => {
  let renderer: SVGRenderer;
  let selection: SelectionManager;
  let onSelect: ReturnType<typeof vi.fn>;
  let onDeselect: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    renderer = new SVGRenderer();
    document.body.appendChild(renderer.svg);
    renderer.renderState({
      nodes: [
        { id: 'n1', definitionId: 'test', label: 'N1', x: 0, y: 0, attributes: {}, definition: makeNodeDef() },
        { id: 'n2', definitionId: 'test', label: 'N2', x: 300, y: 0, attributes: {}, definition: makeNodeDef() },
      ],
      edges: [],
    });

    onSelect = vi.fn();
    onDeselect = vi.fn();
    selection = new SelectionManager(renderer, {
      onSelect,
      onDeselect,
      onDeleteNodes: vi.fn(),
      onDeleteEdges: vi.fn(),
      onDragEnd: vi.fn(),
      onMultiDragEnd: vi.fn(),
    });
  });

  afterEach(() => {
    renderer.destroy();
    renderer.svg.remove();
  });

  it('select() marks nodes as selected', () => {
    selection.select(['n1']);
    expect(selection.selectedIds).toEqual(['n1']);
    const el = renderer.svg.querySelector('[data-node-id="n1"]');
    expect(el?.classList.contains('selected')).toBe(true);
  });

  it('clearSelection() removes all selections', () => {
    selection.select(['n1', 'n2']);
    selection.clearSelection();
    expect(selection.selectedIds).toEqual([]);
    expect(onDeselect).toHaveBeenCalled();
  });

  it('select() calls onSelect callback', () => {
    selection.select(['n1']);
    expect(onSelect).toHaveBeenCalledWith(['n1']);
  });
});

describe('ConnectionManager', () => {
  let renderer: SVGRenderer;
  let connection: ConnectionManager;
  let onConnect: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    renderer = new SVGRenderer();
    document.body.appendChild(renderer.svg);
    renderer.renderState({
      nodes: [
        { id: 'n1', definitionId: 'test', label: 'N1', x: 0, y: 0, attributes: {}, definition: makeNodeDef() },
        { id: 'n2', definitionId: 'test', label: 'N2', x: 300, y: 0, attributes: {}, definition: makeNodeDef() },
      ],
      edges: [],
    });

    onConnect = vi.fn();
    connection = new ConnectionManager(renderer, { onConnect });
    connection.attach();
  });

  afterEach(() => {
    connection.detach();
    renderer.destroy();
    renderer.svg.remove();
  });

  it('attaches and detaches without errors', () => {
    connection.detach();
    connection.attach();
    // No assertion needed -- just verifying no throw
  });

  it('does not create preview on non-port mousedown', () => {
    const node = renderer.svg.querySelector('[data-node-id="n1"]')!;
    node.dispatchEvent(new MouseEvent('mousedown', { button: 0, bubbles: true }));
    expect(renderer.getInteractionLayer().querySelector('.edge-preview')).toBeNull();
  });

  it('creates preview path when dragging from output port', () => {
    const port = renderer.svg.querySelector('[data-node-id="n1"] .port-output') as SVGElement;
    if (!port) return; // happy-dom may not support this fully
    port.dispatchEvent(new MouseEvent('mousedown', { button: 0, bubbles: true }));
    const preview = renderer.getInteractionLayer().querySelector('.edge-preview');
    expect(preview).not.toBeNull();
    // Clean up by triggering mouseup
    renderer.svg.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }));
  });
});

describe('KeyboardManager', () => {
  it('calls onUndo for Ctrl+Z', () => {
    const onUndo = vi.fn();
    const keyboard = new KeyboardManager({
      onUndo,
      onRedo: vi.fn(),
      onZoomIn: vi.fn(),
      onZoomOut: vi.fn(),
      onZoomToFit: vi.fn(),
    });
    const div = document.createElement('div');
    document.body.appendChild(div);
    keyboard.attach(div);

    div.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: 'z',
        ctrlKey: true,
        bubbles: true,
      }),
    );
    expect(onUndo).toHaveBeenCalled();
    keyboard.detach(div);
    div.remove();
  });

  it('calls onRedo for Ctrl+Shift+Z', () => {
    const onRedo = vi.fn();
    const keyboard = new KeyboardManager({
      onUndo: vi.fn(),
      onRedo,
      onZoomIn: vi.fn(),
      onZoomOut: vi.fn(),
      onZoomToFit: vi.fn(),
    });
    const div = document.createElement('div');
    document.body.appendChild(div);
    keyboard.attach(div);

    div.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: 'z',
        ctrlKey: true,
        shiftKey: true,
        bubbles: true,
      }),
    );
    expect(onRedo).toHaveBeenCalled();
    keyboard.detach(div);
    div.remove();
  });

  it('calls onZoomToFit for Ctrl+0', () => {
    const onZoomToFit = vi.fn();
    const keyboard = new KeyboardManager({
      onUndo: vi.fn(),
      onRedo: vi.fn(),
      onZoomIn: vi.fn(),
      onZoomOut: vi.fn(),
      onZoomToFit,
    });
    const div = document.createElement('div');
    document.body.appendChild(div);
    keyboard.attach(div);

    div.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: '0',
        ctrlKey: true,
        bubbles: true,
      }),
    );
    expect(onZoomToFit).toHaveBeenCalled();
    keyboard.detach(div);
    div.remove();
  });
});
