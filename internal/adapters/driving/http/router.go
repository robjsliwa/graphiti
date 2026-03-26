package http

import (
	"io/fs"
	"net/http"
	"time"

	"graphiti/internal/app"
	"graphiti/internal/ports/driven"
	"graphiti/internal/ports/driving"
	"graphiti/web/templates"
)

// RouterDeps holds all dependencies needed by the HTTP router.
type RouterDeps struct {
	WorkflowSvc  driving.WorkflowService
	ExecutionSvc *app.ExecutionService
	NodeRegistry driving.NodeRegistryService
	AuthProvider driven.AuthProvider
	UserRepo     driven.UserRepository
	SessionStore *SessionStore
	WSHub        *WebSocketHub
	HMACSecret       string // for callback signature verification
	Branding         templates.Branding
	FaviconFilePath  string // filesystem path to favicon file (for serving)
	CustomCSSFilePath string // filesystem path to custom CSS file (for serving)
	StaticFS         fs.FS  // embedded static assets (if nil, serves from web/static/ on disk)

	// TokenValidator enables bearer token auth for API clients.
	// If nil, only session-based auth is used.
	TokenValidator driven.TokenValidator

	// CORS controls Cross-Origin Resource Sharing for /api/ routes.
	CORS CORSConfig
}

// NewRouter creates the HTTP handler with all routes configured.
func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	// Rate limiter for auth endpoints
	authLimiter := NewRateLimiter(100, time.Minute)

	// Static files
	if deps.StaticFS != nil {
		mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(deps.StaticFS)))
	} else {
		mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	}

	// Favicon (if configured)
	if deps.FaviconFilePath != "" {
		faviconPath := deps.FaviconFilePath
		mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, faviconPath)
		})
	}

	// Custom CSS (if configured and outside /static/)
	if deps.CustomCSSFilePath != "" {
		cssFilePath := deps.CustomCSSFilePath
		mux.HandleFunc("GET /branding/custom.css", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/css")
			http.ServeFile(w, r, cssFilePath)
		})
	}

	// Auth routes (with rate limiting, no session auth)
	mux.Handle("GET /auth/login", authLimiter.RateLimitMiddleware(handleLogin(deps.AuthProvider, deps.SessionStore, deps.Branding)))
	mux.Handle("GET /auth/callback", authLimiter.RateLimitMiddleware(handleCallback(deps.AuthProvider, deps.UserRepo, deps.SessionStore)))
	mux.HandleFunc("POST /auth/logout", handleLogout(deps.SessionStore))

	// Execution callback endpoint (authenticated by HMAC, not session)
	if deps.ExecutionSvc != nil && deps.WSHub != nil {
		mux.HandleFunc("POST /api/callbacks/execution", handleExecutionCallback(deps.ExecutionSvc, deps.WSHub, deps.HMACSecret))
	}

	// Protected routes - wrap in auth middleware (with optional token validator)
	protected := http.NewServeMux()
	protected.HandleFunc("GET /", handleDashboard(deps.WorkflowSvc, deps.Branding))
	protected.HandleFunc("POST /workflows", handleCreateWorkflow(deps.WorkflowSvc))
	protected.HandleFunc("GET /workflows/{id}", handleWorkflowBuilder(deps.WorkflowSvc, deps.NodeRegistry, deps.Branding))
	protected.HandleFunc("DELETE /workflows/{id}", handleDeleteWorkflow(deps.WorkflowSvc))
	protected.HandleFunc("PATCH /api/workflows/{id}/name", handleRenameWorkflow(deps.WorkflowSvc))

	// Command API routes
	protected.HandleFunc("POST /api/workflows/{id}/commands", handleExecuteCommand(deps.WorkflowSvc, deps.NodeRegistry))
	protected.HandleFunc("POST /api/workflows/{id}/undo", handleUndo(deps.WorkflowSvc))
	protected.HandleFunc("POST /api/workflows/{id}/redo", handleRedo(deps.WorkflowSvc))

	// Help modal (keyboard shortcuts)
	protected.HandleFunc("GET /help", handleHelp())

	// Node definition routes
	protected.HandleFunc("GET /api/nodes/search", handleNodeSearch(deps.NodeRegistry))
	protected.HandleFunc("GET /api/nodes", handleNodeList(deps.NodeRegistry))

	// Node config, attributes, and sub-workflow reference
	protected.HandleFunc("GET /api/workflows/{id}/nodes/{nodeId}/config", handleNodeConfig(deps.WorkflowSvc))
	protected.HandleFunc("GET /api/workflows/{id}/nodes/{nodeId}/ref", handleNodeRef(deps.WorkflowSvc))
	protected.HandleFunc("PATCH /api/workflows/{id}/nodes/{nodeId}/attributes", handleAttributeUpdate(deps.WorkflowSvc, deps.NodeRegistry))

	// Clipboard
	protected.HandleFunc("POST /api/workflows/{id}/clipboard/copy", handleClipboardCopy(deps.WorkflowSvc))
	protected.HandleFunc("POST /api/workflows/{id}/clipboard/paste", handleClipboardPaste(deps.WorkflowSvc))

	// Validation, deploy, and export
	protected.HandleFunc("POST /api/workflows/{id}/validate", handleValidate(deps.WorkflowSvc))
	protected.HandleFunc("POST /api/workflows/{id}/deploy", handleDeploy(deps.WorkflowSvc))
	protected.HandleFunc("GET /api/workflows/{id}/deploy/status", handleCheckDeployStatus(deps.WorkflowSvc))
	protected.HandleFunc("GET /api/workflows/{id}/export", handleExport(deps.WorkflowSvc))
	protected.HandleFunc("GET /api/workflows/{id}/versions", handleVersionHistory(deps.WorkflowSvc))

	// JSON API routes (headless mode)
	protected.HandleFunc("GET /api/workflows", handleAPIListWorkflows(deps.WorkflowSvc))
	protected.HandleFunc("POST /api/workflows", handleAPICreateWorkflow(deps.WorkflowSvc))
	protected.HandleFunc("GET /api/workflows/{id}/overview", handleAPIGetWorkflow(deps.WorkflowSvc, deps.NodeRegistry))
	protected.HandleFunc("DELETE /api/workflows/{id}", handleAPIDeleteWorkflow(deps.WorkflowSvc))
	protected.HandleFunc("GET /api/workflows/{id}/state", handleAPIWorkflowState(deps.WorkflowSvc))

	// Execution mode routes
	if deps.ExecutionSvc != nil {
		protected.HandleFunc("GET /api/workflows/{id}/runs", handleListRuns(deps.ExecutionSvc))
		protected.HandleFunc("GET /api/runs/{runId}", handleGetRun(deps.ExecutionSvc))
	}

	// WebSocket for live execution updates
	if deps.WSHub != nil {
		protected.HandleFunc("GET /api/ws/workflows/{id}", handleWebSocket(deps.WSHub))
	}

	// Apply auth middleware (with optional token validator)
	authMw := AuthMiddlewareWithToken(deps.SessionStore, deps.TokenValidator, deps.UserRepo)

	// Apply CORS middleware
	corsMw := CORSMiddleware(deps.CORS)

	mux.Handle("/", corsMw(authMw(protected)))

	return mux
}
