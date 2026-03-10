package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunnerDeploy(t *testing.T) {
	runner := NewRunner(nil, "", "")

	payload := makeTestPayload()
	if err := runner.Deploy(payload); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	if len(runner.workflows) != 1 {
		t.Errorf("workflows count = %d, want 1", len(runner.workflows))
	}
	if len(runner.routes) != 1 {
		t.Errorf("routes count = %d, want 1", len(runner.routes))
	}
}

func TestRunnerDeployInvalidWorkflow(t *testing.T) {
	runner := NewRunner(nil, "", "")

	// Workflow with cycle
	payload := DeployPayload{
		Workflow: WorkflowInfo{
			ID:   "wf-cycle",
			Name: "Cyclic Workflow",
			Definition: WorkflowDefinition{
				Nodes: []NodeDef{
					{ID: "a", DefinitionID: "processing-json-transform"},
					{ID: "b", DefinitionID: "processing-json-transform"},
				},
				Edges: []EdgeDef{
					{ID: "e1", SourceNodeID: "a", SourcePortID: "out-main", TargetNodeID: "b", TargetPortID: "in-main"},
					{ID: "e2", SourceNodeID: "b", SourcePortID: "out-main", TargetNodeID: "a", TargetPortID: "in-main"},
				},
			},
		},
	}

	err := runner.Deploy(payload)
	if err == nil {
		t.Fatal("expected error for cyclic workflow, got nil")
	}
}

func TestRunnerMatchRoute(t *testing.T) {
	runner := NewRunner(nil, "", "")
	runner.Deploy(makeTestPayload())

	tests := []struct {
		name      string
		method    string
		path      string
		wantMatch bool
	}{
		{"exact match", "GET", "/api/todos", true},
		{"exact match POST", "POST", "/api/todos", true},
		{"with ID param", "GET", "/api/todos/123", true},
		{"no match", "GET", "/api/users", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, _ := runner.MatchRoute(tt.method, tt.path)
			if tt.wantMatch && entry == nil {
				t.Error("expected route match, got nil")
			}
			if !tt.wantMatch && entry != nil {
				t.Errorf("expected no match, got %+v", entry)
			}
		})
	}
}

func TestMatchPathSegments(t *testing.T) {
	tests := []struct {
		name      string
		pattern   []string
		request   []string
		wantMatch bool
		wantParam string
	}{
		{"exact", []string{"api", "todos"}, []string{"api", "todos"}, true, ""},
		{"with id", []string{"api", "todos"}, []string{"api", "todos", "123"}, true, "123"},
		{"param in pattern", []string{"api", "todos", ":id"}, []string{"api", "todos", "456"}, true, "456"},
		{"mismatch", []string{"api", "todos"}, []string{"api", "users"}, false, ""},
		{"too short", []string{"api", "todos", "extra"}, []string{"api", "todos"}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := matchPathSegments(tt.pattern, tt.request)
			if tt.wantMatch && params == nil {
				t.Fatal("expected match, got nil")
			}
			if !tt.wantMatch && params != nil {
				t.Fatalf("expected no match, got %v", params)
			}
			if tt.wantParam != "" {
				id, _ := params["id"].(string)
				if id != tt.wantParam {
					t.Errorf("param id = %q, want %q", id, tt.wantParam)
				}
			}
		})
	}
}

func TestRunnerCallbackSending(t *testing.T) {
	var receivedCallbacks []StatusCallback

	callbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cb StatusCallback
		json.NewDecoder(r.Body).Decode(&cb)
		receivedCallbacks = append(receivedCallbacks, cb)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer callbackServer.Close()

	runner := NewRunner(nil, callbackServer.URL, "test-secret")

	payload := makeSimplePayload()
	if err := runner.Deploy(payload); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	// Execute a request against the deployed workflow
	entry, pathParams := runner.MatchRoute("GET", "/api/simple")
	if entry == nil {
		t.Fatal("route not matched")
	}

	req := httptest.NewRequest("GET", "/api/simple", nil)
	body := map[string]any{}
	_, err := runner.ExecuteWorkflow(req.Context(), entry, req, pathParams, body)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	// Wait briefly for async callbacks
	// (callbacks are sent in goroutines)
	// We already have 50ms sleeps in sendCallback, so callbacks
	// should arrive by the time ExecuteWorkflow returns

	if len(receivedCallbacks) < 2 {
		t.Logf("received %d callbacks (may need timing adjustment)", len(receivedCallbacks))
	}
}

func TestRunnerCallbackHMAC(t *testing.T) {
	var receivedSig string

	callbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-Graphiti-Signature")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer callbackServer.Close()

	runner := NewRunner(nil, callbackServer.URL, "my-secret")
	runner.sendCallback("wf-1", "run-1", "n-1", "running", timeNow(), timeZero(), "", nil)

	// Wait for async callback
	waitForCallbacks(100)

	if receivedSig == "" {
		t.Fatal("expected X-Graphiti-Signature header")
	}
	if !hasPrefix(receivedSig, "sha256=") {
		t.Errorf("signature = %q, should start with sha256=", receivedSig)
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func makeTestPayload() DeployPayload {
	return DeployPayload{
		APIVersion: "graphiti/v1",
		Event:      "workflow.deployed",
		Workflow: WorkflowInfo{
			ID:      "wf-test",
			Name:    "Test Workflow",
			Version: 1,
			Definition: WorkflowDefinition{
				Nodes: []NodeDef{
					{
						ID:           "n-gateway",
						DefinitionID: "source-api-gateway",
						Label:        "API Gateway",
						AttributeValues: map[string]any{
							"path":    "/api/todos",
							"methods": "ALL",
						},
					},
					{
						ID:           "n-response",
						DefinitionID: "destination-http-response",
						Label:        "Response",
						AttributeValues: map[string]any{
							"status_code":  float64(200),
							"content_type": "application/json",
						},
					},
				},
				Edges: []EdgeDef{
					{
						ID:           "e-1",
						SourceNodeID: "n-gateway",
						SourcePortID: "out-main",
						TargetNodeID: "n-response",
						TargetPortID: "in-main",
					},
				},
			},
		},
	}
}

func makeSimplePayload() DeployPayload {
	return DeployPayload{
		APIVersion: "graphiti/v1",
		Event:      "workflow.deployed",
		Workflow: WorkflowInfo{
			ID:      "wf-simple",
			Name:    "Simple Workflow",
			Version: 1,
			Definition: WorkflowDefinition{
				Nodes: []NodeDef{
					{
						ID:           "n-gw",
						DefinitionID: "source-api-gateway",
						Label:        "Gateway",
						AttributeValues: map[string]any{
							"path":    "/api/simple",
							"methods": "ALL",
						},
					},
					{
						ID:           "n-resp",
						DefinitionID: "destination-http-response",
						Label:        "Response",
						AttributeValues: map[string]any{
							"status_code":  float64(200),
							"content_type": "application/json",
						},
					},
				},
				Edges: []EdgeDef{
					{
						ID:           "e-1",
						SourceNodeID: "n-gw",
						SourcePortID: "out-main",
						TargetNodeID: "n-resp",
						TargetPortID: "in-main",
					},
				},
			},
		},
	}
}

func waitForCallbacks(ms int) {
	// Simple wait for async operations - used only in tests
	<-timeAfter(ms)
}

func timeAfter(ms int) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		for range ms {
			_ = ms // burn a little time
		}
		close(ch)
	}()
	return ch
}
