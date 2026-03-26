package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driving"
	"graphiti/web/templates"
	"graphiti/web/templates/pages"
	"graphiti/web/templates/partials"
)

func handleDashboard(svc driving.WorkflowService, branding templates.Branding) http.HandlerFunc {
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
			Branding:  branding,
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

func handleWorkflowBuilder(svc driving.WorkflowService, registry driving.NodeRegistryService, branding templates.Branding) http.HandlerFunc {
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
			Branding:        branding,
		}

		// Sub-workflow navigation: parent context
		parentID := r.URL.Query().Get("parent")
		parentNodeID := r.URL.Query().Get("parentNode")
		if parentID != "" {
			data.ParentWorkflowID = parentID
			// Resolve parent workflow name for breadcrumb
			parentWf, err := svc.GetWorkflow(r.Context(), parentID)
			if err == nil {
				data.ParentWorkflowName = parentWf.Name
			}
			data.ParentNodeID = parentNodeID
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
			if wantsJSON(r) {
				writeJSONError(w, http.StatusInternalServerError, "search failed")
			} else {
				http.Error(w, "search failed", http.StatusInternalServerError)
			}
			return
		}

		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]any{"items": nodes})
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
			if wantsJSON(r) {
				writeJSONError(w, http.StatusNotFound, "workflow not found")
			} else {
				http.Error(w, "workflow not found", http.StatusNotFound)
			}
			return
		}

		node := wf.FindNode(nodeID)
		if node == nil {
			if wantsJSON(r) {
				writeJSONError(w, http.StatusNotFound, "node not found")
			} else {
				http.Error(w, "node not found", http.StatusNotFound)
			}
			return
		}

		if wantsJSON(r) {
			// Build JSON response with masked secrets
			attrs := make(map[string]any)
			for k, v := range node.AttributeValues {
				attrs[k] = v
			}
			maskSecretAttributes(attrs, node.Definition)

			nodeResp := map[string]any{
				"id":           node.ID,
				"definitionId": node.DefinitionID,
				"label":        node.Label,
				"x":            node.X,
				"y":            node.Y,
				"attributes":   attrs,
			}
			var defResp any
			if node.Definition != nil {
				defResp = buildNodeDefJSON(node.Definition)
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"node":       nodeResp,
				"definition": defResp,
			})
			return
		}

		data := partials.ConfigPanelData{
			WorkflowID: workflowID,
			Node:       node,
		}
		partials.ConfigPanel(data).Render(r.Context(), w)
	}
}

// buildNodeDefJSON converts a domain NodeDefinition to its JSON representation.
func buildNodeDefJSON(def *domain.NodeDefinition) *nodeDefJSON {
	result := &nodeDefJSON{
		Icon: def.Icon,
		Shape: shapeJSON{
			Type: string(def.Shape.Type), Width: def.Shape.Width,
			HeaderColor: def.Shape.HeaderColor, HeaderBackground: def.Shape.HeaderBackground,
		},
		Category:   categoryJSON{Group: def.Category.Group},
		Inputs:     make([]portDefJSON, 0, len(def.Inputs)),
		Outputs:    make([]portDefJSON, 0, len(def.Outputs)),
		Attributes: make([]attrDefJSON, 0, len(def.Attributes)),
	}
	for _, p := range def.Inputs {
		result.Inputs = append(result.Inputs, portDefJSON{
			ID: p.ID, Label: p.Label, Type: string(p.Type),
			Position: string(p.Position), MaxConnections: p.MaxConnections,
		})
	}
	for _, p := range def.Outputs {
		result.Outputs = append(result.Outputs, portDefJSON{
			ID: p.ID, Label: p.Label, Type: string(p.Type),
			Position: string(p.Position), MaxConnections: p.MaxConnections,
		})
	}
	for _, a := range def.Attributes {
		result.Attributes = append(result.Attributes, attrDefJSON{
			ID: a.ID, Label: a.Label, Type: string(a.Type), Display: string(a.Display),
		})
	}
	return result
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

func handleRenameWorkflow(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")

		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		wf, err := svc.GetWorkflow(r.Context(), workflowID)
		if err != nil {
			slog.Error("get workflow for rename failed", "error", err, "id", workflowID)
			http.Error(w, "workflow not found", http.StatusNotFound)
			return
		}

		wf.Name = name
		if err := svc.SaveWorkflow(r.Context(), wf); err != nil {
			slog.Error("rename workflow failed", "error", err, "id", workflowID)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func handleNodeRef(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")
		nodeID := r.PathValue("nodeId")

		wf, err := svc.GetWorkflow(r.Context(), workflowID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) || strings.Contains(err.Error(), "not found") {
				http.Error(w, "workflow not found", http.StatusNotFound)
				return
			}
			slog.Error("get workflow failed", "error", err, "id", workflowID)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		node := wf.FindNode(nodeID)
		if node == nil {
			http.Error(w, "node not found", http.StatusNotFound)
			return
		}

		workflowRef := ""
		if val, ok := node.AttributeValues["workflow_ref"]; ok {
			if s, ok := val.(string); ok {
				workflowRef = s
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(struct {
			WorkflowRef string `json:"workflowRef"`
		}{WorkflowRef: workflowRef}); err != nil {
			slog.Error("failed to encode node ref response", "error", err)
		}
	}
}
