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

The project ships with four sample nodes:

| Node | Category | Shape | Description |
|---|---|---|---|
| Twilio | Sources | Rounded rect | Receives call recording callbacks |
| Transcribe | Processing | Rounded rect | Speech-to-text conversion |
| S3 Storage | Destinations | Rounded rect | Store files to S3 |
| Condition | Control | Diamond | Branch based on expression |

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

### Themes

Light and dark themes are defined via CSS custom properties. The UI respects `data-theme` on `<html>`:

```css
/* All colors reference variables */
[data-theme="light"] { --bg: #F8F9FB; --surface: #FFFFFF; --accent: #3B82F6; }
[data-theme="dark"]  { --bg: #0F1117; --surface: #1A1D27; --accent: #60A5FA; }
```

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

### Phase 3: Persistence & Deploy — Planned

- [ ] Workflow CRUD with dashboard listing
- [ ] Auto-save on every canvas mutation (debounced)
- [ ] Pre-deploy validation with error highlighting
- [ ] Webhook deploy with HMAC signing and retry
- [ ] Export workflow as YAML/JSON
- [ ] Workflow versioning on deploy

### Phase 4: Auth & Execution View — Planned

- [ ] GitHub OAuth2 adapter
- [ ] Production session management with CSRF protection
- [ ] Execution mode with run history
- [ ] Per-node status overlays (running, completed, failed)
- [ ] Execution callback endpoint for engine status updates
- [ ] WebSocket for live execution updates

### Phase 5: Advanced Features — Planned

- [ ] Sub-workflow node type with drill-in navigation
- [x] Theme system toggle (light/dark) — *completed in Phase 2*
- [ ] Viewport culling and level-of-detail for large workflows
- [x] Full keyboard shortcut map with help overlay — *completed in Phase 2*
- [ ] Accessibility audit (WCAG 2.1 AA)

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
