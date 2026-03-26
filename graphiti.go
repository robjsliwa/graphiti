package graphiti

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"graphiti/internal/adapters/driven/filesystem"
	httpAdapter "graphiti/internal/adapters/driving/http"
	"graphiti/internal/app"
	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
	"graphiti/internal/ports/driving"
	"graphiti/web/templates"
)

// UserContextKey is the context key the host's auth middleware must use
// to store the authenticated user session on the request context.
// Graphiti's handlers read from this key.
var UserContextKey = httpAdapter.ExportedUserContextKey

// Config controls how the embedded Graphiti instance behaves.
//
// All fields are optional. Zero values provide sensible defaults:
// BasePath defaults to "/", node definitions use the embedded default set,
// and the undo stack holds 100 entries.
type Config struct {
	// BasePath records the URL prefix where Graphiti is mounted.
	// Default: "/"
	//
	// Note: Graphiti's internal routes always start at "/". If you mount
	// Graphiti at a sub-path (e.g., "/workflows"), use http.StripPrefix:
	//
	//	mux.Handle("/workflows/", http.StripPrefix("/workflows", app.Handler()))
	BasePath string

	// NodeDefinitionsPath is the filesystem path to YAML node definitions.
	// If empty, uses the embedded default set.
	NodeDefinitionsPath string

	// CommandHistoryMaxDepth controls the undo stack size.
	// Default: 100
	CommandHistoryMaxDepth int

	// Logger is an optional structured logger.
	// If nil, logs to slog.Default().
	Logger *slog.Logger

	// Branding configures the UI appearance.
	Branding templates.Branding
}

// Deps holds the interfaces the host application must provide.
//
// Required fields are WorkflowRepo, UserRepo, and AuthProvider.
// Optional fields disable features when nil: a nil ExecutionRepo hides
// execution mode, and a nil DeployTarget hides the deploy button.
type Deps struct {
	// Required: where workflows are stored.
	WorkflowRepo driven.WorkflowRepository

	// Required: where users are stored.
	UserRepo driven.UserRepository

	// Required: who the current user is.
	AuthProvider driven.AuthProvider

	// Optional: where execution runs are stored.
	// If nil, execution mode is disabled.
	ExecutionRepo driven.ExecutionRepository

	// Optional: deploy target.
	// If nil, the deploy button is hidden.
	DeployTarget driven.DeployTarget

	// Optional: node definition loader.
	// If nil, loads from Config.NodeDefinitionsPath or embedded defaults.
	NodeDefinitionRepo driven.NodeDefinitionRepository

	// Optional: HMAC secret for callback signature verification.
	HMACSecret string

	// Optional: session configuration.
	SessionSecret string
	SessionMaxAge time.Duration
	SessionSecure bool

	// Optional: token validator for bearer token auth (API clients).
	// If nil, only session-based auth is used.
	TokenValidator driven.TokenValidator

	// Optional: CORS configuration for /api/ routes.
	// Empty AllowedOrigins disables CORS headers entirely.
	CORS httpAdapter.CORSConfig
}

// ExecutionUpdate is the data the host engine sends when a node's
// execution status changes.
type ExecutionUpdate struct {
	RunID         string         `json:"runId"`
	WorkflowID    string         `json:"workflowId"`
	NodeID        string         `json:"nodeId"`
	Status        string         `json:"status"` // "running", "completed", "failed", "skipped"
	StartedAt     *time.Time     `json:"startedAt,omitempty"`
	CompletedAt   *time.Time     `json:"completedAt,omitempty"`
	ErrorMessage  string         `json:"errorMessage,omitempty"`
	OutputSummary map[string]any `json:"outputSummary,omitempty"`
}

// Services provides programmatic access to Graphiti's use cases.
type Services struct {
	Workflows    driving.WorkflowService
	NodeRegistry driving.NodeRegistryService
	Executions   driving.ExecutionService
}

// App is an initialized Graphiti instance ready to serve HTTP requests.
type App struct {
	config        Config
	handler       http.Handler
	workflowSvc   driving.WorkflowService
	nodeRegistry  driving.NodeRegistryService
	execSvc       *app.ExecutionService
	hub           *httpAdapter.WebSocketHub
	executionRepo driven.ExecutionRepository
}

// New creates a new Graphiti App from the given configuration and
// dependencies. It initializes the node registry, builds the HTTP handler
// tree, and returns an App ready to be mounted on any router.
//
// Returns an error if required dependencies are nil or if node definition
// loading fails.
func New(cfg Config, deps Deps) (*App, error) {
	// Validate required deps
	if deps.WorkflowRepo == nil {
		return nil, errors.New("graphiti: WorkflowRepo is required")
	}
	if deps.UserRepo == nil {
		return nil, errors.New("graphiti: UserRepo is required")
	}
	if deps.AuthProvider == nil {
		return nil, errors.New("graphiti: AuthProvider is required")
	}

	// Apply defaults
	if cfg.BasePath == "" {
		cfg.BasePath = "/"
	}
	if cfg.CommandHistoryMaxDepth <= 0 {
		cfg.CommandHistoryMaxDepth = 100
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Branding.AppName == "" {
		cfg.Branding.AppName = "Graphiti"
	}
	if cfg.Branding.Tagline == "" {
		cfg.Branding.Tagline = "Visual Workflow Builder"
	}
	if cfg.Branding.TitleSuffix == "" {
		cfg.Branding.TitleSuffix = cfg.Branding.AppName
	}
	if cfg.Branding.ThemeStorageKey == "" {
		cfg.Branding.ThemeStorageKey = "graphiti-theme"
	}

	// Set up node definition loader
	var nodeLoader driven.NodeDefinitionRepository
	if deps.NodeDefinitionRepo != nil {
		nodeLoader = deps.NodeDefinitionRepo
	} else if cfg.NodeDefinitionsPath != "" {
		nodeLoader = filesystem.NewNodeLoader(cfg.NodeDefinitionsPath, cfg.Logger)
	} else {
		// Use embedded defaults
		nodeLoader = filesystem.NewEmbeddedNodeLoader(defaultNodeDefs, "config/nodes", cfg.Logger)
	}

	// Initialize node registry
	nodeRegistry := app.NewNodeRegistry(nodeLoader)
	if err := nodeRegistry.Load(context.Background()); err != nil {
		return nil, fmt.Errorf("graphiti: loading node definitions: %w", err)
	}

	// Initialize app services
	workflowSvc := app.NewWorkflowService(deps.WorkflowRepo, nodeRegistry, cfg.CommandHistoryMaxDepth)
	if deps.DeployTarget != nil {
		workflowSvc.SetDeployTarget(deps.DeployTarget)
	}

	var execSvc *app.ExecutionService
	if deps.ExecutionRepo != nil {
		execSvc = app.NewExecutionService(deps.ExecutionRepo)
	}

	// WebSocket hub
	wsHub := httpAdapter.NewWebSocketHub()

	// Wire WebSocket notifier to execution service
	if execSvc != nil {
		execSvc.SetNotifier(func(workflowID string, cb app.ExecutionCallback) {
			wsHub.BroadcastToWorkflow(workflowID, httpAdapter.WebSocketMessage{
				Type:   "node_status",
				RunID:  cb.RunID,
				NodeID: cb.NodeID,
				Status: cb.Status,
			})
		})
	}

	// Session store
	sessionSecret := deps.SessionSecret
	if sessionSecret == "" {
		cfg.Logger.Warn("no SessionSecret provided, sessions use an insecure default — set Deps.SessionSecret in production")
		sessionSecret = "graphiti-dev-secret"
	}
	sessionMaxAge := deps.SessionMaxAge
	if sessionMaxAge == 0 {
		sessionMaxAge = 24 * time.Hour
	}
	sessionStore := httpAdapter.NewSessionStore(sessionSecret, sessionMaxAge, deps.SessionSecure)

	// Prepare embedded static FS — strip the "web/static" prefix so files
	// are served from the root of the sub-filesystem.
	embeddedStaticFS, err := fs.Sub(staticFS, "web/static")
	if err != nil {
		return nil, fmt.Errorf("graphiti: preparing static assets: %w", err)
	}

	// Build HTTP handler
	router := httpAdapter.NewRouter(httpAdapter.RouterDeps{
		WorkflowSvc:    workflowSvc,
		ExecutionSvc:   execSvc,
		NodeRegistry:   nodeRegistry,
		AuthProvider:   deps.AuthProvider,
		UserRepo:       deps.UserRepo,
		SessionStore:   sessionStore,
		WSHub:          wsHub,
		HMACSecret:     deps.HMACSecret,
		Branding:       cfg.Branding,
		StaticFS:       embeddedStaticFS,
		TokenValidator: deps.TokenValidator,
		CORS:           deps.CORS,
	})

	return &App{
		config:        cfg,
		handler:       router,
		workflowSvc:   workflowSvc,
		nodeRegistry:  nodeRegistry,
		execSvc:       execSvc,
		hub:           wsHub,
		executionRepo: deps.ExecutionRepo,
	}, nil
}

// Handler returns an http.Handler that serves the entire Graphiti UI
// and API. Mount this on your router at the configured BasePath.
func (a *App) Handler() http.Handler {
	return a.handler
}

// Services returns the driving port interfaces so the host application
// can interact with Graphiti programmatically.
func (a *App) Services() *Services {
	var execSvc driving.ExecutionService
	if a.execSvc != nil {
		execSvc = a.execSvc
	}
	return &Services{
		Workflows:    a.workflowSvc,
		NodeRegistry: a.nodeRegistry,
		Executions:   execSvc,
	}
}

// WebSocketHub returns the WebSocket hub for pushing execution updates.
// The host calls hub.BroadcastToWorkflow() when execution status changes,
// instead of sending HTTP callbacks.
func (a *App) WebSocketHub() *httpAdapter.WebSocketHub {
	return a.hub
}

// ReportNodeStatus pushes an execution status update to all browsers
// viewing the given workflow. This is the embedded-mode equivalent of
// the POST /api/callbacks/execution endpoint.
//
// Valid status values are: "pending", "running", "completed", "failed", "skipped".
func (a *App) ReportNodeStatus(ctx context.Context, update ExecutionUpdate) error {
	// Validate status
	switch domain.NodeExecStatus(update.Status) {
	case domain.NodeExecPending, domain.NodeExecRunning, domain.NodeExecCompleted,
		domain.NodeExecFailed, domain.NodeExecSkipped:
		// valid
	default:
		return fmt.Errorf("graphiti: invalid status %q", update.Status)
	}

	// Persist to execution repo
	if a.executionRepo != nil {
		nodeStatus := domain.NodeExecutionStatus{
			NodeID:       update.NodeID,
			Status:       domain.NodeExecStatus(update.Status),
			ErrorMessage: update.ErrorMessage,
			OutputData:   update.OutputSummary,
		}
		if update.StartedAt != nil {
			nodeStatus.StartedAt = *update.StartedAt
		}
		if update.CompletedAt != nil {
			nodeStatus.CompletedAt = *update.CompletedAt
		}

		if err := a.executionRepo.UpdateNodeStatus(ctx, update.RunID, update.NodeID, nodeStatus); err != nil {
			return fmt.Errorf("failed to persist node status: %w", err)
		}
	}

	// Push to connected browsers via WebSocket
	a.hub.BroadcastToWorkflow(update.WorkflowID, httpAdapter.WebSocketMessage{
		Type:   "node_status",
		RunID:  update.RunID,
		NodeID: update.NodeID,
		Status: update.Status,
	})

	return nil
}
