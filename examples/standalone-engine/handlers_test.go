package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleDeploy(t *testing.T) {
	runner := NewRunner(nil, "", "")
	handler := handleDeploy(runner, "")

	payload := makeTestPayload()
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/deploy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["status"] != "deployed" {
		t.Errorf("status = %v, want deployed", resp["status"])
	}
}

func TestHandleDeployWithHMAC(t *testing.T) {
	secret := "test-secret-123"
	runner := NewRunner(nil, "", "")
	handler := handleDeploy(runner, secret)

	payload := makeTestPayload()
	body, _ := json.Marshal(payload)
	sig := computeHMAC(body, secret)

	t.Run("valid signature", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/deploy", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Graphiti-Signature", "sha256="+sig)
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/deploy", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Graphiti-Signature", "sha256=invalid")
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("missing signature", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/deploy", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

func TestHandleDeployInvalidJSON(t *testing.T) {
	runner := NewRunner(nil, "", "")
	handler := handleDeploy(runner, "")

	req := httptest.NewRequest("POST", "/deploy", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleWorkflowRequestNoMatch(t *testing.T) {
	runner := NewRunner(nil, "", "")
	handler := handleWorkflowRequest(runner)

	req := httptest.NewRequest("GET", "/api/nothing", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleWorkflowRequestSuccess(t *testing.T) {
	runner := NewRunner(nil, "", "")
	runner.Deploy(makeTestPayload())

	handler := handleWorkflowRequest(runner)

	req := httptest.NewRequest("GET", "/api/todos", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestHealthEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	runner := NewRunner(nil, "", "")

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"workflows": len(runner.workflows),
		})
	})

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
}

func TestHandleWorkflowStatus_Deployed(t *testing.T) {
	runner := NewRunner(nil, "", "")
	runner.Deploy(makeTestPayload())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /status/workflows/{id}", handleWorkflowStatus(runner))

	req := httptest.NewRequest("GET", "/status/workflows/wf-test", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["deployed"] != true {
		t.Errorf("deployed = %v, want true", resp["deployed"])
	}
	if resp["workflowId"] != "wf-test" {
		t.Errorf("workflowId = %v, want wf-test", resp["workflowId"])
	}
}

func TestHandleWorkflowStatus_NotFound(t *testing.T) {
	runner := NewRunner(nil, "", "")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /status/workflows/{id}", handleWorkflowStatus(runner))

	req := httptest.NewRequest("GET", "/status/workflows/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["deployed"] != false {
		t.Errorf("deployed = %v, want false", resp["deployed"])
	}
}

func TestEndToEndSimpleWorkflow(t *testing.T) {
	// Deploy a simple API Gateway -> HTTP Response workflow
	runner := NewRunner(nil, "", "")
	runner.Deploy(makeTestPayload())

	handler := handleWorkflowRequest(runner)

	// GET request
	req := httptest.NewRequest("GET", "/api/todos", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET status = %d, want %d", rec.Code, http.StatusOK)
	}

	// POST request with body
	postBody := map[string]any{"title": "Test todo"}
	bodyBytes, _ := json.Marshal(postBody)
	req = httptest.NewRequest("POST", "/api/todos", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("POST status = %d, want %d", rec.Code, http.StatusOK)
	}
}
