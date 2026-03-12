package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandleRenameWorkflow_Success(t *testing.T) {
	svc, _, wfID := testSetup(t)

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/workflows/{id}/name", handleRenameWorkflow(svc))

	form := url.Values{"name": {"My New Name"}}
	req := httptest.NewRequest(http.MethodPatch, "/api/workflows/"+wfID+"/name", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify the name was persisted
	wf, err := svc.GetWorkflow(req.Context(), wfID)
	if err != nil {
		t.Fatalf("get workflow: %v", err)
	}
	if wf.Name != "My New Name" {
		t.Errorf("expected name %q, got %q", "My New Name", wf.Name)
	}
}

func TestHandleRenameWorkflow_EmptyName(t *testing.T) {
	svc, _, wfID := testSetup(t)

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/workflows/{id}/name", handleRenameWorkflow(svc))

	form := url.Values{"name": {"  "}}
	req := httptest.NewRequest(http.MethodPatch, "/api/workflows/"+wfID+"/name", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRenameWorkflow_NotFound(t *testing.T) {
	svc, _, _ := testSetup(t)

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/workflows/{id}/name", handleRenameWorkflow(svc))

	form := url.Values{"name": {"New Name"}}
	req := httptest.NewRequest(http.MethodPatch, "/api/workflows/nonexistent/name", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleNodeRef_Success(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	// Add a sub-workflow node (using "src" def for simplicity, we'll set workflow_ref attribute)
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "subwf-1", X: 100, Y: 200,
	})

	// Manually set the workflow_ref attribute on the node
	form := url.Values{"url": {"https://example.com"}}
	req := httptest.NewRequest(http.MethodPatch, "/api/workflows/"+wfID+"/nodes/subwf-1/attributes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Now set workflow_ref directly via the service
	wf, err := svc.GetWorkflow(req.Context(), wfID)
	if err != nil {
		t.Fatalf("get workflow: %v", err)
	}
	node := wf.FindNode("subwf-1")
	if node == nil {
		t.Fatal("node subwf-1 not found")
	}
	node.AttributeValues["workflow_ref"] = "referenced-wf-id-123"
	if err := svc.SaveWorkflow(req.Context(), wf); err != nil {
		t.Fatalf("save workflow: %v", err)
	}

	// Now test the node ref endpoint
	refMux := http.NewServeMux()
	refMux.HandleFunc("GET /api/workflows/{id}/nodes/{nodeId}/ref", handleNodeRef(svc))

	req = httptest.NewRequest(http.MethodGet, "/api/workflows/"+wfID+"/nodes/subwf-1/ref", nil)
	rec = httptest.NewRecorder()
	refMux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		WorkflowRef string `json:"workflowRef"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.WorkflowRef != "referenced-wf-id-123" {
		t.Errorf("workflowRef = %q, want %q", resp.WorkflowRef, "referenced-wf-id-123")
	}
}

func TestHandleNodeRef_NoRef(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	// Add a node without workflow_ref
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "node-1", X: 100, Y: 200,
	})

	refMux := http.NewServeMux()
	refMux.HandleFunc("GET /api/workflows/{id}/nodes/{nodeId}/ref", handleNodeRef(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/"+wfID+"/nodes/node-1/ref", nil)
	rec := httptest.NewRecorder()
	refMux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		WorkflowRef string `json:"workflowRef"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.WorkflowRef != "" {
		t.Errorf("workflowRef = %q, want empty string", resp.WorkflowRef)
	}
}

func TestHandleNodeRef_WorkflowNotFound(t *testing.T) {
	svc, _, _ := testSetup(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/workflows/{id}/nodes/{nodeId}/ref", handleNodeRef(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/nonexistent/nodes/node-1/ref", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleNodeRef_NodeNotFound(t *testing.T) {
	svc, _, wfID := testSetup(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/workflows/{id}/nodes/{nodeId}/ref", handleNodeRef(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/"+wfID+"/nodes/nonexistent/ref", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
