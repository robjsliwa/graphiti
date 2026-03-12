<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go" alt="Go 1.25"/>
  <img src="https://img.shields.io/badge/SQLite-WAL-003B57?style=flat-square&logo=sqlite" alt="SQLite"/>
  <img src="https://img.shields.io/badge/HTMX-2.0-3366CC?style=flat-square" alt="HTMX"/>
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License"/>
</p>

# Graphiti

**A visual workflow builder for designing, connecting, and deploying node-based pipelines.**

Graphiti is a standalone, open-source UI where you drag nodes onto an SVG canvas, wire them together, configure each node's settings, and deploy the resulting workflow definition to any external execution engine via webhook. It's the cockpit, not the engine — deliberately decoupled from any specific backend.

<p align="center">
  <em>Build workflows visually. Deploy anywhere.</em>
</p>

---

## Why Graphiti?

Most workflow tools lock you into a specific execution runtime. Graphiti takes a different approach: it's a pure builder UI that outputs a portable workflow definition. You bring your own engine.

- **Framework-agnostic** — Deploys via signed webhooks to any HTTP endpoint
- **YAML-driven nodes** — Define new node types in config, not code
- **Full undo/redo** — Every canvas action is reversible via the Command Pattern
- **Server-rendered** — HTMX + Templ for a snappy UI with minimal JavaScript (~975 LOC)
- **Single binary** — One Go binary, one SQLite file, zero infrastructure

## Architecture

Graphiti follows **hexagonal architecture** (ports & adapters). The domain layer is pure Go with zero external dependencies. Storage, auth, and deployment are pluggable adapters behind interface boundaries.

```
Driving Adapters → Ports → App Services → Domain ← Ports ← Driven Adapters
  (HTTP, HTMX)            (use cases)    (pure Go)         (SQLite, Auth, Webhook)
```

```
graphiti/
├── cmd/server/          # Entry point — wires adapters to ports
├── internal/
│   ├── domain/          # Pure business logic, zero external imports
│   │   └── commands/    # Command Pattern implementations (add, move, connect, etc.)
│   ├── ports/           # Interface contracts
│   │   ├── driving/     # What the outside world asks us to do
│   │   └── driven/      # What we need from the outside world
│   ├── app/             # Use case implementations
│   └── adapters/
│       ├── driving/     # HTTP handlers, CLI config loader
│       └── driven/      # SQLite, filesystem, auth, webhook
├── web/
│   ├── templates/       # Templ templates (.templ → Go)
│   └── static/
│       ├── js/          # ES modules: app, canvas, commands, connect, drag, select, clipboard, theme
│       └── css/         # Stylesheets with CSS custom property theming
├── config/
│   ├── app.yaml         # Server, storage, deploy settings
│   ├── auth.yaml        # Auth provider config
│   └── nodes/           # YAML node type definitions
└── migrations/          # SQLite schema
```

## Quick Start

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Task](https://taskfile.dev/) (task runner)
- [Templ](https://templ.guide/) (template compiler)

```bash
# Install tools (if needed)
go install github.com/go-task/task/v3/cmd/task@latest
go install github.com/a-h/templ/cmd/templ@latest
```

### Run

```bash
git clone https://github.com/robjsliwa/graphiti.git
cd graphiti

# Set up environment
cp .env.example .env    # Edit .env with your secrets

# Build and run
task run                # Starts at http://localhost:8080
```

In development mode (`auth.provider: fake`), you're auto-authenticated as a dev user — no OAuth setup needed.

### Available Commands

| Command | Description |
|---|---|
| `task run` | Build and start the server |
| `task build` | Compile the binary to `./bin/graphiti` |
| `task test` | Run all tests |
| `task test:domain` | Run domain tests only |
| `task test:app` | Run app service tests only |
| `task test:adapters` | Run adapter tests only |
| `task test:cover` | Run tests with coverage report |
| `task generate` | Compile `.templ` files to Go |
| `task lint` | Run golangci-lint |
| `task dev` | Run directly with `go run` |
| `task clean` | Remove build artifacts |

## How It Works

### Node Definitions

Nodes are defined in YAML, not Go code. Drop a file in `config/nodes/` and it's available in the palette on next startup.

```yaml
# config/nodes/sources/twilio.yaml
apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: source-twilio
  name: Twilio
  description: Receives call recording callbacks
  version: "1.0.0"
  icon: "T"
category:
  group: Sources
  order: 20
shape:
  type: rounded-rect          # rounded-rect | pill | diamond | hexagon | sub-workflow
  width: 200
  headerColor: "var(--cyan)"
  headerBackground: "var(--cyan-dim)"
ports:
  inputs: []
  outputs:
    - id: out-main
      label: Output
      type: data               # data | control | error
      position: right-center
      maxConnections: -1       # -1 = unlimited
attributes:
  - id: endpoint
    label: Endpoint
    type: string               # string | number | boolean | enum | secret | json | expression
    default: "/ingest/twilio"
    required: true
    display: node-body         # node-body | config-panel | both
    group: Connection
validation:
  connectionRules:
    - outputPort: out-main
      allowedTargetCategories: ["Processing", "Destinations"]
```

The project ships with 11 node definitions across four categories:

| Node | Category | Shape | Description |
|---|---|---|---|
| Twilio | Sources | Rounded rect | Receives call recording callbacks |
| API Gateway | Sources | Rounded rect | HTTP endpoint that receives API requests |
| Transcribe | Processing | Rounded rect | Speech-to-text conversion |
| Validate Payload | Processing | Rounded rect | Validates request body fields |
| JSON Transform | Processing | Rounded rect | Reshapes JSON data using mapping templates |
| S3 Storage | Destinations | Rounded rect | Store files to S3 |
| PostgreSQL | Destinations | Rounded rect | Execute parameterized SQL queries |
| HTTP Response | Destinations | Rounded rect | Format and return HTTP responses |
| Condition | Control | Diamond | Branch based on expression |
| HTTP Router | Control | Diamond | Route requests by HTTP method |
| Sub-Workflow | Control | Double-border rect | References another workflow as a reusable building block |

### Command Pattern & Undo/Redo

Every canvas mutation flows through the Command Pattern. Commands are executed via `CommandHistory`, which maintains undo/redo stacks. Each command returns its inverse — undo is just executing the inverse command.

```
User action → Command.Execute(workflow) → returns inverse Command
                                        → pushes inverse onto undo stack
                                        → clears redo stack

Ctrl+Z      → pop undo stack → Execute inverse → push result onto redo stack
Ctrl+Shift+Z → pop redo stack → Execute it     → push result onto undo stack
```

Supported commands: `AddNode`, `RemoveNode`, `MoveNode`, `MoveNodes`, `AddEdge`, `RemoveEdge`, `UpdateAttribute`, `RenameNode`, `PasteNodes`.

### Webhook Deploy

When you deploy a workflow, Graphiti sends a signed JSON payload to your configured endpoint:

```json
{
  "apiVersion": "graphiti/v1",
  "event": "workflow.deployed",
  "timestamp": "2026-03-09T14:30:00Z",
  "deployment": {
    "id": "deploy-abc123",
    "target": "production",
    "triggeredBy": { "userID": "user-1", "username": "alice" }
  },
  "workflow": {
    "id": "wf-001",
    "name": "Support Call Pipeline",
    "version": 5,
    "definition": { "nodes": [], "edges": [] }
  },
  "previousVersion": 4,
  "checksum": "sha256:a1b2c3..."
}
```

The payload is signed with HMAC-SHA256 (header: `X-Graphiti-Signature`), retried with exponential backoff, and includes the full workflow definition for your engine to execute.

### Authentication

Auth is always active — even in dev mode. The `FakeAuthAdapter` auto-authenticates a hardcoded dev user so every request flows through the same auth pipeline that production uses. When you're ready for real auth, switch one config value:

```yaml
# config/auth.yaml
auth:
  provider: github    # swap "fake" → "github", no code changes
  github:
    clientId: "${GITHUB_CLIENT_ID}"
    clientSecret: "${GITHUB_CLIENT_SECRET}"
```

### GitHub OAuth2 Setup

To enable GitHub OAuth in production:

1. Create a GitHub OAuth App at **Settings > Developer settings > OAuth Apps**
2. Set the authorization callback URL to `https://your-domain.com/auth/callback`
3. Configure your environment:

```bash
# .env
GITHUB_CLIENT_ID=your-client-id
GITHUB_CLIENT_SECRET=your-client-secret
```

4. Switch the auth provider in config:

```yaml
# config/auth.yaml
auth:
  provider: github
  github:
    clientId: "${GITHUB_CLIENT_ID}"
    clientSecret: "${GITHUB_CLIENT_SECRET}"
    scopes: ["read:user", "read:org"]
    allowedOrgs: ["your-org"]    # Optional: restrict access to org members
```

When `allowedOrgs` is set, only members of those GitHub organizations can log in.

### Execution Mode

Graphiti has two UI modes, toggled via tabs below the nav bar:

- **Builder** — Design and edit workflows (default)
- **Execution** — Monitor workflow runs in real time

In execution mode, the canvas becomes read-only and shows live status overlays on nodes:
- Pending (gray), Running (yellow pulse), Completed (green), Failed (red), Skipped (dashed gray)

The left panel switches from the node palette to a run history list.

### Execution Callback Endpoint

Your execution engine reports status updates back to Graphiti via:

```
POST /api/callbacks/execution
X-Graphiti-Signature: sha256=<HMAC-SHA256 hex digest>
Content-Type: application/json

{
  "apiVersion": "graphiti/v1",
  "event": "node.status",
  "runID": "run-abc123",
  "workflowID": "wf-001",
  "nodeID": "node-transcribe-1",
  "status": "completed",
  "startedAt": "2026-03-10T14:30:00Z",
  "completedAt": "2026-03-10T14:30:05Z"
}
```

The signature is verified using the same `DEPLOY_HMAC_SECRET` from your `.env`. If the run doesn't exist yet, Graphiti creates it automatically on the first callback.

### WebSocket Live Updates

When in execution mode, the browser connects via WebSocket to receive real-time node status updates:

```
ws://localhost:8080/api/ws/workflows/{workflowID}
```

The server broadcasts status changes to all connected clients for that workflow, enabling live collaboration and monitoring.

### CSRF Protection

All state-mutating requests (POST, PUT, PATCH, DELETE) require a CSRF token. The server sets a `csrf_token` cookie on every response. Include the token in your requests:

```
X-CSRF-Token: <value from csrf_token cookie>
```

GET and HEAD requests are exempt. Auth callback routes are also exempt.

### Themes

Light and dark themes are defined via CSS custom properties. The UI respects `data-theme` on `<html>`:

```css
/* All colors reference variables */
[data-theme="light"] { --bg: #F8F9FB; --surface: #FFFFFF; --accent: #3B82F6; }
[data-theme="dark"]  { --bg: #0F1117; --surface: #1A1D27; --accent: #60A5FA; }
```

The theme toggle button (top-right of every page) cycles through three modes:

| Mode | Icon | Behavior |
|---|---|---|
| Light | Sun | Always light theme |
| Dark | Moon | Always dark theme |
| System | Gear | Follows OS `prefers-color-scheme` setting |

The selected mode is persisted to `localStorage` and applied instantly on page load.

### Sub-Workflows

Sub-workflows let you compose workflows by referencing other workflows as reusable building blocks. This is the key to managing complexity — instead of one giant workflow, you break logic into smaller, testable pieces and wire them together.

#### How to Use Sub-Workflows

**1. Create the child workflow first.**

Go to the dashboard, create a new workflow (e.g., "Email Notification"), build it out with its own nodes and edges, and deploy it. This is the workflow you'll reuse.

**2. Add a Sub-Workflow node to the parent.**

Open your parent workflow. In the **Components** palette on the left, find **Sub-Workflow** under the **Control Flow** category. Drag it onto the canvas.

The node renders with a distinctive double-border rectangle to visually distinguish it from regular nodes:

```
┌─────────────────────┐
│ ┌─────────────────┐ │
│ │  📦 Sub-Workflow │ │
│ │                 │ │
│ │  Ref: Email...  │ │
│ └─────────────────┘ │
└─────────────────────┘
```

**3. Configure the reference.**

Click the Sub-Workflow node to open the config panel on the right. Set these fields:

| Field | Description |
|---|---|
| **Referenced Workflow** | The ID of the child workflow to execute (required) |
| **Input Mapping** | JSON mapping parent data to child inputs, e.g., `{"recipient": "upstream.email"}` |
| **Output Mapping** | JSON mapping child outputs back to parent, e.g., `{"status": "child.result"}` |

**4. Wire it into your flow.**

The Sub-Workflow node has three ports:

| Port | Type | Description |
|---|---|---|
| **Input** (left) | Data | Receives data from upstream nodes |
| **Output** (right) | Data | Passes results to downstream nodes |
| **Error** (bottom) | Error | Routes errors to error handlers |

Connect it like any other node — wire data in, wire results out.

**5. Drill into the child workflow.**

**Double-click** the Sub-Workflow node to navigate into the referenced workflow. The breadcrumb trail at the top updates to show:

```
Workflows / Parent Workflow / Child Workflow (sub-workflow)
```

Click the parent name in the breadcrumb to navigate back.

#### Validation

Sub-workflow validation catches common mistakes before deploy:

| Validation | Severity | Description |
|---|---|---|
| `SUBWORKFLOW_NO_REFERENCE` | Error | No workflow selected in the Referenced Workflow field |
| `SUBWORKFLOW_SELF_REFERENCE` | Error | Workflow references itself |
| `SUBWORKFLOW_CIRCULAR_REFERENCE` | Error | A → B → A (or longer cycles like A → B → C → A) |
| `SUBWORKFLOW_NOT_FOUND` | Error | Referenced workflow doesn't exist |
| `SUBWORKFLOW_NESTING_DEPTH` | Warning | Nesting exceeds 5 levels deep |

#### Sub-Workflow YAML Definition

The built-in sub-workflow node is defined in `config/nodes/control/sub-workflow.yaml`:

```yaml
apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: control-sub-workflow
  name: Sub-Workflow
  description: "References another workflow as a reusable building block"
  icon: "📦"
category:
  group: Control Flow
  order: 30
shape:
  type: sub-workflow        # renders with double-border rectangle
  width: 220
ports:
  inputs:
    - id: in-main
      label: "Input"
      type: data
  outputs:
    - id: out-main
      label: "Output"
      type: data
    - id: out-error
      label: "Error"
      type: error
attributes:
  - id: workflow_ref
    label: "Referenced Workflow"
    type: workflow-reference
    required: true
    display: both
  - id: input_mapping
    label: "Input Mapping"
    type: json
    default: "{}"
  - id: output_mapping
    label: "Output Mapping"
    type: json
    default: "{}"
```

### Keyboard Shortcuts

Press `?` to see the full shortcut map. Key bindings:

| Shortcut | Action |
|---|---|
| Ctrl/Cmd+Z | Undo |
| Ctrl/Cmd+Shift+Z | Redo |
| Ctrl/Cmd+C/X/V | Copy/Cut/Paste |
| Ctrl/Cmd+D | Duplicate |
| Ctrl/Cmd+A | Select All |
| Del/Backspace | Delete selection |
| Tab / Shift+Tab | Cycle through nodes |
| Ctrl/Cmd+=/-/0 | Zoom in/out/fit |
| Space+Drag | Pan canvas |
| Esc | Deselect |
| ? | Toggle shortcut help |

### Accessibility

Graphiti targets WCAG 2.1 AA compliance:

- All SVG nodes and edges have `aria-label` descriptions
- Canvas operations are announced to screen readers via a live region
- `:focus-visible` outlines are visible in both light and dark themes
- The theme toggle includes dynamic `aria-label` describing current mode
- Form labels in the config panel are properly associated with inputs

## Sample Execution Engine

Graphiti ships with a reference execution engine in `examples/engine/` that demonstrates the full deploy-execute-callback loop. It turns Graphiti from a visual editor into a live system where you can build a REST API backed by PostgreSQL entirely by wiring nodes together.

### What It Does

The sample engine is a standalone Go HTTP server that:

1. **Receives deploy webhooks** from Graphiti (with HMAC signature verification)
2. **Registers API routes** from the workflow's API Gateway nodes
3. **Executes workflows** in topological order when HTTP requests arrive
4. **Reports status back** to Graphiti via callbacks, lighting up nodes in real-time

When you deploy a workflow containing an API Gateway node with `path: /api/todos`, the engine starts accepting requests at that path and runs each request through the workflow graph.

### Quick Start with Docker Compose

The fastest way to see everything working together:

```bash
docker compose up --build
```

This starts three services:

| Service | Port | Description |
|---|---|---|
| Graphiti | `localhost:8080` | Workflow builder UI |
| Engine | `localhost:9090` | Sample execution engine |
| PostgreSQL | `localhost:5432` | Database for the ToDo API |

### Building the ToDo API

Once the stack is running, open `http://localhost:8080` (auto-authenticated in dev mode) and create a new workflow named "ToDo API". Then follow the steps below.

#### Step 1: Add the API Gateway

Drag **API Gateway** from the **Sources** category onto the canvas. Click on it to open the config panel, set these values, and press **Save**:

| Field | Value |
|---|---|
| Path | `/api/todos` |
| Allowed Methods | `ALL` |
| Authentication | `none` |

This is the entry point — every HTTP request to `/api/todos` enters the workflow here.

#### Step 2: Add the HTTP Router

Drag **HTTP Router** from the **Control** category to the right of the gateway. Configure it:

| Field | Value |
|---|---|
| ID Path Parameter | `id` |

Wire the API Gateway's **Request** output port to the HTTP Router's **Request** input port. The router inspects each request's HTTP method and sends it to the matching output port (GET, GET :id, POST, PUT, DELETE).

#### Step 3: Add the "List Todos" branch (GET)

Drag a **PostgreSQL** node from **Destinations**. Rename it to "List Todos" and configure:

| Field | Value |
|---|---|
| Connection String | `${DATABASE_URL}` |
| Operation | `query` |
| SQL | `SELECT id, title, completed, created_at, updated_at FROM todos ORDER BY created_at DESC` |
| Parameters | `[]` |

Drag an **HTTP Response** node. Rename it to "200 OK (List)" and configure:

| Field | Value |
|---|---|
| Status Code | `200` |
| Content Type | `application/json` |

Wire: **HTTP Router** `GET` → **List Todos** → **200 OK (List)**

#### Step 4: Add the "Get Todo" branch (GET :id)

Drag another **PostgreSQL** node. Rename it to "Get Todo" and configure:

| Field | Value |
|---|---|
| Connection String | `${DATABASE_URL}` |
| Operation | `query-row` |
| SQL | `SELECT id, title, completed, created_at, updated_at FROM todos WHERE id = $1` |
| Parameters | `["pathParams.id"]` |

Drag another **HTTP Response** node. Rename it to "200 OK (Get)" with Status Code `200`.

Wire: **HTTP Router** `GET :id` → **Get Todo** → **200 OK (Get)**

#### Step 5: Add the "Create Todo" branch (POST)

Drag a **Validate Payload** node from **Processing**. Rename it to "Validate Create" and configure:

| Field | Value |
|---|---|
| Required Fields | `title` |
| Field Types | `{"title": "string"}` |
| Max Body Size | `256` |

Drag a **PostgreSQL** node. Rename it to "Create Todo" and configure:

| Field | Value |
|---|---|
| Connection String | `${DATABASE_URL}` |
| Operation | `query-row` |
| SQL | `INSERT INTO todos (title, completed) VALUES ($1, false) RETURNING id, title, completed, created_at, updated_at` |
| Parameters | `["body.title"]` |

Drag an **HTTP Response** node. Rename it to "201 Created" with Status Code `201`.

Wire: **HTTP Router** `POST` → **Validate Create** → **Create Todo** → **201 Created**

#### Step 6: Add the "Update Todo" branch (PUT)

Drag another **Validate Payload** node. Rename it to "Validate Update" and configure:

| Field | Value |
|---|---|
| Required Fields | `title` |
| Field Types | `{"title": "string", "completed": "boolean"}` |
| Max Body Size | `256` |

Drag a **PostgreSQL** node. Rename it to "Update Todo" and configure:

| Field | Value |
|---|---|
| Connection String | `${DATABASE_URL}` |
| Operation | `query-row` |
| SQL | `UPDATE todos SET title = $1, completed = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3 RETURNING id, title, completed, created_at, updated_at` |
| Parameters | `["body.title", "body.completed", "pathParams.id"]` |

Drag an **HTTP Response** node. Rename it to "200 OK (Update)" with Status Code `200`.

Wire: **HTTP Router** `PUT` → **Validate Update** → **Update Todo** → **200 OK (Update)**

#### Step 7: Add the "Delete Todo" branch (DELETE)

Drag a **PostgreSQL** node. Rename it to "Delete Todo" and configure:

| Field | Value |
|---|---|
| Connection String | `${DATABASE_URL}` |
| Operation | `exec` |
| SQL | `DELETE FROM todos WHERE id = $1` |
| Parameters | `["pathParams.id"]` |

Drag an **HTTP Response** node. Rename it to "204 No Content" with Status Code `204`.

Wire: **HTTP Router** `DELETE` → **Delete Todo** → **204 No Content**

#### Step 8: Add validation error handling

Drag one more **HTTP Response** node. Rename it to "400 Bad Request" with Status Code `400`.

Wire both validators' **Invalid** error ports to this node:
- **Validate Create** `Invalid` → **400 Bad Request**
- **Validate Update** `Invalid` → **400 Bad Request**

#### Step 9: Deploy

Click **Deploy** in the toolbar. Graphiti signs the workflow definition with HMAC-SHA256 and sends it to the engine. You should see a success toast.

#### Loading the pre-built workflow

If you prefer to skip the manual build, a ready-made workflow JSON is included at `examples/workflows/todo-api.json`. Currently, the way to load it is to import it via the API:

```bash
curl -X POST http://localhost:8080/api/workflows \
  -H "Content-Type: application/json" \
  -d @examples/workflows/todo-api.json
```

Then refresh the dashboard and open the "ToDo REST API" workflow.

### Testing the API

After deploying, use curl to interact with your workflow-powered API:

```bash
# Create a todo
curl -X POST http://localhost:9090/api/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Buy milk"}'

# List all todos
curl http://localhost:9090/api/todos

# Get a specific todo
curl http://localhost:9090/api/todos/1

# Update a todo
curl -X PUT http://localhost:9090/api/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"title": "Buy oat milk", "completed": true}'

# Delete a todo
curl -X DELETE http://localhost:9090/api/todos/1
```

While the API processes requests, switch to **Execution mode** in the Graphiti UI to watch nodes light up in real-time as data flows through the workflow.

### Running the Engine Standalone

If you prefer to run without Docker:

```bash
# Start PostgreSQL (any method)
# export DATABASE_URL=postgres://user:pass@localhost:5432/graphiti?sslmode=disable

# Start Graphiti
task run

# Start the engine
cd examples/engine
export DATABASE_URL=postgres://user:pass@localhost:5432/graphiti?sslmode=disable
export GRAPHITI_CALLBACK_URL=http://localhost:8080/api/callbacks/execution
export DEPLOY_HMAC_SECRET=your-shared-secret
go run .
```

### How the Engine Works

The engine processes each HTTP request through the deployed workflow graph:

```
curl POST /api/todos {"title":"Buy milk"}
  |
  v
API Gateway (receives request, emits structured data)
  |
  v
HTTP Router (inspects method, routes to POST branch)
  |
  v
Validate Payload (checks required fields)
  |-- valid --> PostgreSQL INSERT (creates the todo)
  |                |
  |                v
  |             HTTP Response (201 Created)
  |
  |-- invalid --> HTTP Response (400 Bad Request)
```

Each node is executed by a **NodeExecutor** that implements the node type's logic:

| Node Type | Executor | What It Does |
|---|---|---|
| `source-api-gateway` | `APIGatewayExecutor` | Passes through incoming request data |
| `control-http-router` | `HTTPRouterExecutor` | Routes to output port based on HTTP method |
| `processing-validate-payload` | `ValidatePayloadExecutor` | Checks required fields, routes valid/invalid |
| `destination-postgresql` | `PostgreSQLExecutor` | Runs parameterized SQL against Postgres |
| `destination-http-response` | `HTTPResponseExecutor` | Packages data into HTTP response shape |
| `processing-json-transform` | `JSONTransformExecutor` | Reshapes JSON using a mapping template |

### Building Your Own Engine

The sample engine demonstrates the contract between Graphiti and any execution engine:

**1. Receive deploys** via `POST /deploy` with HMAC-signed payload:
```
X-Graphiti-Signature: sha256=<hex digest>
X-Graphiti-Event: workflow.deployed
```

**2. Parse the workflow definition** from `payload.workflow.definition` which contains `nodes` and `edges`.

**3. Build an execution graph** and run nodes in topological order, passing data along edges.

**4. Report status** back to Graphiti via `POST /api/callbacks/execution`:
```json
{
  "apiVersion": "graphiti/v1",
  "event": "node.status",
  "runID": "run-123",
  "workflowID": "wf-001",
  "nodeID": "node-abc",
  "status": "completed",
  "startedAt": "2026-03-10T14:30:00Z",
  "completedAt": "2026-03-10T14:30:01Z"
}
```

Sign callbacks with the same HMAC secret using `X-Graphiti-Signature`.

The engine is ~600 lines of Go across 5 files. Read the source in `examples/engine/` for the complete reference implementation.

### Engine Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `9090` | Engine HTTP port |
| `DATABASE_URL` | *(none)* | PostgreSQL connection string |
| `GRAPHITI_CALLBACK_URL` | `http://localhost:8080/api/callbacks/execution` | Graphiti callback endpoint |
| `DEPLOY_HMAC_SECRET` | *(none)* | Shared secret for webhook signing |

## Configuration

### Environment Variables

Create a `.env` file (see `.env.example`):

```bash
# Required
SESSION_SECRET=your-random-secret-here

# Required for GitHub OAuth (when auth.provider is "github")
GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...

# Required for deploy functionality
DEPLOY_HMAC_SECRET=your-webhook-signing-secret
DEPLOY_WEBHOOK_PROD=https://your-engine.example.com/deploy
DEPLOY_WEBHOOK_STAGING=https://staging-engine.example.com/deploy
```

### App Config

See [`config/app.yaml`](config/app.yaml) for server, storage, and deploy settings. All `${VAR}` references are expanded from environment variables (loaded from `.env`).

## Implementation Status

### Phase 1: Foundation — **Complete**

- [x] Domain models (Workflow, Node, Edge, Ports, Attributes)
- [x] Command Pattern with all 9 command types and inverse logic
- [x] Undo/redo stack with configurable depth
- [x] Port interfaces (driving and driven)
- [x] YAML node definition parser with validation
- [x] In-memory node registry with search
- [x] Fake auth adapter with session middleware
- [x] SQLite adapter with migrations and WAL mode
- [x] Templ-based three-panel layout (palette, canvas, config panel)
- [x] SVG canvas with dot grid, node shapes, ports, and Bezier edges
- [x] HTTP router with protected routes and HTMX endpoints
- [x] Application service layer wiring everything together
- [x] Config loader with `.env` support
- [x] 4 sample node definitions (source, processing, destination, control)
- [x] Light and dark CSS themes

### Phase 2: Interactive Canvas — **Complete**

- [x] Pan/zoom canvas engine (mouse wheel, trackpad, middle-click drag, fit-to-view)
- [x] Drag nodes from palette to canvas with ghost preview
- [x] Move single and multiple nodes with grid snapping
- [x] Draw edges between ports with live Bezier curve preview
- [x] Connection validation (port type matching, self-connection prevention)
- [x] Single select, Ctrl+click toggle, rubber band multi-select
- [x] Delete selection (Delete/Backspace), Escape to deselect, Ctrl+A select all
- [x] Undo/redo wired to Ctrl+Z / Ctrl+Shift+Z with full state sync
- [x] Copy/cut/paste/duplicate node groups via server-side clipboard
- [x] Config panel loads via HTMX on node selection, saves via PATCH with debounce
- [x] Theme toggle (light/dark) with localStorage persistence
- [x] Keyboard shortcut help overlay (press `?`)
- [x] Full workflow state sync after every command (server returns state, JS diffs SVG DOM)
- [x] HTTP handler tests for all canvas endpoints (commands, undo/redo, clipboard, attributes)

### Phase 3: Persistence & Deploy — **Complete**

- [x] Workflow CRUD with dashboard listing (create, open, delete with confirmation)
- [x] Auto-save on every canvas mutation (saves to SQLite after each command)
- [x] Pre-deploy validation with error highlighting (nodes highlighted red on failure)
- [x] Webhook deploy adapter with HMAC-SHA256 signing and exponential backoff retry
- [x] Deploy dropdown menu (Production, Staging, Save as Draft)
- [x] Export workflow as YAML/JSON with file download
- [x] Workflow versioning on deploy (version snapshots stored in `workflow_versions` table)
- [x] Version badge displayed in builder nav bar
- [x] Deploy/export HTTP handlers with tests
- [x] SQLite `CreateVersion` repository method with cascade delete tests

### Phase 4: Auth & Execution View — **Complete**

- [x] GitHub OAuth2 adapter with org membership verification
- [x] Production session management with CSRF protection and rate limiting
- [x] Execution mode with run history (builder/execution mode tabs)
- [x] Per-node status overlays (pending, running, completed, failed, skipped)
- [x] Execution callback endpoint for engine status updates (HMAC-authenticated)
- [x] WebSocket for live execution updates (raw HTTP hijack, no external deps)

### Phase 4.5: Sample Execution Engine — **Complete**

- [x] 6 new node definitions (API Gateway, HTTP Router, Validate Payload, PostgreSQL, HTTP Response, JSON Transform)
- [x] Sample execution engine with topological-order workflow runner
- [x] Node executors for all node types (router, validator, SQL, response, transform)
- [x] HMAC-signed deploy webhook receiver with signature verification
- [x] Status callback sender with HMAC signing for live execution updates
- [x] Path parameter matching for REST-style routes (`/api/todos/:id`)
- [x] Pre-built ToDo REST API workflow definition
- [x] Docker Compose stack (Graphiti + Engine + PostgreSQL)
- [x] 24 engine tests covering types, graph, executors, runner, and HTTP handlers

### Phase 5: Advanced Features — **Complete**

- [x] Sub-workflow node type with YAML definition, double-border rendering, and validation (circular reference detection, nesting depth limits)
- [x] Sub-workflow drill-in navigation with parent context breadcrumbs and `handleNodeRef` API endpoint
- [x] Theme system with Light/Dark/System toggle (persisted to localStorage, respects `prefers-color-scheme`)
- [x] Viewport culling and level-of-detail (nodes outside viewport are hidden; text/ports simplified at low zoom)
- [x] Full keyboard shortcut map with Tab/Shift+Tab node cycling and `?` help overlay
- [x] Accessibility: ARIA labels on SVG nodes/edges, screen reader live region announcements, `:focus-visible` outlines, `.sr-only` utility, accessible theme toggle

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.25 |
| Templates | [Templ](https://templ.guide/) |
| UI Interactivity | [HTMX](https://htmx.org/) + vanilla JS |
| Canvas | SVG |
| Database | SQLite ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), pure Go) |
| Config | YAML ([gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)) |
| Auth | Pluggable (fake for dev, GitHub OAuth for prod) |
| Build | [Task](https://taskfile.dev/) |

## Contributing

Contributions are welcome! Please read the [architecture guidelines](CLAUDE.md) before submitting changes.

Key rules:
1. **TDD** — Write tests first, watch them fail, then implement
2. **Hex boundary** — `internal/domain/` must never import external packages
3. **Command Pattern** — Every canvas mutation goes through a Command, never modify workflows directly
4. **Auth always on** — Every handler sits behind auth middleware, even in dev

```bash
# Run tests before submitting
task test

# Check coverage
task test:cover

# Lint
task lint
```

## License

MIT

---

<p align="center">
  <sub>Built with Go, HTMX, Templ, and SVG. No React was harmed in the making of this project.</sub>
</p>
