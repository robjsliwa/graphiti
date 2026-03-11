package http

import (
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
