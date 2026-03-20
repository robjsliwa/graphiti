package domain

import (
	"fmt"
	"strings"
)

// validateGraphStructure checks workflow-level graph invariants:
// connectivity (single connected component), source/terminal nodes, and cycles.
func (w *Workflow) validateGraphStructure() []ValidationResult {
	var results []ValidationResult

	if len(w.Nodes) == 0 {
		results = append(results, ValidationResult{
			Severity: SeverityWarning,
			Category: CategoryGraph,
			Code:     "EMPTY_WORKFLOW",
			Message:  "This workflow has no nodes. Add at least one node to get started.",
		})
		return results
	}

	// Check connectivity (treat edges as undirected)
	components := w.findConnectedComponents()
	if len(components) > 1 {
		// Find the largest component
		largest := 0
		for i, c := range components {
			if len(c) > len(components[largest]) {
				largest = i
			}
		}
		for i, component := range components {
			if i == largest {
				continue
			}
			for _, nodeID := range component {
				results = append(results, ValidationResult{
					Severity: SeverityError,
					Category: CategoryGraph,
					NodeID:   nodeID,
					Code:     "DISCONNECTED_SUBGRAPH",
					Message:  "This node is not connected to the main flow. Every node must be reachable from a source node. Consider moving disconnected nodes into a separate workflow or connecting them to the main flow.",
				})
			}
		}
	}

	// Must have at least one source node (no incoming edges)
	sources := w.findSourceNodes()
	if len(sources) == 0 {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryGraph,
			Code:     "NO_SOURCE_NODE",
			Message:  "This workflow has no entry point. Add a source node (a node with no inputs) to define where data enters the workflow.",
		})
	}

	// Should have at least one terminal node (no outgoing edges)
	terminals := w.findTerminalNodes()
	if len(terminals) == 0 && len(w.Nodes) > 0 {
		results = append(results, ValidationResult{
			Severity: SeverityWarning,
			Category: CategoryGraph,
			Code:     "NO_TERMINAL_NODE",
			Message:  "This workflow has no endpoint. Every flow should end somewhere. Check if all output ports are connected but the workflow never reaches a final destination.",
		})
	}

	// Cycle detection
	if cycle := w.detectCycle(); cycle != nil {
		nodeNames := make([]string, len(cycle))
		for i, id := range cycle {
			if n := w.FindNode(id); n != nil {
				nodeNames[i] = n.Label
				if nodeNames[i] == "" {
					nodeNames[i] = id
				}
			} else {
				nodeNames[i] = id
			}
		}
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryGraph,
			Code:     "CYCLE_DETECTED",
			Message:  fmt.Sprintf("Circular dependency detected: %s. Workflows must be directed acyclic graphs (no loops). Use a Loop control node if you need iteration.", strings.Join(nodeNames, " → ")),
		})
	}

	return results
}

// findConnectedComponents returns groups of node IDs that are connected.
// Treats edges as undirected for connectivity purposes.
func (w *Workflow) findConnectedComponents() [][]string {
	if len(w.Nodes) == 0 {
		return nil
	}

	// Build undirected adjacency list
	adj := make(map[string][]string)
	for _, node := range w.Nodes {
		adj[node.ID] = []string{}
	}
	for _, edge := range w.Edges {
		adj[edge.SourceNodeID] = append(adj[edge.SourceNodeID], edge.TargetNodeID)
		adj[edge.TargetNodeID] = append(adj[edge.TargetNodeID], edge.SourceNodeID)
	}

	visited := make(map[string]bool)
	var components [][]string

	for _, node := range w.Nodes {
		if visited[node.ID] {
			continue
		}
		// BFS from this node
		var component []string
		queue := []string{node.ID}
		visited[node.ID] = true
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			component = append(component, current)
			for _, neighbor := range adj[current] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
		components = append(components, component)
	}

	return components
}

// findSourceNodes returns node IDs that have no incoming edges.
func (w *Workflow) findSourceNodes() []string {
	hasIncoming := make(map[string]bool)
	for _, edge := range w.Edges {
		hasIncoming[edge.TargetNodeID] = true
	}

	var sources []string
	for _, node := range w.Nodes {
		if !hasIncoming[node.ID] {
			sources = append(sources, node.ID)
		}
	}
	return sources
}

// findTerminalNodes returns node IDs that have no outgoing edges.
func (w *Workflow) findTerminalNodes() []string {
	hasOutgoing := make(map[string]bool)
	for _, edge := range w.Edges {
		hasOutgoing[edge.SourceNodeID] = true
	}

	var terminals []string
	for _, node := range w.Nodes {
		if !hasOutgoing[node.ID] {
			terminals = append(terminals, node.ID)
		}
	}
	return terminals
}

// detectCycle returns the cycle path if one exists, nil otherwise.
// Uses DFS with coloring (white/gray/black).
func (w *Workflow) detectCycle() []string {
	if len(w.Nodes) == 0 {
		return nil
	}

	// Build directed adjacency list
	adj := make(map[string][]string)
	for _, node := range w.Nodes {
		adj[node.ID] = []string{}
	}
	for _, edge := range w.Edges {
		adj[edge.SourceNodeID] = append(adj[edge.SourceNodeID], edge.TargetNodeID)
	}

	const (
		white = 0 // unvisited
		gray  = 1 // in current DFS path
		black = 2 // fully processed
	)

	color := make(map[string]int)
	parent := make(map[string]string)

	var dfs func(node string) []string
	dfs = func(node string) []string {
		color[node] = gray
		for _, neighbor := range adj[node] {
			if color[neighbor] == gray {
				// Found a cycle - reconstruct path
				path := []string{neighbor, node}
				cur := node
				for cur != neighbor {
					cur = parent[cur]
					if cur == "" || cur == neighbor {
						break
					}
					path = append(path, cur)
				}
				// Reverse to get forward order
				for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
					path[i], path[j] = path[j], path[i]
				}
				return path
			}
			if color[neighbor] == white {
				parent[neighbor] = node
				if cycle := dfs(neighbor); cycle != nil {
					return cycle
				}
			}
		}
		color[node] = black
		return nil
	}

	for _, node := range w.Nodes {
		if color[node.ID] == white {
			if cycle := dfs(node.ID); cycle != nil {
				return cycle
			}
		}
	}

	return nil
}
