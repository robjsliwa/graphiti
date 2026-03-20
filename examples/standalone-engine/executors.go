package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// NodeExecutor defines how a node type processes data.
type NodeExecutor interface {
	Execute(node *GraphNode, input map[string]any) (map[string]any, error)
}

// GetNodeExecutor returns the appropriate executor for a node definition type.
func GetNodeExecutor(definitionID string) NodeExecutor {
	switch definitionID {
	case "source-api-gateway":
		return &APIGatewayExecutor{}
	case "control-http-router":
		return &HTTPRouterExecutor{}
	case "processing-validate-payload":
		return &ValidatePayloadExecutor{}
	case "destination-postgresql":
		return &PostgreSQLExecutor{}
	case "destination-http-response":
		return &HTTPResponseExecutor{}
	case "processing-json-transform":
		return &JSONTransformExecutor{}
	default:
		return &PassthroughExecutor{}
	}
}

// APIGatewayExecutor passes through the incoming HTTP request data.
type APIGatewayExecutor struct{}

func (e *APIGatewayExecutor) Execute(node *GraphNode, input map[string]any) (map[string]any, error) {
	// The API gateway node just passes through the request data it receives.
	// The actual HTTP serving is handled by the engine's HTTP server.
	return input, nil
}

// HTTPRouterExecutor routes data based on HTTP method.
type HTTPRouterExecutor struct{}

func (e *HTTPRouterExecutor) Execute(node *GraphNode, input map[string]any) (map[string]any, error) {
	method, _ := input["method"].(string)
	method = strings.ToUpper(method)

	idParam := "id"
	if v, ok := node.AttributeValues["id_param"].(string); ok && v != "" {
		idParam = v
	}

	// Check if there's a path param for the ID
	hasID := false
	if pathParams, ok := input["pathParams"].(map[string]any); ok {
		if _, exists := pathParams[idParam]; exists {
			hasID = true
		}
	}

	var routedPort string
	switch method {
	case "GET":
		if hasID {
			routedPort = "out-get-by-id"
		} else {
			routedPort = "out-get"
		}
	case "POST":
		routedPort = "out-post"
	case "PUT", "PATCH":
		routedPort = "out-put"
	case "DELETE":
		routedPort = "out-delete"
	default:
		routedPort = "out-unmatched"
	}

	result := make(map[string]any)
	for k, v := range input {
		result[k] = v
	}
	result["_routedPort"] = routedPort
	return result, nil
}

// ValidatePayloadExecutor validates request body fields.
type ValidatePayloadExecutor struct{}

func (e *ValidatePayloadExecutor) Execute(node *GraphNode, input map[string]any) (map[string]any, error) {
	requiredStr, _ := node.AttributeValues["required_fields"].(string)

	result := make(map[string]any)
	for k, v := range input {
		result[k] = v
	}

	if requiredStr == "" {
		result["_routedPort"] = "out-main"
		return result, nil
	}

	body, _ := input["body"].(map[string]any)
	if body == nil {
		body = make(map[string]any)
	}

	required := strings.Split(requiredStr, ",")
	var missing []string
	for _, field := range required {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if _, ok := body[field]; !ok {
			missing = append(missing, field)
		}
	}

	if len(missing) > 0 {
		result["_routedPort"] = "out-error"
		result["_validationError"] = fmt.Sprintf("missing required fields: %s", strings.Join(missing, ", "))
		return result, nil
	}

	result["_routedPort"] = "out-main"
	return result, nil
}

// PostgreSQLExecutor executes SQL queries against PostgreSQL.
type PostgreSQLExecutor struct {
	DB *sql.DB // Set by the engine when the executor is used
}

func (e *PostgreSQLExecutor) Execute(node *GraphNode, input map[string]any) (map[string]any, error) {
	operation, _ := node.AttributeValues["operation"].(string)
	sqlQuery, _ := node.AttributeValues["sql"].(string)
	paramsJSON, _ := node.AttributeValues["params"].(string)

	if sqlQuery == "" {
		return nil, fmt.Errorf("SQL query is empty")
	}

	// Parse parameter mapping
	var paramPaths []string
	if paramsJSON != "" {
		if err := json.Unmarshal([]byte(paramsJSON), &paramPaths); err != nil {
			return nil, fmt.Errorf("invalid params JSON: %w", err)
		}
	}

	// Resolve parameter values from input
	args := make([]any, len(paramPaths))
	for i, path := range paramPaths {
		args[i] = resolveJSONPath(input, path)
	}

	if e.DB == nil {
		return nil, fmt.Errorf("no database connection configured")
	}

	result := make(map[string]any)
	for k, v := range input {
		result[k] = v
	}

	switch operation {
	case "query":
		rows, err := e.DB.Query(sqlQuery, args...)
		if err != nil {
			result["_routedPort"] = "out-error"
			result["_error"] = err.Error()
			return result, nil
		}
		defer rows.Close()

		records, err := scanRows(rows)
		if err != nil {
			result["_routedPort"] = "out-error"
			result["_error"] = err.Error()
			return result, nil
		}
		result["rows"] = records
		result["_routedPort"] = "out-main"

	case "query-row":
		rows, err := e.DB.Query(sqlQuery, args...)
		if err != nil {
			result["_routedPort"] = "out-error"
			result["_error"] = err.Error()
			return result, nil
		}
		defer rows.Close()

		records, err := scanRows(rows)
		if err != nil {
			result["_routedPort"] = "out-error"
			result["_error"] = err.Error()
			return result, nil
		}
		if len(records) > 0 {
			result["row"] = records[0]
		} else {
			result["row"] = nil
		}
		result["_routedPort"] = "out-main"

	case "exec":
		res, err := e.DB.Exec(sqlQuery, args...)
		if err != nil {
			result["_routedPort"] = "out-error"
			result["_error"] = err.Error()
			return result, nil
		}
		affected, _ := res.RowsAffected()
		lastID, _ := res.LastInsertId()
		result["rowsAffected"] = affected
		result["lastInsertId"] = lastID
		result["_routedPort"] = "out-main"

		// For INSERT RETURNING queries, try to read the returned row
		// PostgreSQL uses RETURNING, which returns rows via Query, not Exec
		// If the SQL contains RETURNING, re-execute as query
		if strings.Contains(strings.ToUpper(sqlQuery), "RETURNING") {
			rows, err := e.DB.Query(sqlQuery, args...)
			if err == nil {
				defer rows.Close()
				records, err := scanRows(rows)
				if err == nil && len(records) > 0 {
					result["row"] = records[0]
					result["rows"] = records
				}
			}
		}

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	return result, nil
}

// scanRows converts sql.Rows to a slice of maps.
func scanRows(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var records []map[string]any
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		record := make(map[string]any, len(cols))
		for i, col := range cols {
			val := values[i]
			// Convert []byte to string for readability
			if b, ok := val.([]byte); ok {
				record[col] = string(b)
			} else {
				record[col] = val
			}
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

// HTTPResponseExecutor formats data into an HTTP response shape.
type HTTPResponseExecutor struct{}

func (e *HTTPResponseExecutor) Execute(node *GraphNode, input map[string]any) (map[string]any, error) {
	statusCode := 200
	if v, ok := node.AttributeValues["status_code"].(float64); ok {
		statusCode = int(v)
	}

	contentType := "application/json"
	if v, ok := node.AttributeValues["content_type"].(string); ok {
		contentType = v
	}

	result := map[string]any{
		"statusCode":  statusCode,
		"contentType": contentType,
		"body":        input,
	}

	// If there's a single row, use it as the response body
	if row, ok := input["row"]; ok {
		result["body"] = row
	} else if rows, ok := input["rows"]; ok {
		result["body"] = rows
	}

	// If there's a validation error, format it
	if errMsg, ok := input["_validationError"].(string); ok {
		result["body"] = map[string]any{"error": errMsg}
		if statusCode == 200 {
			result["statusCode"] = 400
		}
	}

	return result, nil
}

// JSONTransformExecutor reshapes JSON data using a mapping template.
type JSONTransformExecutor struct{}

func (e *JSONTransformExecutor) Execute(node *GraphNode, input map[string]any) (map[string]any, error) {
	mappingStr, _ := node.AttributeValues["mapping"].(string)
	mode, _ := node.AttributeValues["mode"].(string)
	if mode == "" {
		mode = "merge"
	}

	var mapping map[string]string
	if mappingStr != "" {
		if err := json.Unmarshal([]byte(mappingStr), &mapping); err != nil {
			return nil, fmt.Errorf("invalid mapping JSON: %w", err)
		}
	}

	var result map[string]any
	if mode == "merge" {
		result = make(map[string]any)
		for k, v := range input {
			result[k] = v
		}
	} else {
		result = make(map[string]any)
	}

	for outputField, inputPath := range mapping {
		result[outputField] = resolveJSONPath(input, inputPath)
	}

	return result, nil
}

// PassthroughExecutor passes data through unchanged. Used for unknown node types.
type PassthroughExecutor struct{}

func (e *PassthroughExecutor) Execute(node *GraphNode, input map[string]any) (map[string]any, error) {
	slog.Warn("using passthrough executor for unknown node type", "definitionId", node.DefinitionID)
	return input, nil
}

// resolveJSONPath extracts a value from nested maps using dot notation.
// e.g. "body.title" from {"body": {"title": "Hello"}} returns "Hello"
func resolveJSONPath(data map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[part]
	}

	return current
}

// computeHMAC creates an HMAC-SHA256 signature for the given body.
func computeHMAC(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
