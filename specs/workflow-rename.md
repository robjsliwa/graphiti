# Workflow Rename Test Plan

## Scope
Inline workflow name editing, save on blur, cancel on Escape, validation.

## Base URL
`http://localhost:8080/workflows/{id}`

## Preconditions
- Authenticated via fake auth
- Workflow open in builder

## Tests

### Enter Edit Mode
- Click on `[data-testid="workflow-name"]`
- `[data-testid="workflow-name-edit"]` input appears (or element becomes editable)
- Input is focused and contains current workflow name
- Text is selected for easy replacement

### Save on Blur
- Enter edit mode
- Type new name "Renamed Workflow"
- Click elsewhere (blur the input)
- Expected: `htmx.ajax PATCH` sends new name to server
- `[data-testid="workflow-name"]` updates to "Renamed Workflow"
- Refresh page — new name persists

### Save on Enter
- Enter edit mode
- Type new name
- Press `Enter`
- Name saves and edit mode exits

### Cancel on Escape
- Enter edit mode
- Type partial new name
- Press `Escape`
- Edit mode exits
- `[data-testid="workflow-name"]` reverts to original name
- No PATCH request sent

### Empty Name Rejection
- Enter edit mode
- Clear input (empty string)
- Blur or press Enter
- Expect validation error — name not saved
- Original name preserved in `[data-testid="workflow-name"]`

### Whitespace-Only Name Rejection
- Enter edit mode
- Enter "   " (spaces only)
- Blur — should reject as invalid
- Original name preserved

### Long Name Handling
- Enter a very long name (200+ characters)
- Verify it saves or is truncated per server validation
- UI does not break — name display is contained/ellipsized

### Name Reflects on Dashboard
- Rename workflow to "Updated Name"
- Navigate to dashboard
- `[data-testid^="workflow-card-"]` for this workflow shows "Updated Name"

## Key Selectors
| Selector | Purpose |
|---|---|
| `[data-testid="workflow-name"]` | Display name (click to edit) |
| `[data-testid="workflow-name-edit"]` | Edit input field |
