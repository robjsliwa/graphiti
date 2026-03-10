# Graphiti Implementation Plan

**Version:** 0.1.0
**Based on:** Graphiti PRD v0.1.0-draft (2026-03-09)
**Approach:** Phase-based, TDD-first, hexagonal architecture from the ground up

## How to Read This Plan

Each phase builds on the one before it, like stacking bricks. At the end of every phase you'll have something you can run, click around in, and demo. No phase leaves you with a pile of parts that don't work together yet.

Every story follows a strict TDD cycle: write the test first, watch it fail, write the minimum code to pass, then refactor. No exceptions.

## Architecture Foundation

Before diving into phases, here's the architectural contract that every line of code must respect. Think of hexagonal architecture like a medieval castle. The keep (domain) sits at the center and knows nothing about the outside world. The walls (ports) define the shape of every doorway in and out. The buildings outside the walls (adapters) can be swapped without the keep ever knowing.

```mermaid
graph TB
    subgraph "Driving Side - Who Calls Us"
        HTTP["HTTP/HTMX Handlers"]
        WS["WebSocket Handler"]
        CLI["CLI / Config Loader"]
    end

    subgraph "Application Core"
        subgraph "Ports"
            DP_IN["Driving Ports<br/>(Use Case Interfaces)"]
            DP_OUT["Driven Ports<br/>(Repository Interfaces)"]
        end
        subgraph "Domain - Pure, Zero Dependencies"
            WF["Workflow Aggregate"]
            ND["Node Definition"]
            CMD["Command + History"]
            CB["Clipboard"]
            EX["Execution Run"]
            USR["User"]
        end
        subgraph "App Services"
            AS["Use Case Implementations"]
        end
    end

    subgraph "Driven Side - Who We Call"
        SQL["SQLite Adapter"]
        MEM["In-Memory Adapter"]
        FS["Filesystem Adapter"]
        AUTH["Auth Adapter (Fake → GitHub)"]
        HOOK["Webhook Adapter"]
    end

    HTTP --> DP_IN
    WS --> DP_IN
    CLI --> DP_IN
    DP_IN --> AS
    AS --> WF
    AS --> ND
    AS --> CMD
    AS --> EX
    AS --> DP_OUT
    DP_OUT --> SQL
    DP_OUT --> MEM
    DP_OUT --> FS
    DP_OUT --> AUTH
    DP_OUT --> HOOK
```

**The Golden Rule:** The `internal/domain/` package imports nothing from `internal/ports/`, `internal/adapters/`, or any external library. It's pure Go with zero dependencies. If you find yourself importing `net/http` or `database/sql` inside domain, you've crossed the wall.

### Project Structure

Every file lives in a deliberate location. This structure is non-negotiable because it enforces the hexagonal boundaries at the filesystem level.

```
graphiti/
├── cmd/
│   └── server/
│       └── main.go                    # Entry point, wires adapters to ports
├── internal/
│   ├── domain/                        # Pure domain models, NO external imports
│   │   ├── workflow.go
│   │   ├── workflow_test.go
│   │   ├── node_definition.go
│   │   ├── node_definition_test.go
│   │   ├── node_instance.go
│   │   ├── edge.go
│   │   ├── command.go                 # Command interface
│   │   ├── command_test.go
│   │   ├── commands/                  # Concrete command implementations
│   │   │   ├── add_node.go
│   │   │   ├── add_node_test.go
│   │   │   ├── remove_node.go
│   │   │   ├── remove_node_test.go
│   │   │   ├── move_node.go
│   │   │   ├── move_node_test.go
│   │   │   ├── add_edge.go
│   │   │   ├── add_edge_test.go
│   │   │   ├── remove_edge.go
│   │   │   ├── remove_edge_test.go
│   │   │   ├── update_attribute.go
│   │   │   ├── update_attribute_test.go
│   │   │   ├── paste_nodes.go
│   │   │   ├── paste_nodes_test.go
│   │   │   └── rename_node.go
│   │   ├── command_history.go
│   │   ├── command_history_test.go
│   │   ├── clipboard.go
│   │   ├── clipboard_test.go
│   │   ├── execution.go
│   │   ├── user.go
│   │   └── validation.go
│   ├── ports/                         # Interface definitions only
│   │   ├── driving/
│   │   │   ├── workflow_service.go
│   │   │   ├── node_registry.go
│   │   │   └── execution_service.go
│   │   └── driven/
│   │       ├── workflow_repo.go
│   │       ├── execution_repo.go
│   │       ├── user_repo.go
│   │       ├── auth_provider.go
│   │       └── deploy_target.go
│   ├── app/                           # Application services (use case impl)
│   │   ├── workflow_service.go
│   │   ├── workflow_service_test.go
│   │   ├── node_registry.go
│   │   ├── node_registry_test.go
│   │   ├── execution_service.go
│   │   └── execution_service_test.go
│   └── adapters/
│       ├── driving/
│       │   ├── http/
│       │   │   ├── router.go
│       │   │   ├── handlers_workflow.go
│       │   │   ├── handlers_node.go
│       │   │   ├── handlers_command.go
│       │   │   ├── handlers_auth.go
│       │   │   ├── handlers_execution.go
│       │   │   ├── middleware.go
│       │   │   └── websocket.go
│       │   └── cli/
│       │       └── config_loader.go
│       └── driven/
│           ├── sqlite/
│           │   ├── workflow_repo.go
│           │   ├── workflow_repo_test.go
│           │   ├── execution_repo.go
│           │   ├── user_repo.go
│           │   └── migrations.go
│           ├── memory/
│           │   ├── workflow_repo.go    # In-memory impl for tests
│           │   └── user_repo.go
│           ├── filesystem/
│           │   └── node_loader.go
│           ├── webhook/
│           │   ├── deploy_target.go
│           │   └── deploy_target_test.go
│           └── auth/
│               ├── fake.go            # Fake auth for local dev
│               ├── fake_test.go
│               ├── github.go
│               └── github_test.go
├── web/
│   ├── templates/                     # Templ files
│   │   ├── layouts/
│   │   │   └── base.templ
│   │   ├── pages/
│   │   │   ├── dashboard.templ
│   │   │   ├── workflow_builder.templ
│   │   │   └── login.templ
│   │   ├── partials/
│   │   │   ├── node_palette.templ
│   │   │   ├── node_palette_item.templ
│   │   │   ├── config_panel.templ
│   │   │   ├── execution_list.templ
│   │   │   └── deploy_menu.templ
│   │   └── svg/
│   │       ├── canvas.templ
│   │       ├── node.templ
│   │       ├── edge.templ
│   │       └── shapes/
│   │           ├── rounded_rect.templ
│   │           ├── diamond.templ
│   │           ├── hexagon.templ
│   │           └── sub_workflow.templ
│   └── static/
│       ├── js/
│       │   ├── canvas.js
│       │   ├── drag.js
│       │   ├── connect.js
│       │   ├── select.js
│       │   ├── commands.js
│       │   ├── clipboard.js
│       │   └── theme.js
│       └── css/
│           ├── base.css
│           ├── components.css
│           ├── panels.css
│           ├── canvas.css
│           ├── nodes.css
│           └── themes/
│               ├── light.css
│               └── dark.css
├── config/
│   ├── app.yaml
│   ├── auth.yaml
│   ├── nodes/
│   │   ├── sources/
│   │   ├── processing/
│   │   ├── destinations/
│   │   └── control/
│   └── themes/
│       ├── light.yaml
│       └── dark.yaml
├── migrations/
│   └── 001_initial.sql
├── go.mod
├── Makefile
├── Dockerfile
└── README.md
```

### Authentication from Day One

Authentication is present in every phase, even the first. In early phases it's a fake adapter that auto-authenticates a hardcoded dev user. This means the auth middleware, session handling, and protected routes are all exercised from the start. When real OAuth2 lands in Phase 4, it's just swapping the adapter behind the same port interface.

```mermaid
graph LR
    subgraph "Phase 1-3"
        FA["FakeAuthAdapter<br/>Auto-login as dev@local"]
    end
    subgraph "Phase 4+"
        GA["GitHubAuthAdapter<br/>Real OAuth2 flow"]
    end
    subgraph "Always Present"
        AP["AuthProvider Port Interface"]
        MW["Auth Middleware"]
        SM["Session Management"]
    end

    FA --> AP
    GA --> AP
    AP --> MW
    MW --> SM
```

## Phase 1: Foundation (The Skeleton Walks)

**Goal:** A running Go server that loads node definitions from YAML, stores workflows in SQLite, renders a three-panel layout with a categorized node palette, and displays static SVG nodes on a canvas. Fake auth protects all routes. Every domain model has tests. At demo time you can open a browser, see the palette, and see nodes rendered on a canvas.

**Estimated Duration:** 3-4 weeks

```mermaid
graph LR
    A["Domain Models<br/>+ Commands"] --> B["YAML Parser<br/>+ Node Registry"]
    B --> C["SQLite Adapter<br/>+ Migrations"]
    C --> D["Fake Auth<br/>+ Middleware"]
    D --> E["Templ Layout<br/>+ Palette"]
    E --> F["Static SVG<br/>Node Rendering"]
    style F fill:#10B981,color:#fff
```

### Story 1.1: Domain Models

**What:** Define the core domain types that everything else builds on. These are pure Go structs with no external dependencies.

**Code structure:**

```go
// internal/domain/workflow.go
package domain

import "time"

type WorkflowStatus string

const (
    WorkflowStatusDraft    WorkflowStatus = "draft"
    WorkflowStatusDeployed WorkflowStatus = "deployed"
    WorkflowStatusArchived WorkflowStatus = "archived"
)

type Workflow struct {
    ID          string
    Name        string
    Description string
    Status      WorkflowStatus
    Version     int
    Nodes       []NodeInstance
    Edges       []Edge
    CreatedBy   string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// AddNode creates a new node instance on this workflow at the given position.
func (w *Workflow) AddNode(def *NodeDefinition, x, y float64, instanceID string) *NodeInstance { ... }

// RemoveNode removes a node and all its connected edges.
func (w *Workflow) RemoveNode(nodeID string) (*NodeInstance, []Edge, error) { ... }

// AddEdge connects two ports after validating compatibility.
func (w *Workflow) AddEdge(sourceNodeID, sourcePortID, targetNodeID, targetPortID, edgeID string) error { ... }

// RemoveEdge disconnects two ports.
func (w *Workflow) RemoveEdge(edgeID string) (*Edge, error) { ... }

// FindNode returns the node with the given ID or nil.
func (w *Workflow) FindNode(nodeID string) *NodeInstance { ... }

// FindEdge returns the edge with the given ID or nil.
func (w *Workflow) FindEdge(edgeID string) *Edge { ... }

// Validate checks all invariants and returns any errors.
func (w *Workflow) Validate() []ValidationError { ... }
```

```go
// internal/domain/node_definition.go
package domain

type NodeDefinition struct {
    APIVersion  string
    Kind        string
    ID          string
    Name        string
    Description string
    Version     string
    Icon        string
    IconSvg     string
    Category    Category
    Shape       Shape
    Inputs      []PortDefinition
    Outputs     []PortDefinition
    Attributes  []AttributeDefinition
    Validation  ValidationRules
}

type Category struct {
    Group string
    Order int
}

type ShapeType string

const (
    ShapeRoundedRect ShapeType = "rounded-rect"
    ShapePill        ShapeType = "pill"
    ShapeDiamond     ShapeType = "diamond"
    ShapeHexagon     ShapeType = "hexagon"
    ShapeSubWorkflow ShapeType = "sub-workflow"
    ShapeCustom      ShapeType = "custom"
)

type Shape struct {
    Type            ShapeType
    Width           int
    MinWidth        int
    MaxWidth        int
    HeaderColor     string
    HeaderBackground string
    CustomSvg       string
}

type PortType string

const (
    PortTypeData    PortType = "data"
    PortTypeControl PortType = "control"
    PortTypeError   PortType = "error"
)

type PortPosition string

const (
    PortPositionLeftCenter   PortPosition = "left-center"
    PortPositionRightCenter  PortPosition = "right-center"
    PortPositionTopCenter    PortPosition = "top-center"
    PortPositionBottomCenter PortPosition = "bottom-center"
    PortPositionRightTop     PortPosition = "right-top"
    PortPositionRightBottom  PortPosition = "right-bottom"
)

type PortDefinition struct {
    ID             string
    Label          string
    Type           PortType
    Position       PortPosition
    MaxConnections int // -1 = unlimited
}

type AttributeType string

const (
    AttrTypeString      AttributeType = "string"
    AttrTypeText        AttributeType = "text"
    AttrTypeNumber      AttributeType = "number"
    AttrTypeBoolean     AttributeType = "boolean"
    AttrTypeEnum        AttributeType = "enum"
    AttrTypeSecret      AttributeType = "secret"
    AttrTypeJSON        AttributeType = "json"
    AttrTypeExpression  AttributeType = "expression"
    AttrTypePortMapping AttributeType = "port-mapping"
    AttrTypeWorkflowRef AttributeType = "workflow-reference"
)

type DisplayLocation string

const (
    DisplayNodeBody    DisplayLocation = "node-body"
    DisplayConfigPanel DisplayLocation = "config-panel"
    DisplayBoth        DisplayLocation = "both"
)

type AttributeDefinition struct {
    ID       string
    Label    string
    Type     AttributeType
    Default  any
    Required bool
    Display  DisplayLocation
    Group    string
    Options  []string    // for enum type
    Min      *float64    // for number type
    Max      *float64    // for number type
    Hint     string
}

type ConnectionRule struct {
    OutputPort              string
    AllowedTargetCategories []string
    AllowedTargetPorts      []string
}

type AttributeRule struct {
    Expression string
    Message    string
}

type ValidationRules struct {
    ConnectionRules []ConnectionRule
    AttributeRules  []AttributeRule
}
```

```go
// internal/domain/node_instance.go
package domain

type NodeInstance struct {
    ID              string
    DefinitionID    string
    Label           string
    X               float64
    Y               float64
    AttributeValues map[string]any
    Definition      *NodeDefinition // resolved reference
}
```

```go
// internal/domain/edge.go
package domain

type Edge struct {
    ID           string
    SourceNodeID string
    SourcePortID string
    TargetNodeID string
    TargetPortID string
}
```

```go
// internal/domain/user.go
package domain

import "time"

type User struct {
    ID             string
    Username       string
    Email          string
    AvatarURL      string
    AuthProvider   string
    AuthProviderID string
    CreatedAt      time.Time
    LastLoginAt    time.Time
}
```

```go
// internal/domain/execution.go
package domain

import "time"

type ExecutionStatus string

const (
    ExecStatusPending   ExecutionStatus = "pending"
    ExecStatusRunning   ExecutionStatus = "running"
    ExecStatusCompleted ExecutionStatus = "completed"
    ExecStatusFailed    ExecutionStatus = "failed"
    ExecStatusCancelled ExecutionStatus = "cancelled"
)

type NodeExecStatus string

const (
    NodeExecPending   NodeExecStatus = "pending"
    NodeExecRunning   NodeExecStatus = "running"
    NodeExecCompleted NodeExecStatus = "completed"
    NodeExecFailed    NodeExecStatus = "failed"
    NodeExecSkipped   NodeExecStatus = "skipped"
)

type ExecutionRun struct {
    ID              string
    WorkflowID      string
    WorkflowVersion int
    Status          ExecutionStatus
    StartedAt       time.Time
    CompletedAt     time.Time
    TriggerType     string
    TriggerData     map[string]any
    NodeStatuses    map[string]*NodeExecutionStatus
}

type NodeExecutionStatus struct {
    NodeID       string
    Status       NodeExecStatus
    StartedAt    time.Time
    CompletedAt  time.Time
    InputData    map[string]any
    OutputData   map[string]any
    ErrorMessage string
    Logs         []LogEntry
}

type LogEntry struct {
    Timestamp time.Time
    Level     string
    Message   string
}
```

```go
// internal/domain/validation.go
package domain

type ValidationError struct {
    NodeID  string // empty for workflow-level errors
    Field   string
    Message string
}
```

**Acceptance Criteria:**

- [ ] All domain types compile with zero external imports (only stdlib `time`, `encoding/json`, `errors`)
- [ ] `Workflow.AddNode` creates a `NodeInstance` with a UUID, default attribute values from the definition, and adds it to the workflow's node list
- [ ] `Workflow.RemoveNode` removes the node AND all edges connected to it, returning the removed node and edges for undo
- [ ] `Workflow.AddEdge` rejects edges where port types don't match (data-to-data only, etc.)
- [ ] `Workflow.AddEdge` rejects edges that would exceed a port's `maxConnections`
- [ ] `Workflow.Validate` detects cycles using topological sort and returns a `ValidationError`
- [ ] `Workflow.Validate` flags nodes with missing required attributes
- [ ] `Workflow.Validate` enforces connection rules from node definitions (allowed target categories)
- [ ] Unit tests cover all of the above, including edge cases (empty workflow, self-referencing edge, duplicate node IDs)
- [ ] Tests are written FIRST, following TDD red-green-refactor cycle
- [ ] Test coverage for the `domain` package is 95%+

### Story 1.2: Command Interface and All Command Implementations

**What:** Implement the Command Pattern infrastructure and every concrete command type. This is the backbone that enables undo/redo and prepares for collaborative editing. Think of each command as a recorded chess move: it knows how to play itself forward and how to rewind.

**Code structure:**

```go
// internal/domain/command.go
package domain

type Command interface {
    // Execute applies the mutation and returns the inverse command for undo.
    Execute(wf *Workflow) (Command, error)

    // Type returns a string identifier for serialization.
    Type() string

    // Serialize converts the command to JSON.
    Serialize() ([]byte, error)
}

// DeserializeCommand reconstructs a Command from its type string and JSON bytes.
func DeserializeCommand(cmdType string, data []byte) (Command, error) { ... }
```

```go
// internal/domain/commands/add_node.go
package commands

import "graphiti/internal/domain"

type AddNodeCommand struct {
    DefinitionID string  `json:"definitionId"`
    InstanceID   string  `json:"instanceId"`
    Label        string  `json:"label"`
    X            float64 `json:"x"`
    Y            float64 `json:"y"`
}

func (c *AddNodeCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
    node := wf.AddNode(/* resolved def */, c.X, c.Y, c.InstanceID)
    if node == nil {
        return nil, errors.New("failed to add node")
    }
    // Return inverse: a RemoveNodeCommand that captures the full node state
    return &RemoveNodeCommand{
        NodeID:         node.ID,
        SerializedNode: node, // full snapshot for restore
    }, nil
}

func (c *AddNodeCommand) Type() string { return "add_node" }
func (c *AddNodeCommand) Serialize() ([]byte, error) { return json.Marshal(c) }
```

```go
// internal/domain/commands/move_node.go
package commands

type MoveNodeCommand struct {
    NodeID string  `json:"nodeId"`
    FromX  float64 `json:"fromX"`
    FromY  float64 `json:"fromY"`
    ToX    float64 `json:"toX"`
    ToY    float64 `json:"toY"`
}

func (c *MoveNodeCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
    node := wf.FindNode(c.NodeID)
    if node == nil {
        return nil, ErrNodeNotFound
    }
    node.X = c.ToX
    node.Y = c.ToY
    // Inverse: move back to original position
    return &MoveNodeCommand{
        NodeID: c.NodeID,
        FromX:  c.ToX, FromY: c.ToY,
        ToX:    c.FromX, ToY: c.FromY,
    }, nil
}
```

**All command types to implement:**

| Command | Key Fields | Inverse |
|---|---|---|
| `AddNodeCommand` | definitionID, instanceID, x, y | `RemoveNodeCommand` with full node state |
| `RemoveNodeCommand` | nodeID, serializedNode, removedEdges | `AddNodeCommand` with full state restore |
| `MoveNodeCommand` | nodeID, fromX, fromY, toX, toY | `MoveNodeCommand` with positions swapped |
| `MoveNodesCommand` | []nodeID, []fromPos, []toPos | `MoveNodesCommand` with positions swapped |
| `AddEdgeCommand` | edgeID, sourceNodeID, sourcePortID, targetNodeID, targetPortID | `RemoveEdgeCommand` |
| `RemoveEdgeCommand` | edgeID, serializedEdge | `AddEdgeCommand` with full edge state |
| `UpdateAttributeCommand` | nodeID, attrID, oldValue, newValue | `UpdateAttributeCommand` with values swapped |
| `PasteNodesCommand` | []serializedNodes, []serializedEdges, idMapping | `RemoveNodesCommand` (batch) |
| `RenameNodeCommand` | nodeID, oldLabel, newLabel | `RenameNodeCommand` with labels swapped |

**Acceptance Criteria:**

- [ ] `Command` interface exists in `domain/command.go` with `Execute`, `Type`, and `Serialize` methods
- [ ] All 9 command types listed above are implemented with their inverse logic
- [ ] Every command is JSON-serializable and deserializable (round-trip test)
- [ ] `AddNodeCommand.Execute` returns a `RemoveNodeCommand` that captures the complete node state
- [ ] `RemoveNodeCommand.Execute` restores the node and all its connected edges
- [ ] `MoveNodeCommand.Execute` returns a `MoveNodeCommand` with from/to swapped
- [ ] `MoveNodesCommand` handles batch moves as a single operation
- [ ] `AddEdgeCommand.Execute` validates port compatibility before creating the edge
- [ ] `UpdateAttributeCommand` stores both old and new values for clean reversal
- [ ] `PasteNodesCommand` generates new UUIDs and remaps internal edges
- [ ] `DeserializeCommand` can reconstruct any command from its type string and JSON
- [ ] Each command type has its own `_test.go` file with table-driven tests
- [ ] Tests verify the full round-trip: execute → check state → undo (execute inverse) → check state restored → redo → check state applied again

### Story 1.3: Command History (Undo/Redo Stack)

**What:** The `CommandHistory` struct manages the undo and redo stacks. Every mutation flows through it. It's the single choke point that makes the whole system auditable and reversible.

**Code structure:**

```go
// internal/domain/command_history.go
package domain

import "errors"

var (
    ErrNothingToUndo = errors.New("nothing to undo")
    ErrNothingToRedo = errors.New("nothing to redo")
)

type CommandHistory struct {
    undoStack []Command
    redoStack []Command
    maxSize   int
}

func NewCommandHistory(maxSize int) *CommandHistory {
    return &CommandHistory{
        undoStack: make([]Command, 0),
        redoStack: make([]Command, 0),
        maxSize:   maxSize,
    }
}

func (h *CommandHistory) Execute(wf *Workflow, cmd Command) error {
    inverse, err := cmd.Execute(wf)
    if err != nil {
        return err
    }
    h.undoStack = append(h.undoStack, inverse)
    h.redoStack = nil // new action clears redo
    if len(h.undoStack) > h.maxSize {
        h.undoStack = h.undoStack[1:]
    }
    return nil
}

func (h *CommandHistory) Undo(wf *Workflow) error {
    if len(h.undoStack) == 0 {
        return ErrNothingToUndo
    }
    cmd := h.undoStack[len(h.undoStack)-1]
    h.undoStack = h.undoStack[:len(h.undoStack)-1]
    inverse, err := cmd.Execute(wf)
    if err != nil {
        return err
    }
    h.redoStack = append(h.redoStack, inverse)
    return nil
}

func (h *CommandHistory) Redo(wf *Workflow) error {
    if len(h.redoStack) == 0 {
        return ErrNothingToRedo
    }
    cmd := h.redoStack[len(h.redoStack)-1]
    h.redoStack = h.redoStack[:len(h.redoStack)-1]
    inverse, err := cmd.Execute(wf)
    if err != nil {
        return err
    }
    h.undoStack = append(h.undoStack, inverse)
    return nil
}

func (h *CommandHistory) CanUndo() bool { return len(h.undoStack) > 0 }
func (h *CommandHistory) CanRedo() bool { return len(h.redoStack) > 0 }
```

**Acceptance Criteria:**

- [ ] `CommandHistory.Execute` pushes the inverse command onto the undo stack
- [ ] `CommandHistory.Execute` clears the redo stack
- [ ] `CommandHistory.Undo` pops from the undo stack, executes the inverse, and pushes the result onto redo
- [ ] `CommandHistory.Redo` pops from the redo stack, executes it, and pushes the inverse onto undo
- [ ] Undo on an empty stack returns `ErrNothingToUndo`
- [ ] Redo on an empty stack returns `ErrNothingToRedo`
- [ ] Stack respects `maxSize`, dropping the oldest entries when full
- [ ] Integration test: add 3 nodes → undo all 3 → redo 2 → add new node → redo stack is cleared
- [ ] Integration test: move node A→B → undo → verify position is A → redo → verify position is B

### Story 1.4: Port Interfaces (Driving and Driven)

**What:** Define every port interface. These are the contracts that adapters must fulfill. No implementation yet, just the shapes of the doorways.

```go
// internal/ports/driving/workflow_service.go
package driving

import (
    "context"
    "graphiti/internal/domain"
)

type WorkflowService interface {
    CreateWorkflow(ctx context.Context, name, description, userID string) (*domain.Workflow, error)
    GetWorkflow(ctx context.Context, id string) (*domain.Workflow, error)
    ListWorkflows(ctx context.Context, userID string) ([]*domain.WorkflowSummary, error)
    SaveWorkflow(ctx context.Context, wf *domain.Workflow) error
    DeleteWorkflow(ctx context.Context, id string) error

    // Command execution
    ExecuteCommand(ctx context.Context, workflowID string, cmd domain.Command) (*CommandResult, error)
    Undo(ctx context.Context, workflowID string) (*CommandResult, error)
    Redo(ctx context.Context, workflowID string) (*CommandResult, error)

    // Clipboard
    CopyNodes(ctx context.Context, workflowID string, nodeIDs []string) (*domain.ClipboardPayload, error)
    PasteNodes(ctx context.Context, workflowID string, payload *domain.ClipboardPayload, x, y float64) (*CommandResult, error)

    // Deploy
    DeployWorkflow(ctx context.Context, workflowID, target, userID string) (*domain.DeployResult, error)

    // Export
    ExportWorkflow(ctx context.Context, workflowID, format string) ([]byte, error)
}

type CommandResult struct {
    AffectedNodeIDs []string
    AffectedEdgeIDs []string
    CanUndo         bool
    CanRedo         bool
}

// WorkflowSummary is a lightweight projection for listing.
type WorkflowSummary struct {
    ID        string
    Name      string
    Status    domain.WorkflowStatus
    Version   int
    UpdatedAt time.Time
}
```

```go
// internal/ports/driving/node_registry.go
package driving

type NodeRegistryService interface {
    GetAllDefinitions(ctx context.Context) ([]*domain.NodeDefinition, error)
    GetByCategory(ctx context.Context, category string) ([]*domain.NodeDefinition, error)
    GetByID(ctx context.Context, id string) (*domain.NodeDefinition, error)
    Search(ctx context.Context, query string) ([]*domain.NodeDefinition, error)
}
```

```go
// internal/ports/driven/workflow_repo.go
package driven

type WorkflowRepository interface {
    Create(ctx context.Context, wf *domain.Workflow) error
    GetByID(ctx context.Context, id string) (*domain.Workflow, error)
    List(ctx context.Context, filter WorkflowFilter) ([]*WorkflowSummary, error)
    Update(ctx context.Context, wf *domain.Workflow) error
    Delete(ctx context.Context, id string) error
    GetVersionHistory(ctx context.Context, id string) ([]*domain.WorkflowVersion, error)
}
```

```go
// internal/ports/driven/auth_provider.go
package driven

type AuthProvider interface {
    GetAuthURL(state string) string
    ExchangeCode(ctx context.Context, code string) (*TokenPair, error)
    GetUserInfo(ctx context.Context, accessToken string) (*domain.UserInfo, error)
    RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
}

type TokenPair struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
}
```

```go
// internal/ports/driven/deploy_target.go
package driven

type DeployTarget interface {
    Deploy(ctx context.Context, payload domain.DeployPayload) (*domain.DeployResult, error)
}
```

```go
// internal/ports/driven/user_repo.go
package driven

type UserRepository interface {
    Upsert(ctx context.Context, user *domain.User) error
    GetByID(ctx context.Context, id string) (*domain.User, error)
    GetByProviderID(ctx context.Context, provider, providerID string) (*domain.User, error)
}
```

```go
// internal/ports/driven/execution_repo.go
package driven

type ExecutionRepository interface {
    Create(ctx context.Context, run *domain.ExecutionRun) error
    GetByID(ctx context.Context, id string) (*domain.ExecutionRun, error)
    ListByWorkflow(ctx context.Context, workflowID string, filter ExecutionFilter) ([]*domain.ExecutionRunSummary, error)
    UpdateNodeStatus(ctx context.Context, runID, nodeID string, status domain.NodeExecutionStatus) error
    AppendNodeLog(ctx context.Context, runID, nodeID string, entry domain.LogEntry) error
}
```

```go
// internal/ports/driven/node_definition_repo.go
package driven

type NodeDefinitionRepository interface {
    LoadAll(ctx context.Context) ([]*domain.NodeDefinition, error)
    GetByID(ctx context.Context, id string) (*domain.NodeDefinition, error)
    GetByCategory(ctx context.Context, category string) ([]*domain.NodeDefinition, error)
    Search(ctx context.Context, query string) ([]*domain.NodeDefinition, error)
}
```

**Acceptance Criteria:**

- [ ] All port interfaces compile and are documented with GoDoc comments
- [ ] Port packages import only from `domain` (never from `adapters` or `app`)
- [ ] Driving ports define what the outside world can ask us to do (use cases)
- [ ] Driven ports define what we need the outside world to provide (storage, auth, deploy)
- [ ] Each port interface has a corresponding mock generated (using testify/mock or manual fakes) for testing

### Story 1.5: YAML Node Definition Parser and In-Memory Registry

**What:** Read YAML files from disk, validate them, and build an in-memory node registry that the palette and canvas use.

```go
// internal/adapters/driven/filesystem/node_loader.go
package filesystem

type YAMLNodeLoader struct {
    basePath string
}

func NewYAMLNodeLoader(basePath string) *YAMLNodeLoader { ... }

// LoadAll walks the directory tree, parses every .yaml file,
// validates it, and returns the valid definitions.
// Invalid files are logged and skipped.
func (l *YAMLNodeLoader) LoadAll(ctx context.Context) ([]*domain.NodeDefinition, error) { ... }
```

**Sample YAML for testing:**

```yaml
# config/nodes/sources/twilio.yaml
apiVersion: graphiti/v1
kind: NodeDefinition
metadata:
  id: source-twilio
  name: Twilio
  description: "Receives call recording callbacks from Twilio"
  version: "1.0.0"
  icon: "T"
category:
  group: Sources
  order: 20
shape:
  type: rounded-rect
  width: 200
  minWidth: 160
  maxWidth: 320
  headerColor: "var(--cyan)"
  headerBackground: "var(--cyan-dim)"
ports:
  inputs: []
  outputs:
    - id: out-main
      label: "Output"
      type: data
      position: right-center
      maxConnections: -1
attributes:
  - id: endpoint
    label: "Endpoint"
    type: string
    default: "/ingest/twilio"
    required: true
    display: node-body
    group: "Connection"
  - id: format
    label: "Audio Format"
    type: enum
    options: ["wav", "mp3", "ogg", "flac"]
    default: "wav"
    display: node-body
    group: "Connection"
validation:
  connectionRules:
    - outputPort: out-main
      allowedTargetCategories: ["Processing", "Destinations"]
      allowedTargetPorts: ["data"]
```

**Acceptance Criteria:**

- [ ] Parser reads `.yaml` files recursively from the configured directory
- [ ] Parser validates `apiVersion` is `graphiti/v1` and `kind` is `NodeDefinition`
- [ ] Parser validates required fields: `metadata.id`, `metadata.name`, `category.group`, `shape.type`
- [ ] Parser validates `shape.type` against the allowed set (rounded-rect, pill, diamond, hexagon, sub-workflow, custom)
- [ ] Parser validates port types against the allowed set (data, control, error)
- [ ] Parser validates attribute types against the allowed set
- [ ] Invalid files are logged as warnings but don't prevent startup
- [ ] Duplicate `metadata.id` values across files are rejected with a clear error
- [ ] The in-memory registry supports `GetByID`, `GetByCategory`, `Search`, and `LoadAll`
- [ ] `Search` matches against name, description, and category (case-insensitive)
- [ ] At least 4 sample YAML definitions ship with the project (one per category: source, processing, destination, control flow)
- [ ] Tests use temporary directories with test YAML files, not real config files

### Story 1.6: Fake Auth Adapter, Middleware, and Session Management

**What:** Build the auth port interface, a fake adapter for local dev, session middleware, and route protection. The fake adapter auto-authenticates a dev user so every request passes through the same auth pipeline that production will use.

```go
// internal/adapters/driven/auth/fake.go
package auth

type FakeAuthAdapter struct {
    devUser *domain.User
}

func NewFakeAuth() *FakeAuthAdapter {
    return &FakeAuthAdapter{
        devUser: &domain.User{
            ID:             "dev-user-001",
            Username:       "dev",
            Email:          "dev@local",
            AuthProvider:   "fake",
            AuthProviderID: "fake-001",
        },
    }
}

func (f *FakeAuthAdapter) GetAuthURL(state string) string {
    return "/auth/callback?code=fake-code&state=" + state
}

func (f *FakeAuthAdapter) ExchangeCode(ctx context.Context, code string) (*driven.TokenPair, error) {
    return &driven.TokenPair{
        AccessToken: "fake-token",
        ExpiresAt:   time.Now().Add(24 * time.Hour),
    }, nil
}

func (f *FakeAuthAdapter) GetUserInfo(ctx context.Context, token string) (*domain.UserInfo, error) {
    return &domain.UserInfo{
        ID:       f.devUser.AuthProviderID,
        Username: f.devUser.Username,
        Email:    f.devUser.Email,
        Provider: "fake",
    }, nil
}
```

```go
// internal/adapters/driving/http/middleware.go
package http

func AuthMiddleware(sessionStore SessionStore) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            session, err := sessionStore.Get(r)
            if err != nil || session.UserID == "" {
                http.Redirect(w, r, "/auth/login", http.StatusFound)
                return
            }
            ctx := context.WithValue(r.Context(), userContextKey, session)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

**Acceptance Criteria:**

- [ ] `FakeAuthAdapter` implements the `AuthProvider` port interface
- [ ] The auth middleware reads session cookies and redirects unauthenticated users to `/auth/login`
- [ ] In dev mode, navigating to `/auth/login` auto-completes the fake flow and sets a session cookie
- [ ] All API routes (except `/auth/*`) require authentication
- [ ] The session contains user ID, username, and expiry
- [ ] Session expiry is configurable (default 24 hours)
- [ ] A `POST /auth/logout` endpoint clears the session cookie
- [ ] The auth adapter to use (fake vs. github) is selected by config: `auth.provider: fake`
- [ ] Tests verify that unauthenticated requests to protected routes get redirected
- [ ] Tests verify that authenticated requests pass through middleware successfully

### Story 1.7: SQLite Adapter with Migrations

**What:** Implement the SQLite storage adapter for workflows, users, and execution runs. Migrations run automatically on startup.

**Schema (migrations/001_initial.sql):**

```sql
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    email TEXT,
    avatar_url TEXT,
    auth_provider TEXT NOT NULL,
    auth_provider_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_login_at DATETIME,
    UNIQUE(auth_provider, auth_provider_id)
);

CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    definition JSON NOT NULL,
    status TEXT DEFAULT 'draft',
    version INTEGER DEFAULT 1,
    created_by TEXT REFERENCES users(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS workflow_versions (
    id TEXT PRIMARY KEY,
    workflow_id TEXT REFERENCES workflows(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    definition JSON NOT NULL,
    deployed_at DATETIME,
    deployed_by TEXT REFERENCES users(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS execution_runs (
    id TEXT PRIMARY KEY,
    workflow_id TEXT REFERENCES workflows(id),
    workflow_version INTEGER NOT NULL,
    status TEXT DEFAULT 'pending',
    started_at DATETIME,
    completed_at DATETIME,
    trigger_type TEXT,
    trigger_data JSON
);

CREATE TABLE IF NOT EXISTS execution_node_statuses (
    id TEXT PRIMARY KEY,
    run_id TEXT REFERENCES execution_runs(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    started_at DATETIME,
    completed_at DATETIME,
    input_data JSON,
    output_data JSON,
    error_message TEXT,
    logs JSON
);
```

**Acceptance Criteria:**

- [ ] SQLite adapter implements `WorkflowRepository`, `UserRepository`, and `ExecutionRepository`
- [ ] Database file path is configurable (default `./data/graphiti.db`)
- [ ] WAL mode is enabled for concurrent read/write
- [ ] Migrations run automatically on first startup
- [ ] `WorkflowRepository.Create` stores the full workflow definition as JSON
- [ ] `WorkflowRepository.GetByID` deserializes the JSON back into domain objects with all nodes, edges, and attributes
- [ ] `WorkflowRepository.List` returns lightweight summaries (no full definition)
- [ ] `UserRepository.Upsert` creates or updates a user record
- [ ] All adapter tests use `:memory:` SQLite databases so they're fast and isolated
- [ ] Tests verify round-trip: create → get → verify fields match
- [ ] Tests verify cascading deletes (deleting a workflow removes its versions)

### Story 1.8: Templ Layout, Node Palette, and Static SVG Canvas

**What:** Build the three-panel layout in Templ, render the node palette from the in-memory registry, and render static SVG nodes on the canvas for a saved workflow.

**Acceptance Criteria:**

- [ ] The base layout renders a top navigation bar, left panel (260px), center canvas (flexible), and right panel (300px)
- [ ] Top nav shows: logo, breadcrumb (`/ Workflows / {name}`), placeholder Deploy button
- [ ] Left panel in builder mode shows "Components" header, search input, and categorized node list
- [ ] Node palette items show: colored icon badge, node name, short description
- [ ] Categories are collapsible, sorted by the `order` field
- [ ] Search input is wired with `hx-get="/api/nodes/search"` and `hx-trigger="keyup changed delay:200ms"`
- [ ] Center panel contains an SVG element with a dot-grid pattern (24px spacing)
- [ ] When a workflow is loaded, its nodes render as SVG groups with correct shapes per category
- [ ] Source nodes have a left accent bar, processors are plain rounded rects, destinations have a right accent bar, diamonds for conditions, hexagons for merge/split, double-border for sub-workflows
- [ ] Each SVG node shows: icon badge, title text, status badge, attribute rows for `display: node-body` attributes
- [ ] Ports render as circles at their defined positions
- [ ] Edges render as SVG `<path>` elements with Bezier curves and arrowheads
- [ ] The right panel shows a placeholder message ("Select a node to configure")
- [ ] All pages are served behind the auth middleware
- [ ] The page loads in under 200ms with 20 nodes

### Story 1.9: Application Service Wiring and Main Entry Point

**What:** Wire everything together in `cmd/server/main.go`. Load config, initialize adapters, plug them into ports, start the HTTP server.

```go
// cmd/server/main.go (simplified)
func main() {
    cfg := config.Load("config/app.yaml")

    // Driven adapters
    db := sqlite.NewDB(cfg.Storage.SQLite.Path)
    workflowRepo := sqlite.NewWorkflowRepo(db)
    userRepo := sqlite.NewUserRepo(db)
    nodeLoader := filesystem.NewYAMLNodeLoader(cfg.NodeDefinitions.Path)
    var authAdapter driven.AuthProvider
    if cfg.Auth.Provider == "fake" {
        authAdapter = auth.NewFakeAuth()
    } else {
        authAdapter = auth.NewGitHub(cfg.Auth.GitHub)
    }

    // App services
    nodeRegistry := app.NewNodeRegistry(nodeLoader)
    workflowSvc := app.NewWorkflowService(workflowRepo, nodeRegistry)

    // Driving adapters
    router := httpAdapter.NewRouter(workflowSvc, nodeRegistry, authAdapter, userRepo)
    server := &http.Server{Addr: cfg.Server.Addr(), Handler: router}
    server.ListenAndServe()
}
```

**Acceptance Criteria:**

- [ ] `go run cmd/server/main.go` starts the server with zero manual setup beyond having Go installed
- [ ] Config is loaded from `config/app.yaml` with env var interpolation for secrets
- [ ] The YAML node loader runs at startup and populates the registry
- [ ] SQLite migrations run automatically
- [ ] Fake auth is active when `auth.provider: fake`
- [ ] Navigating to `http://localhost:8080` auto-authenticates (fake mode) and shows the dashboard
- [ ] The dashboard lists existing workflows (or an empty state)
- [ ] Clicking "New Workflow" creates one and opens the builder view
- [ ] The builder view shows the three-panel layout with palette and an empty canvas
- [ ] A `Makefile` exists with targets: `build`, `run`, `test`, `lint`

**Phase 1 Demo Checkpoint:** Open a browser to `localhost:8080`. You're automatically logged in. You see a dashboard. You create a new workflow. The builder opens with a palette of nodes on the left, an empty SVG canvas with a dot grid in the center, and a placeholder config panel on the right. Nodes in the palette are grouped by category with icons. If you load a pre-seeded workflow, you see nodes rendered with their correct shapes and edges drawn between them. Nothing is interactive yet, but the skeleton walks.

## Phase 2: Interactive Canvas

**Goal:** The canvas comes alive. You can drag nodes from the palette, move them around, draw connections between ports, select things, delete them, undo/redo everything, and copy/paste groups of nodes. The config panel loads when you select a node and saves changes. At demo time, you can build a complete workflow by hand.

**Estimated Duration:** 4-5 weeks

```mermaid
graph LR
    A["Drag from Palette"] --> B["Node Positioning<br/>+ Grid Snap"]
    B --> C["Edge Drawing<br/>+ Validation"]
    C --> D["Selection<br/>+ Delete"]
    D --> E["Undo/Redo<br/>Wired to UI"]
    E --> F["Copy/Paste<br/>+ Duplicate"]
    F --> G["Config Panel<br/>HTMX Forms"]
    style G fill:#10B981,color:#fff
```

### Story 2.1: Vanilla JS Canvas Engine (Pan, Zoom, Viewport)

**What:** Write the core `canvas.js` module that manages the SVG viewport. Pan, zoom, and all the coordinate math that every other interaction depends on.

**Code structure:**

```javascript
// web/static/js/canvas.js

export class CanvasEngine {
    constructor(svgElement) {
        this.svg = svgElement;
        this.contentGroup = svgElement.querySelector('.canvas-content');
        this.scale = 1.0;
        this.panX = 0;
        this.panY = 0;
        this.minScale = 0.25;
        this.maxScale = 2.0;
        this.gridSize = 24;
        this._bindEvents();
    }

    // Convert screen coordinates to canvas coordinates
    screenToCanvas(screenX, screenY) { ... }

    // Convert canvas coordinates to screen coordinates
    canvasToScreen(canvasX, canvasY) { ... }

    // Snap a position to the nearest grid point
    snapToGrid(x, y) {
        return {
            x: Math.round(x / this.gridSize) * this.gridSize,
            y: Math.round(y / this.gridSize) * this.gridSize,
        };
    }

    zoom(delta, centerX, centerY) { ... }
    pan(dx, dy) { ... }
    fitToView() { ... }

    _applyTransform() {
        this.contentGroup.setAttribute('transform',
            `translate(${this.panX}, ${this.panY}) scale(${this.scale})`);
    }
}
```

**Acceptance Criteria:**

- [ ] Mouse wheel zooms in/out centered on cursor position
- [ ] Zoom range is clamped between 25% and 200%
- [ ] Middle-click drag pans the canvas
- [ ] Space+left-click drag also pans
- [ ] Trackpad pinch-to-zoom is supported
- [ ] Pan and zoom are smooth (transforms applied via `requestAnimationFrame`)
- [ ] The dot grid scales and translates with the content
- [ ] The current zoom percentage is displayed in the bottom-right HUD
- [ ] Zoom control buttons work: + (zoom in), - (zoom out), fit-to-view
- [ ] Fit-to-view calculates the bounding box of all nodes and adjusts to show everything with 48px padding
- [ ] `screenToCanvas` and `canvasToScreen` conversions are accurate at all zoom levels
- [ ] `snapToGrid` rounds to the nearest 24px boundary

### Story 2.2: Drag from Palette to Canvas

**What:** Write `drag.js` to handle dragging a node type from the palette and dropping it onto the canvas. The drop dispatches an `AddNodeCommand` to the server.

**Code structure:**

```javascript
// web/static/js/drag.js

export class DragManager {
    constructor(canvas, commandDispatcher) {
        this.canvas = canvas;
        this.dispatcher = commandDispatcher;
        this.ghost = null;
        this._bindPaletteItems();
    }

    _onPaletteMouseDown(e, definitionId) {
        // Create ghost element, start tracking
    }

    _onMouseMove(e) {
        // Move ghost to cursor position
    }

    _onMouseUp(e) {
        if (!this._isOverCanvas(e)) { this._cancel(); return; }
        const pos = this.canvas.screenToCanvas(e.clientX, e.clientY);
        const snapped = this.canvas.snapToGrid(pos.x, pos.y);
        this.dispatcher.dispatch('add_node', {
            definitionId: this.draggedDefinitionId,
            x: snapped.x,
            y: snapped.y,
        });
    }
}
```

```javascript
// web/static/js/commands.js

export class CommandDispatcher {
    constructor(workflowId) {
        this.workflowId = workflowId;
    }

    async dispatch(type, payload) {
        const resp = await fetch(`/api/workflows/${this.workflowId}/commands`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ type, ...payload }),
        });
        const result = await resp.json();
        this._applySvgUpdates(result.svgFragments);
        this._updateUndoRedoState(result.canUndo, result.canRedo);
        return result;
    }

    async undo() {
        const resp = await fetch(`/api/workflows/${this.workflowId}/undo`, { method: 'POST' });
        // apply updates...
    }

    async redo() {
        const resp = await fetch(`/api/workflows/${this.workflowId}/redo`, { method: 'POST' });
        // apply updates...
    }
}
```

**Acceptance Criteria:**

- [ ] Palette items have `cursor: grab` and switch to `cursor: grabbing` on mousedown
- [ ] Dragging creates a semi-transparent ghost node that follows the cursor
- [ ] The ghost shows the node's shape and name at the correct size
- [ ] Dropping on the canvas sends `POST /api/workflows/{id}/commands` with `type: "add_node"`
- [ ] The server creates the node via `AddNodeCommand`, which goes through `CommandHistory.Execute`
- [ ] The response includes the new node's SVG fragment, which is injected into the canvas DOM
- [ ] The node appears at the drop position, snapped to the nearest grid point
- [ ] Dropping outside the canvas cancels the drag (no server call, ghost disappears)
- [ ] The new node gets a UUID and default attribute values from the definition
- [ ] The operation is on the undo stack (Ctrl+Z removes the just-added node)

### Story 2.3: Move Nodes on Canvas

**What:** Let users reposition nodes by clicking and dragging them.

**Acceptance Criteria:**

- [ ] Clicking and dragging a node moves it smoothly (updating SVG `transform` on each frame)
- [ ] Position updates use `requestAnimationFrame` to maintain 60fps
- [ ] Connected edges update their Bezier paths in real-time during the drag
- [ ] On mouse release, the final position snaps to the grid
- [ ] A `MoveNodeCommand` is dispatched with `fromX/fromY/toX/toY`
- [ ] Moving multiple selected nodes dispatches a single `MoveNodesCommand`
- [ ] The move is undoable (undo snaps back to the original position)
- [ ] Dragging a node that isn't selected first selects it then starts the drag

### Story 2.4: Edge Drawing Between Ports

**What:** Write `connect.js` to handle drawing connections. The user clicks an output port, drags to an input port, and an edge is created if the connection is valid.

**Acceptance Criteria:**

- [ ] Hovering over a port highlights it (scale-up, border color change)
- [ ] Clicking and dragging from an output port draws a temporary Bezier curve from the port to the cursor
- [ ] Valid target input ports highlight in green as the cursor approaches within 20px
- [ ] Invalid ports (wrong type, at max connections) appear dimmed with a red tint
- [ ] Releasing on a valid input port dispatches `AddEdgeCommand`
- [ ] The server validates port compatibility (type matching, max connections, connection rules, no cycles)
- [ ] On success, the new edge SVG `<path>` is added to the edge layer with an arrowhead marker
- [ ] Releasing on empty space or an invalid port cancels the operation (no server call)
- [ ] A tooltip briefly explains why a port is invalid when hovering during a drag
- [ ] The edge creation is undoable

### Story 2.5: Connection Validation

**What:** Server-side validation that prevents invalid edges from being created.

```go
// internal/domain/workflow.go (AddEdge method)
func (w *Workflow) AddEdge(sourceNodeID, sourcePortID, targetNodeID, targetPortID, edgeID string) error {
    // 1. Both nodes must exist
    // 2. Both ports must exist on their respective nodes
    // 3. Port types must match (data→data, control→control, error→error)
    // 4. Target port must not exceed maxConnections
    // 5. Connection rules from source node definition must allow the target category
    // 6. Adding this edge must not create a cycle (topological sort check)
}
```

**Acceptance Criteria:**

- [ ] Edges are only allowed between matching port types
- [ ] An input port at max connections (default 1 for inputs) rejects new connections
- [ ] The node definition's `connectionRules.allowedTargetCategories` is enforced
- [ ] The node definition's `connectionRules.allowedTargetPorts` is enforced
- [ ] A cycle detection algorithm (Kahn's or DFS-based) prevents edges that would create loops
- [ ] All validation errors return specific, user-friendly messages
- [ ] Each validation rule has dedicated unit tests with both passing and failing cases

### Story 2.6: Selection and Deletion

**What:** Write `select.js` to manage single-select, multi-select, and keyboard-driven deletion.

**Acceptance Criteria:**

- [ ] Clicking a node selects it (accent-colored border highlight) and deselects everything else
- [ ] Clicking the canvas background deselects all
- [ ] Ctrl/Cmd+click toggles a node in/out of the selection
- [ ] Click-dragging on empty canvas space draws a selection rectangle
- [ ] All nodes within the rectangle on release are selected
- [ ] Clicking an edge selects it (thicker stroke, accent color)
- [ ] Pressing Delete/Backspace removes selected nodes and their connected edges via `RemoveNodeCommand`
- [ ] A confirmation dialog appears before deleting nodes (skippable with Shift+Delete)
- [ ] Deleting multiple selected nodes is a single undoable command
- [ ] Selected node triggers an HTMX load of the config panel in the right panel

### Story 2.7: Undo/Redo Keyboard Shortcuts

**What:** Wire Ctrl+Z and Ctrl+Shift+Z to the undo/redo endpoints.

**Acceptance Criteria:**

- [ ] Ctrl/Cmd+Z calls `POST /api/workflows/{id}/undo` and applies the returned SVG updates
- [ ] Ctrl/Cmd+Shift+Z (or Ctrl+Y) calls `POST /api/workflows/{id}/redo`
- [ ] Undo/redo buttons in the canvas toolbar are enabled/disabled based on stack state
- [ ] The response from undo/redo includes `canUndo` and `canRedo` booleans
- [ ] Keyboard shortcuts are prevented from triggering browser defaults (e.g., Ctrl+Z doesn't undo text input)
- [ ] Undo/redo of a node add removes/restores the SVG element
- [ ] Undo/redo of a move smoothly transitions the node back to its previous position
- [ ] Undo/redo of an edge add removes/restores the SVG path

### Story 2.8: Copy, Paste, and Duplicate

**What:** Write `clipboard.js` and the server-side clipboard endpoints.

```go
// internal/domain/clipboard.go
package domain

type ClipboardPayload struct {
    Version int                    `json:"version"`
    Source  string                 `json:"source"`
    Nodes   []ClipboardNode       `json:"nodes"`
    Edges   []ClipboardEdge       `json:"edges"`
}

type ClipboardNode struct {
    OriginalID   string            `json:"originalId"`
    DefinitionID string            `json:"definitionId"`
    Label        string            `json:"label"`
    RelativeX    float64           `json:"relativeX"`
    RelativeY    float64           `json:"relativeY"`
    Attributes   map[string]any    `json:"attributes"`
}

type ClipboardEdge struct {
    SourceOriginalID string `json:"sourceOriginalId"`
    SourcePortID     string `json:"sourcePortId"`
    TargetOriginalID string `json:"targetOriginalId"`
    TargetPortID     string `json:"targetPortId"`
}
```

**Acceptance Criteria:**

- [ ] Ctrl/Cmd+C with nodes selected sends `POST /api/workflows/{id}/clipboard/copy` with node IDs
- [ ] The server serializes selected nodes and their internal edges (both endpoints in the selection) into `ClipboardPayload`
- [ ] Node positions in the payload are relative to the selection's top-left bounding box corner
- [ ] Ctrl/Cmd+V sends `POST /api/workflows/{id}/clipboard/paste` with the payload and cursor position
- [ ] The server generates new UUIDs for every pasted node, using a `oldID → newID` mapping
- [ ] Internal edges are remapped to use the new node IDs
- [ ] Edges referencing nodes outside the selection are silently dropped
- [ ] Pasted nodes appear at the cursor position, preserving relative layout
- [ ] The paste is a single undoable `PasteNodesCommand`
- [ ] Ctrl/Cmd+X copies then deletes (single undoable command), undo restores everything
- [ ] Ctrl/Cmd+D duplicates in-place, offset by 24px diagonally
- [ ] Pasted/duplicated nodes are automatically selected after the operation
- [ ] Clipboard tests verify ID remapping with complex multi-node, multi-edge selections

### Story 2.9: Config Panel with HTMX Forms

**What:** When a node is selected, the right panel loads its configuration form via HTMX. Changes auto-save.

**Acceptance Criteria:**

- [ ] Selecting a node triggers `hx-get="/api/workflows/{id}/nodes/{nodeId}/config"` on the right panel
- [ ] The panel header shows the node's icon and title
- [ ] Attributes are rendered in groups (matching the `group` field from the definition)
- [ ] Each attribute type renders as the correct widget (see table below)
- [ ] Current values are pre-populated from the node instance
- [ ] Changes trigger `hx-patch` with `hx-trigger="change"` for dropdowns/toggles and `hx-trigger="keyup changed delay:500ms"` for text inputs
- [ ] The PATCH handler creates an `UpdateAttributeCommand` through `CommandHistory.Execute`
- [ ] Attribute changes are undoable
- [ ] Required fields show a red asterisk indicator
- [ ] The panel has tabs: Configure (default), Logs (placeholder), YAML (shows raw config)
- [ ] Attributes with `display: node-body` or `display: both` update the SVG node on the canvas after save

| Attribute Type | Widget |
|---|---|
| `string` | Text input |
| `text` | Textarea |
| `number` | Number input with optional min/max |
| `boolean` | Toggle switch |
| `enum` | Dropdown select |
| `secret` | Password input with visibility toggle + env var hint |
| `json` | Monospace textarea |
| `expression` | Monospace input |
| `workflow-reference` | Dropdown listing saved workflows |
| `port-mapping` | Dynamic key-value list |

**Phase 2 Demo Checkpoint:** Open the browser, create a new workflow. Drag a Twilio source node from the palette onto the canvas. Drag a Transcribe processor next to it. Draw a connection between Twilio's output port and Transcribe's input port. The Bezier curve appears. Select the Twilio node, configure its endpoint in the right panel. Move a node, undo the move with Ctrl+Z, redo with Ctrl+Shift+Z. Copy both nodes, paste them below. Delete the copies. The entire workflow was built interactively with full undo/redo support.

## Phase 3: Persistence, Deploy, and Workflow Management

**Goal:** Workflows persist across sessions. You can create, list, open, save, and delete workflows. The deploy button validates the workflow and sends it via signed webhook to a configured endpoint. Workflows are versioned. At demo time, you can build a workflow, close the browser, reopen it, and your work is still there. You can deploy and see the webhook hit a mock endpoint.

**Estimated Duration:** 3-4 weeks

```mermaid
graph LR
    A["Workflow CRUD<br/>+ Dashboard"] --> B["Auto-Save<br/>on Mutations"]
    B --> C["Workflow Validation<br/>Pre-Deploy"]
    C --> D["Webhook Deploy<br/>+ HMAC Signing"]
    D --> E["Export<br/>YAML/JSON"]
    E --> F["Versioning<br/>+ History"]
    style F fill:#10B981,color:#fff
```

### Story 3.1: Workflow CRUD and Dashboard

**What:** Full create, read, update, delete operations for workflows. The dashboard page lists all workflows with status badges.

**Acceptance Criteria:**

- [ ] `GET /` renders the dashboard with a list of the authenticated user's workflows
- [ ] Each workflow card shows: name, status badge (draft/deployed/archived), last updated timestamp, version number
- [ ] A "New Workflow" button creates a blank workflow and redirects to the builder
- [ ] Clicking an existing workflow opens it in the builder with all nodes, edges, and config loaded
- [ ] A delete option (with confirmation) removes a workflow
- [ ] The dashboard is paginated or uses infinite scroll for large lists
- [ ] Empty state shows a friendly message with a prominent "Create your first workflow" button
- [ ] All operations go through the `WorkflowService` port, never directly to the repository

### Story 3.2: Auto-Save on Canvas Mutations

**What:** Every command that passes through `CommandHistory.Execute` triggers a save of the full workflow state to SQLite.

**Acceptance Criteria:**

- [ ] After each successful command execution, the workflow's `definition` JSON is updated in SQLite
- [ ] The `updated_at` timestamp is refreshed on every save
- [ ] Saves are debounced: if multiple commands arrive within 500ms, only one write happens (with the latest state)
- [ ] A save failure doesn't crash the session, it's logged and retried once
- [ ] Reopening a workflow after a browser crash restores the last saved state
- [ ] Node positions, attribute values, and all edges are preserved across saves

### Story 3.3: Pre-Deploy Validation

**What:** Before deploying, the system validates every invariant the domain model enforces.

**Acceptance Criteria:**

- [ ] Validation checks: all required attributes are filled, all connections are valid, no cycles, port types match
- [ ] Validation runs entirely in the domain layer (`Workflow.Validate()`)
- [ ] Orphan nodes (no connections) generate warnings, not errors
- [ ] If validation fails, the response includes a list of `ValidationError` objects with `nodeID`, `field`, and `message`
- [ ] The UI highlights problematic nodes with a red border and shows a toast with the error list
- [ ] If validation passes, the deploy proceeds

### Story 3.4: Webhook Deploy Adapter with HMAC Signing

**What:** The webhook adapter sends the workflow definition to a configured HTTP endpoint, signed with HMAC-SHA256.

```go
// internal/adapters/driven/webhook/deploy_target.go
package webhook

type WebhookDeployTarget struct {
    url        string
    hmacSecret string
    timeout    time.Duration
    maxRetries int
    backoff    BackoffStrategy
    client     *http.Client
}

func (w *WebhookDeployTarget) Deploy(ctx context.Context, payload domain.DeployPayload) (*domain.DeployResult, error) {
    body, _ := json.Marshal(payload)
    signature := w.sign(body)

    for attempt := 0; attempt <= w.maxRetries; attempt++ {
        req, _ := http.NewRequestWithContext(ctx, "POST", w.url, bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("X-Graphiti-Signature", "sha256="+signature)
        req.Header.Set("X-Graphiti-Event", payload.Event)

        resp, err := w.client.Do(req)
        if err == nil && resp.StatusCode < 500 {
            // Parse response, return result
        }
        // Exponential backoff before retry
        time.Sleep(w.backoff.Duration(attempt))
    }
    return nil, ErrDeployFailed
}

func (w *WebhookDeployTarget) sign(body []byte) string {
    mac := hmac.New(sha256.New, []byte(w.hmacSecret))
    mac.Write(body)
    return hex.EncodeToString(mac.Sum(nil))
}
```

**Webhook payload structure:**

```json
{
  "apiVersion": "graphiti/v1",
  "event": "workflow.deployed",
  "timestamp": "2026-03-09T14:30:00Z",
  "deployment": {
    "id": "deploy-abc123",
    "target": "production",
    "triggeredBy": { "userID": "user-xyz", "username": "alice" }
  },
  "workflow": {
    "id": "wf-001",
    "name": "Support Call Pipeline",
    "version": 5,
    "definition": { "nodes": [...], "edges": [...] }
  },
  "previousVersion": 4,
  "checksum": "sha256:a1b2c3..."
}
```

**Acceptance Criteria:**

- [ ] The Deploy button sends `POST /api/workflows/{id}/deploy` with `target` (production/staging)
- [ ] The handler validates the workflow first, rejecting invalid workflows
- [ ] On valid workflow, a new version record is created in the database
- [ ] The `DeployPayload` is constructed with all fields from the PRD spec (section 10.2)
- [ ] The payload is HMAC-SHA256 signed with the configured secret
- [ ] The signature is sent in the `X-Graphiti-Signature` header
- [ ] Retries happen with exponential backoff (1s, 2s, 4s) up to 3 attempts
- [ ] The webhook timeout is configurable (default 30s)
- [ ] If the engine returns a run ID in the response body, it's stored for execution tracking
- [ ] The Deploy button shows a green flash + checkmark on success
- [ ] The Deploy button shows a red flash + error toast on failure
- [ ] Deploy targets (production, staging) are configured in `config/app.yaml`
- [ ] Tests use `httptest.NewServer` to mock the webhook endpoint

### Story 3.5: Deploy Dropdown Options

**What:** The deploy button's dropdown chevron opens a menu with multiple actions.

**Acceptance Criteria:**

- [ ] The dropdown chevron reveals: "Deploy to Production", "Deploy to Staging", "Save as Draft", "Export as YAML", "Export as JSON"
- [ ] "Deploy to Production" and "Deploy to Staging" each trigger the webhook to the respective configured URL
- [ ] "Save as Draft" saves the workflow without deploying (just persists current state)
- [ ] "Export as YAML" triggers a file download of the workflow definition as YAML
- [ ] "Export as JSON" triggers a file download of the workflow definition as JSON
- [ ] The default deploy target is configurable in `app.yaml`

### Story 3.6: Workflow Versioning

**What:** Every deploy creates a numbered version snapshot.

**Acceptance Criteria:**

- [ ] Each deploy increments the workflow's `version` field and creates a `workflow_versions` record
- [ ] The version record stores the complete workflow definition JSON at the time of deploy
- [ ] The current version number is visible in the top navigation bar next to the workflow name
- [ ] A version history view (accessible from settings or a menu) lists all versions with timestamp, version number, and deployer username
- [ ] The data model supports rollback (loading a previous version's definition), even though the rollback UI is deferred

**Phase 3 Demo Checkpoint:** Build a workflow with several nodes and connections. Close the browser. Reopen it. Your workflow is still there. Click Deploy. Watch a mock webhook server receive the signed payload. Export the workflow as YAML. Create a second workflow from the dashboard. Deploy it to staging. Check the version history, it shows version 1.

## Phase 4: Authentication and Execution View

**Goal:** Replace the fake auth with real GitHub OAuth2. Build the execution mode UI that shows run history and per-node status overlays. WebSocket pushes live execution updates. At demo time, you log in with GitHub, deploy a workflow, and watch a mock execution light up nodes on the canvas in real-time.

**Estimated Duration:** 3-4 weeks

```mermaid
graph LR
    A["GitHub OAuth2<br/>Adapter"] --> B["Session Management<br/>+ Protected Routes"]
    B --> C["Execution Mode<br/>Run History"]
    C --> D["Status Overlays<br/>on Canvas"]
    D --> E["Execution Status<br/>Callback Endpoint"]
    E --> F["WebSocket<br/>Live Updates"]
    style F fill:#10B981,color:#fff
```

### Story 4.1: GitHub OAuth2 Adapter

**What:** Implement the real `GitHubAuthAdapter` that performs the full OAuth2 authorization code flow.

```go
// internal/adapters/driven/auth/github.go
package auth

type GitHubAuthAdapter struct {
    clientID     string
    clientSecret string
    scopes       []string
    allowedOrgs  []string
    httpClient   *http.Client
}

func (g *GitHubAuthAdapter) GetAuthURL(state string) string {
    return fmt.Sprintf(
        "https://github.com/login/oauth/authorize?client_id=%s&scope=%s&state=%s",
        g.clientID, strings.Join(g.scopes, " "), state,
    )
}

func (g *GitHubAuthAdapter) ExchangeCode(ctx context.Context, code string) (*driven.TokenPair, error) {
    // POST to https://github.com/login/oauth/access_token
    // Parse response for access_token
}

func (g *GitHubAuthAdapter) GetUserInfo(ctx context.Context, token string) (*domain.UserInfo, error) {
    // GET https://api.github.com/user with Bearer token
    // Parse response for id, login, email, avatar_url
}
```

**Config (`config/auth.yaml`):**

```yaml
auth:
  provider: github    # or "fake" for local dev
  github:
    clientId: "${GITHUB_CLIENT_ID}"
    clientSecret: "${GITHUB_CLIENT_SECRET}"
    scopes: ["user:email", "read:org"]
    allowedOrgs: []
  session:
    secret: "${SESSION_SECRET}"
    maxAge: 86400
    secure: true
```

**Acceptance Criteria:**

- [ ] `GitHubAuthAdapter` implements the same `AuthProvider` port interface as the fake adapter
- [ ] `GetAuthURL` returns a properly formatted GitHub OAuth2 authorization URL with client ID, scopes, and state parameter
- [ ] `ExchangeCode` posts to GitHub's token endpoint and returns an access token
- [ ] `GetUserInfo` calls the GitHub API `/user` endpoint and maps the response to `domain.UserInfo`
- [ ] If `allowedOrgs` is configured, the adapter checks org membership via GitHub API and rejects non-members
- [ ] The adapter is selected by config: switching from `provider: fake` to `provider: github` requires no code changes
- [ ] Tests mock GitHub's API endpoints using `httptest.NewServer`
- [ ] Error handling: network failures, invalid tokens, and rate limits are handled gracefully with clear error messages

### Story 4.2: Session Management and Route Protection

**What:** Production-quality session handling with secure cookies and proper expiration.

**Acceptance Criteria:**

- [ ] Sessions use encrypted, HTTP-only, Secure, SameSite=Strict cookies
- [ ] Session cookie contains: user ID, username, expiry timestamp
- [ ] Sessions expire after `session.maxAge` (default 24 hours)
- [ ] Expired sessions redirect to `/auth/login` with a friendly "Session expired" message
- [ ] `POST /auth/logout` clears the session cookie and redirects to the login page
- [ ] All routes except `/auth/login`, `/auth/callback`, and `/static/*` require authentication
- [ ] CSRF protection is active on all state-changing requests (POST, PATCH, DELETE)
- [ ] Rate limiting is applied to `/auth/login` and `/auth/callback` (max 10 requests/minute per IP)
- [ ] The login page shows a "Login with GitHub" button (or the configured provider name)

### Story 4.3: Execution Mode, Run History List

**What:** Switch between Builder and Execution modes. The execution mode left panel shows a chronological list of runs.

**Acceptance Criteria:**

- [ ] A tab bar below the top nav offers "Builder" and "Execution" mode tabs
- [ ] Switching to Execution mode replaces the left panel content with execution run history
- [ ] Each run entry shows: short run ID (first 8 chars), timestamp, duration, status badge (color-coded)
- [ ] Status badges: green for completed, red for failed, amber for running, gray for pending/cancelled
- [ ] The list is sorted newest-first
- [ ] Pagination is handled via HTMX infinite scroll (`hx-trigger="revealed"`)
- [ ] The run history loads from `ExecutionRepository.ListByWorkflow`
- [ ] Clicking a run loads its execution graph onto the canvas

### Story 4.4: Execution Status Overlays on Canvas

**What:** In execution mode, the canvas is read-only and nodes show their execution status visually.

**Acceptance Criteria:**

- [ ] The canvas in execution mode is read-only (no drag, no connect, no edit, no delete)
- [ ] Each node's border color reflects its execution status: green (completed), red (failed), amber (running), gray (pending/skipped)
- [ ] Edges that carried data are colored green, unexecuted edges stay gray
- [ ] Currently-running nodes have a pulsing border animation (CSS keyframes on the SVG stroke)
- [ ] Nodes display their execution duration as a small overlay badge
- [ ] Clicking a node loads its execution details in the right panel
- [ ] The right panel shows: status badge, start/end times, duration, input data (collapsible JSON), output data (collapsible JSON), error message (if failed), log entries (scrollable, timestamped)

### Story 4.5: Execution Status Callback Endpoint

**What:** An endpoint that the external execution engine calls to report node-level status updates.

```go
// POST /api/callbacks/execution
// Accepts the execution status payload from the engine
{
    "apiVersion": "graphiti/v1",
    "event": "execution.node_status",
    "runID": "run-789",
    "workflowID": "wf-001",
    "nodeID": "node-123",
    "status": "completed",
    "startedAt": "2026-03-09T14:30:01Z",
    "completedAt": "2026-03-09T14:30:03Z",
    "outputSummary": { "recordsProcessed": 42 }
}
```

**Acceptance Criteria:**

- [ ] `POST /api/callbacks/execution` accepts status payloads from the engine
- [ ] The endpoint validates the payload structure and `apiVersion`
- [ ] The payload is authenticated (HMAC signature or API key, configurable)
- [ ] Node status is updated in `ExecutionRepository.UpdateNodeStatus`
- [ ] If the run doesn't exist, it's created with status "running"
- [ ] When all nodes in a run are completed/failed/skipped, the overall run status is computed and updated
- [ ] The endpoint returns `202 Accepted` immediately (processing is async)

### Story 4.6: WebSocket for Live Execution Updates

**What:** When execution status callbacks arrive, push updates to connected browsers in real-time via WebSocket.

```go
// internal/adapters/driving/http/websocket.go
type WebSocketHub struct {
    connections map[string]map[*websocket.Conn]bool // workflowID → connections
    mu          sync.RWMutex
}

func (h *WebSocketHub) BroadcastToWorkflow(workflowID string, msg WebSocketMessage) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    for conn := range h.connections[workflowID] {
        conn.WriteJSON(msg)
    }
}
```

**Acceptance Criteria:**

- [ ] Clients connect to `ws:///api/ws/workflows/{id}` when viewing a workflow in execution mode
- [ ] When an execution callback arrives, the hub broadcasts the status update to all clients viewing that workflow
- [ ] The WebSocket message includes: run ID, node ID, new status, timing data
- [ ] The client JS receives the message and updates the node's status overlay (border color, badge) without a full page reload
- [ ] Running nodes start their pulse animation when the "running" status arrives
- [ ] Connection cleanup happens automatically when the client disconnects
- [ ] The hub handles concurrent connections gracefully

**Phase 4 Demo Checkpoint:** Go to `localhost:8080`. You see a login page. Click "Login with GitHub." Complete the OAuth flow. You're on the dashboard. Open a workflow. Deploy it. Switch to Execution mode. Use a curl command to simulate the engine sending status callbacks. Watch nodes light up in real-time, first amber (running) with a pulse, then green (completed) or red (failed). Click a node to see its execution details.

## Phase 4.5: Sample Execution Engine and Developer Guide

**Goal:** Build a working execution engine that turns Graphiti from a visual editor into a live system. The engine receives deployed workflows via webhook, executes nodes in topological order, reports status back to Graphiti in real-time, and actually processes HTTP requests through a Postgres-backed ToDo API. Ship it with docker-compose so anyone can run the entire stack with one command. Write a developer guide that teaches others how to build their own engine.

At demo time, you docker-compose up, open Graphiti, build the ToDo workflow from scratch (or load the pre-built one), deploy it, and then use curl to create, read, update, and delete ToDo items. Every request flows through the workflow visibly, with nodes lighting up in execution mode as data passes through them.

**Estimated Duration:** 3-4 weeks

**Depends on:** Phase 4 (execution mode UI, WebSocket, status callbacks)

```mermaid
graph LR
    A["New Node<br/>Definitions"] --> B["Sample Engine<br/>Core"]
    B --> C["Node Executors<br/>(API, PG, etc.)"]
    C --> D["ToDo Workflow<br/>Definition"]
    D --> E["Docker Compose<br/>Full Stack"]
    E --> F["Developer Guide<br/>Documentation"]
    style F fill:#10B981,color:#fff
```

### The Big Picture

Here's what the full running stack looks like. Graphiti is the UI. The sample engine is the runtime. Postgres is where data lives. All three are independent services that communicate over HTTP and SQL.

```mermaid
graph TB
    subgraph "Browser"
        UI["Graphiti UI<br/>localhost:8080"]
    end

    subgraph "Docker Network"
        G["Graphiti Server<br/>:8080"]
        E["Sample Engine<br/>:9090"]
        PG["PostgreSQL<br/>:5432"]
    end

    subgraph "External"
        CURL["curl / Postman<br/>API Client"]
    end

    UI -->|"HTMX, WebSocket"| G
    G -->|"Deploy webhook<br/>(workflow definition)"| E
    E -->|"Status callbacks<br/>(node execution updates)"| G
    G -->|"WebSocket push"| UI
    E -->|"SQL queries<br/>(from PostgreSQL nodes)"| PG
    CURL -->|"HTTP requests<br/>POST /api/todos"| E
    E -->|"HTTP responses<br/>201 Created"| CURL
```

The flow when someone sends a curl request to the ToDo API:

```mermaid
sequenceDiagram
    participant C as curl
    participant E as Sample Engine
    participant PG as PostgreSQL
    participant G as Graphiti Server
    participant B as Browser (Execution Mode)

    C->>E: POST /api/todos {"title": "Buy milk"}
    E->>G: callback: API Gateway node → running
    G->>B: WebSocket: node status update
    E->>G: callback: API Gateway node → completed
    E->>G: callback: HTTP Router node → running
    E->>G: callback: HTTP Router node → completed (route: POST)
    E->>G: callback: Validate Payload node → running
    E->>G: callback: Validate Payload node → completed
    E->>G: callback: PostgreSQL Insert node → running
    E->>PG: INSERT INTO todos (title, completed) VALUES ('Buy milk', false)
    PG->>E: {id: 1, title: "Buy milk", completed: false}
    E->>G: callback: PostgreSQL Insert node → completed
    E->>G: callback: HTTP Response node → running
    E->>G: callback: HTTP Response node → completed
    E->>C: 201 Created {"id": 1, "title": "Buy milk", "completed": false}
    Note over B: All nodes visibly light up<br/>in sequence during execution
```

### The ToDo Workflow

This is the workflow users build (or load from the pre-built example) to create a full CRUD REST API backed by Postgres. It uses every category of node: sources, processing, control flow, and destinations.

```mermaid
graph TD
    API["🌐 API Gateway<br/><i>Source</i><br/>Path: /api/todos/*"]

    ROUTER{"🔀 HTTP Router<br/><i>Control Flow</i>"}

    VAL_POST["✅ Validate Payload<br/><i>Processing</i><br/>Required: title"]
    VAL_PUT["✅ Validate Payload<br/><i>Processing</i><br/>Required: title, completed"]

    PG_LIST["🐘 PostgreSQL<br/><i>Destination</i><br/>SELECT * FROM todos"]
    PG_GET["🐘 PostgreSQL<br/><i>Destination</i><br/>SELECT * FROM todos WHERE id = $1"]
    PG_INSERT["🐘 PostgreSQL<br/><i>Destination</i><br/>INSERT INTO todos ..."]
    PG_UPDATE["🐘 PostgreSQL<br/><i>Destination</i><br/>UPDATE todos SET ... WHERE id = $1"]
    PG_DELETE["🐘 PostgreSQL<br/><i>Destination</i><br/>DELETE FROM todos WHERE id = $1"]

    RES_LIST["📤 HTTP Response<br/><i>Destination</i><br/>200 OK"]
    RES_GET["📤 HTTP Response<br/><i>Destination</i><br/>200 OK"]
    RES_CREATE["📤 HTTP Response<br/><i>Destination</i><br/>201 Created"]
    RES_UPDATE["📤 HTTP Response<br/><i>Destination</i><br/>200 OK"]
    RES_DELETE["📤 HTTP Response<br/><i>Destination</i><br/>204 No Content"]
    RES_ERR["📤 HTTP Response<br/><i>Destination</i><br/>400 Bad Request"]

    API -->|"data"| ROUTER
    ROUTER -->|"GET (no id)"| PG_LIST
    ROUTER -->|"GET (with id)"| PG_GET
    ROUTER -->|"POST"| VAL_POST
    ROUTER -->|"PUT"| VAL_PUT
    ROUTER -->|"DELETE"| PG_DELETE

    VAL_POST -->|"valid"| PG_INSERT
    VAL_POST -->|"error"| RES_ERR
    VAL_PUT -->|"valid"| PG_UPDATE
    VAL_PUT -->|"error"| RES_ERR

    PG_LIST --> RES_LIST
    PG_GET --> RES_GET
    PG_INSERT --> RES_CREATE
    PG_UPDATE --> RES_UPDATE
    PG_DELETE --> RES_DELETE
```

This single workflow exercises:
- **Source node** with configurable path, methods, and optional auth
- **Control flow** (diamond shape) with multiple conditional output ports
- **Processing** nodes for validation with error output ports
- **Destination** nodes for both database operations and HTTP responses
- **Error paths** that route validation failures to error responses
- **Multiple port types** (data ports for the main flow, error ports for failures)

### New Node Definitions

These YAML definitions ship in `config/nodes/` alongside the existing ones. They're what make the ToDo workflow possible, but they're generic enough to build any HTTP-to-database workflow.

#### Story 4.5.1: API Gateway Node Definition

**What:** A source node that represents an HTTP endpoint. The engine listens on the configured path and emits incoming requests as data. Think of it as a funnel, HTTP requests pour in at the top and structured data flows out the right side.

**YAML definition:**

```yaml
# config/nodes/sources/api-gateway.yaml
apiVersion: graphiti/v1
kind: NodeDefinition

metadata:
  id: source-api-gateway
  name: API Gateway
  description: "Receives HTTP requests on a configurable endpoint"
  version: "1.0.0"
  icon: "🌐"

category:
  group: Sources
  order: 10

shape:
  type: rounded-rect
  width: 220
  minWidth: 180
  maxWidth: 320
  headerColor: "var(--node-source)"
  headerBackground: "var(--node-source-dim)"

ports:
  inputs: []
  outputs:
    - id: out-main
      label: "Request"
      type: data
      position: right-center
      maxConnections: 1

attributes:
  - id: path
    label: "Path"
    type: string
    default: "/api/resource"
    required: true
    display: node-body
    group: "Endpoint"
    hint: "Supports path params like /api/todos/:id"

  - id: methods
    label: "Allowed Methods"
    type: enum
    options: ["ALL", "GET", "POST", "PUT", "DELETE", "PATCH"]
    default: "ALL"
    display: node-body
    group: "Endpoint"

  - id: auth_type
    label: "Authentication"
    type: enum
    options: ["none", "api-key", "bearer-token", "basic"]
    default: "none"
    display: both
    group: "Security"

  - id: auth_secret
    label: "Auth Secret / API Key"
    type: secret
    required: false
    display: config-panel
    group: "Security"
    hint: "Use ${API_KEY} for env var reference"

  - id: rate_limit
    label: "Rate Limit (req/min)"
    type: number
    default: 0
    min: 0
    max: 10000
    display: config-panel
    group: "Security"
    hint: "0 = unlimited"

  - id: cors_origins
    label: "CORS Origins"
    type: string
    default: "*"
    display: config-panel
    group: "Security"

validation:
  connectionRules:
    - outputPort: out-main
      allowedTargetCategories: ["Processing", "Control Flow"]
      allowedTargetPorts: ["data"]
  attributeRules:
    - expression: "path.startsWith('/')"
      message: "Path must start with /"
```

**The data the API Gateway node emits downstream:**

```json
{
  "method": "POST",
  "path": "/api/todos",
  "pathParams": {},
  "queryParams": {},
  "headers": { "content-type": "application/json", "authorization": "Bearer ..." },
  "body": { "title": "Buy milk", "completed": false },
  "clientIp": "192.168.1.100",
  "timestamp": "2026-03-10T14:30:00Z"
}
```

**Acceptance Criteria:**

- [ ] YAML file parses and validates correctly by the existing node loader
- [ ] Node appears in the "Sources" category in the palette with the globe icon
- [ ] Shape renders as a rounded-rect with left accent bar (source style)
- [ ] Path and Allowed Methods display on the node body
- [ ] Auth type shows on both node body and config panel
- [ ] Secret field is write-only after save (never sent back to browser as plaintext)
- [ ] Connection rules allow connecting only to Processing or Control Flow nodes
- [ ] Path validation rule rejects paths that don't start with `/`

#### Story 4.5.2: HTTP Method Router Node Definition

**What:** A control flow node that examines the incoming HTTP method and routes data to the matching output port. It's a traffic cop: one road in, multiple roads out, and it reads the method sign to decide which way to send the car.

```yaml
# config/nodes/control/http-router.yaml
apiVersion: graphiti/v1
kind: NodeDefinition

metadata:
  id: control-http-router
  name: HTTP Router
  description: "Routes requests based on HTTP method and path pattern"
  version: "1.0.0"
  icon: "🔀"

category:
  group: Control Flow
  order: 10

shape:
  type: diamond
  width: 200
  minWidth: 160
  maxWidth: 280
  headerColor: "var(--node-control)"
  headerBackground: "var(--node-control-dim)"

ports:
  inputs:
    - id: in-main
      label: "Request"
      type: data
      position: left-center
      maxConnections: 1
  outputs:
    - id: out-get
      label: "GET"
      type: data
      position: right-top
      maxConnections: 1
    - id: out-get-by-id
      label: "GET :id"
      type: data
      position: right-center
      maxConnections: 1
    - id: out-post
      label: "POST"
      type: data
      position: right-bottom
      maxConnections: 1
    - id: out-put
      label: "PUT"
      type: data
      position: bottom-center
      maxConnections: 1
    - id: out-delete
      label: "DELETE"
      type: data
      position: bottom-center
      maxConnections: 1
    - id: out-unmatched
      label: "No Match"
      type: error
      position: bottom-center
      maxConnections: 1

attributes:
  - id: id_param
    label: "ID Path Parameter"
    type: string
    default: "id"
    display: config-panel
    group: "Routing"
    hint: "Name of the path parameter that distinguishes GET-all from GET-by-id"

validation:
  connectionRules:
    - outputPort: out-get
      allowedTargetCategories: ["Processing", "Destinations"]
      allowedTargetPorts: ["data"]
    - outputPort: out-post
      allowedTargetCategories: ["Processing", "Destinations"]
      allowedTargetPorts: ["data"]
    - outputPort: out-put
      allowedTargetCategories: ["Processing", "Destinations"]
      allowedTargetPorts: ["data"]
    - outputPort: out-delete
      allowedTargetCategories: ["Processing", "Destinations"]
      allowedTargetPorts: ["data"]
    - outputPort: out-unmatched
      allowedTargetCategories: ["Processing", "Destinations"]
      allowedTargetPorts: ["data", "error"]
```

**Acceptance Criteria:**

- [ ] Renders as a diamond shape (control flow convention)
- [ ] Shows 6 output ports: GET, GET :id, POST, PUT, DELETE, No Match
- [ ] The "No Match" port uses the `error` port type
- [ ] Connecting to the input requires a `data` type output port from the source
- [ ] Each output port allows connecting to Processing or Destination nodes

#### Story 4.5.3: Validate Payload Node Definition

**What:** A processing node that checks incoming JSON against configurable rules. If the data passes, it flows out the main output. If it fails, it flows out the error port with a description of what went wrong.

```yaml
# config/nodes/processing/validate-payload.yaml
apiVersion: graphiti/v1
kind: NodeDefinition

metadata:
  id: processing-validate-payload
  name: Validate Payload
  description: "Validates request body fields against configurable rules"
  version: "1.0.0"
  icon: "✅"

category:
  group: Processing
  order: 15

shape:
  type: rounded-rect
  width: 200
  minWidth: 160
  maxWidth: 320
  headerColor: "var(--node-processor)"
  headerBackground: "var(--node-processor-dim)"

ports:
  inputs:
    - id: in-main
      label: "Input"
      type: data
      position: left-center
      maxConnections: 1
  outputs:
    - id: out-main
      label: "Valid"
      type: data
      position: right-center
      maxConnections: 1
    - id: out-error
      label: "Invalid"
      type: error
      position: bottom-center
      maxConnections: 1

attributes:
  - id: required_fields
    label: "Required Fields"
    type: string
    default: ""
    required: true
    display: node-body
    group: "Rules"
    hint: "Comma-separated field names, e.g.: title,email"

  - id: field_types
    label: "Field Type Rules"
    type: json
    default: "{}"
    display: config-panel
    group: "Rules"
    hint: '{"title": "string", "completed": "boolean", "priority": "number"}'

  - id: max_body_size
    label: "Max Body Size (KB)"
    type: number
    default: 256
    min: 1
    max: 10240
    display: config-panel
    group: "Limits"

validation:
  connectionRules:
    - outputPort: out-main
      allowedTargetCategories: ["Processing", "Destinations"]
      allowedTargetPorts: ["data"]
    - outputPort: out-error
      allowedTargetCategories: ["Processing", "Destinations"]
      allowedTargetPorts: ["data", "error"]
```

**Acceptance Criteria:**

- [ ] Renders as a standard rounded rectangle (processing category)
- [ ] Shows two output ports: "Valid" (data type) and "Invalid" (error type)
- [ ] Required fields display on the node body
- [ ] Field type rules are editable as JSON in the config panel
- [ ] Connection rules allow the error port to connect to both data and error inputs

#### Story 4.5.4: PostgreSQL Node Definition

**What:** A destination node that runs a parameterized SQL query against a Postgres database. The user configures the connection string, the SQL template, and how parameters map from the incoming data.

```yaml
# config/nodes/destinations/postgresql.yaml
apiVersion: graphiti/v1
kind: NodeDefinition

metadata:
  id: destination-postgresql
  name: PostgreSQL
  description: "Executes a parameterized SQL query against a PostgreSQL database"
  version: "1.0.0"
  icon: "🐘"

category:
  group: Destinations
  order: 20

shape:
  type: rounded-rect
  width: 220
  minWidth: 180
  maxWidth: 360
  headerColor: "var(--node-destination)"
  headerBackground: "var(--node-destination-dim)"

ports:
  inputs:
    - id: in-main
      label: "Input"
      type: data
      position: left-center
      maxConnections: 1
  outputs:
    - id: out-main
      label: "Result"
      type: data
      position: right-center
      maxConnections: 1
    - id: out-error
      label: "Error"
      type: error
      position: bottom-center
      maxConnections: 1

attributes:
  - id: connection_string
    label: "Connection String"
    type: secret
    required: true
    display: config-panel
    group: "Connection"
    hint: "Use ${DATABASE_URL} for env var reference"

  - id: operation
    label: "Operation"
    type: enum
    options: ["query", "query-row", "exec"]
    default: "query"
    required: true
    display: node-body
    group: "Query"
    hint: "query = multiple rows, query-row = single row, exec = INSERT/UPDATE/DELETE"

  - id: sql
    label: "SQL"
    type: text
    required: true
    display: config-panel
    group: "Query"
    hint: "Use $1, $2, ... for parameters"

  - id: params
    label: "Parameter Mapping"
    type: json
    default: "[]"
    display: config-panel
    group: "Query"
    hint: '["body.title", "body.completed", "pathParams.id"]'

  - id: timeout_seconds
    label: "Query Timeout (s)"
    type: number
    default: 30
    min: 1
    max: 300
    display: config-panel
    group: "Connection"

validation:
  connectionRules:
    - outputPort: out-main
      allowedTargetCategories: ["Processing", "Destinations"]
      allowedTargetPorts: ["data"]
    - outputPort: out-error
      allowedTargetCategories: ["Processing", "Destinations"]
      allowedTargetPorts: ["data", "error"]
```

**Acceptance Criteria:**

- [ ] Renders as a rounded-rect with right accent bar (destination category)
- [ ] Operation type is visible on the node body
- [ ] Connection string uses the `secret` type (never echoed back to browser)
- [ ] SQL attribute uses the `text` type for multi-line editing
- [ ] Parameter mapping uses JSON to express which incoming data fields map to $1, $2, etc.
- [ ] Has both a data output (for query results) and an error output

#### Story 4.5.5: HTTP Response Node Definition

**What:** A destination node that packages the upstream data into an HTTP response with a configurable status code and headers. It's the last stop in the workflow, the exit door that sends a response back to whoever called the API.

```yaml
# config/nodes/destinations/http-response.yaml
apiVersion: graphiti/v1
kind: NodeDefinition

metadata:
  id: destination-http-response
  name: HTTP Response
  description: "Formats and returns an HTTP response to the API caller"
  version: "1.0.0"
  icon: "📤"

category:
  group: Destinations
  order: 30

shape:
  type: rounded-rect
  width: 200
  minWidth: 160
  maxWidth: 280
  headerColor: "var(--node-destination)"
  headerBackground: "var(--node-destination-dim)"

ports:
  inputs:
    - id: in-main
      label: "Input"
      type: data
      position: left-center
      maxConnections: 1
    - id: in-error
      label: "Error"
      type: error
      position: left-center
      maxConnections: 1
  outputs: []

attributes:
  - id: status_code
    label: "Status Code"
    type: number
    default: 200
    required: true
    min: 100
    max: 599
    display: node-body
    group: "Response"

  - id: content_type
    label: "Content-Type"
    type: enum
    options: ["application/json", "text/plain", "text/html", "application/xml"]
    default: "application/json"
    display: config-panel
    group: "Response"

  - id: headers
    label: "Custom Headers"
    type: json
    default: "{}"
    display: config-panel
    group: "Response"
    hint: '{"X-Request-Id": "{{requestId}}"}'

  - id: body_template
    label: "Body Template"
    type: text
    default: ""
    display: config-panel
    group: "Response"
    hint: "Leave empty to forward upstream data as-is. Use {{field}} for templating."

validation:
  connectionRules: []
```

**Acceptance Criteria:**

- [ ] Renders as a destination node (right accent bar) with no output ports (it's a terminal node)
- [ ] Accepts both `data` and `error` type input ports (can be wired to either main flow or error paths)
- [ ] Status code is visible on the node body
- [ ] Body template supports simple mustache-style templating or passthrough

#### Story 4.5.6: JSON Transform Node Definition

**What:** A processing node that reshapes JSON data. Takes an input object, applies a mapping template, and outputs the transformed result. Useful for extracting fields from the request body to pass to a SQL query, or for reformatting a database result into an API response shape.

```yaml
# config/nodes/processing/json-transform.yaml
apiVersion: graphiti/v1
kind: NodeDefinition

metadata:
  id: processing-json-transform
  name: JSON Transform
  description: "Reshapes JSON data using a mapping template"
  version: "1.0.0"
  icon: "🔄"

category:
  group: Processing
  order: 20

shape:
  type: rounded-rect
  width: 200
  minWidth: 160
  maxWidth: 320
  headerColor: "var(--node-processor)"
  headerBackground: "var(--node-processor-dim)"

ports:
  inputs:
    - id: in-main
      label: "Input"
      type: data
      position: left-center
      maxConnections: 1
  outputs:
    - id: out-main
      label: "Output"
      type: data
      position: right-center
      maxConnections: 1

attributes:
  - id: mapping
    label: "Mapping Template"
    type: json
    required: true
    display: config-panel
    group: "Transform"
    hint: '{"title": "body.title", "done": "body.completed"}'

  - id: mode
    label: "Mode"
    type: enum
    options: ["map-fields", "passthrough-with-additions", "expression"]
    default: "map-fields"
    display: node-body
    group: "Transform"

validation:
  connectionRules:
    - outputPort: out-main
      allowedTargetCategories: ["Processing", "Destinations", "Control Flow"]
      allowedTargetPorts: ["data"]
```

**Acceptance Criteria:**

- [ ] Renders as a standard processor rounded-rect
- [ ] Mode is visible on the node body
- [ ] Mapping template is editable as JSON in the config panel
- [ ] Supports three modes: field mapping, passthrough with additions, and raw expression

### Sample Execution Engine

#### Story 4.5.7: Engine Core, Webhook Receiver, and Workflow Runner

**What:** A standalone Go service that lives in `examples/engine/`. It receives workflow definitions from Graphiti's deploy webhook, stores them, and executes them when HTTP requests arrive. The engine is deliberately simple, it's reference code, not a production runtime.

**Project structure:**

```
examples/
└── engine/
    ├── cmd/
    │   └── engine/
    │       └── main.go                 # Entry point
    ├── internal/
    │   ├── receiver/
    │   │   └── webhook.go              # Receives deploy payloads from Graphiti
    │   ├── runner/
    │   │   ├── runner.go               # Topological sort, node-by-node execution
    │   │   ├── runner_test.go
    │   │   └── context.go              # Execution context (carries data between nodes)
    │   ├── executors/
    │   │   ├── registry.go             # Maps node definition IDs to executor functions
    │   │   ├── api_gateway.go          # Handles incoming HTTP, emits request data
    │   │   ├── api_gateway_test.go
    │   │   ├── http_router.go          # Routes by HTTP method
    │   │   ├── http_router_test.go
    │   │   ├── validate_payload.go     # Field validation logic
    │   │   ├── validate_payload_test.go
    │   │   ├── postgresql.go           # Executes parameterized SQL
    │   │   ├── postgresql_test.go
    │   │   ├── http_response.go        # Builds and sends HTTP response
    │   │   ├── http_response_test.go
    │   │   └── json_transform.go       # JSON reshaping
    │   ├── callback/
    │   │   └── reporter.go             # Sends status updates back to Graphiti
    │   └── config/
    │       └── config.go
    ├── config.yaml
    ├── go.mod
    ├── Dockerfile
    └── README.md
```

**Engine config:**

```yaml
# examples/engine/config.yaml
server:
  port: 9090

graphiti:
  callbackURL: "http://graphiti:8080/api/callbacks/execution"
  hmacSecret: "${DEPLOY_HMAC_SECRET}"

database:
  url: "${DATABASE_URL}"

logging:
  level: debug
  format: text
```

**Core engine flow:**

```go
// examples/engine/internal/runner/runner.go
package runner

// Run executes a workflow definition against an incoming request.
// It walks the graph in topological order, passing data between nodes
// through an ExecutionContext.
func (r *Runner) Run(ctx context.Context, workflow WorkflowDef, request *http.Request) (*http.Response, error) {
    runID := uuid.New().String()
    execCtx := NewExecutionContext(runID, request)

    // Topological sort to get execution order
    order, err := TopologicalSort(workflow.Nodes, workflow.Edges)
    if err != nil {
        return nil, fmt.Errorf("cycle detected in workflow: %w", err)
    }

    for _, nodeID := range order {
        node := workflow.FindNode(nodeID)
        executor := r.registry.Get(node.DefinitionID)
        if executor == nil {
            r.reporter.ReportStatus(runID, nodeID, "failed", "no executor for: "+node.DefinitionID)
            continue
        }

        // Report: node is running
        r.reporter.ReportStatus(runID, nodeID, "running", "")

        // Gather input data from upstream edges
        inputData := execCtx.GatherInputs(nodeID, workflow.Edges)

        // Execute the node
        result, err := executor.Execute(execCtx, node, inputData)
        if err != nil {
            r.reporter.ReportStatus(runID, nodeID, "failed", err.Error())
            // Route to error port if available
            execCtx.SetOutput(nodeID, "out-error", map[string]any{
                "error":   err.Error(),
                "nodeId":  nodeID,
                "inputData": inputData,
            })
            continue
        }

        // Store output for downstream nodes
        execCtx.SetOutput(nodeID, result.OutputPort, result.Data)

        // Report: node completed
        r.reporter.ReportStatusWithData(runID, nodeID, "completed", result.Summary)
    }

    return execCtx.GetHTTPResponse(), nil
}
```

```go
// examples/engine/internal/runner/context.go
package runner

// ExecutionContext carries data between nodes during a single workflow run.
// Think of it as a conveyor belt: each node picks up data from the belt,
// transforms it, and puts the result back for the next node.
type ExecutionContext struct {
    RunID      string
    Request    *http.Request
    Response   *ResponseBuilder
    outputs    map[string]map[string]any // nodeID → portID → data
    mu         sync.RWMutex
}

// GatherInputs finds all edges targeting a given node and collects
// the output data from their source nodes/ports.
func (ec *ExecutionContext) GatherInputs(nodeID string, edges []EdgeDef) map[string]any {
    merged := make(map[string]any)
    for _, edge := range edges {
        if edge.TargetNodeID == nodeID {
            if data, ok := ec.GetOutput(edge.SourceNodeID, edge.SourcePortID); ok {
                merged[edge.TargetPortID] = data
            }
        }
    }
    // If there's only one input, unwrap it for convenience
    if len(merged) == 1 {
        for _, v := range merged {
            return v.(map[string]any)
        }
    }
    return merged
}
```

**Executor interface:**

```go
// examples/engine/internal/executors/registry.go
package executors

// NodeExecutor is the interface every node type must implement.
// It takes an execution context, the node's configuration, and input data,
// then returns the output data and which port it should flow through.
type NodeExecutor interface {
    Execute(ctx *runner.ExecutionContext, node NodeDef, input map[string]any) (*ExecutionResult, error)
}

type ExecutionResult struct {
    OutputPort string         // which output port the data flows to (e.g., "out-main", "out-get")
    Data       map[string]any // the data to pass downstream
    Summary    map[string]any // brief summary for Graphiti's execution display
}

// Registry maps node definition IDs to their executor implementations.
type Registry struct {
    executors map[string]NodeExecutor
}

func NewRegistry(db *sql.DB) *Registry {
    r := &Registry{executors: make(map[string]NodeExecutor)}
    r.Register("source-api-gateway", &APIGatewayExecutor{})
    r.Register("control-http-router", &HTTPRouterExecutor{})
    r.Register("processing-validate-payload", &ValidatePayloadExecutor{})
    r.Register("processing-json-transform", &JSONTransformExecutor{})
    r.Register("destination-postgresql", &PostgreSQLExecutor{DB: db})
    r.Register("destination-http-response", &HTTPResponseExecutor{})
    return r
}
```

**Acceptance Criteria:**

- [ ] The engine starts with `go run cmd/engine/main.go` and listens on port 9090
- [ ] `POST /webhooks/deploy` receives Graphiti's deploy payload and stores the workflow definition in memory
- [ ] The engine verifies the HMAC signature on incoming deploy payloads
- [ ] When a request arrives at a path matching a deployed workflow's API Gateway, the engine executes the workflow
- [ ] Execution follows topological order (nodes with no unsatisfied dependencies run first)
- [ ] Each node's executor is looked up by `definitionID` from the registry
- [ ] Data flows between nodes through the `ExecutionContext` based on edge definitions
- [ ] The engine sends status callbacks to Graphiti for every node transition (running, completed, failed)
- [ ] Cycle detection prevents infinite loops (returns error before execution)
- [ ] The runner handles the "conditional routing" pattern (HTTP Router sends data to exactly one output port per request)
- [ ] Tests cover: topological sort, execution context data flow, each executor in isolation

#### Story 4.5.8: Node Executor Implementations

**What:** The concrete executors for each node type. Each one is straightforward, they read config from the node definition's attributes and process data.

**API Gateway Executor:**

```go
// examples/engine/internal/executors/api_gateway.go
package executors

type APIGatewayExecutor struct{}

func (e *APIGatewayExecutor) Execute(
    ctx *runner.ExecutionContext, node NodeDef, _ map[string]any,
) (*ExecutionResult, error) {
    req := ctx.Request
    pathParams := extractPathParams(node.Attributes["path"].(string), req.URL.Path)

    var body map[string]any
    if req.Body != nil {
        json.NewDecoder(req.Body).Decode(&body)
    }

    return &ExecutionResult{
        OutputPort: "out-main",
        Data: map[string]any{
            "method":      req.Method,
            "path":        req.URL.Path,
            "pathParams":  pathParams,
            "queryParams": flattenQuery(req.URL.Query()),
            "headers":     flattenHeaders(req.Header),
            "body":        body,
            "clientIp":    req.RemoteAddr,
            "timestamp":   time.Now().UTC().Format(time.RFC3339),
        },
        Summary: map[string]any{
            "method": req.Method,
            "path":   req.URL.Path,
        },
    }, nil
}
```

**HTTP Router Executor:**

```go
// examples/engine/internal/executors/http_router.go
package executors

type HTTPRouterExecutor struct{}

func (e *HTTPRouterExecutor) Execute(
    ctx *runner.ExecutionContext, node NodeDef, input map[string]any,
) (*ExecutionResult, error) {
    method := input["method"].(string)
    pathParams, _ := input["pathParams"].(map[string]any)
    idParam := node.Attributes["id_param"].(string)

    _, hasID := pathParams[idParam]

    var outputPort string
    switch {
    case method == "GET" && !hasID:
        outputPort = "out-get"
    case method == "GET" && hasID:
        outputPort = "out-get-by-id"
    case method == "POST":
        outputPort = "out-post"
    case method == "PUT":
        outputPort = "out-put"
    case method == "DELETE":
        outputPort = "out-delete"
    default:
        outputPort = "out-unmatched"
    }

    return &ExecutionResult{
        OutputPort: outputPort,
        Data:       input, // pass through all request data
        Summary:    map[string]any{"route": outputPort, "method": method},
    }, nil
}
```

**PostgreSQL Executor:**

```go
// examples/engine/internal/executors/postgresql.go
package executors

type PostgreSQLExecutor struct {
    DB *sql.DB
}

func (e *PostgreSQLExecutor) Execute(
    ctx *runner.ExecutionContext, node NodeDef, input map[string]any,
) (*ExecutionResult, error) {
    sqlTemplate := node.Attributes["sql"].(string)
    operation := node.Attributes["operation"].(string)
    paramPaths := parseJSONArray(node.Attributes["params"])
    timeoutSec := intOrDefault(node.Attributes["timeout_seconds"], 30)

    // Resolve parameter values from input data using dot-path notation
    // e.g., "body.title" extracts input["body"]["title"]
    params := make([]any, len(paramPaths))
    for i, path := range paramPaths {
        params[i] = resolveDotPath(input, path)
    }

    queryCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
    defer cancel()

    switch operation {
    case "query":
        rows, err := e.DB.QueryContext(queryCtx, sqlTemplate, params...)
        if err != nil {
            return nil, fmt.Errorf("query failed: %w", err)
        }
        defer rows.Close()
        results := rowsToMaps(rows)
        return &ExecutionResult{
            OutputPort: "out-main",
            Data:       map[string]any{"rows": results, "count": len(results)},
            Summary:    map[string]any{"rowCount": len(results)},
        }, nil

    case "query-row":
        rows, err := e.DB.QueryContext(queryCtx, sqlTemplate, params...)
        if err != nil {
            return nil, fmt.Errorf("query-row failed: %w", err)
        }
        defer rows.Close()
        results := rowsToMaps(rows)
        if len(results) == 0 {
            return nil, fmt.Errorf("not found")
        }
        return &ExecutionResult{
            OutputPort: "out-main",
            Data:       results[0],
            Summary:    map[string]any{"found": true},
        }, nil

    case "exec":
        result, err := e.DB.ExecContext(queryCtx, sqlTemplate, params...)
        if err != nil {
            return nil, fmt.Errorf("exec failed: %w", err)
        }
        affected, _ := result.RowsAffected()
        lastID, _ := result.LastInsertId()
        // For INSERT with RETURNING, use query instead
        // This handles simple INSERT/UPDATE/DELETE
        return &ExecutionResult{
            OutputPort: "out-main",
            Data:       map[string]any{"rowsAffected": affected, "lastInsertId": lastID},
            Summary:    map[string]any{"rowsAffected": affected},
        }, nil
    }

    return nil, fmt.Errorf("unknown operation: %s", operation)
}
```

**Validate Payload Executor:**

```go
// examples/engine/internal/executors/validate_payload.go
package executors

type ValidatePayloadExecutor struct{}

func (e *ValidatePayloadExecutor) Execute(
    ctx *runner.ExecutionContext, node NodeDef, input map[string]any,
) (*ExecutionResult, error) {
    requiredStr := node.Attributes["required_fields"].(string)
    required := splitAndTrim(requiredStr, ",")

    body, _ := input["body"].(map[string]any)
    if body == nil {
        return &ExecutionResult{
            OutputPort: "out-error",
            Data:       map[string]any{"error": "request body is empty", "status": 400},
        }, nil
    }

    var missing []string
    for _, field := range required {
        if _, exists := body[field]; !exists {
            missing = append(missing, field)
        }
    }

    if len(missing) > 0 {
        return &ExecutionResult{
            OutputPort: "out-error",
            Data: map[string]any{
                "error":  fmt.Sprintf("missing required fields: %s", strings.Join(missing, ", ")),
                "status": 400,
                "fields": missing,
            },
        }, nil
    }

    // Validation passed, forward the full input
    return &ExecutionResult{
        OutputPort: "out-main",
        Data:       input,
        Summary:    map[string]any{"valid": true, "fieldsChecked": len(required)},
    }, nil
}
```

**HTTP Response Executor:**

```go
// examples/engine/internal/executors/http_response.go
package executors

type HTTPResponseExecutor struct{}

func (e *HTTPResponseExecutor) Execute(
    ctx *runner.ExecutionContext, node NodeDef, input map[string]any,
) (*ExecutionResult, error) {
    statusCode := intOrDefault(node.Attributes["status_code"], 200)
    contentType := stringOrDefault(node.Attributes["content_type"], "application/json")

    // If input came through the error port, override status from error data
    if errStatus, ok := input["status"]; ok {
        statusCode = int(errStatus.(float64))
    }

    // Build the response on the execution context
    ctx.Response.SetStatus(statusCode)
    ctx.Response.SetHeader("Content-Type", contentType)

    // Apply custom headers
    if headers, ok := node.Attributes["headers"].(map[string]any); ok {
        for k, v := range headers {
            ctx.Response.SetHeader(k, fmt.Sprint(v))
        }
    }

    // Body: use template if provided, otherwise forward input as-is
    bodyTemplate, _ := node.Attributes["body_template"].(string)
    if bodyTemplate != "" {
        ctx.Response.SetBodyFromTemplate(bodyTemplate, input)
    } else {
        ctx.Response.SetBodyJSON(input)
    }

    return &ExecutionResult{
        OutputPort: "", // terminal node, no output port
        Data:       nil,
        Summary:    map[string]any{"statusCode": statusCode},
    }, nil
}
```

**Acceptance Criteria:**

- [ ] **API Gateway:** extracts method, path, pathParams (`:id` → `{"id": "123"}`), queryParams, headers, body, clientIp, timestamp
- [ ] **API Gateway:** supports path parameter extraction (e.g., `/api/todos/:id` matches `/api/todos/42` and yields `{"id": "42"}`)
- [ ] **HTTP Router:** routes to the correct output port based on method and presence of an ID param
- [ ] **HTTP Router:** sends unmatched methods to the "out-unmatched" error port
- [ ] **Validate Payload:** checks required fields, returns error data through `out-error` port on failure
- [ ] **Validate Payload:** passes through full input data on success through `out-main`
- [ ] **PostgreSQL:** resolves parameter values from input using dot-path notation (`body.title` → `input["body"]["title"]`)
- [ ] **PostgreSQL:** handles all three operation modes: `query` (returns rows), `query-row` (returns single row or error), `exec` (returns affected count)
- [ ] **PostgreSQL:** enforces configurable query timeout
- [ ] **PostgreSQL:** returns through `out-error` port on SQL errors
- [ ] **HTTP Response:** sets status code, content-type, custom headers on the execution context response
- [ ] **HTTP Response:** accepts input from both data and error ports (error port input can override status code)
- [ ] **JSON Transform:** maps fields using dot-path notation from input to output
- [ ] Each executor has unit tests with table-driven test cases
- [ ] Tests for PostgreSQL executor use a test Postgres instance (via docker or testcontainers)

#### Story 4.5.9: Status Callback Reporter

**What:** The component that reports execution progress back to Graphiti, so the execution mode UI can show nodes lighting up in real-time.

```go
// examples/engine/internal/callback/reporter.go
package callback

type Reporter struct {
    callbackURL string
    hmacSecret  string
    client      *http.Client
}

func (r *Reporter) ReportStatus(runID, nodeID, status, errorMsg string) {
    r.ReportStatusWithData(runID, nodeID, status, nil)
}

func (r *Reporter) ReportStatusWithData(runID, nodeID, status string, summary map[string]any) {
    payload := map[string]any{
        "apiVersion":  "graphiti/v1",
        "event":       "execution.node_status",
        "runID":       runID,
        "nodeID":      nodeID,
        "status":      status,
        "timestamp":   time.Now().UTC().Format(time.RFC3339),
    }

    if status == "running" {
        payload["startedAt"] = time.Now().UTC().Format(time.RFC3339)
    }
    if status == "completed" || status == "failed" {
        payload["completedAt"] = time.Now().UTC().Format(time.RFC3339)
    }
    if errorMsg != "" {
        payload["error"] = errorMsg
    }
    if summary != nil {
        payload["outputSummary"] = summary
    }

    body, _ := json.Marshal(payload)

    // Fire and forget (with retry). Don't block execution waiting for Graphiti.
    go func() {
        for attempt := 0; attempt < 3; attempt++ {
            req, _ := http.NewRequest("POST", r.callbackURL, bytes.NewReader(body))
            req.Header.Set("Content-Type", "application/json")
            if r.hmacSecret != "" {
                sig := hmacSign(body, r.hmacSecret)
                req.Header.Set("X-Graphiti-Signature", "sha256="+sig)
            }
            resp, err := r.client.Do(req)
            if err == nil && resp.StatusCode < 500 {
                resp.Body.Close()
                return
            }
            time.Sleep(time.Duration(1<<attempt) * time.Second)
        }
    }()
}
```

**Acceptance Criteria:**

- [ ] Reporter sends callbacks asynchronously (doesn't block workflow execution)
- [ ] Callbacks are HMAC-signed when a shared secret is configured
- [ ] Retries up to 3 times with exponential backoff on failure
- [ ] The payload matches the `execution.node_status` schema from the PRD (section 10.5)
- [ ] Includes `startedAt` when status is "running" and `completedAt` when status is "completed" or "failed"
- [ ] Error messages are included in the payload when status is "failed"
- [ ] Output summaries are included when available (e.g., `{"rowCount": 5}`)

#### Story 4.5.10: Pre-Built ToDo Workflow Definition

**What:** A ready-to-load workflow YAML that wires up the complete ToDo CRUD API. Users can import this to see a fully working example without building it by hand.

```yaml
# examples/engine/workflows/todo-api.yaml
apiVersion: graphiti/v1
kind: WorkflowDefinition

metadata:
  name: "ToDo REST API"
  description: "A complete CRUD API for ToDo items, backed by PostgreSQL"

nodes:
  - id: node-api-gateway
    definitionID: source-api-gateway
    label: "ToDo API"
    x: 50
    y: 250
    attributes:
      path: "/api/todos/:id"
      methods: "ALL"
      auth_type: "none"

  - id: node-router
    definitionID: control-http-router
    label: "Method Router"
    x: 320
    y: 250
    attributes:
      id_param: "id"

  # GET all todos
  - id: node-pg-list
    definitionID: destination-postgresql
    label: "List Todos"
    x: 620
    y: 50
    attributes:
      connection_string: "${DATABASE_URL}"
      operation: "query"
      sql: "SELECT id, title, completed, created_at, updated_at FROM todos ORDER BY created_at DESC"
      params: "[]"

  - id: node-res-list
    definitionID: destination-http-response
    label: "200 OK (List)"
    x: 920
    y: 50
    attributes:
      status_code: 200
      content_type: "application/json"

  # GET single todo
  - id: node-pg-get
    definitionID: destination-postgresql
    label: "Get Todo"
    x: 620
    y: 150
    attributes:
      connection_string: "${DATABASE_URL}"
      operation: "query-row"
      sql: "SELECT id, title, completed, created_at, updated_at FROM todos WHERE id = $1"
      params: '["pathParams.id"]'

  - id: node-res-get
    definitionID: destination-http-response
    label: "200 OK (Get)"
    x: 920
    y: 150
    attributes:
      status_code: 200
      content_type: "application/json"

  # POST create todo
  - id: node-val-post
    definitionID: processing-validate-payload
    label: "Validate Create"
    x: 620
    y: 270
    attributes:
      required_fields: "title"

  - id: node-pg-insert
    definitionID: destination-postgresql
    label: "Insert Todo"
    x: 920
    y: 250
    attributes:
      connection_string: "${DATABASE_URL}"
      operation: "query-row"
      sql: "INSERT INTO todos (title, completed) VALUES ($1, $2) RETURNING id, title, completed, created_at, updated_at"
      params: '["body.title", "body.completed"]'

  - id: node-res-create
    definitionID: destination-http-response
    label: "201 Created"
    x: 1220
    y: 250
    attributes:
      status_code: 201
      content_type: "application/json"

  # PUT update todo
  - id: node-val-put
    definitionID: processing-validate-payload
    label: "Validate Update"
    x: 620
    y: 400
    attributes:
      required_fields: "title,completed"

  - id: node-pg-update
    definitionID: destination-postgresql
    label: "Update Todo"
    x: 920
    y: 380
    attributes:
      connection_string: "${DATABASE_URL}"
      operation: "query-row"
      sql: "UPDATE todos SET title = $1, completed = $2, updated_at = NOW() WHERE id = $3 RETURNING id, title, completed, created_at, updated_at"
      params: '["body.title", "body.completed", "pathParams.id"]'

  - id: node-res-update
    definitionID: destination-http-response
    label: "200 OK (Update)"
    x: 1220
    y: 380
    attributes:
      status_code: 200
      content_type: "application/json"

  # DELETE todo
  - id: node-pg-delete
    definitionID: destination-postgresql
    label: "Delete Todo"
    x: 620
    y: 520
    attributes:
      connection_string: "${DATABASE_URL}"
      operation: "exec"
      sql: "DELETE FROM todos WHERE id = $1"
      params: '["pathParams.id"]'

  - id: node-res-delete
    definitionID: destination-http-response
    label: "204 No Content"
    x: 920
    y: 520
    attributes:
      status_code: 204
      content_type: "application/json"

  # Error response (shared by validation failures)
  - id: node-res-error
    definitionID: destination-http-response
    label: "400 Bad Request"
    x: 920
    y: 470
    attributes:
      status_code: 400
      content_type: "application/json"

edges:
  # API Gateway → Router
  - id: edge-1
    sourceNodeID: node-api-gateway
    sourcePortID: out-main
    targetNodeID: node-router
    targetPortID: in-main

  # Router → branches
  - id: edge-2
    sourceNodeID: node-router
    sourcePortID: out-get
    targetNodeID: node-pg-list
    targetPortID: in-main

  - id: edge-3
    sourceNodeID: node-router
    sourcePortID: out-get-by-id
    targetNodeID: node-pg-get
    targetPortID: in-main

  - id: edge-4
    sourceNodeID: node-router
    sourcePortID: out-post
    targetNodeID: node-val-post
    targetPortID: in-main

  - id: edge-5
    sourceNodeID: node-router
    sourcePortID: out-put
    targetNodeID: node-val-put
    targetPortID: in-main

  - id: edge-6
    sourceNodeID: node-router
    sourcePortID: out-delete
    targetNodeID: node-pg-delete
    targetPortID: in-main

  # GET paths → responses
  - id: edge-7
    sourceNodeID: node-pg-list
    sourcePortID: out-main
    targetNodeID: node-res-list
    targetPortID: in-main

  - id: edge-8
    sourceNodeID: node-pg-get
    sourcePortID: out-main
    targetNodeID: node-res-get
    targetPortID: in-main

  # POST path: validate → insert → response
  - id: edge-9
    sourceNodeID: node-val-post
    sourcePortID: out-main
    targetNodeID: node-pg-insert
    targetPortID: in-main

  - id: edge-10
    sourceNodeID: node-pg-insert
    sourcePortID: out-main
    targetNodeID: node-res-create
    targetPortID: in-main

  # PUT path: validate → update → response
  - id: edge-11
    sourceNodeID: node-val-put
    sourcePortID: out-main
    targetNodeID: node-pg-update
    targetPortID: in-main

  - id: edge-12
    sourceNodeID: node-pg-update
    sourcePortID: out-main
    targetNodeID: node-res-update
    targetPortID: in-main

  # DELETE path → response
  - id: edge-13
    sourceNodeID: node-pg-delete
    sourcePortID: out-main
    targetNodeID: node-res-delete
    targetPortID: in-main

  # Validation errors → error response
  - id: edge-14
    sourceNodeID: node-val-post
    sourcePortID: out-error
    targetNodeID: node-res-error
    targetPortID: in-error

  - id: edge-15
    sourceNodeID: node-val-put
    sourcePortID: out-error
    targetNodeID: node-res-error
    targetPortID: in-error
```

**Acceptance Criteria:**

- [ ] The YAML file can be imported into Graphiti and renders all 14 nodes with correct shapes and categories
- [ ] All 15 edges render as Bezier curves connecting the correct ports
- [ ] The visual layout is clean and readable without manual rearrangement
- [ ] The workflow passes Graphiti's pre-deploy validation (no missing required fields, no cycles, valid connections)
- [ ] When deployed and running, all 5 CRUD operations work end-to-end

#### Story 4.5.11: Docker Compose and Database Setup

**What:** A single `docker-compose.yml` that spins up Graphiti, the sample engine, and Postgres. One command, full stack.

```yaml
# docker-compose.yml (project root)
version: "3.8"

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: graphiti
      POSTGRES_PASSWORD: graphiti_dev
      POSTGRES_DB: graphiti_todos
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./examples/engine/init.sql:/docker-entrypoint-initdb.d/01-init.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U graphiti"]
      interval: 5s
      timeout: 5s
      retries: 5

  graphiti:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      DEPLOY_HMAC_SECRET: "dev-secret-change-in-prod"
      DEPLOY_WEBHOOK_PROD: "http://engine:9090/webhooks/deploy"
      DEPLOY_WEBHOOK_STAGING: "http://engine:9090/webhooks/deploy"
      SESSION_SECRET: "dev-session-secret"
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - ./config:/app/config

  engine:
    build:
      context: ./examples/engine
      dockerfile: Dockerfile
    ports:
      - "9090:9090"
    environment:
      DATABASE_URL: "postgres://graphiti:graphiti_dev@postgres:5432/graphiti_todos?sslmode=disable"
      GRAPHITI_CALLBACK_URL: "http://graphiti:8080/api/callbacks/execution"
      DEPLOY_HMAC_SECRET: "dev-secret-change-in-prod"
    depends_on:
      postgres:
        condition: service_healthy
      graphiti:
        condition: service_started

volumes:
  pgdata:
```

**Postgres init script:**

```sql
-- examples/engine/init.sql
CREATE TABLE IF NOT EXISTS todos (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    completed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed some sample data so the demo isn't empty
INSERT INTO todos (title, completed) VALUES
    ('Read the Graphiti docs', true),
    ('Build a sample workflow', false),
    ('Deploy to production', false),
    ('Write integration tests', false);
```

**Engine Dockerfile:**

```dockerfile
# examples/engine/Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o engine ./cmd/engine/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/engine /usr/local/bin/engine
COPY --from=builder /app/config.yaml /etc/engine/config.yaml
EXPOSE 9090
CMD ["engine"]
```

**Acceptance Criteria:**

- [ ] `docker-compose up` starts all three services with no manual steps
- [ ] Postgres initializes with the `todos` table and seed data
- [ ] Graphiti is accessible at `http://localhost:8080`
- [ ] The sample engine is accessible at `http://localhost:9090`
- [ ] Graphiti's deploy webhook is pre-configured to point at the engine
- [ ] The engine's callback URL is pre-configured to point at Graphiti
- [ ] Health checks ensure Postgres is ready before the engine starts
- [ ] `docker-compose down -v` cleanly removes everything including data
- [ ] All three services share the same HMAC secret for webhook signing
- [ ] The engine's `/webhooks/deploy` endpoint accepts Graphiti's deploy payload
- [ ] After deploying the ToDo workflow, the following curl commands work:

```bash
# List all todos
curl http://localhost:9090/api/todos

# Get a single todo
curl http://localhost:9090/api/todos/1

# Create a new todo
curl -X POST http://localhost:9090/api/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Try Graphiti", "completed": false}'

# Update a todo
curl -X PUT http://localhost:9090/api/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"title": "Read the Graphiti docs", "completed": true}'

# Delete a todo
curl -X DELETE http://localhost:9090/api/todos/1

# Validation error (missing title)
curl -X POST http://localhost:9090/api/todos \
  -H "Content-Type: application/json" \
  -d '{"completed": true}'
# → 400 Bad Request: {"error": "missing required fields: title"}
```

#### Story 4.5.12: Developer Guide for Building Graphiti Backends

**What:** A comprehensive guide that teaches developers how to build their own execution engine for Graphiti. Lives in `docs/building-an-engine.md`. Uses the sample engine as a running example but explains the general principles.

**Document outline:**

```
docs/building-an-engine.md

1. How Graphiti Talks to Engines
   - The webhook contract (deploy payload schema)
   - HMAC signature verification
   - The callback contract (execution status schema)
   - What each field means and when it matters

2. Receiving a Workflow Definition
   - Parsing the deploy payload
   - Understanding the node/edge graph structure
   - Storing deployed workflows (in memory, database, wherever)
   - Handling re-deploys (version bumps)

3. Executing a Workflow
   - Topological sorting: why order matters
   - Walking the graph node by node
   - The execution context pattern (passing data between nodes)
   - Handling conditional routing (one input, multiple output ports)
   - Error paths and error ports
   - Parallel execution (future consideration)

4. Building Node Executors
   - The executor interface pattern
   - Reading node configuration from attributes
   - Resolving attribute values (dot-path notation, env vars)
   - Input/output data contracts
   - Error handling: when to use error ports vs. exceptions

5. Reporting Status Back to Graphiti
   - The callback endpoint and payload format
   - When to send "running" vs "completed" vs "failed"
   - Async reporting (don't block execution)
   - Retry and error handling for callbacks
   - What Graphiti does with the status (WebSocket → execution mode UI)

6. Creating Custom Node Types
   - Writing a YAML node definition
   - Choosing shape, ports, and attributes
   - Writing the matching executor
   - Registering the executor in the engine
   - Testing the node type end-to-end

7. Running the Example
   - docker-compose quickstart
   - Building the ToDo workflow step-by-step
   - Deploying and testing with curl
   - Watching execution in Graphiti's execution mode

8. Reference
   - Full deploy payload schema
   - Full callback payload schema
   - Supported attribute types and their runtime values
   - Port type compatibility matrix
```

**Acceptance Criteria:**

- [ ] The guide is written for developers who've never seen Graphiti before
- [ ] Every code example compiles and works (extracted from the sample engine)
- [ ] The webhook payload schema is documented field-by-field with types and examples
- [ ] The callback payload schema is documented the same way
- [ ] The "Building Node Executors" section includes a complete example of a custom node from YAML to executor to test
- [ ] The "Running the Example" section has copy-pasteable commands for the full demo
- [ ] The guide explains the topological sort requirement and why it matters (with a diagram)
- [ ] Error handling patterns are explained: when to route to an error port vs. when to fail the whole run
- [ ] The guide links to the PRD sections for deeper context on the webhook contract and execution status

**Phase 4.5 Demo Checkpoint:**

```bash
# Terminal 1: start the full stack
docker-compose up

# Terminal 2: open Graphiti and watch execution
open http://localhost:8080
# → Log in, create or import the ToDo workflow
# → Click Deploy
# → Switch to Execution Mode

# Terminal 3: fire requests at the API
curl -s http://localhost:9090/api/todos | jq .
# → Watch nodes light up in the browser:
#   API Gateway (amber → green) → HTTP Router (amber → green) → List Todos (amber → green) → 200 OK (amber → green)

curl -s -X POST http://localhost:9090/api/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Demo Graphiti at standup"}' | jq .
# → Watch the POST path light up:
#   API Gateway → HTTP Router → Validate Create → Insert Todo → 201 Created

curl -s -X POST http://localhost:9090/api/todos \
  -H "Content-Type: application/json" \
  -d '{"completed": true}' | jq .
# → Watch the error path light up:
#   API Gateway → HTTP Router → Validate Create → 400 Bad Request (error port, red)
```

The demo tells the full story: you design a workflow visually, deploy it with a click, and then real HTTP requests flow through it with every step visible in real-time. That's the pitch for Graphiti in 60 seconds.

## Phase 5: Advanced Features

**Goal:** Sub-workflows, the theme system, performance optimizations for large canvases, the full keyboard shortcut map, and accessibility improvements. At demo time, you can double-click a sub-workflow node to drill in, switch between light and dark themes, and navigate a 200-node workflow without lag.

**Estimated Duration:** 4-5 weeks

```mermaid
graph LR
    A["Sub-Workflow<br/>Node Type"] --> B["Sub-Workflow<br/>Navigation"]
    B --> C["Theme System<br/>Light/Dark/System"]
    C --> D["Viewport Culling<br/>+ LOD"]
    D --> E["Full Keyboard<br/>Shortcut Map"]
    E --> F["Accessibility<br/>Audit + Fixes"]
    style F fill:#10B981,color:#fff
```

### Story 5.1: Sub-Workflow Node Type

**What:** A special node that references another workflow. It shows up in the Control Flow category with a double-bordered visual.

**Acceptance Criteria:**

- [ ] A "Sub-workflow" node definition ships in `config/nodes/control/sub-workflow.yaml`
- [ ] The node uses `shape.type: sub-workflow` which renders as a double-bordered rounded rectangle
- [ ] The config panel shows a "Referenced Workflow" dropdown listing all saved workflows (type `workflow-reference`)
- [ ] Selecting a workflow binds the sub-workflow node to it
- [ ] The sub-workflow node's ports match: one data input, one data output, one error output
- [ ] Input/output mapping attributes (`json` type) are configurable for data transformation between parent and child
- [ ] The sub-workflow node displays the referenced workflow's name inside the node body
- [ ] Validation ensures the referenced workflow exists and isn't circular (workflow A can't reference B if B references A)

### Story 5.2: Sub-Workflow Navigation

**What:** Double-click a sub-workflow node to drill into the referenced workflow's canvas. Breadcrumbs track the nesting path.

**Acceptance Criteria:**

- [ ] Double-clicking a sub-workflow node transitions the canvas to show the inner workflow
- [ ] The transition is animated (quick fade or slide)
- [ ] The breadcrumb updates to show the nesting path: `/ Workflows / Main Pipeline / Data Enrichment (sub)`
- [ ] All builder features work inside the sub-workflow (drag, connect, configure, undo/redo)
- [ ] Clicking a breadcrumb segment navigates back to that level
- [ ] A "Back" button/icon is visible for quick return to the parent
- [ ] Nesting depth is unlimited, but a practical limit of 5 levels is enforced with a warning
- [ ] Changes to the inner workflow are saved independently (they're separate workflow records)
- [ ] The sub-workflow's undo/redo stack is independent from the parent's

### Story 5.3: Theme System (Light, Dark, System)

**What:** CSS custom properties driven by YAML theme definitions, with a three-way toggle.

**Theme YAML structure:**

```yaml
# config/themes/light.yaml
name: light
variables:
  bg: "#F8F9FB"
  surface: "#FFFFFF"
  surface-2: "#F1F3F5"
  border: "#E2E5EA"
  text: "#1A1D23"
  text-muted: "#6B7280"
  accent: "#3B82F6"
  node-source: "#0891B2"
  node-processor: "#9333EA"
  node-destination: "#D97706"
  node-control: "#059669"
```

**Acceptance Criteria:**

- [ ] The application defaults to light mode on first visit
- [ ] A theme toggle (in settings or top nav) offers: Light, Dark, System
- [ ] "System" follows the OS `prefers-color-scheme` media query and reacts to changes
- [ ] Theme is implemented via CSS custom properties on `:root`, toggled with `data-theme` attribute on `<html>`
- [ ] Theme YAML files in `config/themes/` are compiled into CSS variable blocks at startup
- [ ] All UI elements respect the theme: panels, buttons, inputs, SVG canvas, nodes, edges, ports
- [ ] SVG node header backgrounds, port colors, and edge colors update with the theme
- [ ] The transition between themes uses CSS transitions on `background-color`, `color`, and `border-color` (200ms ease)
- [ ] Theme preference is persisted in localStorage (client-side) and optionally in the user's session (server-side)
- [ ] Custom themes can be added by creating new YAML files in `config/themes/`
- [ ] Theme variables cover all colors listed in PRD section 5.10

### Story 5.4: Viewport Culling and Level-of-Detail for Large Workflows

**What:** Performance optimizations that keep the canvas smooth with hundreds of nodes.

**Acceptance Criteria:**

- [ ] Nodes outside the visible viewport are replaced with lightweight placeholder `<rect>` elements (no text rendering, no attribute rows)
- [ ] As the user pans, nodes entering the viewport are replaced with their full SVG representation
- [ ] At zoom levels below 50%, nodes simplify to colored rectangles without text or attributes (level-of-detail)
- [ ] At zoom levels below 30%, ports become invisible
- [ ] Edge path strings are cached and only recalculated when connected nodes move
- [ ] Position updates during drag use `requestAnimationFrame` batching
- [ ] A workflow with 100 nodes renders initial load in under 200ms
- [ ] A workflow with 500 nodes renders initial load in under 1 second with culling active
- [ ] Node drag maintains 60fps (under 16ms frame budget) regardless of workflow size
- [ ] The implementation uses an R-tree or spatial index to quickly determine which nodes are in the viewport

### Story 5.5: Full Keyboard Shortcut Map

**What:** Every interaction has a keyboard shortcut. A help overlay (triggered by `?`) shows all shortcuts.

| Shortcut | Action |
|---|---|
| Ctrl/Cmd+Z | Undo |
| Ctrl/Cmd+Shift+Z | Redo |
| Ctrl/Cmd+C | Copy |
| Ctrl/Cmd+X | Cut |
| Ctrl/Cmd+V | Paste |
| Ctrl/Cmd+D | Duplicate |
| Ctrl/Cmd+A | Select all |
| Delete / Backspace | Delete selection |
| Ctrl/Cmd+S | Save (explicit) |
| Ctrl/Cmd+= | Zoom in |
| Ctrl/Cmd+- | Zoom out |
| Ctrl/Cmd+0 | Fit to view |
| Space+drag | Pan |
| Escape | Deselect all / Cancel current operation |
| Tab | Cycle selection to next node |
| ? | Show keyboard shortcut help overlay |

**Acceptance Criteria:**

- [ ] All shortcuts in the table above are implemented and functional
- [ ] Shortcuts don't conflict with browser defaults (browser defaults are prevented via `e.preventDefault()`)
- [ ] Mac users see Cmd, Windows/Linux users see Ctrl (detected via `navigator.platform`)
- [ ] Pressing `?` shows a modal overlay listing all shortcuts, organized by category
- [ ] The shortcut overlay is dismissible with Escape or clicking outside
- [ ] Shortcuts are disabled when a text input or textarea is focused (to avoid conflicts with typing)

### Story 5.6: Accessibility Audit and Fixes

**What:** Make the application usable for people with disabilities, meeting WCAG 2.1 AA standards for key flows.

**Acceptance Criteria:**

- [ ] All interactive elements are keyboard-navigable (Tab order makes sense)
- [ ] SVG nodes have `aria-label` attributes describing the node type and name
- [ ] SVG edges have `aria-label` describing the connection (e.g., "Connection from Twilio output to Transcribe input")
- [ ] Color is never the sole indicator of state. Status overlays include icons or patterns alongside color (checkmark for success, X for failure, spinner for running)
- [ ] Focus indicators are clearly visible in both light and dark themes (2px solid outline with offset)
- [ ] Screen reader announcements fire for canvas operations: "Node added: Twilio", "Edge created", "Node deleted"
- [ ] The node palette is navigable with arrow keys
- [ ] The config panel's form fields have proper `<label>` associations
- [ ] Contrast ratios meet WCAG AA (4.5:1 for normal text, 3:1 for large text) in both themes
- [ ] The theme toggle itself is accessible

**Phase 5 Demo Checkpoint:** Build a workflow with a sub-workflow node. Double-click it. You drill into the inner workflow, the breadcrumb updates. Build something inside, navigate back. Toggle to dark mode. The entire UI smoothly transitions. Load a stress-test workflow with 300 nodes. Pan and zoom are still smooth thanks to viewport culling. Press `?` to see the full shortcut map. Navigate the palette with the keyboard using Tab and arrow keys.

## Testing Strategy Summary

Testing isn't a phase, it's continuous. But here's the expected coverage at each layer:

```mermaid
graph TB
    subgraph "Test Pyramid"
        E2E["E2E Tests (Playwright)<br/>Key user flows<br/>~20 tests"]
        HTTP["HTTP Handler Tests (httptest)<br/>Request/response contracts<br/>80%+ coverage"]
        SVC["Service Tests (mocked ports)<br/>Use case logic<br/>90%+ coverage"]
        DOM["Domain Unit Tests (pure functions)<br/>Commands, validation, clipboard<br/>95%+ coverage"]
    end

    E2E --> HTTP
    HTTP --> SVC
    SVC --> DOM

    style DOM fill:#10B981,color:#fff
    style SVC fill:#3B82F6,color:#fff
    style HTTP fill:#F59E0B,color:#fff
    style E2E fill:#EF4444,color:#fff
```

| Layer | Approach | Target | When |
|---|---|---|---|
| Domain | Unit tests, pure functions, table-driven, zero mocks | 95%+ | Phase 1, maintained forever |
| App Services | Unit tests with mocked port interfaces (testify/mock) | 90%+ | Phase 1+, grows with features |
| Adapters (SQLite) | Integration tests against `:memory:` SQLite | 85%+ | Phase 1+, grows with schema |
| Adapters (Webhook) | Integration tests using `httptest.NewServer` | 85%+ | Phase 3 |
| HTTP Handlers | `httptest` with mocked services | 80%+ | Phase 1+, grows with routes |
| HTMX Interactions | Playwright E2E tests | Key flows | Phase 2+ |
| SVG Rendering | Visual regression (screenshot comparison) | Key node types | Phase 2+ |

**TDD Enforcement:** Every pull request must include tests that were written before the implementation code. CI fails if coverage drops below the targets above. The test file must exist in the same commit as (or before) the implementation file.

## Appendix: Configuration Reference

### app.yaml

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  readTimeout: 30s
  writeTimeout: 30s

storage:
  adapter: sqlite
  sqlite:
    path: "./data/graphiti.db"
    walMode: true

nodeDefinitions:
  path: "./config/nodes"
  watchForChanges: true

theme:
  default: light

deploy:
  defaultTarget: production
  hmacSecret: "${DEPLOY_HMAC_SECRET}"
  targets:
    production:
      webhookURL: "${DEPLOY_WEBHOOK_PROD}"
      timeout: 30s
      retries: 3
      retryBackoff: exponential
    staging:
      webhookURL: "${DEPLOY_WEBHOOK_STAGING}"
      timeout: 30s
      retries: 3
      retryBackoff: exponential

commandHistory:
  maxUndoDepth: 100

logging:
  level: info
  format: json
```

### auth.yaml

```yaml
auth:
  provider: fake        # fake | github
  github:
    clientId: "${GITHUB_CLIENT_ID}"
    clientSecret: "${GITHUB_CLIENT_SECRET}"
    scopes: ["user:email", "read:org"]
    allowedOrgs: []
  session:
    secret: "${SESSION_SECRET}"
    maxAge: 86400
    secure: true
```

## Appendix: Open Questions Carried Forward

These decisions from the PRD are deferred but should be resolved during implementation:

1. **Edge routing algorithm:** Start with simple horizontal Bezier curves. Revisit orthogonal routing if user feedback demands it.
2. **Node definition hot-reload:** Implement `watchForChanges` as dev-only. Use a build-time flag or config toggle.
3. **Multi-tenancy:** Start single-tenant. The schema supports `created_by` on workflows, so adding workspace isolation later is a migration, not a rewrite.
4. **Plugin system:** Defer. YAML definitions are enough for v1.
5. **Command batching for config changes:** Use 500ms debounce on input blur. Collapse rapid keystrokes into a single `UpdateAttributeCommand`.
