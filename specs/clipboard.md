# Clipboard Test Plan

## Scope
Copy, cut, paste, and duplicate nodes via keyboard shortcuts and commands API.

## Base URL
`http://localhost:8080/workflows/{id}`

## Preconditions
- Authenticated via fake auth
- Workflow open in builder with nodes on canvas

## Tests

### Copy Single Node
- Select one `[data-node-id]` node
- Press `Ctrl+C`
- Expected: clipboard state stored (server-side serialization via commands API)
- Original node remains on canvas unchanged

### Paste Single Node
- Copy a node, then press `Ctrl+V`
- Expected: `POST /api/workflows/{id}/commands` with `PasteNodes` command
- New node appears at offset position from original
- New node has unique `[data-node-id]` (different from source)
- New node has same type and attributes as source

### Paste Multiple Times
- Copy a node
- Press `Ctrl+V` three times
- Three new nodes created, each at incrementally offset positions
- All have unique IDs

### Copy Multiple Nodes
- Select 2+ nodes (rubber band or Ctrl+click)
- Press `Ctrl+C`
- Press `Ctrl+V`
- All selected nodes are pasted as a group
- Relative positions between pasted nodes match original layout

### Copy Nodes with Edges
- Select two connected nodes (with edge between them)
- Press `Ctrl+C`, then `Ctrl+V`
- Pasted group includes both nodes AND the edge between them
- New `[data-edge-id]` connects new `[data-node-id]` elements

### Cut Single Node
- Select a node
- Press `Ctrl+X`
- Node removed from canvas
- Press `Ctrl+V`
- Node appears at new position
- Original position is empty

### Cut Removes Connected Edges
- Node A connected to Node B and Node C
- Select Node A, press `Ctrl+X`
- Node A removed
- Edges from A to B and A to C also removed

### Duplicate — Ctrl+D
- Select a node
- Press `Ctrl+D`
- Equivalent to copy + paste in one action
- New node appears offset from original
- Both original and duplicate visible

### Duplicate Multiple Nodes
- Select 2+ nodes
- Press `Ctrl+D`
- All selected nodes duplicated as a group with preserved layout

### Paste with Nothing Copied
- Without copying anything, press `Ctrl+V`
- No-op — no error, no new nodes

### Undo Paste
- Paste a node
- Press `Ctrl+Z`
- Pasted node removed

### Undo Cut
- Cut a node
- Press `Ctrl+Z`
- Node restored to original position with its edges

### Clipboard Persists Across Selections
- Copy a node
- Deselect, select different node, deselect
- Press `Ctrl+V`
- Originally copied node is pasted (clipboard not cleared by selection changes)

## Key Selectors
| Selector | Purpose |
|---|---|
| `[data-node-id]` | Node elements — verify creation/removal |
| `[data-edge-id]` | Edge elements — verify copy with edges |
| `[data-testid="workflow-canvas"]` | Canvas container |
| `[data-testid="toast-container"]` | Error feedback |
