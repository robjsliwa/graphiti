package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Runner executes deployed workflows. It receives HTTP requests,
// matches them to a deployed workflow's API Gateway node, and
// executes the workflow graph in topological order.
type Runner struct {
	mu          sync.RWMutex
	workflows   map[string]*DeployedWorkflow // workflowID -> deployed workflow
	routes      map[string]*RouteEntry       // "METHOD /path" -> route entry
	db          *sql.DB
	callbackURL string
	hmacSecret  string
	client      *http.Client
}

// DeployedWorkflow holds a deployed workflow and its execution graph.
type DeployedWorkflow struct {
	Payload DeployPayload
	Graph   *ExecutionGraph
}

// RouteEntry maps an HTTP route to the workflow that handles it.
type RouteEntry struct {
	WorkflowID   string
	GatewayNode  string
	Path         string
	Methods      string
	PathSegments []string // for path parameter matching
}

// NewRunner creates a new workflow runner.
func NewRunner(db *sql.DB, callbackURL, hmacSecret string) *Runner {
	return &Runner{
		workflows:   make(map[string]*DeployedWorkflow),
		routes:      make(map[string]*RouteEntry),
		db:          db,
		callbackURL: callbackURL,
		hmacSecret:  hmacSecret,
		client:      &http.Client{Timeout: 10 * time.Second},
	}
}

// Deploy registers a workflow for execution.
func (r *Runner) Deploy(payload DeployPayload) error {
	graph := BuildExecutionGraph(payload.Workflow.Definition)
	if err := graph.Validate(); err != nil {
		return fmt.Errorf("invalid workflow: %w", err)
	}

	deployed := &DeployedWorkflow{
		Payload: payload,
		Graph:   graph,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.workflows[payload.Workflow.ID] = deployed

	// Register routes from API Gateway nodes
	for _, nodeID := range graph.FindSourceNodes() {
		node := graph.Nodes[nodeID]
		if node.DefinitionID != "source-api-gateway" {
			continue
		}

		path, _ := node.AttributeValues["path"].(string)
		methods, _ := node.AttributeValues["methods"].(string)
		if path == "" {
			path = "/api/resource"
		}
		if methods == "" {
			methods = "ALL"
		}

		entry := &RouteEntry{
			WorkflowID:   payload.Workflow.ID,
			GatewayNode:  nodeID,
			Path:         path,
			Methods:      methods,
			PathSegments: strings.Split(strings.TrimPrefix(path, "/"), "/"),
		}

		// Register both exact and parameterized routes
		r.routes[path] = entry
	}

	slog.Info("workflow deployed", "id", payload.Workflow.ID, "name", payload.Workflow.Name, "version", payload.Workflow.Version)
	return nil
}

// MatchRoute finds the route entry and extracted path params for an incoming request.
func (r *Runner) MatchRoute(method, path string) (*RouteEntry, map[string]any) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try exact match first
	if entry, ok := r.routes[path]; ok {
		if entry.Methods == "ALL" || strings.Contains(entry.Methods, method) {
			return entry, nil
		}
	}

	// Try parameterized matching
	reqSegments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for _, entry := range r.routes {
		if entry.Methods != "ALL" && !strings.Contains(entry.Methods, method) {
			continue
		}
		params := matchPathSegments(entry.PathSegments, reqSegments)
		if params != nil {
			return entry, params
		}
	}

	return nil, nil
}

// matchPathSegments matches request path segments against a pattern with :param placeholders.
func matchPathSegments(pattern, request []string) map[string]any {
	// Allow request to be longer (extra segments become params)
	if len(request) < len(pattern) {
		return nil
	}

	params := make(map[string]any)
	for i, seg := range pattern {
		if strings.HasPrefix(seg, ":") {
			params[seg[1:]] = request[i]
		} else if seg != request[i] {
			return nil
		}
	}

	// If request has extra segments beyond pattern, treat as ID
	if len(request) > len(pattern) && len(request) == len(pattern)+1 {
		params["id"] = request[len(pattern)]
	} else if len(request) > len(pattern)+1 {
		return nil // too many extra segments
	}

	return params
}

// ExecuteWorkflow runs a workflow for an incoming HTTP request and returns the response.
func (r *Runner) ExecuteWorkflow(ctx context.Context, entry *RouteEntry, req *http.Request, pathParams map[string]any, body map[string]any) (map[string]any, error) {
	r.mu.RLock()
	deployed, ok := r.workflows[entry.WorkflowID]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("workflow %s not found", entry.WorkflowID)
	}

	runID := generateID()
	graph := deployed.Graph
	workflowID := deployed.Payload.Workflow.ID

	// Build initial request data
	requestData := map[string]any{
		"method":      req.Method,
		"path":        req.URL.Path,
		"pathParams":  pathParams,
		"queryParams": flattenQueryParams(req.URL.Query()),
		"headers":     flattenHeaders(req.Header),
		"body":        body,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	// Get topological order
	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("topological sort: %w", err)
	}

	// Track node outputs for passing data downstream
	nodeOutputs := make(map[string]map[string]any)
	nodeOutputs[entry.GatewayNode] = requestData

	var finalResult map[string]any

	for _, nodeID := range sorted {
		node := graph.Nodes[nodeID]

		// Determine input for this node
		var input map[string]any
		if nodeID == entry.GatewayNode {
			input = requestData
		} else {
			// Merge inputs from all incoming edges
			input = make(map[string]any)
			for _, edge := range node.Incoming {
				if upstream, ok := nodeOutputs[edge.FromNodeID]; ok {
					// Check if the upstream node routed to a specific port
					if routedPort, ok := upstream["_routedPort"].(string); ok {
						if routedPort != edge.FromPortID {
							continue // This edge's port wasn't the routed output
						}
					}
					for k, v := range upstream {
						if !strings.HasPrefix(k, "_") {
							input[k] = v
						}
					}
				}
			}
			if len(input) == 0 {
				// No data reached this node (conditional routing bypassed it)
				continue
			}
		}

		// Send "running" callback
		r.sendCallback(workflowID, runID, nodeID, "running", time.Now(), time.Time{}, "", nil)

		// Execute the node
		startTime := time.Now()
		executor := r.getExecutor(node)
		output, err := executor.Execute(node, input)
		endTime := time.Now()

		if err != nil {
			r.sendCallback(workflowID, runID, nodeID, "failed", startTime, endTime, err.Error(), nil)
			return nil, fmt.Errorf("node %s (%s) failed: %w", nodeID, node.Label, err)
		}

		// Send "completed" callback
		summary := makeSummary(output)
		r.sendCallback(workflowID, runID, nodeID, "completed", startTime, endTime, "", summary)

		nodeOutputs[nodeID] = output

		// If this is an HTTP Response node (terminal), capture the result
		if node.DefinitionID == "destination-http-response" {
			finalResult = output
		}
	}

	if finalResult == nil {
		finalResult = map[string]any{
			"statusCode":  500,
			"contentType": "application/json",
			"body":        map[string]any{"error": "no response node reached"},
		}
	}

	return finalResult, nil
}

// getExecutor returns the node executor, injecting the DB connection for PostgreSQL nodes.
func (r *Runner) getExecutor(node *GraphNode) NodeExecutor {
	executor := GetNodeExecutor(node.DefinitionID)
	if pgExec, ok := executor.(*PostgreSQLExecutor); ok {
		pgExec.DB = r.db
	}
	return executor
}

// sendCallback sends a status update to the Graphiti server.
func (r *Runner) sendCallback(workflowID, runID, nodeID, status string, started, completed time.Time, errMsg string, summary map[string]any) {
	if r.callbackURL == "" {
		return
	}

	cb := StatusCallback{
		APIVersion:    "graphiti/v1",
		Event:         "node.status",
		RunID:         runID,
		WorkflowID:    workflowID,
		NodeID:        nodeID,
		Status:        status,
		StartedAt:     started,
		ErrorMessage:  errMsg,
		OutputSummary: summary,
	}
	if !completed.IsZero() {
		cb.CompletedAt = completed
	}

	body, err := json.Marshal(cb)
	if err != nil {
		slog.Error("marshal callback failed", "error", err)
		return
	}

	req, err := http.NewRequest("POST", r.callbackURL, bytes.NewReader(body))
	if err != nil {
		slog.Error("create callback request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	if r.hmacSecret != "" {
		sig := computeHMAC(body, r.hmacSecret)
		req.Header.Set("X-Graphiti-Signature", "sha256="+sig)
	}

	go func() {
		resp, err := r.client.Do(req)
		if err != nil {
			slog.Error("callback request failed", "error", err, "nodeID", nodeID, "status", status)
			return
		}
		resp.Body.Close()
	}()

	// Small delay to let Graphiti process the status update for visual effect
	time.Sleep(50 * time.Millisecond)
}

// makeSummary creates a small summary of the output for the callback.
func makeSummary(output map[string]any) map[string]any {
	summary := make(map[string]any)
	if rows, ok := output["rows"].([]map[string]any); ok {
		summary["rowCount"] = len(rows)
	}
	if _, ok := output["row"]; ok {
		summary["hasRow"] = true
	}
	if rp, ok := output["_routedPort"].(string); ok {
		summary["routedPort"] = rp
	}
	if sc, ok := output["statusCode"]; ok {
		summary["statusCode"] = sc
	}
	return summary
}

func flattenQueryParams(params map[string][]string) map[string]any {
	result := make(map[string]any, len(params))
	for k, v := range params {
		if len(v) == 1 {
			result[k] = v[0]
		} else {
			result[k] = v
		}
	}
	return result
}

func flattenHeaders(headers http.Header) map[string]any {
	result := make(map[string]any, len(headers))
	for k, v := range headers {
		if len(v) == 1 {
			result[strings.ToLower(k)] = v[0]
		} else {
			result[strings.ToLower(k)] = v
		}
	}
	return result
}
