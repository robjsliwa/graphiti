package http

import (
	"net/http"

	"graphiti/internal/ports/driven"
	"graphiti/internal/ports/driving"
)

// RouterDeps holds all dependencies needed by the HTTP router.
type RouterDeps struct {
	WorkflowSvc  driving.WorkflowService
	NodeRegistry driving.NodeRegistryService
	AuthProvider driven.AuthProvider
	UserRepo     driven.UserRepository
	SessionStore *SessionStore
}

// NewRouter creates the HTTP handler with all routes configured.
func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Auth routes (no middleware)
	mux.HandleFunc("GET /auth/login", handleLogin(deps.AuthProvider, deps.SessionStore))
	mux.HandleFunc("GET /auth/callback", handleCallback(deps.AuthProvider, deps.UserRepo, deps.SessionStore))
	mux.HandleFunc("POST /auth/logout", handleLogout(deps.SessionStore))

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

	// Node definition routes
	protected.HandleFunc("GET /api/nodes/search", handleNodeSearch(deps.NodeRegistry))
	protected.HandleFunc("GET /api/nodes", handleNodeList(deps.NodeRegistry))

	// Node config and attributes
	protected.HandleFunc("GET /api/workflows/{id}/nodes/{nodeId}/config", handleNodeConfig(deps.WorkflowSvc))
	protected.HandleFunc("PATCH /api/workflows/{id}/nodes/{nodeId}/attributes", handleAttributeUpdate(deps.WorkflowSvc, deps.NodeRegistry))

	// Clipboard
	protected.HandleFunc("POST /api/workflows/{id}/clipboard/copy", handleClipboardCopy(deps.WorkflowSvc))
	protected.HandleFunc("POST /api/workflows/{id}/clipboard/paste", handleClipboardPaste(deps.WorkflowSvc))

	mux.Handle("/", AuthMiddleware(deps.SessionStore, deps.UserRepo)(protected))

	return mux
}
