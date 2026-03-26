# Builder Deploy Test Plan

## Scope
Workflow validation, deploy action, export JSON/YAML, deploy menu interactions.

## Base URL
`http://localhost:8080/workflows/{id}`

## Preconditions
- Authenticated via fake auth
- Workflow open in builder

## Tests

### Validate Workflow — Valid
- Build a complete workflow (nodes + edges forming valid graph)
- Click `[data-testid="validate-btn"]`
- Expect success response
- Toast success in `[data-testid="toast-container"]`

### Validate Workflow — Invalid (No Nodes)
- Open empty workflow
- Click `[data-testid="validate-btn"]`
- Expect validation error
- Toast error in `[data-testid="toast-container"]` with message

### Validate Workflow — Invalid (Disconnected Nodes)
- Add nodes but no edges
- Click `[data-testid="validate-btn"]`
- Expect validation warning or error

### Deploy Menu Opens
- Click `[data-testid="deploy-btn"]`
- `[data-testid="deploy-menu"]` becomes visible
- Menu contains deploy action and export options

### Deploy Menu Closes
- Open deploy menu
- Click outside menu — menu closes
- Press `Escape` — menu closes

### Deploy Workflow
- Build valid workflow
- Click `[data-testid="deploy-btn"]` to open menu
- Click deploy action within `[data-testid="deploy-menu"]`
- Expected: `POST /api/workflows/{id}/deploy`
- Toast notification shows deploy result (success or webhook failure)

### Deploy — Validation Failure
- Open empty/invalid workflow
- Attempt deploy
- Expect validation errors before deploy fires
- No POST to deploy endpoint

### Export JSON
- Open deploy menu
- Click JSON export option
- Browser downloads `.json` file
- File content matches workflow definition

### Export YAML
- Open deploy menu
- Click YAML export option
- Browser downloads `.yaml` file
- File content matches workflow definition in YAML format

## Key Selectors
| Selector | Purpose |
|---|---|
| `[data-testid="validate-btn"]` | Validate workflow button |
| `[data-testid="deploy-btn"]` | Deploy button (opens menu) |
| `[data-testid="deploy-menu"]` | Deploy dropdown menu |
| `[data-testid="toast-container"]` | Feedback notifications |
