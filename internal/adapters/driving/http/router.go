package http

import (
	"net/http"
	"time"

	"graphiti/internal/app"
	"graphiti/internal/ports/driven"
	"graphiti/internal/ports/driving"
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
	HMACSecret   string // for callback signature verification
}

// NewRouter creates the HTTP handler with all routes configured.
func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	// Rate limiter for auth endpoints
	authLimiter := NewRateLimiter(10, time.Minute)

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Auth routes (with rate limiting, no session auth)
	mux.Handle("GET /auth/login", authLimiter.RateLimitMiddleware(handleLogin(deps.AuthProvider, deps.SessionStore)))
	mux.Handle("GET /auth/callback", authLimiter.RateLimitMiddleware(handleCallback(deps.AuthProvider, deps.UserRepo, deps.SessionStore)))
	mux.HandleFunc("POST /auth/logout", handleLogout(deps.SessionStore))

	// Execution callback endpoint (authenticated by HMAC, not session)
	if deps.ExecutionSvc != nil && deps.WSHub != nil {
		mux.HandleFunc("POST /api/callbacks/execution", handleExecutionCallback(deps.ExecutionSvc, deps.WSHub, deps.HMACSecret))
	}

	// Protected routes - wrap in auth middleware
	protected := http.NewServeMux()
	protected.HandleFunc("GET /", handleDashboard(deps.WorkflowSvc))
	protected.HandleFunc("POST /workflows", handleCreateWorkflow(deps.WorkflowSvc))
	protected.HandleFunc("GET /workflows/{id}", handleWorkflowBuilder(deps.WorkflowSvc, deps.NodeRegistry))
	protected.HandleFunc("DELETE /workflows/{id}", handleDeleteWorkflow(deps.WorkflowSvc))

	// Command API routes
	protected.HandleFunc("POST /api/workflows/{id}/commands", handleExecuteCommand(deps.WorkflowSvc, deps.NodeRegistry))
	protected.HandleFunc("POST /api/workflows/{id}/undo", handleUndo(deps.WorkflowSvc))
	protected.HandleFunc("POST /api/workflows/{id}/redo", handleRedo(deps.WorkflowSvc))

	// Help modal (keyboard shortcuts)
	protected.HandleFunc("GET /help", handleHelp())

	// Node definition routes
	protected.HandleFunc("GET /api/nodes/search", handleNodeSearch(deps.NodeRegistry))
	protected.HandleFunc("GET /api/nodes", handleNodeList(deps.NodeRegistry))

	// Node config and attributes
	protected.HandleFunc("GET /api/workflows/{id}/nodes/{nodeId}/config", handleNodeConfig(deps.WorkflowSvc))
	protected.HandleFunc("PATCH /api/workflows/{id}/nodes/{nodeId}/attributes", handleAttributeUpdate(deps.WorkflowSvc, deps.NodeRegistry))

	// Clipboard
	protected.HandleFunc("POST /api/workflows/{id}/clipboard/copy", handleClipboardCopy(deps.WorkflowSvc))
	protected.HandleFunc("POST /api/workflows/{id}/clipboard/paste", handleClipboardPaste(deps.WorkflowSvc))

	// Deploy and export
	protected.HandleFunc("POST /api/workflows/{id}/deploy", handleDeploy(deps.WorkflowSvc))
	protected.HandleFunc("GET /api/workflows/{id}/export", handleExport(deps.WorkflowSvc))
	protected.HandleFunc("GET /api/workflows/{id}/versions", handleVersionHistory(deps.WorkflowSvc))

	// Execution mode routes
	if deps.ExecutionSvc != nil {
		protected.HandleFunc("GET /api/workflows/{id}/runs", handleListRuns(deps.ExecutionSvc))
		protected.HandleFunc("GET /api/runs/{runId}", handleGetRun(deps.ExecutionSvc))
	}

	// WebSocket for live execution updates
	if deps.WSHub != nil {
		protected.HandleFunc("GET /api/ws/workflows/{id}", handleWebSocket(deps.WSHub))
	}

	mux.Handle("/", AuthMiddleware(deps.SessionStore, deps.UserRepo)(protected))

	return mux
}
