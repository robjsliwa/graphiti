package http

import (
	"log/slog"
	"net/http"

	"graphiti/internal/ports/driving"
)

func handleAPIListWorkflows(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r)
		if session == nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		workflows, err := svc.ListWorkflows(r.Context(), session.UserID)
		if err != nil {
			slog.Error("api: list workflows failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": workflows})
	}
}

func handleAPICreateWorkflow(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r)
		if session == nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if !decodeJSONBody(w, r, &body) {
			return
		}
		if body.Name == "" {
			body.Name = "Untitled Workflow"
		}
		wf, err := svc.CreateWorkflow(r.Context(), body.Name, body.Description, session.UserID)
		if err != nil {
			slog.Error("api: create workflow failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"workflow": wf})
	}
}

func handleAPIGetWorkflow(svc driving.WorkflowService, registry driving.NodeRegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r)
		if session == nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		workflowID := r.PathValue("id")
		wf, err := svc.GetWorkflow(r.Context(), workflowID)
		if err != nil {
			if isNotFoundError(err) {
				writeJSONError(w, http.StatusNotFound, "workflow not found")
				return
			}
			slog.Error("api: get workflow failed", "error", err, "id", workflowID)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		nodes, err := registry.GetAllDefinitions(r.Context())
		if err != nil {
			slog.Error("api: get node definitions failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"workflow":        wf,
			"nodeDefinitions": nodes,
		})
	}
}

func handleAPIDeleteWorkflow(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r)
		if session == nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		workflowID := r.PathValue("id")

		// Verify workflow exists first
		_, err := svc.GetWorkflow(r.Context(), workflowID)
		if err != nil {
			if isNotFoundError(err) {
				writeJSONError(w, http.StatusNotFound, "workflow not found")
				return
			}
			slog.Error("api: get workflow for delete failed", "error", err, "id", workflowID)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if err := svc.DeleteWorkflow(r.Context(), workflowID); err != nil {
			slog.Error("api: delete workflow failed", "error", err, "id", workflowID)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAPIWorkflowState(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r)
		if session == nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		workflowID := r.PathValue("id")
		wf, err := svc.GetWorkflow(r.Context(), workflowID)
		if err != nil {
			if isNotFoundError(err) {
				writeJSONError(w, http.StatusNotFound, "workflow not found")
				return
			}
			slog.Error("api: get workflow state failed", "error", err, "id", workflowID)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		state := buildWorkflowState(wf)
		writeJSON(w, http.StatusOK, state)
	}
}
