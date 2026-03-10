package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"graphiti/internal/domain"
)

func TestWebhookDeployTarget_Deploy_Success(t *testing.T) {
	var received []byte
	var sigHeader string
	var eventHeader string
	var contentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received, _ = io.ReadAll(r.Body)
		sigHeader = r.Header.Get("X-Graphiti-Signature")
		eventHeader = r.Header.Get("X-Graphiti-Event")
		contentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"runId": "run-123"})
	}))
	defer server.Close()

	target := NewWebhookDeployTarget(Config{
		URL:        server.URL,
		HMACSecret: "test-secret",
		Timeout:    5 * time.Second,
		MaxRetries: 3,
	})

	payload := domain.DeployPayload{
		APIVersion: "graphiti/v1",
		Event:      "workflow.deployed",
		Timestamp:  time.Now(),
		Deployment: domain.DeploymentInfo{
			ID:     "deploy-1",
			Target: "production",
			TriggeredBy: domain.TriggerUser{
				UserID:   "user-1",
				Username: "alice",
			},
		},
		Workflow: domain.WorkflowInfo{
			ID:      "wf-1",
			Name:    "Test Pipeline",
			Version: 1,
		},
		PreviousVersion: 0,
	}

	result, err := target.Deploy(context.Background(), payload)
	if err != nil {
		t.Fatalf("deploy failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.RunID != "run-123" {
		t.Errorf("expected runId 'run-123', got %q", result.RunID)
	}

	// Verify HMAC signature
	if sigHeader == "" {
		t.Fatal("missing X-Graphiti-Signature header")
	}
	expectedSig := computeHMAC(received, "test-secret")
	if sigHeader != "sha256="+expectedSig {
		t.Errorf("signature mismatch: got %q, want %q", sigHeader, "sha256="+expectedSig)
	}

	// Verify event header
	if eventHeader != "workflow.deployed" {
		t.Errorf("expected X-Graphiti-Event 'workflow.deployed', got %q", eventHeader)
	}

	// Verify content type
	if contentType != "application/json" {
		t.Errorf("expected application/json, got %q", contentType)
	}

	// Verify payload was sent
	var sent domain.DeployPayload
	if err := json.Unmarshal(received, &sent); err != nil {
		t.Fatalf("unmarshal received payload: %v", err)
	}
	if sent.Workflow.Name != "Test Pipeline" {
		t.Errorf("expected workflow name 'Test Pipeline', got %q", sent.Workflow.Name)
	}
}

func TestWebhookDeployTarget_Deploy_Checksum(t *testing.T) {
	var received []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer server.Close()

	target := NewWebhookDeployTarget(Config{
		URL:        server.URL,
		HMACSecret: "secret",
		Timeout:    5 * time.Second,
	})

	payload := domain.DeployPayload{
		APIVersion: "graphiti/v1",
		Event:      "workflow.deployed",
		Workflow: domain.WorkflowInfo{
			ID:         "wf-1",
			Name:       "Test",
			Version:    1,
			Definition: map[string]any{"nodes": []any{}, "edges": []any{}},
		},
	}

	_, err := target.Deploy(context.Background(), payload)
	if err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Verify checksum was included in payload
	var sent map[string]any
	json.Unmarshal(received, &sent)
	checksum, ok := sent["checksum"].(string)
	if !ok || checksum == "" {
		t.Error("expected non-empty checksum in payload")
	}
	if len(checksum) < 10 {
		t.Errorf("checksum too short: %q", checksum)
	}
}

func TestWebhookDeployTarget_Deploy_RetriesOnServerError(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"runId": "run-456"})
	}))
	defer server.Close()

	target := NewWebhookDeployTarget(Config{
		URL:            server.URL,
		HMACSecret:     "secret",
		Timeout:        5 * time.Second,
		MaxRetries:     3,
		InitialBackoff: 1 * time.Millisecond, // fast for tests
	})

	result, err := target.Deploy(context.Background(), domain.DeployPayload{
		APIVersion: "graphiti/v1",
		Event:      "workflow.deployed",
	})
	if err != nil {
		t.Fatalf("deploy should have succeeded after retries: %v", err)
	}
	if !result.Success {
		t.Error("expected success after retries")
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestWebhookDeployTarget_Deploy_FailsAfterMaxRetries(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	target := NewWebhookDeployTarget(Config{
		URL:            server.URL,
		HMACSecret:     "secret",
		Timeout:        5 * time.Second,
		MaxRetries:     3,
		InitialBackoff: 1 * time.Millisecond,
	})

	_, err := target.Deploy(context.Background(), domain.DeployPayload{
		APIVersion: "graphiti/v1",
		Event:      "workflow.deployed",
	})
	if err == nil {
		t.Fatal("expected error after max retries")
	}
	// 1 initial + 3 retries = 4 attempts
	if attempts.Load() != 4 {
		t.Errorf("expected 4 attempts (1 initial + 3 retries), got %d", attempts.Load())
	}
}

func TestWebhookDeployTarget_Deploy_DoesNotRetryClientErrors(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "bad payload"})
	}))
	defer server.Close()

	target := NewWebhookDeployTarget(Config{
		URL:            server.URL,
		HMACSecret:     "secret",
		Timeout:        5 * time.Second,
		MaxRetries:     3,
		InitialBackoff: 1 * time.Millisecond,
	})

	_, err := target.Deploy(context.Background(), domain.DeployPayload{
		APIVersion: "graphiti/v1",
		Event:      "workflow.deployed",
	})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	// Client errors should not be retried
	if attempts.Load() != 1 {
		t.Errorf("expected 1 attempt (no retry on 400), got %d", attempts.Load())
	}
}

func TestWebhookDeployTarget_Deploy_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // slow server
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	target := NewWebhookDeployTarget(Config{
		URL:        server.URL,
		HMACSecret: "secret",
		Timeout:    10 * time.Second, // long client timeout so ctx cancels first
		MaxRetries: 0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := target.Deploy(ctx, domain.DeployPayload{
		APIVersion: "graphiti/v1",
		Event:      "workflow.deployed",
	})
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

func computeHMAC(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
