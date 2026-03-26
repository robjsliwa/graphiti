package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"graphiti/internal/domain"
)

// newAPIMux creates a mux with all JSON API routes wired to the given services.
func newAPIMux(t *testing.T) (*http.ServeMux, string) {
	t.Helper()
	svc, registry, wfID := testSetup(t)
	mux := http.NewServeMux()

	// Wrap handlers with session context for API routes
	withSession := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			session := &Session{UserID: "user-1", Username: "testuser"}
			ctx := context.WithValue(r.Context(), userContextKey, session)
			h.ServeHTTP(w, r.WithContext(ctx))
		}
	}

	mux.HandleFunc("GET /api/workflows", withSession(handleAPIListWorkflows(svc)))
	mux.HandleFunc("POST /api/workflows", withSession(handleAPICreateWorkflow(svc)))
	mux.HandleFunc("GET /api/workflows/{id}/overview", withSession(handleAPIGetWorkflow(svc, registry)))
	mux.HandleFunc("DELETE /api/workflows/{id}", withSession(handleAPIDeleteWorkflow(svc)))
	mux.HandleFunc("GET /api/workflows/{id}/state", withSession(handleAPIWorkflowState(svc)))

	// Also wire command routes for setup
	mux.HandleFunc("POST /api/workflows/{id}/commands", withSession(handleExecuteCommand(svc, registry)))

	return mux, wfID
}

// newAPIMuxNoSession creates a mux WITHOUT session context for auth failure tests.
func newAPIMuxNoSession(t *testing.T) *http.ServeMux {
	t.Helper()
	svc, registry, _ := testSetup(t)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/workflows", handleAPIListWorkflows(svc))
	mux.HandleFunc("POST /api/workflows", handleAPICreateWorkflow(svc))
	mux.HandleFunc("GET /api/workflows/{id}/overview", handleAPIGetWorkflow(svc, registry))
	mux.HandleFunc("DELETE /api/workflows/{id}", handleAPIDeleteWorkflow(svc))
	mux.HandleFunc("GET /api/workflows/{id}/state", handleAPIWorkflowState(svc))
	return mux
}

func TestAPIListWorkflows(t *testing.T) {
	mux, _ := newAPIMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var resp map[string]json.RawMessage
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if _, ok := resp["items"]; !ok {
		t.Error("response missing 'items' key")
	}

	var items []*domain.WorkflowSummary
	if err := json.Unmarshal(resp["items"], &items); err != nil {
		t.Fatalf("failed to decode items: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(items))
	}
}

func TestAPIListWorkflows_Unauthorized(t *testing.T) {
	mux := newAPIMuxNoSession(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	var resp apiErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Error.Code != 401 {
		t.Errorf("error code = %d, want 401", resp.Error.Code)
	}
}

func TestAPICreateWorkflow(t *testing.T) {
	mux, _ := newAPIMux(t)

	body := map[string]string{"name": "Test API Workflow", "description": "created via API"}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/workflows", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var resp map[string]json.RawMessage
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if _, ok := resp["workflow"]; !ok {
		t.Error("response missing 'workflow' key")
	}
}

func TestAPICreateWorkflow_EmptyBody(t *testing.T) {
	mux, _ := newAPIMux(t)

	body := map[string]string{}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/workflows", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	// Should default to "Untitled Workflow"
	var resp struct {
		Workflow struct {
			Name string `json:"name"`
		} `json:"workflow"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Workflow.Name != "Untitled Workflow" {
		t.Errorf("name = %q, want %q", resp.Workflow.Name, "Untitled Workflow")
	}
}

func TestAPICreateWorkflow_InvalidJSON(t *testing.T) {
	mux, _ := newAPIMux(t)

	req := httptest.NewRequest(http.MethodPost, "/api/workflows", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAPICreateWorkflow_Unauthorized(t *testing.T) {
	mux := newAPIMuxNoSession(t)

	body := map[string]string{"name": "Test"}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/workflows", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAPIGetWorkflow(t *testing.T) {
	mux, wfID := newAPIMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/"+wfID+"/overview", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var resp map[string]json.RawMessage
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if _, ok := resp["workflow"]; !ok {
		t.Error("response missing 'workflow' key")
	}
	if _, ok := resp["nodeDefinitions"]; !ok {
		t.Error("response missing 'nodeDefinitions' key")
	}
}

func TestAPIGetWorkflow_NotFound(t *testing.T) {
	mux, _ := newAPIMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/nonexistent/overview", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	var resp apiErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Error.Code != 404 {
		t.Errorf("error code = %d, want 404", resp.Error.Code)
	}
}

func TestAPIGetWorkflow_Unauthorized(t *testing.T) {
	mux := newAPIMuxNoSession(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/some-id/overview", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAPIDeleteWorkflow(t *testing.T) {
	mux, wfID := newAPIMux(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/workflows/"+wfID, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body, got: %s", rec.Body.String())
	}
}

func TestAPIDeleteWorkflow_NotFound(t *testing.T) {
	mux, _ := newAPIMux(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/workflows/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestAPIDeleteWorkflow_Unauthorized(t *testing.T) {
	mux := newAPIMuxNoSession(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/workflows/some-id", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// --- Story 1.8: Workflow State Endpoint ---

func TestAPIWorkflowState(t *testing.T) {
	mux, wfID := newAPIMux(t)

	// Add a node so we have state to return
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "node-1", X: 100, Y: 200,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/"+wfID+"/state", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var state workflowStateResponse
	if err := json.NewDecoder(rec.Body).Decode(&state); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(state.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(state.Nodes))
	}
	if state.Nodes[0].ID != "node-1" {
		t.Errorf("node ID = %q, want %q", state.Nodes[0].ID, "node-1")
	}
	if state.Nodes[0].Definition == nil {
		t.Error("expected node definition to be populated")
	}
}

func TestAPIWorkflowState_EmptyWorkflow(t *testing.T) {
	mux, wfID := newAPIMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/"+wfID+"/state", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var state workflowStateResponse
	if err := json.NewDecoder(rec.Body).Decode(&state); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	// Empty arrays, not null
	if state.Nodes == nil {
		t.Error("expected empty nodes array, got nil")
	}
	if state.Edges == nil {
		t.Error("expected empty edges array, got nil")
	}
}

func TestAPIWorkflowState_NotFound(t *testing.T) {
	mux, _ := newAPIMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/nonexistent/state", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAPIWorkflowState_Unauthorized(t *testing.T) {
	mux := newAPIMuxNoSession(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/some-id/state", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAPIWorkflowState_WithEdges(t *testing.T) {
	mux, wfID := newAPIMux(t)

	// Add two nodes and an edge
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "node-1", X: 100, Y: 200,
	})
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "proc", InstanceID: "node-2", X: 400, Y: 200,
	})
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_edge", EdgeID: "edge-1",
		SourceNodeID: "node-1", SourcePortID: "out",
		TargetNodeID: "node-2", TargetPortID: "in",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/"+wfID+"/state", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var state workflowStateResponse
	if err := json.NewDecoder(rec.Body).Decode(&state); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(state.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(state.Nodes))
	}
	if len(state.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(state.Edges))
	}
	if state.Edges[0].SourceNodeID != "node-1" || state.Edges[0].TargetNodeID != "node-2" {
		t.Errorf("edge = %s -> %s, want node-1 -> node-2", state.Edges[0].SourceNodeID, state.Edges[0].TargetNodeID)
	}
}
