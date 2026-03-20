package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driving"
)

type validateResponse struct {
	Valid   bool                       `json:"valid"`
	Results []validationResultResponse `json:"results"`
	Summary validationSummary          `json:"summary"`
}

type validationResultResponse struct {
	Severity string `json:"severity"`
	Category string `json:"category"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	NodeID   string `json:"nodeId,omitempty"`
	EdgeID   string `json:"edgeId,omitempty"`
	Field    string `json:"field,omitempty"`
}

type validationSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

func handleValidate(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")

		results, err := svc.ValidateWorkflow(r.Context(), workflowID)
		if err != nil {
			slog.Error("validation failed", "error", err, "workflowId", workflowID)
			if errors.Is(err, domain.ErrNotFound) || strings.Contains(err.Error(), "not found") {
				writeJSON(w, http.StatusNotFound, map[string]string{"message": "workflow not found"})
			} else {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "validation failed"})
			}
			return
		}

		resp := buildValidateResponse(results)
		writeJSON(w, http.StatusOK, resp)
	}
}

func buildValidateResponse(results []domain.ValidationResult) validateResponse {
	resp := validateResponse{
		Valid:   !domain.HasErrors(results),
		Results: make([]validationResultResponse, 0, len(results)),
	}

	for _, r := range results {
		resp.Results = append(resp.Results, validationResultResponse{
			Severity: string(r.Severity),
			Category: string(r.Category),
			Code:     r.Code,
			Message:  r.Message,
			NodeID:   r.NodeID,
			EdgeID:   r.EdgeID,
			Field:    r.Field,
		})
		switch r.Severity {
		case domain.SeverityError:
			resp.Summary.Errors++
		case domain.SeverityWarning:
			resp.Summary.Warnings++
		case domain.SeverityInfo:
			resp.Summary.Info++
		}
	}

	return resp
}
