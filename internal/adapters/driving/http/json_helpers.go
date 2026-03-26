package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"graphiti/internal/domain"
)

// maxRequestBodySize is the maximum allowed request body size (1 MB).
const maxRequestBodySize = 1 << 20

// wantsJSON returns true when the client prefers a JSON response.
// Checks Accept header for "application/json".
func wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

// apiError is the standard JSON error envelope for all API error responses.
type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// apiErrorResponse wraps apiError for the top-level JSON response.
type apiErrorResponse struct {
	Error apiError `json:"error"`
}

// writeJSONError writes a JSON error response with the given status code.
func writeJSONError(w http.ResponseWriter, status int, message string, detail ...string) {
	resp := apiErrorResponse{
		Error: apiError{
			Code:    status,
			Message: message,
		},
	}
	if len(detail) > 0 && detail[0] != "" {
		resp.Error.Detail = detail[0]
	}
	writeJSON(w, status, resp)
}

// decodeJSONBody reads and decodes a JSON request body with a size limit.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// isNotFoundError returns true if the error indicates a resource was not found.
func isNotFoundError(err error) bool {
	return errors.Is(err, domain.ErrNotFound) || strings.Contains(err.Error(), "not found")
}

// maskSecretAttributes replaces secret attribute values with "***".
func maskSecretAttributes(attrs map[string]any, def *domain.NodeDefinition) map[string]any {
	if def == nil {
		return attrs
	}
	for _, a := range def.Attributes {
		if a.Type == domain.AttrTypeSecret {
			if _, ok := attrs[a.ID]; ok {
				attrs[a.ID] = "***"
			}
		}
	}
	return attrs
}
