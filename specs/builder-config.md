# Builder Config Panel Test Plan

## Scope
Node config panel loading, attribute editing, saving, toast feedback.

## Base URL
`http://localhost:8080/workflows/{id}`

## Preconditions
- Authenticated via fake auth
- Workflow with at least one node on canvas

## Tests

### Config Panel Loads on Node Click
- Click on `[data-node-id]` element
- `[data-testid="config-panel"]` populates via `hx-get /api/workflows/{id}/nodes/{nodeId}/config`
- Panel shows node name and attribute fields
- Panel title matches selected node type

### Config Panel Clears on Deselect
- With config panel open, click empty canvas
- `[data-testid="config-panel"]` clears or shows empty state

### Edit String Attribute
- Select a node with string attributes
- Modify a text input in `[data-testid="config-panel"]`
- Expected: `hx-patch` fires (with debounce) to save
- Toast success in `[data-testid="toast-container"]`

### Edit Enum Attribute
- Select a node with enum attribute (e.g., dropdown)
- Change dropdown selection
- Expect save via `hx-patch`
- Reload page — new value persists

### Edit Boolean Attribute
- Select a node with boolean attribute (checkbox/toggle)
- Toggle the value
- Expect save via `hx-patch`
- Verify updated value on reload

### Edit Number Attribute
- Enter valid number — saves successfully
- Enter non-numeric value — validation error shown

### Edit JSON Attribute
- Enter valid JSON string — saves
- Enter invalid JSON — validation error shown

### Secret Attribute — Write-Only
- Select a node with `type: secret` attribute
- Field shows placeholder (not actual value)
- Enter new value — saves successfully
- Reload config panel — field shows placeholder again, never plaintext

### Config Panel Switches Between Nodes
- Click node A — config panel shows A's attributes
- Click node B — config panel updates to B's attributes
- Verify no stale data from node A remains

### Toast Notification on Save
- Edit an attribute and trigger save
- `[data-testid="toast-container"]` shows success toast
- Toast auto-dismisses after timeout

### Toast Notification on Error
- Trigger a save that fails (e.g., invalid data)
- `[data-testid="toast-container"]` shows error toast

## Key Selectors
| Selector | Purpose |
|---|---|
| `[data-testid="config-panel"]` | Config panel container |
| `[data-node-id]` | Node to click for config |
| `[data-testid="toast-container"]` | Toast notifications |
