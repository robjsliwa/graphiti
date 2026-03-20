package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"graphiti/internal/app"
	"graphiti/internal/domain"
	"graphiti/internal/domain/commands"
)

func TestHandleValidate_EmptyWorkflow(t *testing.T) {
	svc, _, wfID := setupDeployTestService(t)
	handler := handleValidate(svc)

	req := httptest.NewRequest("POST", "/api/workflows/"+wfID+"/validate", nil)
	req.SetPathValue("id", wfID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp validateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Results) == 0 {
		t.Error("expected validation results for empty workflow")
	}

	found := false
	for _, r := range resp.Results {
		if r.Code == "EMPTY_WORKFLOW" {
			found = true
			if r.Severity != "warning" {
				t.Errorf("expected warning severity for EMPTY_WORKFLOW, got %s", r.Severity)
			}
			if r.Category != "graph" {
				t.Errorf("expected graph category for EMPTY_WORKFLOW, got %s", r.Category)
			}
		}
	}
	if !found {
		t.Error("expected EMPTY_WORKFLOW in results")
	}
}

func TestHandleValidate_NonexistentWorkflow(t *testing.T) {
	svc, _, _ := setupDeployTestService(t)
	handler := handleValidate(svc)

	req := httptest.NewRequest("POST", "/api/workflows/nonexistent/validate", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent workflow, got %d", w.Code)
	}
}

// nodeDefResolver wraps a NodeRegistry for use as a NodeDefinitionResolver.
type nodeDefResolver struct {
	registry *app.NodeRegistry
}

func (r *nodeDefResolver) GetByID(id string) (*domain.NodeDefinition, error) {
	return r.registry.GetByID(context.Background(), id)
}

// addNodeViaService adds a node to the workflow using the app service directly.
func addNodeViaService(t *testing.T, svc *app.WorkflowService, registry *app.NodeRegistry, wfID, defID string, x, y int) {
	t.Helper()
	resolver := &nodeDefResolver{registry: registry}
	instanceID := fmt.Sprintf("test-%s-%d-%d", t.Name(), x, y)
	cmd := commands.NewAddNodeCommand(defID, instanceID, "", float64(x), float64(y), resolver)
	_, err := svc.ExecuteCommand(context.Background(), wfID, cmd)
	if err != nil {
		t.Fatalf("failed to add node: %v", err)
	}
}

func TestHandleValidate_SummaryCountsCorrect(t *testing.T) {
	svc, registry, wfID := setupDeployTestService(t)

	// Add a node via service
	addNodeViaService(t, svc, registry, wfID, "src", 100, 100)

	// Validate
	handler := handleValidate(svc)
	req := httptest.NewRequest("POST", "/api/workflows/"+wfID+"/validate", nil)
	req.SetPathValue("id", wfID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp validateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// Verify summary counts match actual results
	var errors, warnings, infos int
	for _, r := range resp.Results {
		switch r.Severity {
		case "error":
			errors++
		case "warning":
			warnings++
		case "info":
			infos++
		}
	}
	if resp.Summary.Errors != errors {
		t.Errorf("summary errors %d != actual errors %d", resp.Summary.Errors, errors)
	}
	if resp.Summary.Warnings != warnings {
		t.Errorf("summary warnings %d != actual warnings %d", resp.Summary.Warnings, warnings)
	}
	if resp.Summary.Info != infos {
		t.Errorf("summary info %d != actual info %d", resp.Summary.Info, infos)
	}
}

func TestHandleValidate_ValidBooleanCorrect(t *testing.T) {
	svc, _, wfID := setupDeployTestService(t)
	handler := handleValidate(svc)

	req := httptest.NewRequest("POST", "/api/workflows/"+wfID+"/validate", nil)
	req.SetPathValue("id", wfID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp validateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// Empty workflow has warnings but no errors → valid should be true
	hasErrors := false
	for _, r := range resp.Results {
		if r.Severity == "error" {
			hasErrors = true
			break
		}
	}
	if resp.Valid == hasErrors {
		t.Errorf("valid=%v but hasErrors=%v — these should be opposite", resp.Valid, hasErrors)
	}
}

func TestHandleValidate_ResultFieldsPopulated(t *testing.T) {
	svc, registry, wfID := setupDeployTestService(t)

	// Add two disconnected nodes to get a DISCONNECTED_SUBGRAPH error
	addNodeViaService(t, svc, registry, wfID, "src", 100, 100)
	addNodeViaService(t, svc, registry, wfID, "src", 300, 100)

	// Validate
	handler := handleValidate(svc)
	req := httptest.NewRequest("POST", "/api/workflows/"+wfID+"/validate", nil)
	req.SetPathValue("id", wfID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp validateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// Should not be valid (disconnected subgraph)
	if resp.Valid {
		t.Error("expected workflow with disconnected nodes to be invalid")
	}

	// Check that we have a DISCONNECTED_SUBGRAPH error with proper fields
	found := false
	for _, r := range resp.Results {
		if r.Code == "DISCONNECTED_SUBGRAPH" {
			found = true
			if r.Severity != "error" {
				t.Errorf("expected error severity, got %s", r.Severity)
			}
			if r.Category != "graph" {
				t.Errorf("expected graph category, got %s", r.Category)
			}
			if r.Message == "" {
				t.Error("expected non-empty message")
			}
		}
	}
	if !found {
		t.Errorf("expected DISCONNECTED_SUBGRAPH in results, got %+v", resp.Results)
	}

	// Summary should have at least 1 error
	if resp.Summary.Errors < 1 {
		t.Errorf("expected at least 1 error in summary, got %d", resp.Summary.Errors)
	}
}

func TestHandleValidate_DeployBlockedByErrors(t *testing.T) {
	svc, registry, wfID := setupDeployTestService(t)
	fakeTarget := &fakeDeployTargetHTTP{success: true, runID: "run-1"}
	svc.SetDeployTarget(fakeTarget)

	// Add two disconnected nodes to cause validation errors
	addNodeViaService(t, svc, registry, wfID, "src", 100, 100)
	addNodeViaService(t, svc, registry, wfID, "src", 300, 100)

	// Try to deploy — should be blocked
	deployHandler := handleDeploy(svc)
	body, _ := json.Marshal(map[string]string{"target": "production"})
	req := httptest.NewRequest("POST", "/api/workflows/"+wfID+"/deploy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", wfID)
	w := httptest.NewRecorder()
	deployHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}

	var resp deployResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Success {
		t.Error("expected deploy to fail due to validation errors")
	}

	if len(resp.ValidationResults) == 0 {
		t.Error("expected validation results in deploy response when blocked")
	}
}
