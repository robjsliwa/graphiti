import { describe, it, expect } from 'vitest';
import { GraphitiCanvasElement } from '../src/graphiti-canvas.js';

describe('GraphitiCanvasElement registration', () => {
  it('is registered as graphiti-canvas custom element', () => {
    const Ctor = customElements.get('graphiti-canvas');
    expect(Ctor).toBeDefined();
    expect(Ctor).toBe(GraphitiCanvasElement);
  });

  it('creates an element with Shadow DOM', () => {
    const el = document.createElement('graphiti-canvas');
    document.body.appendChild(el);
    expect(el.shadowRoot).not.toBeNull();
    expect(el.shadowRoot!.mode).toBe('open');
    el.remove();
  });

  it('injects styles into Shadow DOM', () => {
    const el = document.createElement('graphiti-canvas');
    document.body.appendChild(el);
    const style = el.shadowRoot!.querySelector('style');
    expect(style).not.toBeNull();
    expect(style!.textContent).toContain('--gc-bg');
    el.remove();
  });

  it('contains an SVG element in Shadow DOM', () => {
    const el = document.createElement('graphiti-canvas');
    document.body.appendChild(el);
    const svg = el.shadowRoot!.querySelector('svg');
    expect(svg).not.toBeNull();
    expect(svg!.classList.contains('workflow-canvas')).toBe(true);
    el.remove();
  });

  it('reflects api-url, workflow-id, token, theme, read-only attributes as properties', () => {
    const el = document.createElement('graphiti-canvas') as GraphitiCanvasElement;
    el.setAttribute('api-url', 'http://localhost:8080');
    el.setAttribute('workflow-id', 'wf-123');
    el.setAttribute('token', 'test-token');
    el.setAttribute('theme', 'dark');
    el.setAttribute('read-only', '');
    document.body.appendChild(el);

    expect(el.apiUrl).toBe('http://localhost:8080');
    expect(el.workflowId).toBe('wf-123');
    expect(el.token).toBe('test-token');
    expect(el.theme).toBe('dark');
    expect(el.readOnly).toBe(true);
    el.remove();
  });

  it('exposes public methods', () => {
    const el = document.createElement('graphiti-canvas') as GraphitiCanvasElement;
    document.body.appendChild(el);

    expect(typeof el.loadWorkflow).toBe('function');
    expect(typeof el.executeCommand).toBe('function');
    expect(typeof el.undo).toBe('function');
    expect(typeof el.redo).toBe('function');
    expect(typeof el.exportJSON).toBe('function');
    expect(typeof el.zoomToFit).toBe('function');
    el.remove();
  });
});
