# Builder Edges Test Plan

## Scope
Connecting ports, edge type validation, deleting edges, edge SVG rendering.

## Base URL
`http://localhost:8080/workflows/{id}`

## Preconditions
- Authenticated via fake auth
- Workflow with 2+ nodes on canvas (added via commands API)

## Tests

### Connect Two Ports
- Mouse down on a source `[data-port-id]` on node A
- Drag to a target `[data-port-id]` on node B
- Release — expected: `POST /api/workflows/{id}/commands` with `AddEdge` command
- New `[data-edge-id]` path element appears in SVG edges layer
- Edge rendered as cubic Bezier curve between the two ports

### Edge Preview During Drag
- Mouse down on source port, drag without releasing
- Preview edge line follows mouse cursor
- Release on empty canvas — preview disappears, no edge created

### Edge Type Validation — Matching Types
- Connect data output port to data input port — succeeds
- Connect control output port to control input port — succeeds
- Verify `[data-edge-id]` exists after each

### Edge Type Validation — Mismatch Rejected
- Attempt to connect data output to control input
- Expect error response from server (type mismatch)
- No `[data-edge-id]` created
- Toast error in `[data-testid="toast-container"]`

### Edge Type Validation — Error Port
- Connect error output to error input — succeeds
- Connect error output to data input — rejected

### Delete Edge — Context Menu
- Right-click on `[data-edge-id]` element
- Context menu appears with delete option
- Click delete
- Expected: `POST /api/workflows/{id}/commands` with `RemoveEdge` command
- `[data-edge-id]` removed from SVG

### Delete Edge — Select and Delete Key
- Click on `[data-edge-id]` to select it
- Press `Delete` key
- Edge removed from SVG

### Edge Rendering
- Create an edge between two nodes
- Verify `[data-edge-id]` is a `<path>` element
- Verify path has arrowhead marker (`marker-end`)
- Move source node — edge path updates to follow node position

### Undo Edge Creation
- Create an edge
- Press `Ctrl+Z`
- Edge removed from SVG
- Press `Ctrl+Shift+Z`
- Edge restored

### Prevent Duplicate Edges
- Connect port A to port B
- Attempt to connect same ports again
- Expect error or no-op — only one `[data-edge-id]` between same ports

## Key Selectors
| Selector | Purpose |
|---|---|
| `[data-port-id]` | Port circle on node (source or target) |
| `[data-port-type]` | Port type attribute (data/control/error) |
| `[data-edge-id]` | Edge path element in SVG |
| `[data-node-id]` | Node group for verifying edge endpoints |
| `[data-testid="toast-container"]` | Error notification display |
