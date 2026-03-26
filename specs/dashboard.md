# Dashboard Test Plan

## Scope
Workflow list, create/delete workflows, empty state, theme toggle.

## Base URL
`http://localhost:8080`

## Preconditions
- Authenticated via fake auth (storageState loaded)
- Dashboard page loaded at `/`

## Tests

### Empty State
- With no workflows in DB, navigate to dashboard
- Verify empty state message is displayed
- Verify `[data-testid="create-workflow-btn"]` is visible
- No `[data-testid^="workflow-card-"]` elements exist

### Create Workflow
- Fill `[data-testid="workflow-name-input"]` with "Test Workflow"
- Click `[data-testid="create-workflow-btn"]` (or submit `[data-testid="create-workflow-form"]`)
- Expected: POST to `/workflows`, redirect to builder page `/workflows/{id}`
- Navigate back to dashboard — new `[data-testid^="workflow-card-"]` appears

### Create Workflow — Empty Name Rejection
- Leave `[data-testid="workflow-name-input"]` empty
- Submit `[data-testid="create-workflow-form"]`
- Expect validation error or no workflow created

### Workflow List
- Create 3 workflows
- Navigate to dashboard
- Verify 3 `[data-testid^="workflow-card-"]` elements exist
- Each card shows workflow name

### Delete Workflow
- Create a workflow, note its ID
- On dashboard, find `[data-testid="delete-workflow-btn"]` within the workflow card
- Click delete — expect `hx-delete` with confirmation dialog
- Accept confirm
- Verify `[data-testid="workflow-card-{id}"]` is removed from DOM
- Refresh page — workflow still gone

### Delete Workflow — Cancel
- Click `[data-testid="delete-workflow-btn"]`
- Cancel the confirmation
- Workflow card remains in DOM

### Theme Toggle
- Click `[data-testid="theme-toggle"]`
- Verify `<html>` `data-theme` attribute changes (cycles through light/dark/system)
- Check that CSS custom properties update (e.g., background color changes)
- Toggle again — cycles to next theme
- Refresh page — theme persists (stored in localStorage)

## Key Selectors
| Selector | Purpose |
|---|---|
| `[data-testid="create-workflow-form"]` | Workflow creation form |
| `[data-testid="workflow-name-input"]` | Name input field |
| `[data-testid="create-workflow-btn"]` | Submit button |
| `[data-testid="workflow-card-{id}"]` | Individual workflow card |
| `[data-testid="delete-workflow-btn"]` | Delete button on card |
| `[data-testid="theme-toggle"]` | Theme cycle button |
