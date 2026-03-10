package main

import (
	"encoding/json"
	"testing"
)

func TestParseDeployPayload(t *testing.T) {
	raw := `{
		"apiVersion": "graphiti/v1",
		"event": "workflow.deployed",
		"timestamp": "2026-03-09T14:30:00Z",
		"deployment": {
			"id": "deploy-001",
			"target": "production",
			"triggeredBy": {"userID": "user-1", "username": "alice"}
		},
		"workflow": {
			"id": "wf-001",
			"name": "Test Workflow",
			"version": 1,
			"definition": {
				"nodes": [
					{"id": "n1", "definitionId": "source-api-gateway", "label": "Gateway", "x": 0, "y": 0, "attributeValues": {"path": "/api/test", "methods": "ALL"}}
				],
				"edges": []
			}
		}
	}`

	var payload DeployPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if payload.APIVersion != "graphiti/v1" {
		t.Errorf("apiVersion = %q, want %q", payload.APIVersion, "graphiti/v1")
	}
	if payload.Workflow.ID != "wf-001" {
		t.Errorf("workflow ID = %q, want %q", payload.Workflow.ID, "wf-001")
	}
	if len(payload.Workflow.Definition.Nodes) != 1 {
		t.Fatalf("nodes count = %d, want 1", len(payload.Workflow.Definition.Nodes))
	}
	if payload.Workflow.Definition.Nodes[0].DefinitionID != "source-api-gateway" {
		t.Errorf("node def ID = %q, want %q", payload.Workflow.Definition.Nodes[0].DefinitionID, "source-api-gateway")
	}
}

func TestBuildExecutionGraph(t *testing.T) {
	def := WorkflowDefinition{
		Nodes: []NodeDef{
			{ID: "n1", DefinitionID: "source-api-gateway", Label: "API", AttributeValues: map[string]any{"path": "/api/todos"}},
			{ID: "n2", DefinitionID: "control-http-router", Label: "Router"},
			{ID: "n3", DefinitionID: "destination-http-response", Label: "Response", AttributeValues: map[string]any{"status_code": float64(200)}},
		},
		Edges: []EdgeDef{
			{ID: "e1", SourceNodeID: "n1", SourcePortID: "out-main", TargetNodeID: "n2", TargetPortID: "in-main"},
			{ID: "e2", SourceNodeID: "n2", SourcePortID: "out-get", TargetNodeID: "n3", TargetPortID: "in-main"},
		},
	}

	graph := BuildExecutionGraph(def)

	if len(graph.Nodes) != 3 {
		t.Fatalf("graph nodes = %d, want 3", len(graph.Nodes))
	}

	// n1 should have no incoming edges
	n1 := graph.Nodes["n1"]
	if len(n1.Incoming) != 0 {
		t.Errorf("n1 incoming = %d, want 0", len(n1.Incoming))
	}

	// n2 should have 1 incoming edge from n1
	n2 := graph.Nodes["n2"]
	if len(n2.Incoming) != 1 {
		t.Errorf("n2 incoming = %d, want 1", len(n2.Incoming))
	}

	// n3 should have 1 incoming edge from n2
	n3 := graph.Nodes["n3"]
	if len(n3.Incoming) != 1 {
		t.Errorf("n3 incoming = %d, want 1", len(n3.Incoming))
	}
}

func TestTopologicalSort(t *testing.T) {
	def := WorkflowDefinition{
		Nodes: []NodeDef{
			{ID: "a", DefinitionID: "source-api-gateway"},
			{ID: "b", DefinitionID: "control-http-router"},
			{ID: "c", DefinitionID: "destination-http-response"},
		},
		Edges: []EdgeDef{
			{ID: "e1", SourceNodeID: "a", SourcePortID: "out-main", TargetNodeID: "b", TargetPortID: "in-main"},
			{ID: "e2", SourceNodeID: "b", SourcePortID: "out-get", TargetNodeID: "c", TargetPortID: "in-main"},
		},
	}

	graph := BuildExecutionGraph(def)
	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("topological sort: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("sorted length = %d, want 3", len(sorted))
	}

	// a must come before b, b must come before c
	posA, posB, posC := -1, -1, -1
	for i, id := range sorted {
		switch id {
		case "a":
			posA = i
		case "b":
			posB = i
		case "c":
			posC = i
		}
	}

	if posA >= posB {
		t.Errorf("a (pos %d) should come before b (pos %d)", posA, posB)
	}
	if posB >= posC {
		t.Errorf("b (pos %d) should come before c (pos %d)", posB, posC)
	}
}

func TestFindSourceNodes(t *testing.T) {
	graph := &ExecutionGraph{
		Nodes: map[string]*GraphNode{
			"n1": {NodeDef: NodeDef{ID: "n1", DefinitionID: "source-api-gateway"}, Incoming: nil},
			"n2": {NodeDef: NodeDef{ID: "n2", DefinitionID: "control-http-router"}, Incoming: []GraphEdge{{FromNodeID: "n1"}}},
		},
	}

	sources := graph.FindSourceNodes()
	if len(sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(sources))
	}
	if sources[0] != "n1" {
		t.Errorf("source = %q, want %q", sources[0], "n1")
	}
}

func TestHTTPRouterExecutor(t *testing.T) {
	executor := &HTTPRouterExecutor{}
	node := &GraphNode{
		NodeDef: NodeDef{
			ID:              "router-1",
			DefinitionID:    "control-http-router",
			AttributeValues: map[string]any{"id_param": "id"},
		},
		OutgoingEdges: map[string][]GraphEdge{
			"out-get":       {{FromNodeID: "router-1", FromPortID: "out-get", ToNodeID: "n-list"}},
			"out-get-by-id": {{FromNodeID: "router-1", FromPortID: "out-get-by-id", ToNodeID: "n-get"}},
			"out-post":      {{FromNodeID: "router-1", FromPortID: "out-post", ToNodeID: "n-create"}},
			"out-put":       {{FromNodeID: "router-1", FromPortID: "out-put", ToNodeID: "n-update"}},
			"out-delete":    {{FromNodeID: "router-1", FromPortID: "out-delete", ToNodeID: "n-delete"}},
		},
	}

	tests := []struct {
		name       string
		method     string
		pathParams map[string]any
		wantPort   string
	}{
		{"GET all", "GET", nil, "out-get"},
		{"GET by id", "GET", map[string]any{"id": "123"}, "out-get-by-id"},
		{"POST", "POST", nil, "out-post"},
		{"PUT", "PUT", map[string]any{"id": "123"}, "out-put"},
		{"DELETE", "DELETE", map[string]any{"id": "123"}, "out-delete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]any{
				"method":     tt.method,
				"pathParams": tt.pathParams,
			}
			result, err := executor.Execute(node, input)
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			routedPort, ok := result["_routedPort"].(string)
			if !ok {
				t.Fatal("missing _routedPort in result")
			}
			if routedPort != tt.wantPort {
				t.Errorf("routed to %q, want %q", routedPort, tt.wantPort)
			}
		})
	}
}

func TestValidatePayloadExecutor(t *testing.T) {
	executor := &ValidatePayloadExecutor{}

	tests := []struct {
		name     string
		required string
		body     map[string]any
		wantErr  bool
	}{
		{
			name:     "valid - all fields present",
			required: "title",
			body:     map[string]any{"title": "Buy milk"},
			wantErr:  false,
		},
		{
			name:     "invalid - missing required field",
			required: "title",
			body:     map[string]any{"description": "something"},
			wantErr:  true,
		},
		{
			name:     "valid - multiple required fields",
			required: "title,completed",
			body:     map[string]any{"title": "Buy milk", "completed": false},
			wantErr:  false,
		},
		{
			name:     "invalid - one of multiple missing",
			required: "title,completed",
			body:     map[string]any{"title": "Buy milk"},
			wantErr:  true,
		},
		{
			name:     "valid - empty required means no validation",
			required: "",
			body:     map[string]any{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &GraphNode{
				NodeDef: NodeDef{
					ID:              "val-1",
					DefinitionID:    "processing-validate-payload",
					AttributeValues: map[string]any{"required_fields": tt.required},
				},
			}
			input := map[string]any{"body": tt.body}
			result, err := executor.Execute(node, input)
			if err != nil {
				t.Fatalf("execute: %v", err)
			}

			if tt.wantErr {
				if result["_routedPort"] != "out-error" {
					t.Errorf("expected routing to out-error, got %v", result["_routedPort"])
				}
			} else {
				if result["_routedPort"] == "out-error" {
					t.Errorf("expected routing to out-main, got out-error")
				}
			}
		})
	}
}

func TestHTTPResponseExecutor(t *testing.T) {
	executor := &HTTPResponseExecutor{}
	node := &GraphNode{
		NodeDef: NodeDef{
			ID:           "resp-1",
			DefinitionID: "destination-http-response",
			AttributeValues: map[string]any{
				"status_code":  float64(201),
				"content_type": "application/json",
			},
		},
	}

	input := map[string]any{
		"rows": []any{map[string]any{"id": 1, "title": "Test"}},
	}

	result, err := executor.Execute(node, input)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	statusCode, ok := result["statusCode"].(int)
	if !ok {
		t.Fatal("missing statusCode")
	}
	if statusCode != 201 {
		t.Errorf("statusCode = %d, want 201", statusCode)
	}
}

func TestGetNodeExecutor(t *testing.T) {
	tests := []struct {
		defID string
		want  string
	}{
		{"source-api-gateway", "*main.APIGatewayExecutor"},
		{"control-http-router", "*main.HTTPRouterExecutor"},
		{"processing-validate-payload", "*main.ValidatePayloadExecutor"},
		{"destination-postgresql", "*main.PostgreSQLExecutor"},
		{"destination-http-response", "*main.HTTPResponseExecutor"},
		{"processing-json-transform", "*main.JSONTransformExecutor"},
		{"unknown-node", "*main.PassthroughExecutor"},
	}

	for _, tt := range tests {
		t.Run(tt.defID, func(t *testing.T) {
			executor := GetNodeExecutor(tt.defID)
			if executor == nil {
				t.Fatal("executor is nil")
			}
		})
	}
}

func TestJSONTransformExecutor(t *testing.T) {
	executor := &JSONTransformExecutor{}

	t.Run("merge mode", func(t *testing.T) {
		node := &GraphNode{
			NodeDef: NodeDef{
				ID:           "tr-1",
				DefinitionID: "processing-json-transform",
				AttributeValues: map[string]any{
					"mode":    "merge",
					"mapping": `{"name": "body.title"}`,
				},
			},
		}
		input := map[string]any{
			"body": map[string]any{"title": "Hello", "extra": "data"},
		}
		result, err := executor.Execute(node, input)
		if err != nil {
			t.Fatalf("execute: %v", err)
		}
		if result["name"] != "Hello" {
			t.Errorf("name = %v, want Hello", result["name"])
		}
	})

	t.Run("replace mode", func(t *testing.T) {
		node := &GraphNode{
			NodeDef: NodeDef{
				ID:           "tr-2",
				DefinitionID: "processing-json-transform",
				AttributeValues: map[string]any{
					"mode":    "replace",
					"mapping": `{"name": "body.title"}`,
				},
			},
		}
		input := map[string]any{
			"body":   map[string]any{"title": "Hello"},
			"method": "POST",
		}
		result, err := executor.Execute(node, input)
		if err != nil {
			t.Fatalf("execute: %v", err)
		}
		if result["name"] != "Hello" {
			t.Errorf("name = %v, want Hello", result["name"])
		}
		// In replace mode, original fields should not be present
		if _, ok := result["method"]; ok {
			t.Error("replace mode should not include original fields")
		}
	})
}

func TestCallbackSigning(t *testing.T) {
	body := []byte(`{"test": true}`)
	secret := "test-secret"

	sig := computeHMAC(body, secret)
	if sig == "" {
		t.Fatal("signature is empty")
	}

	// Verify it's a valid hex string
	if len(sig) != 64 {
		t.Errorf("signature length = %d, want 64 (sha256 hex)", len(sig))
	}

	// Same input should produce same output
	sig2 := computeHMAC(body, secret)
	if sig != sig2 {
		t.Error("same input should produce same signature")
	}

	// Different input should produce different output
	sig3 := computeHMAC([]byte(`{"test": false}`), secret)
	if sig == sig3 {
		t.Error("different input should produce different signature")
	}
}
