package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWantsJSON(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   bool
	}{
		{"exact json", "application/json", true},
		{"json with html", "application/json, text/html", true},
		{"wildcard with json", "*/*, application/json", true},
		{"html only", "text/html", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.accept != "" {
				r.Header.Set("Accept", tt.accept)
			}
			got := wantsJSON(r)
			if got != tt.want {
				t.Errorf("wantsJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWantsJSON_NoHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if wantsJSON(r) {
		t.Error("wantsJSON() should return false when no Accept header is set")
	}
}

func TestWriteJSONError_Basic(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSONError(w, http.StatusNotFound, "not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var resp apiErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Error.Code != 404 {
		t.Errorf("error code = %d, want 404", resp.Error.Code)
	}
	if resp.Error.Message != "not found" {
		t.Errorf("error message = %q, want %q", resp.Error.Message, "not found")
	}
	if resp.Error.Detail != "" {
		t.Errorf("error detail = %q, want empty", resp.Error.Detail)
	}
}

func TestWriteJSONError_WithDetail(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSONError(w, http.StatusBadRequest, "bad request", "name is required")

	var resp apiErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Error.Code != 400 {
		t.Errorf("error code = %d, want 400", resp.Error.Code)
	}
	if resp.Error.Message != "bad request" {
		t.Errorf("error message = %q, want %q", resp.Error.Message, "bad request")
	}
	if resp.Error.Detail != "name is required" {
		t.Errorf("error detail = %q, want %q", resp.Error.Detail, "name is required")
	}
}

func TestWriteJSONError_EmptyDetail(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSONError(w, http.StatusInternalServerError, "internal error", "")

	var resp apiErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Error.Detail != "" {
		t.Errorf("error detail = %q, want empty", resp.Error.Detail)
	}
}
