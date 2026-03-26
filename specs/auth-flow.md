# Auth Flow Test Plan

## Scope
Login page rendering, fake auth auto-redirect, logout, session persistence.

## Base URL
`http://localhost:8080`

## Preconditions
- Server running with `auth.provider: fake`
- No existing storageState

## Tests

### Login Page Render
- Navigate to `/auth/login`
- Verify login page loads with a heading or login prompt
- Verify no authenticated UI elements are visible (no `[data-testid="logout-btn"]`)

### Fake Auth Auto-Redirect
- Navigate to `/` (unauthenticated)
- Expect redirect to `/auth/login`
- Fake auth should auto-authenticate and redirect to dashboard
- Verify dashboard loads (presence of `[data-testid="create-workflow-btn"]`)

### Logout Flow
- Authenticate via fake auth
- Click `[data-testid="logout-btn"]`
- Expect redirect to `/auth/login`
- Navigate to `/` — should redirect to login again (session cleared)

### Session Persistence
- Authenticate via fake auth, save storageState
- Open new browser context with saved storageState
- Navigate to `/` — should load dashboard directly without login redirect
- Verify `[data-testid="create-workflow-btn"]` is visible

### Protected Routes Without Auth
- Without storageState, navigate to `/workflows/{any-id}`
- Expect redirect to `/auth/login`
- Verify no workflow data is exposed

## Key Selectors
| Selector | Location |
|---|---|
| `[data-testid="logout-btn"]` | Dashboard and Builder header |
| `[data-testid="create-workflow-btn"]` | Dashboard (proves authenticated) |
