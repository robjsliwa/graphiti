package main

import (
	"errors"
	"fmt"
)

// ExecutionGraph represents the workflow as a directed graph for execution.
type ExecutionGraph struct {
	Nodes map[string]*GraphNode
}

// GraphNode is a node in the execution graph with its connectivity info.
type GraphNode struct {
	NodeDef
	Incoming      []GraphEdge
	OutgoingEdges map[string][]GraphEdge // port ID -> edges from that port
}

// GraphEdge represents a connection between two node ports.
type GraphEdge struct {
	EdgeID     string
	FromNodeID string
	FromPortID string
	ToNodeID   string
	ToPortID   string
}

// BuildExecutionGraph creates an execution graph from a workflow definition.
func BuildExecutionGraph(def WorkflowDefinition) *ExecutionGraph {
	graph := &ExecutionGraph{
		Nodes: make(map[string]*GraphNode, len(def.Nodes)),
	}

	for _, n := range def.Nodes {
		graph.Nodes[n.ID] = &GraphNode{
			NodeDef:       n,
			OutgoingEdges: make(map[string][]GraphEdge),
		}
	}

	for _, e := range def.Edges {
		edge := GraphEdge{
			EdgeID:     e.ID,
			FromNodeID: e.SourceNodeID,
			FromPortID: e.SourcePortID,
			ToNodeID:   e.TargetNodeID,
			ToPortID:   e.TargetPortID,
		}

		if src, ok := graph.Nodes[e.SourceNodeID]; ok {
			src.OutgoingEdges[e.SourcePortID] = append(src.OutgoingEdges[e.SourcePortID], edge)
		}
		if tgt, ok := graph.Nodes[e.TargetNodeID]; ok {
			tgt.Incoming = append(tgt.Incoming, edge)
		}
	}

	return graph
}

// TopologicalSort returns node IDs in execution order using Kahn's algorithm.
func (g *ExecutionGraph) TopologicalSort() ([]string, error) {
	inDegree := make(map[string]int, len(g.Nodes))
	for id := range g.Nodes {
		inDegree[id] = len(g.Nodes[id].Incoming)
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		node := g.Nodes[current]
		for _, edges := range node.OutgoingEdges {
			for _, e := range edges {
				inDegree[e.ToNodeID]--
				if inDegree[e.ToNodeID] == 0 {
					queue = append(queue, e.ToNodeID)
				}
			}
		}
	}

	if len(sorted) != len(g.Nodes) {
		return nil, errors.New("cycle detected in workflow graph")
	}

	return sorted, nil
}

// FindSourceNodes returns IDs of nodes with no incoming edges.
func (g *ExecutionGraph) FindSourceNodes() []string {
	var sources []string
	for id, node := range g.Nodes {
		if len(node.Incoming) == 0 {
			sources = append(sources, id)
		}
	}
	return sources
}

// GetDownstreamNodes returns the node IDs connected to the given node's output port.
func (g *ExecutionGraph) GetDownstreamNodes(nodeID, portID string) []string {
	node, ok := g.Nodes[nodeID]
	if !ok {
		return nil
	}
	edges := node.OutgoingEdges[portID]
	result := make([]string, 0, len(edges))
	for _, e := range edges {
		result = append(result, e.ToNodeID)
	}
	return result
}

// Validate checks the graph for basic structural issues.
func (g *ExecutionGraph) Validate() error {
	sources := g.FindSourceNodes()
	if len(sources) == 0 {
		return fmt.Errorf("workflow has no source nodes (no nodes without incoming edges)")
	}

	_, err := g.TopologicalSort()
	if err != nil {
		return err
	}

	return nil
}
