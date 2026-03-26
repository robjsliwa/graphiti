# Builder Execution Mode Test Plan

## Scope
Mode toggle between builder and execution, execution run list, WebSocket live status.

## Base URL
`http://localhost:8080/workflows/{id}`

## Preconditions
- Authenticated via fake auth
- Workflow open in builder
- Workflow has been deployed at least once (for execution runs)

## Tests

### Mode Tab — Builder Active by Default
- Navigate to builder page
- `[data-testid="mode-tab-builder"]` has active/selected state
- `[data-testid="mode-tab-execution"]` is inactive
- Canvas and node palette are visible

### Switch to Execution Mode
- Click `[data-testid="mode-tab-execution"]`
- Execution tab becomes active
- Builder-specific UI (palette, config panel editing) is hidden or disabled
- `[data-testid="execution-run-list"]` becomes visible

### Switch Back to Builder Mode
- From execution mode, click `[data-testid="mode-tab-builder"]`
- Builder UI restores (palette, editable config)
- Execution run list hides

### Execution Run List — Empty
- Workflow with no runs
- Switch to execution mode
- `[data-testid="execution-run-list"]` shows empty state message

### Execution Run List — With Runs
- Deploy workflow and trigger execution(s)
- Switch to execution mode
- `[data-testid="execution-run-list"]` loads via `hx-get /api/workflows/{id}/runs`
- Each run shows: run ID, status, timestamp
- Runs ordered by most recent first

### Execution Run Selection
- Click on a run in `[data-testid="execution-run-list"]`
- Canvas nodes update to show per-node execution status
- Status indicators (colors/icons) on `[data-node-id]` elements

### WebSocket Live Status Updates
- Start an execution run
- WebSocket connection receives status updates
- Node status indicators update in real time without page refresh
- Statuses progress: pending -> running -> completed/failed

### WebSocket Reconnection
- Disconnect WebSocket (e.g., server restart)
- Client attempts reconnection
- After reconnect, status updates resume

### Execution Mode — Canvas Read-Only
- In execution mode, attempt to drag a node — should not create new node
- Attempt to delete a node — should be blocked
- Canvas pan/zoom still works (view-only interactions allowed)

## Key Selectors
| Selector | Purpose |
|---|---|
| `[data-testid="mode-tab-builder"]` | Builder mode tab |
| `[data-testid="mode-tab-execution"]` | Execution mode tab |
| `[data-testid="execution-run-list"]` | List of execution runs |
| `[data-node-id]` | Node elements showing execution status |
| `[data-testid="workflow-canvas"]` | Canvas (read-only in execution mode) |
