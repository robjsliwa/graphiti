# Graphiti UI Discovery Document

Comprehensive catalog of all interactive UI elements, routes, and testability surface for Playwright E2E tests.

---

## Pages & Routes

| Route | Type | Purpose |
|-------|------|---------|
| `GET /` | Full page | Dashboard — list workflows, create/delete |
| `GET /workflows/{id}` | Full page | Builder — canvas, palette, config panel, deploy |
| `GET /auth/login` | Full page | Login (fake auth auto-redirects in dev) |
| `POST /auth/logout` | Action | Logout, redirect to login |
| `GET /help` | HTMX partial | Help modal with keyboard shortcuts |

---

## HTMX Interactions

| Trigger | URL | Target | Source |
|---------|-----|--------|--------|
| Node search `keyup delay:200ms` | `GET /api/nodes/search` | `#node-palette` | builder.templ |
| Page load | `GET /api/workflows/{id}/runs` | `#execution-run-list` | builder.templ |
| Click run entry | `GET /api/runs/{runId}` | `#config-content` | execution_list.templ |
| Load delay:200ms | `GET /api/workflows/{id}/deploy/status` | `none` (Alpine reads response) | deploy_status.templ |
| Form submit | `POST /workflows` | Redirect | dashboard.templ |
| Delete button (with confirm) | `DELETE /workflows/{id}` | Redirect | dashboard.templ |
| Config panel form submit | `PATCH /api/workflows/{id}/nodes/{nodeId}/attributes` | `none` + toast | config_panel.templ |
| Inline rename blur | `PATCH /api/workflows/{id}/name` (via htmx.ajax) | `none` | builder.templ |

---

## Alpine.js Components

1. **Theme toggle** (base.templ) — `themeMode` cycling light/dark/system
2. **Workflow rename** (builder.templ) — inline edit with `editing`, `name`, `original`
3. **deployManager** (components.js) — validate, deploy, export, validation panel
4. **executionMode** (components.js) — builder/execution tabs, WebSocket lifecycle
5. **canvasContextMenu** (components.js) — right-click edge deletion

---

## Vanilla JS Canvas Modules (9 files)

| File | Purpose |
|------|---------|
| canvas.js | Pan/zoom/viewport/SVG rendering |
| select.js | Click/multi-select/rubber band/node drag |
| connect.js | Port-to-port edge drawing |
| drag.js | Palette-to-canvas drag |
| commands.js | Command dispatch, undo/redo, keyboard shortcuts |
| clipboard.js | Copy/cut/paste/duplicate |
| toast.js | Notifications |
| components.js | Alpine.data() registrations |
| app.js | Module wiring |

---

## WebSocket

- `GET /api/ws/workflows/{id}` — live execution status updates
- Auto-reconnects every 3s on close

---

## JSON API Endpoints

| Method | URL | Purpose |
|--------|-----|---------|
| POST | `/api/workflows/{id}/commands` | Execute canvas commands |
| POST | `/api/workflows/{id}/undo` | Undo last command |
| POST | `/api/workflows/{id}/redo` | Redo last undone command |
| GET | `/api/workflows/{id}/nodes/{nodeId}/config` | Load config panel |
| POST | `/api/workflows/{id}/clipboard/copy` | Copy selected nodes |
| POST | `/api/workflows/{id}/clipboard/cut` | Cut selected nodes |
| POST | `/api/workflows/{id}/clipboard/paste` | Paste nodes |
| POST | `/api/workflows/{id}/validate` | Validate workflow |
| POST | `/api/workflows/{id}/deploy` | Deploy workflow |
| GET | `/api/workflows/{id}/export/{format}` | Export as JSON/YAML |

---

## data-testid Attributes

All `data-testid` attributes added to the codebase:

### Dashboard (dashboard.templ)

| Element | `data-testid` |
|---------|---------------|
| Theme toggle button | `theme-toggle` |
| Logout button | `logout-btn` |
| Create workflow form | `create-workflow-form` |
| Workflow name input | `workflow-name-input` |
| Create workflow button | `create-workflow-btn` |
| Workflow card wrapper | `workflow-card-{id}` (dynamic) |
| Delete workflow button | `delete-workflow-btn` |

### Builder (builder.templ)

| Element | `data-testid` |
|---------|---------------|
| Workflow name display | `workflow-name` |
| Workflow name edit input | `workflow-name-edit` |
| Theme toggle button | `theme-toggle` |
| Validate button | `validate-btn` |
| Deploy button | `deploy-btn` |
| Deploy menu | `deploy-menu` |
| Builder mode tab | `mode-tab-builder` |
| Execution mode tab | `mode-tab-execution` |
| Node search input | `node-search-input` |
| Node palette results | `node-palette-results` |
| Execution run list | `execution-run-list` |
| Zoom out button | `zoom-out-btn` |
| Zoom in button | `zoom-in-btn` |
| Zoom fit button | `zoom-fit-btn` |
| Undo button | `undo-btn` |
| Redo button | `redo-btn` |
| Config panel | `config-panel` |

### Canvas (canvas.templ)

| Element | `data-testid` |
|---------|---------------|
| SVG canvas | `workflow-canvas` |

### Help Modal (help_modal.templ)

| Element | `data-testid` |
|---------|---------------|
| Help modal overlay | `help-modal` |

### Toast (toast.js)

| Element | `data-testid` |
|---------|---------------|
| Toast container | `toast-container` |

---

## Existing Selectors (reusable in tests)

### id-based selectors
- `#workflow-canvas`, `#node-palette`, `#config-content`, `#config-panel`
- `#zoom-in`, `#zoom-out`, `#zoom-fit`, `#zoom-level`
- `#undo-btn`, `#redo-btn`
- `#canvas-container`, `#palette-panel`
- `#graphiti-data` (hidden, holds workflow ID)

### data-* attribute selectors
- `[data-node-id]` — SVG node groups
- `[data-definition-id]` — node definition type
- `[data-port-id]` / `[data-port-type]` / `[data-is-input]` — ports
- `[data-edge-id]` — edge groups
- `[data-source-node]` / `[data-target-node]` — edge endpoints
- `[data-run-id]` — execution run entries

### CSS class selectors
- `.node`, `.edge`, `.port` — SVG canvas elements
- `.palette-item` — draggable node palette entries
- `.workflow-card` — dashboard workflow cards
- `.toast`, `.toast-container` — notification elements
