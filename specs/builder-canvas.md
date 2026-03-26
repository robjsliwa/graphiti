# Builder Canvas Test Plan

## Scope
Canvas SVG loading, pan/zoom, node palette search, drag-to-canvas, select/deselect, delete nodes, undo/redo.

## Base URL
`http://localhost:8080/workflows/{id}`

## Preconditions
- Authenticated via fake auth
- Workflow created and navigated to builder page

## Tests

### Canvas Loads with SVG
- Navigate to builder page
- Verify `[data-testid="workflow-canvas"]` exists and is an `<svg>` element
- Verify SVG contains layer groups (grid, edges, nodes)
- Verify zoom controls are visible: `[data-testid="zoom-in-btn"]`, `[data-testid="zoom-out-btn"]`, `[data-testid="zoom-fit-btn"]`

### Zoom — Buttons
- Click `[data-testid="zoom-in-btn"]` — SVG viewBox or transform scale increases
- Click `[data-testid="zoom-out-btn"]` — scale decreases
- Click `[data-testid="zoom-fit-btn"]` — viewport resets to fit content

### Zoom — Scroll Wheel
- Scroll up on canvas — zoom in (transform scale increases)
- Scroll down on canvas — zoom out
- Verify zoom stays within min/max bounds

### Pan
- Mouse down on empty canvas area, drag — viewport translates
- Release — pan stops, new position persists
- Verify nodes move with viewport (not individually)

### Node Palette Search
- Type in `[data-testid="node-search-input"]`
- Expect `hx-get` to `/api/nodes/search?q={query}` fires after debounce
- `[data-testid="node-palette-results"]` updates with filtered results
- Clear input — full palette restores

### Drag Node to Canvas
- Drag a node from palette onto `[data-testid="workflow-canvas"]`
- Expected: `POST /api/workflows/{id}/commands` with `AddNode` command
- New `[data-node-id]` element appears in SVG
- Node positioned at drop coordinates (snapped to grid)

### Select Node
- Click on a `[data-node-id]` element
- Node gets selected state (CSS class or visual indicator)
- Config panel `[data-testid="config-panel"]` loads for that node

### Deselect Node
- With a node selected, click on empty canvas area
- Selection clears — no node has selected state
- Config panel clears or closes

### Multi-Select (Rubber Band)
- Add 2+ nodes to canvas
- Mouse down on empty area, drag to create selection rectangle
- Nodes within rectangle get selected
- Release — selected nodes remain highlighted

### Delete Node
- Select a node
- Press `Delete` key
- Expected: `POST /api/workflows/{id}/commands` with `RemoveNode` command
- `[data-node-id]` element removed from SVG
- Connected edges also removed

### Undo/Redo — Buttons
- Add a node, then click `[data-testid="undo-btn"]`
- Node disappears from canvas
- Click `[data-testid="redo-btn"]`
- Node reappears

### Undo/Redo — Keyboard
- Add a node, press `Ctrl+Z`
- Node removed
- Press `Ctrl+Shift+Z`
- Node restored

## Key Selectors
| Selector | Purpose |
|---|---|
| `[data-testid="workflow-canvas"]` | Main SVG canvas |
| `[data-node-id]` | Individual node in SVG |
| `[data-testid="node-search-input"]` | Palette search input |
| `[data-testid="node-palette-results"]` | Filtered palette results |
| `[data-testid="zoom-in-btn"]` | Zoom in button |
| `[data-testid="zoom-out-btn"]` | Zoom out button |
| `[data-testid="zoom-fit-btn"]` | Zoom fit button |
| `[data-testid="undo-btn"]` | Undo button |
| `[data-testid="redo-btn"]` | Redo button |
| `[data-testid="config-panel"]` | Node configuration panel |
