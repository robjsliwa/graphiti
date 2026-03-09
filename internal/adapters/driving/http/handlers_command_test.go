package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"graphiti/internal/adapters/driven/memory"
	"graphiti/internal/app"
	"graphiti/internal/domain"
)

// fakeNodeDefRepo implements driven.NodeDefinitionRepository for tests.
type fakeNodeDefRepo struct {
	defs []*domain.NodeDefinition
}

func (r *fakeNodeDefRepo) LoadAll(_ context.Context) ([]*domain.NodeDefinition, error) {
	return r.defs, nil
}

func (r *fakeNodeDefRepo) GetByID(_ context.Context, id string) (*domain.NodeDefinition, error) {
	for _, d := range r.defs {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, domain.ErrNodeNotFound
}

func (r *fakeNodeDefRepo) GetByCategory(_ context.Context, category string) ([]*domain.NodeDefinition, error) {
	var result []*domain.NodeDefinition
	for _, d := range r.defs {
		if d.Category.Group == category {
			result = append(result, d)
		}
	}
	return result, nil
}

func (r *fakeNodeDefRepo) Search(_ context.Context, query string) ([]*domain.NodeDefinition, error) {
	return r.defs, nil
}

// testNodeDefs returns node definitions used across handler tests.
func testNodeDefs() []*domain.NodeDefinition {
	return []*domain.NodeDefinition{
		{
			ID:       "src",
			Name:     "Source",
			Category: domain.Category{Group: "Sources"},
			Shape:    domain.Shape{Type: domain.ShapeRoundedRect, Width: 200},
			Outputs: []domain.PortDefinition{
				{ID: "out", Label: "Output", Type: domain.PortTypeData, Position: domain.PortPositionRightCenter, MaxConnections: -1},
			},
			Attributes: []domain.AttributeDefinition{
				{ID: "url", Label: "URL", Type: domain.AttrTypeString, Display: domain.DisplayConfigPanel},
			},
		},
		{
			ID:       "proc",
			Name:     "Processor",
			Category: domain.Category{Group: "Processing"},
			Shape:    domain.Shape{Type: domain.ShapeRoundedRect, Width: 200},
			Inputs: []domain.PortDefinition{
				{ID: "in", Label: "Input", Type: domain.PortTypeData, Position: domain.PortPositionLeftCenter, MaxConnections: 1},
			},
			Outputs: []domain.PortDefinition{
				{ID: "out", Label: "Output", Type: domain.PortTypeData, Position: domain.PortPositionRightCenter, MaxConnections: -1},
			},
		},
		{
			ID:       "dest",
			Name:     "Destination",
			Category: domain.Category{Group: "Destinations"},
			Shape:    domain.Shape{Type: domain.ShapeRoundedRect, Width: 200},
			Inputs: []domain.PortDefinition{
				{ID: "in", Label: "Input", Type: domain.PortTypeData, Position: domain.PortPositionLeftCenter, MaxConnections: 1},
			},
		},
	}
}

// testSetup creates a WorkflowService, NodeRegistry, and a test workflow.
// Returns the service, registry, and the workflow ID.
func testSetup(t *testing.T) (*app.WorkflowService, *app.NodeRegistry, string) {
	t.Helper()

	repo := &fakeNodeDefRepo{defs: testNodeDefs()}
	registry := app.NewNodeRegistry(repo)
	if err := registry.Load(context.Background()); err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	wfRepo := memory.NewWorkflowRepo()
	svc := app.NewWorkflowService(wfRepo, registry, 100)

	wf, err := svc.CreateWorkflow(context.Background(), "Test Workflow", "test", "user-1")
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	return svc, registry, wf.ID
}

// newCommandMux creates a mux with command routes wired to the given service/registry.
func newCommandMux(svc *app.WorkflowService, registry *app.NodeRegistry) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/workflows/{id}/commands", handleExecuteCommand(svc, registry))
	mux.HandleFunc("POST /api/workflows/{id}/undo", handleUndo(svc))
	mux.HandleFunc("POST /api/workflows/{id}/redo", handleRedo(svc))
	mux.HandleFunc("POST /api/workflows/{id}/clipboard/copy", handleClipboardCopy(svc))
	mux.HandleFunc("POST /api/workflows/{id}/clipboard/paste", handleClipboardPaste(svc))
	mux.HandleFunc("PATCH /api/workflows/{id}/nodes/{nodeId}/attributes", handleAttributeUpdate(svc, registry))
	return mux
}

// postJSON sends a POST request with JSON body and returns the recorder.
func postJSON(t *testing.T, mux http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// decodeCommandResponse parses a commandResponse from the recorder.
func decodeCommandResponse(t *testing.T, rec *httptest.ResponseRecorder) commandResponse {
	t.Helper()
	var resp commandResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return resp
}

func TestHandleExecuteCommand_AddNode(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type:         "add_node",
		DefinitionID: "src",
		InstanceID:   "node-1",
		X:            100,
		Y:            200,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	resp := decodeCommandResponse(t, rec)
	if !resp.OK {
		t.Fatalf("expected ok=true, got error: %s", resp.Error)
	}
	if !resp.CanUndo {
		t.Error("expected canUndo=true after add_node")
	}
	if resp.Workflow == nil {
		t.Fatal("expected workflow state in response")
	}
	if len(resp.Workflow.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(resp.Workflow.Nodes))
	}
	node := resp.Workflow.Nodes[0]
	if node.ID != "node-1" {
		t.Errorf("node ID = %q, want %q", node.ID, "node-1")
	}
	if node.DefinitionID != "src" {
		t.Errorf("node definitionID = %q, want %q", node.DefinitionID, "src")
	}
	if node.X != 100 || node.Y != 200 {
		t.Errorf("node position = (%v, %v), want (100, 200)", node.X, node.Y)
	}
	if node.Definition == nil {
		t.Error("expected node definition to be populated")
	}
}

func TestHandleExecuteCommand_MoveNode(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	// First add a node
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "node-1", X: 100, Y: 200,
	})

	// Move the node
	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "move_node", NodeID: "node-1", FromX: 100, FromY: 200, ToX: 300, ToY: 400,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	resp := decodeCommandResponse(t, rec)
	if !resp.OK {
		t.Fatalf("expected ok=true, got error: %s", resp.Error)
	}
	if resp.Workflow == nil {
		t.Fatal("expected workflow state in response")
	}
	if len(resp.Workflow.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(resp.Workflow.Nodes))
	}
	node := resp.Workflow.Nodes[0]
	if node.X != 300 || node.Y != 400 {
		t.Errorf("node position = (%v, %v), want (300, 400)", node.X, node.Y)
	}
}

func TestHandleExecuteCommand_AddEdge(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	// Add two nodes
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "node-1", X: 100, Y: 200,
	})
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "proc", InstanceID: "node-2", X: 400, Y: 200,
	})

	// Add an edge
	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type:         "add_edge",
		EdgeID:       "edge-1",
		SourceNodeID: "node-1",
		SourcePortID: "out",
		TargetNodeID: "node-2",
		TargetPortID: "in",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	resp := decodeCommandResponse(t, rec)
	if !resp.OK {
		t.Fatalf("expected ok=true, got error: %s", resp.Error)
	}
	if resp.Workflow == nil {
		t.Fatal("expected workflow state in response")
	}
	if len(resp.Workflow.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(resp.Workflow.Edges))
	}
	edge := resp.Workflow.Edges[0]
	if edge.ID != "edge-1" {
		t.Errorf("edge ID = %q, want %q", edge.ID, "edge-1")
	}
	if edge.SourceNodeID != "node-1" || edge.TargetNodeID != "node-2" {
		t.Errorf("edge endpoints = (%s -> %s), want (node-1 -> node-2)", edge.SourceNodeID, edge.TargetNodeID)
	}
}

func TestHandleExecuteCommand_RemoveNode(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	// Add a node
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "node-1", X: 100, Y: 200,
	})

	// Remove the node
	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "remove_node", NodeID: "node-1",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	resp := decodeCommandResponse(t, rec)
	if !resp.OK {
		t.Fatalf("expected ok=true, got error: %s", resp.Error)
	}
	if resp.Workflow == nil {
		t.Fatal("expected workflow state in response")
	}
	if len(resp.Workflow.Nodes) != 0 {
		t.Errorf("expected 0 nodes after remove, got %d", len(resp.Workflow.Nodes))
	}
}

func TestHandleUndo(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	// Add a node
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "node-1", X: 100, Y: 200,
	})

	// Undo
	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/undo", struct{}{})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	resp := decodeCommandResponse(t, rec)
	if !resp.OK {
		t.Fatalf("expected ok=true, got error: %s", resp.Error)
	}
	if !resp.CanRedo {
		t.Error("expected canRedo=true after undo")
	}
	if resp.Workflow == nil {
		t.Fatal("expected workflow state in response")
	}
	if len(resp.Workflow.Nodes) != 0 {
		t.Errorf("expected 0 nodes after undo, got %d", len(resp.Workflow.Nodes))
	}
}

func TestHandleRedo(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	// Add a node, then undo
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "node-1", X: 100, Y: 200,
	})
	postJSON(t, mux, "/api/workflows/"+wfID+"/undo", struct{}{})

	// Redo
	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/redo", struct{}{})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	resp := decodeCommandResponse(t, rec)
	if !resp.OK {
		t.Fatalf("expected ok=true, got error: %s", resp.Error)
	}
	if !resp.CanUndo {
		t.Error("expected canUndo=true after redo")
	}
	if resp.Workflow == nil {
		t.Fatal("expected workflow state in response")
	}
	if len(resp.Workflow.Nodes) != 1 {
		t.Errorf("expected 1 node after redo, got %d", len(resp.Workflow.Nodes))
	}
}

func TestHandleClipboardCopy(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	// Add two nodes
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "node-1", X: 100, Y: 200,
	})
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "proc", InstanceID: "node-2", X: 400, Y: 200,
	})

	// Copy both nodes
	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/clipboard/copy", copyRequest{
		NodeIDs: []string{"node-1", "node-2"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp copyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected ok=true, got error: %s", resp.Error)
	}
	if resp.Payload == nil {
		t.Fatal("expected payload in copy response")
	}

	// Verify the payload structure
	payloadJSON, err := json.Marshal(resp.Payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	var payload struct {
		Version int `json:"version"`
		Nodes   []struct {
			OriginalID   string `json:"originalId"`
			DefinitionID string `json:"definitionId"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload.Version != 1 {
		t.Errorf("payload version = %d, want 1", payload.Version)
	}
	if len(payload.Nodes) != 2 {
		t.Fatalf("expected 2 nodes in payload, got %d", len(payload.Nodes))
	}
}

func TestHandleClipboardPaste(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	// Add a node
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "node-1", X: 100, Y: 200,
	})

	// Copy
	copyRec := postJSON(t, mux, "/api/workflows/"+wfID+"/clipboard/copy", copyRequest{
		NodeIDs: []string{"node-1"},
	})

	var copyResp copyResponse
	json.NewDecoder(copyRec.Body).Decode(&copyResp)

	// Marshal the payload for paste request
	payloadJSON, err := json.Marshal(copyResp.Payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	// Paste at a new position
	pasteBody := pasteRequest{
		Payload: payloadJSON,
		X:       500,
		Y:       300,
	}
	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/clipboard/paste", pasteBody)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	resp := decodeCommandResponse(t, rec)
	if !resp.OK {
		t.Fatalf("expected ok=true, got error: %s", resp.Error)
	}
	if resp.Workflow == nil {
		t.Fatal("expected workflow state in response")
	}
	// Original node + pasted node
	if len(resp.Workflow.Nodes) != 2 {
		t.Fatalf("expected 2 nodes after paste, got %d", len(resp.Workflow.Nodes))
	}

	// The pasted node should have a different ID from the original
	ids := make(map[string]bool)
	for _, n := range resp.Workflow.Nodes {
		ids[n.ID] = true
	}
	if len(ids) != 2 {
		t.Error("pasted node should have a different ID from the original")
	}
}

func TestHandleAttributeUpdate(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	// Add a node with an attribute
	postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "node-1", X: 100, Y: 200,
	})

	// Send PATCH with form data to update the attribute
	form := url.Values{}
	form.Set("url", "https://example.com")
	req := httptest.NewRequest(http.MethodPatch, "/api/workflows/"+wfID+"/nodes/node-1/attributes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// Verify the attribute was updated by fetching the workflow
	wf, err := svc.GetWorkflow(context.Background(), wfID)
	if err != nil {
		t.Fatalf("failed to get workflow: %v", err)
	}
	node := wf.FindNode("node-1")
	if node == nil {
		t.Fatal("node-1 not found in workflow")
	}
	if node.AttributeValues["url"] != "https://example.com" {
		t.Errorf("attribute url = %v, want %q", node.AttributeValues["url"], "https://example.com")
	}
}

func TestHandleExecuteCommand_InvalidBody(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	req := httptest.NewRequest(http.MethodPost, "/api/workflows/"+wfID+"/commands", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleExecuteCommand_UnknownCommandType(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "unknown_command",
	})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleUndo_NothingToUndo(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/undo", struct{}{})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleRedo_NothingToRedo(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/redo", struct{}{})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleExecuteCommand_RemoveNodeAlsoRemovesEdges(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

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

	// Remove node-1 (should also remove edge-1)
	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "remove_node", NodeID: "node-1",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	resp := decodeCommandResponse(t, rec)
	if !resp.OK {
		t.Fatalf("expected ok=true, got error: %s", resp.Error)
	}
	if len(resp.Workflow.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(resp.Workflow.Nodes))
	}
	if len(resp.Workflow.Edges) != 0 {
		t.Errorf("expected 0 edges after removing connected node, got %d", len(resp.Workflow.Edges))
	}
}

func TestHandleExecuteCommand_UndoRedoRoundTrip(t *testing.T) {
	svc, registry, wfID := testSetup(t)
	mux := newCommandMux(svc, registry)

	// Add a node
	rec := postJSON(t, mux, "/api/workflows/"+wfID+"/commands", commandRequest{
		Type: "add_node", DefinitionID: "src", InstanceID: "node-1", X: 100, Y: 200,
	})
	resp := decodeCommandResponse(t, rec)
	if len(resp.Workflow.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(resp.Workflow.Nodes))
	}

	// Undo - node should be gone
	rec = postJSON(t, mux, "/api/workflows/"+wfID+"/undo", struct{}{})
	resp = decodeCommandResponse(t, rec)
	if len(resp.Workflow.Nodes) != 0 {
		t.Fatalf("expected 0 nodes after undo, got %d", len(resp.Workflow.Nodes))
	}
	if !resp.CanRedo {
		t.Error("expected canRedo=true")
	}
	if resp.CanUndo {
		t.Error("expected canUndo=false")
	}

	// Redo - node should be back
	rec = postJSON(t, mux, "/api/workflows/"+wfID+"/redo", struct{}{})
	resp = decodeCommandResponse(t, rec)
	if len(resp.Workflow.Nodes) != 1 {
		t.Fatalf("expected 1 node after redo, got %d", len(resp.Workflow.Nodes))
	}
	if !resp.CanUndo {
		t.Error("expected canUndo=true")
	}
	if resp.CanRedo {
		t.Error("expected canRedo=false")
	}
}
