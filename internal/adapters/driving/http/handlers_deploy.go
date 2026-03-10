package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"graphiti/internal/ports/driving"
)

type deployRequest struct {
	Target string `json:"target"`
}

type deployResponse struct {
	Success          bool                     `json:"success"`
	RunID            string                   `json:"runId,omitempty"`
	Message          string                   `json:"message,omitempty"`
	ValidationErrors []validationErrorResponse `json:"validationErrors,omitempty"`
}

type validationErrorResponse struct {
	NodeID  string `json:"nodeId,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type versionResponse struct {
	ID         string `json:"id"`
	Version    int    `json:"version"`
	DeployedAt string `json:"deployedAt"`
	DeployedBy string `json:"deployedBy"`
}

func handleDeploy(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")
		session := GetSession(r)

		var req deployRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, deployResponse{Message: "invalid request"})
			return
		}
		if req.Target == "" {
			req.Target = "production"
		}

		userID := "anonymous"
		if session != nil {
			userID = session.UserID
		}

		result, err := svc.DeployWorkflow(r.Context(), workflowID, req.Target, userID)
		if err != nil {
			slog.Error("deploy failed", "error", err, "workflowId", workflowID)
			writeJSON(w, http.StatusInternalServerError, deployResponse{Message: err.Error()})
			return
		}

		resp := deployResponse{
			Success: result.Success,
			RunID:   result.RunID,
			Message: result.Message,
		}
		for _, ve := range result.ValidationErrors {
			resp.ValidationErrors = append(resp.ValidationErrors, validationErrorResponse{
				NodeID:  ve.NodeID,
				Field:   ve.Field,
				Message: ve.Message,
			})
		}

		status := http.StatusOK
		if !result.Success {
			status = http.StatusUnprocessableEntity
		}
		writeJSON(w, status, resp)
	}
}

func handleExport(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "json"
		}

		data, err := svc.ExportWorkflow(r.Context(), workflowID, format)
		if err != nil {
			slog.Error("export failed", "error", err, "workflowId", workflowID)
			http.Error(w, "export failed", http.StatusInternalServerError)
			return
		}

		var contentType, ext string
		switch format {
		case "yaml":
			contentType = "application/x-yaml"
			ext = "yaml"
		default:
			contentType = "application/json"
			ext = "json"
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="workflow.%s"`, ext))
		w.Write(data)
	}
}

func handleVersionHistory(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")

		versions, err := svc.GetVersionHistory(r.Context(), workflowID)
		if err != nil {
			slog.Error("version history failed", "error", err, "workflowId", workflowID)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := make([]versionResponse, 0, len(versions))
		for _, v := range versions {
			resp = append(resp, versionResponse{
				ID:         v.ID,
				Version:    v.Version,
				DeployedAt: v.DeployedAt.Format("2006-01-02T15:04:05Z"),
				DeployedBy: v.DeployedBy,
			})
		}
		writeJSON(w, http.StatusOK, resp)
	}
}
