# @graphiti/canvas

Framework-agnostic Web Component for embedding the Graphiti workflow canvas in any web application.

## Installation

```bash
npm install @graphiti/canvas
```

## Quick Start

```html
<script type="module">
  import '@graphiti/canvas';
</script>

<graphiti-canvas
  api-url="https://graphiti.example.com"
  workflow-id="wf-abc123"
  token="your-bearer-token"
  theme="light"
></graphiti-canvas>
```

## Attributes

| HTML Attribute  | JS Property  | Type                | Description                     |
|-----------------|-------------|---------------------|---------------------------------|
| `api-url`       | `apiUrl`    | `string`            | Graphiti server base URL        |
| `workflow-id`   | `workflowId`| `string`            | Workflow to display             |
| `token`         | `token`     | `string`            | Bearer auth token               |
| `theme`         | `theme`     | `'light' \| 'dark'` | Color theme                     |
| `read-only`     | `readOnly`  | `boolean`           | Disable editing when present    |

## Events

| Event                       | Detail                                | When                          |
|-----------------------------|---------------------------------------|-------------------------------|
| `graphiti:node-selected`    | `{ nodeId: string }`                  | User clicks a node            |
| `graphiti:node-deselected`  | `{}`                                  | Selection cleared             |
| `graphiti:workflow-changed` | `{ workflow: WorkflowState }`         | After any state change        |
| `graphiti:command-executed` | `{ type: string, ok: boolean }`       | After command response        |
| `graphiti:error`            | `{ message: string }`                 | API or rendering error        |

```javascript
document.querySelector('graphiti-canvas')
  .addEventListener('graphiti:node-selected', (e) => {
    console.log('Selected node:', e.detail.nodeId);
  });
```

## Methods

```typescript
const canvas = document.querySelector('graphiti-canvas');

// Reload workflow state from the server
await canvas.loadWorkflow();

// Execute a canvas command
await canvas.executeCommand({ type: 'add_node', definitionId: 'api-gateway', x: 200, y: 100 });

// Undo / redo
await canvas.undo();
await canvas.redo();

// Export workflow as JSON
const state = await canvas.exportJSON();

// Zoom to fit all nodes
canvas.zoomToFit();
```

## Interactions

The canvas supports these interactions out of the box:

- **Pan**: Middle-click drag or Space+left-click drag
- **Zoom**: Mouse wheel
- **Select node**: Click on a node
- **Multi-select**: Ctrl/Cmd+click to add/remove from selection
- **Rubber band select**: Shift+drag on empty canvas
- **Drag node**: Click and drag a selected node (snaps to 24px grid)
- **Connect ports**: Drag from an output port to an input port
- **Delete**: Select nodes/edges and press Delete or Backspace
- **Select all**: Ctrl/Cmd+A
- **Undo/Redo**: Ctrl/Cmd+Z / Ctrl/Cmd+Shift+Z
- **Zoom controls**: Ctrl/Cmd+= / Ctrl/Cmd+- / Ctrl/Cmd+0 (fit to view)

## CSS Custom Properties

The component uses Shadow DOM with CSS custom properties. Override them on the host element:

```css
graphiti-canvas {
  --gc-bg: #1a1a2e;
  --gc-surface: #16213e;
  --gc-accent: #6366f1;
  /* See src/styles/canvas.css.ts for full list */
}
```

## Exports

```typescript
// Custom element (auto-registers as <graphiti-canvas>)
export { GraphitiCanvasElement } from '@graphiti/canvas';

// SVG renderer (for advanced use without the custom element)
export { SVGRenderer, Viewport } from '@graphiti/canvas';

// Interaction handlers
export { SelectionManager, ConnectionManager, KeyboardManager } from '@graphiti/canvas';

// Shape rendering
export { renderShape, renderRoundedRect, renderDiamond, renderHexagon, renderPill, renderSubWorkflow } from '@graphiti/canvas';

// Layout calculations
export { calcNodeWidth, calcNodeHeight, portY, findPortY, getBodyAttrs } from '@graphiti/canvas';
```

## Browser Compatibility

Requires browsers with Custom Elements v1 and Shadow DOM support:
Chrome 54+, Firefox 63+, Safari 10.1+, Edge 79+.

## Development

```bash
npm install
npm test          # Run tests (vitest + happy-dom)
npm run typecheck # TypeScript type checking
npm run build     # Build ESM bundle
npm run dev       # Start demo at localhost:5173
```

## Demo

Run `npm run dev` and open `http://localhost:5173`. Enter your Graphiti server URL and token, then select a workflow to render the canvas.
