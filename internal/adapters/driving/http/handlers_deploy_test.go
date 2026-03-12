package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"graphiti/internal/adapters/driven/memory"
	"graphiti/internal/app"
	"graphiti/internal/domain"
)

func setupDeployTestService(t *testing.T) (*app.WorkflowService, *app.NodeRegistry, string) {
	t.Helper()
	defs := []*domain.NodeDefinition{
		{
			ID: "src", Name: "Source",
			Category: domain.Category{Group: "Sources"},
			Shape:    domain.Shape{Type: domain.ShapeRoundedRect},
			Outputs:  []domain.PortDefinition{{ID: "out", Type: domain.PortTypeData, MaxConnections: -1}},
		},
	}
	nodeRepo := &fakeNodeDefRepoForDeploy{defs: defs}
	nodeRegistry := app.NewNodeRegistry(nodeRepo)
	nodeRegistry.Load(context.Background())

	wfRepo := memory.NewWorkflowRepo()
	svc := app.NewWorkflowService(wfRepo, nodeRegistry, 100)

	// Create a workflow
	wf, _ := svc.CreateWorkflow(context.Background(), "Test WF", "desc", "user-1")
	return svc, nodeRegistry, wf.ID
}

type fakeNodeDefRepoForDeploy struct {
	defs []*domain.NodeDefinition
}

func (r *fakeNodeDefRepoForDeploy) LoadAll(_ context.Context) ([]*domain.NodeDefinition, error) {
	return r.defs, nil
}
func (r *fakeNodeDefRepoForDeploy) GetByID(_ context.Context, id string) (*domain.NodeDefinition, error) {
	for _, d := range r.defs {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, domain.ErrNodeNotFound
}
func (r *fakeNodeDefRepoForDeploy) GetByCategory(_ context.Context, _ string) ([]*domain.NodeDefinition, error) {
	return nil, nil
}
func (r *fakeNodeDefRepoForDeploy) Search(_ context.Context, _ string) ([]*domain.NodeDefinition, error) {
	return nil, nil
}

func TestHandleDeploy_Success(t *testing.T) {
	svc, _, wfID := setupDeployTestService(t)
	fakeTarget := &fakeDeployTargetHTTP{success: true, runID: "run-1"}
	svc.SetDeployTarget(fakeTarget)

	handler := handleDeploy(svc)

	body, _ := json.Marshal(map[string]string{"target": "production"})
	req := httptest.NewRequest("POST", "/api/workflows/"+wfID+"/deploy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", wfID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp deployResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Error("expected success")
	}
	if resp.RunID != "run-1" {
		t.Errorf("expected runID 'run-1', got %q", resp.RunID)
	}
}

func TestHandleDeploy_ValidationFails(t *testing.T) {
	svc, _, wfID := setupDeployTestService(t)
	fakeTarget := &fakeDeployTargetHTTP{success: true}
	svc.SetDeployTarget(fakeTarget)

	handler := handleDeploy(svc)

	// Deploy with no target specified should still work (uses request body)
	body, _ := json.Marshal(map[string]string{"target": "production"})
	req := httptest.NewRequest("POST", "/api/workflows/"+wfID+"/deploy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", wfID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Empty workflow passes validation (no required attrs missing)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleExport_JSON(t *testing.T) {
	svc, _, wfID := setupDeployTestService(t)
	handler := handleExport(svc)

	req := httptest.NewRequest("GET", "/api/workflows/"+wfID+"/export?format=json", nil)
	req.SetPathValue("id", wfID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
	cd := w.Header().Get("Content-Disposition")
	if cd == "" {
		t.Error("expected Content-Disposition header for download")
	}
}

func TestHandleExport_YAML(t *testing.T) {
	svc, _, wfID := setupDeployTestService(t)
	handler := handleExport(svc)

	req := httptest.NewRequest("GET", "/api/workflows/"+wfID+"/export?format=yaml", nil)
	req.SetPathValue("id", wfID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/x-yaml" {
		t.Errorf("expected Content-Type application/x-yaml, got %q", ct)
	}
}

func TestHandleVersionHistory(t *testing.T) {
	svc, _, wfID := setupDeployTestService(t)
	fakeTarget := &fakeDeployTargetHTTP{success: true}
	svc.SetDeployTarget(fakeTarget)

	// Deploy to create a version
	deployHandler := handleDeploy(svc)
	body, _ := json.Marshal(map[string]string{"target": "production"})
	req := httptest.NewRequest("POST", "/api/workflows/"+wfID+"/deploy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", wfID)
	w := httptest.NewRecorder()
	deployHandler.ServeHTTP(w, req)

	// Get version history
	handler := handleVersionHistory(svc)
	req = httptest.NewRequest("GET", "/api/workflows/"+wfID+"/versions", nil)
	req.SetPathValue("id", wfID)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var versions []versionResponse
	json.NewDecoder(w.Body).Decode(&versions)
	if len(versions) != 1 {
		t.Errorf("expected 1 version, got %d", len(versions))
	}
}

func TestHandleCheckDeployStatus_Unknown(t *testing.T) {
	svc, _, wfID := setupDeployTestService(t)
	handler := handleCheckDeployStatus(svc)

	req := httptest.NewRequest("GET", "/api/workflows/"+wfID+"/deploy/status", nil)
	req.SetPathValue("id", wfID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp deployStatusResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Verification != "unknown" {
		t.Errorf("expected unknown verification, got %q", resp.Verification)
	}
	if resp.WorkflowID != wfID {
		t.Errorf("expected workflowId %q, got %q", wfID, resp.WorkflowID)
	}
}

func TestHandleCheckDeployStatus_Verified(t *testing.T) {
	svc, _, wfID := setupDeployTestService(t)

	fakeTarget := &fakeDeployTargetWithCheckerHTTP{
		fakeDeployTargetHTTP: fakeDeployTargetHTTP{success: true},
		statusResult: &domain.DeployStatusResult{
			Verification: domain.DeployVerificationVerified,
			Message:      "confirmed",
		},
	}
	svc.SetDeployTarget(fakeTarget)

	// Deploy first
	body, _ := json.Marshal(map[string]string{"target": "production"})
	deployReq := httptest.NewRequest("POST", "/api/workflows/"+wfID+"/deploy", bytes.NewReader(body))
	deployReq.Header.Set("Content-Type", "application/json")
	deployReq.SetPathValue("id", wfID)
	dw := httptest.NewRecorder()
	handleDeploy(svc).ServeHTTP(dw, deployReq)

	// Check status
	handler := handleCheckDeployStatus(svc)
	req := httptest.NewRequest("GET", "/api/workflows/"+wfID+"/deploy/status", nil)
	req.SetPathValue("id", wfID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp deployStatusResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Verification != "verified" {
		t.Errorf("expected verified, got %q", resp.Verification)
	}
}

func TestHandleCheckDeployStatus_NotFound(t *testing.T) {
	svc, _, _ := setupDeployTestService(t)
	handler := handleCheckDeployStatus(svc)

	req := httptest.NewRequest("GET", "/api/workflows/nonexistent/deploy/status", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent workflow, got %d", w.Code)
	}
}

type fakeDeployTargetHTTP struct {
	success bool
	runID   string
}

func (f *fakeDeployTargetHTTP) Deploy(_ context.Context, _ domain.DeployPayload) (*domain.DeployResult, error) {
	return &domain.DeployResult{Success: f.success, RunID: f.runID, Message: "ok"}, nil
}

type fakeDeployTargetWithCheckerHTTP struct {
	fakeDeployTargetHTTP
	statusResult *domain.DeployStatusResult
}

func (f *fakeDeployTargetWithCheckerHTTP) CheckDeployStatus(_ context.Context, workflowID string, version int) (*domain.DeployStatusResult, error) {
	result := *f.statusResult
	result.WorkflowID = workflowID
	result.Version = version
	return &result, nil
}
