package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

// Config holds webhook deploy target settings.
type Config struct {
	URL            string
	HMACSecret     string
	Timeout        time.Duration
	MaxRetries     int
	InitialBackoff time.Duration
	StatusBaseURL  string // Optional URL for status checks; "-" disables; empty auto-derives from URL
}

// WebhookDeployTarget sends workflow definitions to an HTTP endpoint with HMAC signing.
type WebhookDeployTarget struct {
	url            string
	hmacSecret     string
	timeout        time.Duration
	maxRetries     int
	initialBackoff time.Duration
	statusBaseURL  string // resolved status check URL; empty means disabled
	client         *http.Client
}

// NewWebhookDeployTarget creates a new webhook deploy target adapter.
func NewWebhookDeployTarget(cfg Config) *WebhookDeployTarget {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	backoff := cfg.InitialBackoff
	if backoff == 0 {
		backoff = 1 * time.Second
	}
	statusURL := resolveStatusBaseURL(cfg.StatusBaseURL, cfg.URL)

	return &WebhookDeployTarget{
		url:            cfg.URL,
		hmacSecret:     cfg.HMACSecret,
		timeout:        timeout,
		maxRetries:     cfg.MaxRetries,
		initialBackoff: backoff,
		statusBaseURL:  statusURL,
		client:         &http.Client{Timeout: timeout},
	}
}

// Deploy sends the workflow payload to the configured webhook endpoint.
func (w *WebhookDeployTarget) Deploy(ctx context.Context, payload domain.DeployPayload) (*domain.DeployResult, error) {
	// Compute checksum of the definition
	if payload.Workflow.Definition != nil {
		defBytes, _ := json.Marshal(payload.Workflow.Definition)
		hash := sha256.Sum256(defBytes)
		payload.Checksum = "sha256:" + hex.EncodeToString(hash[:])
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal deploy payload: %w", err)
	}

	signature := w.sign(body)

	var lastErr error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := w.initialBackoff * (1 << (attempt - 1)) // exponential: 1s, 2s, 4s
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		result, err := w.doRequest(ctx, body, signature, payload.Event)
		if err != nil {
			// Client errors (4xx) are not retryable
			if _, ok := err.(*errClientError); ok {
				return nil, err
			}
			lastErr = err
			continue // network or server error, retry
		}
		return result, nil
	}

	return nil, fmt.Errorf("deploy failed after %d attempts: %w", w.maxRetries+1, lastErr)
}

// errClientError signals a non-retryable client error (4xx).
type errClientError struct {
	statusCode int
	body       string
}

func (e *errClientError) Error() string {
	return fmt.Sprintf("deploy rejected (HTTP %d): %s", e.statusCode, e.body)
}

// errServerError signals a retryable server error (5xx).
type errServerError struct {
	statusCode int
}

func (e *errServerError) Error() string {
	return fmt.Sprintf("server error (HTTP %d)", e.statusCode)
}

func (w *WebhookDeployTarget) doRequest(ctx context.Context, body []byte, signature, event string) (*domain.DeployResult, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", w.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Graphiti-Signature", "sha256="+signature)
	req.Header.Set("X-Graphiti-Event", event)

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err // network error, retryable
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result := &domain.DeployResult{Success: true}
		var respData struct {
			RunID string `json:"runId"`
		}
		if json.Unmarshal(respBody, &respData) == nil && respData.RunID != "" {
			result.RunID = respData.RunID
		}
		result.Message = "deployed successfully"
		return result, nil
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, &errClientError{statusCode: resp.StatusCode, body: string(respBody)}
	}

	// Server errors (5xx) - retryable
	return nil, &errServerError{statusCode: resp.StatusCode}
}

func (w *WebhookDeployTarget) sign(body []byte) string {
	mac := hmac.New(sha256.New, []byte(w.hmacSecret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// CheckDeployStatus checks whether a workflow is deployed on the engine.
func (w *WebhookDeployTarget) CheckDeployStatus(ctx context.Context, workflowID string, version int) (*domain.DeployStatusResult, error) {
	if w.statusBaseURL == "" {
		return &domain.DeployStatusResult{
			WorkflowID:   workflowID,
			Version:      version,
			Verification: domain.DeployVerificationUnknown,
			Message:      "status checking not configured",
			CheckedAt:    time.Now(),
		}, nil
	}

	statusCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	statusURL, err := neturl.JoinPath(w.statusBaseURL, workflowID)
	if err != nil {
		return &domain.DeployStatusResult{
			WorkflowID:   workflowID,
			Version:      version,
			Verification: domain.DeployVerificationError,
			Message:      "invalid status URL",
			CheckedAt:    time.Now(),
		}, nil
	}

	req, err := http.NewRequestWithContext(statusCtx, "GET", statusURL, nil)
	if err != nil {
		return &domain.DeployStatusResult{
			WorkflowID:   workflowID,
			Version:      version,
			Verification: domain.DeployVerificationError,
			Message:      "failed to create request",
			CheckedAt:    time.Now(),
		}, nil
	}

	// HMAC-sign the request URL as body
	sig := w.sign([]byte(workflowID))
	req.Header.Set("X-Graphiti-Signature", "sha256="+sig)

	resp, err := w.client.Do(req)
	if err != nil {
		return &domain.DeployStatusResult{
			WorkflowID:   workflowID,
			Version:      version,
			Verification: domain.DeployVerificationError,
			Message:      "engine unreachable",
			CheckedAt:    time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &domain.DeployStatusResult{
			WorkflowID:   workflowID,
			Version:      version,
			Verification: domain.DeployVerificationMissing,
			Message:      "workflow not found on engine",
			CheckedAt:    time.Now(),
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return &domain.DeployStatusResult{
			WorkflowID:   workflowID,
			Version:      version,
			Verification: domain.DeployVerificationError,
			Message:      fmt.Sprintf("unexpected status %d from engine", resp.StatusCode),
			CheckedAt:    time.Now(),
		}, nil
	}

	var statusResp struct {
		Deployed   bool   `json:"deployed"`
		WorkflowID string `json:"workflowId"`
		Version    int    `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return &domain.DeployStatusResult{
			WorkflowID:   workflowID,
			Version:      version,
			Verification: domain.DeployVerificationError,
			Message:      "invalid response from engine",
			CheckedAt:    time.Now(),
		}, nil
	}

	if !statusResp.Deployed {
		return &domain.DeployStatusResult{
			WorkflowID:   workflowID,
			Version:      version,
			Verification: domain.DeployVerificationMissing,
			Message:      "engine reports workflow not deployed",
			CheckedAt:    time.Now(),
		}, nil
	}

	return &domain.DeployStatusResult{
		WorkflowID:   workflowID,
		Version:      statusResp.Version,
		Verification: domain.DeployVerificationVerified,
		Message:      "engine confirms deployment",
		CheckedAt:    time.Now(),
	}, nil
}

// resolveStatusBaseURL determines the status check URL from config.
// "-" disables status checks; empty derives from the deploy URL.
func resolveStatusBaseURL(statusBaseURL, deployURL string) string {
	if statusBaseURL == "-" {
		return ""
	}
	if statusBaseURL != "" {
		return strings.TrimRight(statusBaseURL, "/")
	}
	// Auto-derive: replace the deploy URL's path with /status/workflows
	if deployURL == "" {
		return ""
	}
	u, err := neturl.Parse(deployURL)
	if err != nil || u.Host == "" {
		return ""
	}
	u.Path = "/status/workflows"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

// Ensure WebhookDeployTarget implements driven.DeployTarget.
var _ driven.DeployTarget = (*WebhookDeployTarget)(nil)

// Ensure WebhookDeployTarget implements driven.DeployStatusChecker.
var _ driven.DeployStatusChecker = (*WebhookDeployTarget)(nil)
