package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driving"
	"graphiti/web/templates/pages"
	"graphiti/web/templates/partials"
)

func handleDashboard(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		session := GetSession(r)
		if session == nil {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		workflows, err := svc.ListWorkflows(r.Context(), session.UserID)
		if err != nil {
			slog.Error("list workflows failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		data := pages.DashboardData{
			Username:  session.Username,
			Workflows: workflows,
		}
		pages.Dashboard(data).Render(r.Context(), w)
	}
}

func handleCreateWorkflow(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r)
		if session == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		name := r.FormValue("name")
		if name == "" {
			name = "Untitled Workflow"
		}
		description := r.FormValue("description")

		wf, err := svc.CreateWorkflow(r.Context(), name, description, session.UserID)
		if err != nil {
			slog.Error("create workflow failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/workflows/"+wf.ID)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/workflows/"+wf.ID, http.StatusFound)
	}
}

func handleWorkflowBuilder(svc driving.WorkflowService, registry driving.NodeRegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r)
		if session == nil {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		workflowID := r.PathValue("id")
		wf, err := svc.GetWorkflow(r.Context(), workflowID)
		if err != nil {
			slog.Error("get workflow failed", "error", err, "id", workflowID)
			http.Error(w, "workflow not found", http.StatusNotFound)
			return
		}

		nodes, err := registry.GetAllDefinitions(r.Context())
		if err != nil {
			slog.Error("get node definitions failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		data := pages.BuilderData{
			Username:        session.Username,
			Workflow:        wf,
			NodeDefinitions: nodes,
		}
		pages.Builder(data).Render(r.Context(), w)
	}
}

func handleDeleteWorkflow(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")
		if err := svc.DeleteWorkflow(r.Context(), workflowID); err != nil {
			slog.Error("delete workflow failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func handleNodeSearch(registry driving.NodeRegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		nodes, err := registry.Search(r.Context(), query)
		if err != nil {
			http.Error(w, "search failed", http.StatusInternalServerError)
			return
		}

		partials.NodePalette(nodes).Render(r.Context(), w)
	}
}

func handleNodeList(registry driving.NodeRegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodes, err := registry.GetAllDefinitions(r.Context())
		if err != nil {
			http.Error(w, "failed to list nodes", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodes)
	}
}

func handleNodeConfig(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")
		nodeID := r.PathValue("nodeId")

		wf, err := svc.GetWorkflow(r.Context(), workflowID)
		if err != nil {
			http.Error(w, "workflow not found", http.StatusNotFound)
			return
		}

		node := wf.FindNode(nodeID)
		if node == nil {
			http.Error(w, "node not found", http.StatusNotFound)
			return
		}

		data := partials.ConfigPanelData{
			WorkflowID: workflowID,
			Node:       node,
		}
		partials.ConfigPanel(data).Render(r.Context(), w)
	}
}

func handleHelp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		modKey := "Ctrl"
		ua := r.UserAgent()
		if strings.Contains(ua, "Mac") || strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad") {
			modKey = "\u2318"
		}
		partials.HelpModal(modKey).Render(r.Context(), w)
	}
}

// Ensure domain import is used (needed for FindNode return type)
var _ = (*domain.NodeInstance)(nil)
