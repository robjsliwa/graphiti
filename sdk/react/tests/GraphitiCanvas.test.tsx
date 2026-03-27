import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, act } from '@testing-library/react';
import { createRef } from 'react';
import { GraphitiCanvas, type GraphitiCanvasRef } from '../src/GraphitiCanvas.js';

// Mock the custom element to avoid actual Shadow DOM + SVG rendering in tests
class MockGraphitiCanvasElement extends HTMLElement {
  loadWorkflow = vi.fn().mockResolvedValue(undefined);
  executeCommand = vi.fn().mockResolvedValue({ ok: true });
  undo = vi.fn().mockResolvedValue(undefined);
  redo = vi.fn().mockResolvedValue(undefined);
  zoomToFit = vi.fn();
  exportJSON = vi.fn().mockResolvedValue({ nodes: [], edges: [] });
}

beforeEach(() => {
  // Register mock element if not already defined
  if (!customElements.get('graphiti-canvas')) {
    customElements.define('graphiti-canvas', MockGraphitiCanvasElement);
  }
});

describe('GraphitiCanvas', () => {
  it('renders a graphiti-canvas custom element', () => {
    const { container } = render(
      <GraphitiCanvas
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="test-token"
      />,
    );
    const el = container.querySelector('graphiti-canvas');
    expect(el).not.toBeNull();
    expect(el?.getAttribute('api-url')).toBe('http://localhost:8080');
    expect(el?.getAttribute('workflow-id')).toBe('wf-1');
    expect(el?.getAttribute('token')).toBe('test-token');
  });

  it('maps theme prop to attribute', () => {
    const { container } = render(
      <GraphitiCanvas
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="t"
        theme="dark"
      />,
    );
    const el = container.querySelector('graphiti-canvas');
    expect(el?.getAttribute('theme')).toBe('dark');
  });

  it('maps readOnly prop to read-only attribute', () => {
    const { container } = render(
      <GraphitiCanvas
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="t"
        readOnly
      />,
    );
    const el = container.querySelector('graphiti-canvas');
    expect(el?.hasAttribute('read-only')).toBe(true);
  });

  it('omits read-only attribute when readOnly is false', () => {
    const { container } = render(
      <GraphitiCanvas
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="t"
        readOnly={false}
      />,
    );
    const el = container.querySelector('graphiti-canvas');
    expect(el?.hasAttribute('read-only')).toBe(false);
  });

  it('updates attributes when props change', () => {
    const { container, rerender } = render(
      <GraphitiCanvas
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="t"
      />,
    );
    rerender(
      <GraphitiCanvas
        apiUrl="http://localhost:8080"
        workflowId="wf-2"
        token="t"
      />,
    );
    const el = container.querySelector('graphiti-canvas');
    expect(el?.getAttribute('workflow-id')).toBe('wf-2');
  });

  it('bridges graphiti:node-selected event to onNodeSelected', async () => {
    const onNodeSelected = vi.fn();
    const { container } = render(
      <GraphitiCanvas
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="t"
        onNodeSelected={onNodeSelected}
      />,
    );
    const el = container.querySelector('graphiti-canvas')!;

    await act(async () => {
      el.dispatchEvent(
        new CustomEvent('graphiti:node-selected', {
          detail: { nodeId: 'node-42' },
          bubbles: true,
          composed: true,
        }),
      );
    });

    expect(onNodeSelected).toHaveBeenCalledWith({ nodeId: 'node-42' });
  });

  it('bridges graphiti:error event to onError', async () => {
    const onError = vi.fn();
    const { container } = render(
      <GraphitiCanvas
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="t"
        onError={onError}
      />,
    );
    const el = container.querySelector('graphiti-canvas')!;

    await act(async () => {
      el.dispatchEvent(
        new CustomEvent('graphiti:error', {
          detail: { message: 'failed' },
          bubbles: true,
          composed: true,
        }),
      );
    });

    expect(onError).toHaveBeenCalledWith({ message: 'failed' });
  });

  it('bridges graphiti:workflow-changed event', async () => {
    const onWorkflowChanged = vi.fn();
    const { container } = render(
      <GraphitiCanvas
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="t"
        onWorkflowChanged={onWorkflowChanged}
      />,
    );
    const el = container.querySelector('graphiti-canvas')!;

    const state = { nodes: [], edges: [] };
    await act(async () => {
      el.dispatchEvent(
        new CustomEvent('graphiti:workflow-changed', {
          detail: { workflow: state },
          bubbles: true,
          composed: true,
        }),
      );
    });

    expect(onWorkflowChanged).toHaveBeenCalledWith({ workflow: state });
  });

  it('bridges graphiti:command-executed event', async () => {
    const onCommandExecuted = vi.fn();
    const { container } = render(
      <GraphitiCanvas
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="t"
        onCommandExecuted={onCommandExecuted}
      />,
    );
    const el = container.querySelector('graphiti-canvas')!;

    await act(async () => {
      el.dispatchEvent(
        new CustomEvent('graphiti:command-executed', {
          detail: { type: 'add_node', ok: true },
          bubbles: true,
          composed: true,
        }),
      );
    });

    expect(onCommandExecuted).toHaveBeenCalledWith({ type: 'add_node', ok: true });
  });

  it('exposes imperative methods via ref', () => {
    const ref = createRef<GraphitiCanvasRef>();
    render(
      <GraphitiCanvas
        ref={ref}
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="t"
      />,
    );

    expect(ref.current).not.toBeNull();
    expect(typeof ref.current!.loadWorkflow).toBe('function');
    expect(typeof ref.current!.executeCommand).toBe('function');
    expect(typeof ref.current!.undo).toBe('function');
    expect(typeof ref.current!.redo).toBe('function');
    expect(typeof ref.current!.zoomToFit).toBe('function');
    expect(typeof ref.current!.exportJSON).toBe('function');
  });

  it('ref methods delegate to the custom element', async () => {
    const ref = createRef<GraphitiCanvasRef>();
    const { container } = render(
      <GraphitiCanvas
        ref={ref}
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="t"
      />,
    );

    const el = container.querySelector('graphiti-canvas') as any;

    // Spy on the actual instance methods
    const zoomSpy = vi.spyOn(el, 'zoomToFit');
    const undoSpy = vi.spyOn(el, 'undo');
    const redoSpy = vi.spyOn(el, 'redo');

    await act(async () => {
      ref.current!.zoomToFit();
    });
    expect(zoomSpy).toHaveBeenCalled();

    await act(async () => {
      await ref.current!.undo();
    });
    expect(undoSpy).toHaveBeenCalled();

    await act(async () => {
      await ref.current!.redo();
    });
    expect(redoSpy).toHaveBeenCalled();
  });

  it('cleans up event listeners on unmount', () => {
    const onNodeSelected = vi.fn();
    const { container, unmount } = render(
      <GraphitiCanvas
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="t"
        onNodeSelected={onNodeSelected}
      />,
    );
    const el = container.querySelector('graphiti-canvas')!;

    unmount();

    // After unmount, events should not fire callbacks
    el.dispatchEvent(
      new CustomEvent('graphiti:node-selected', {
        detail: { nodeId: 'late' },
        bubbles: true,
      }),
    );
    expect(onNodeSelected).not.toHaveBeenCalled();
  });

  it('passes className and style props', () => {
    const { container } = render(
      <GraphitiCanvas
        apiUrl="http://localhost:8080"
        workflowId="wf-1"
        token="t"
        className="my-canvas"
        style={{ width: '100%', height: '600px' }}
      />,
    );
    const el = container.querySelector('graphiti-canvas') as HTMLElement;
    expect(el.className).toBe('my-canvas');
    expect(el.style.width).toBe('100%');
    expect(el.style.height).toBe('600px');
  });
});
