package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"graphiti/internal/app"
	"graphiti/internal/ports/driving"
	"graphiti/web/templates/partials"
)

// handleExecutionCallback accepts execution status updates from the external engine.
// POST /api/callbacks/execution
func handleExecutionCallback(execSvc *app.ExecutionService, hub *WebSocketHub, hmacSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify HMAC signature if configured
		if hmacSecret != "" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			sig := r.Header.Get("X-Graphiti-Signature")
			if !verifyCallbackSignature(body, sig, hmacSecret) {
				http.Error(w, "invalid signature", http.StatusUnauthorized)
				return
			}

			// Re-parse from body bytes
			var cb app.ExecutionCallback
			if err := json.Unmarshal(body, &cb); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			processCallback(w, r, execSvc, hub, cb)
			return
		}

		var cb app.ExecutionCallback
		if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		processCallback(w, r, execSvc, hub, cb)
	}
}

func processCallback(w http.ResponseWriter, r *http.Request, execSvc *app.ExecutionService, hub *WebSocketHub, cb app.ExecutionCallback) {
	// Validate required fields
	if cb.RunID == "" || cb.WorkflowID == "" || cb.NodeID == "" {
		http.Error(w, "missing required fields: runID, workflowID, nodeID", http.StatusBadRequest)
		return
	}

	// Process asynchronously — use a detached context since the request context
	// will be canceled as soon as we write the 202 response.
	go func() {
		ctx := context.Background()
		if err := execSvc.ProcessCallback(ctx, cb); err != nil {
			slog.Error("execution callback processing failed", "error", err, "runID", cb.RunID)
			return
		}

		// Broadcast to WebSocket clients
		hub.BroadcastToWorkflow(cb.WorkflowID, WebSocketMessage{
			Type:        "node_status",
			RunID:       cb.RunID,
			NodeID:      cb.NodeID,
			Status:      cb.Status,
			StartedAt:   formatTime(cb.StartedAt),
			CompletedAt: formatTime(cb.CompletedAt),
		})
	}()

	w.WriteHeader(http.StatusAccepted)
}

// handleListRuns returns execution run summaries for a workflow.
// GET /api/workflows/{id}/runs
// Returns HTML partial for HTMX or JSON based on Accept header.
func handleListRuns(execSvc driving.ExecutionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")
		runs, err := execSvc.ListRuns(r.Context(), workflowID)
		if err != nil {
			slog.Error("list runs failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Return HTML partial for HTMX requests
		if r.Header.Get("HX-Request") == "true" || r.Header.Get("Accept") == "text/html" {
			partials.ExecutionList(partials.ExecutionListData{
				WorkflowID: workflowID,
				Runs:       runs,
			}).Render(r.Context(), w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(runs)
	}
}

// handleGetRun returns a full execution run with node statuses.
// GET /api/runs/{runId}
func handleGetRun(execSvc driving.ExecutionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runID := r.PathValue("runId")
		run, err := execSvc.GetRun(r.Context(), runID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, "run not found", http.StatusNotFound)
				return
			}
			slog.Error("get run failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Return HTML partial for HTMX requests
		if r.Header.Get("HX-Request") == "true" {
			partials.ExecutionDetails(partials.ExecutionDetailsData{Run: run}).Render(r.Context(), w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(run)
	}
}

func verifyCallbackSignature(body []byte, signature string, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	sig := strings.TrimPrefix(signature, "sha256=")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
