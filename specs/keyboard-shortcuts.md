# Keyboard Shortcuts Test Plan

## Scope
All keyboard shortcuts: undo/redo, clipboard, delete, select all, zoom, help modal, deselect, node navigation.

## Base URL
`http://localhost:8080/workflows/{id}`

## Preconditions
- Authenticated via fake auth
- Workflow open in builder with nodes on canvas
- Canvas is focused (not editing a text input)

## Tests

### Undo — Ctrl+Z
- Add a node to canvas
- Press `Ctrl+Z`
- Node is removed (undo AddNode)
- Verify via absence of `[data-node-id]` in SVG

### Redo — Ctrl+Shift+Z
- Undo an action with `Ctrl+Z`
- Press `Ctrl+Shift+Z`
- Action is re-applied (node reappears)

### Copy — Ctrl+C
- Select a node
- Press `Ctrl+C`
- No visible change (copied to clipboard)
- Paste should produce a duplicate (tested in clipboard spec)

### Cut — Ctrl+X
- Select a node
- Press `Ctrl+X`
- Node is removed from canvas
- Paste should restore it at new position

### Paste — Ctrl+V
- Copy a node, then press `Ctrl+V`
- New node appears on canvas with offset from original position
- New node has a different `[data-node-id]`

### Duplicate — Ctrl+D
- Select a node
- Press `Ctrl+D`
- Duplicate node appears offset from original
- Both nodes visible on canvas

### Delete — Delete Key
- Select a node
- Press `Delete`
- Node removed from canvas
- Connected edges also removed

### Delete — Backspace
- Select a node
- Press `Backspace`
- Same behavior as Delete key

### Select All — Ctrl+A
- Canvas has 3+ nodes
- Press `Ctrl+A`
- All nodes selected (all have selected visual state)
- Prevents default browser select-all behavior

### Zoom In — Ctrl+=
- Press `Ctrl+=`
- Canvas zoom level increases
- Same effect as clicking `[data-testid="zoom-in-btn"]`

### Zoom Out — Ctrl+-
- Press `Ctrl+-`
- Canvas zoom level decreases

### Zoom Fit — Ctrl+0
- Press `Ctrl+0`
- Canvas viewport fits to content

### Help Modal — ?
- Press `?`
- `[data-testid="help-modal"]` appears
- Modal lists all keyboard shortcuts
- Press `?` again or `Escape` — modal closes

### Deselect — Escape
- Select a node
- Press `Escape`
- Selection clears — no nodes selected
- Config panel clears

### Node Navigation — Tab
- Press `Tab`
- Focus moves to next node on canvas
- Node receives selected/focused state

### Node Navigation — Shift+Tab
- Press `Shift+Tab`
- Focus moves to previous node

### Shortcuts Disabled During Text Input
- Click on `[data-testid="workflow-name-edit"]` or a config panel input
- Press `Delete` — should delete text in input, not delete a node
- Press `Ctrl+A` — should select all text in input, not all nodes
- Press `?` — should type "?" in input, not open help modal

## Key Selectors
| Selector | Purpose |
|---|---|
| `[data-testid="workflow-canvas"]` | Canvas must be focused |
| `[data-node-id]` | Nodes affected by shortcuts |
| `[data-testid="help-modal"]` | Help keyboard shortcut modal |
| `[data-testid="undo-btn"]` | Visual confirmation undo state |
| `[data-testid="redo-btn"]` | Visual confirmation redo state |
| `[data-testid="zoom-in-btn"]` | Zoom reference |
| `[data-testid="zoom-out-btn"]` | Zoom reference |
| `[data-testid="zoom-fit-btn"]` | Zoom reference |
