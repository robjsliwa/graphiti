package domain

import (
	"errors"
	"fmt"
	"time"
)

// WorkflowStatus represents the lifecycle state of a workflow.
type WorkflowStatus string

const (
	WorkflowStatusDraft    WorkflowStatus = "draft"
	WorkflowStatusDeployed WorkflowStatus = "deployed"
	WorkflowStatusArchived WorkflowStatus = "archived"
)

// Workflow is the aggregate root for the canvas. It contains all nodes and edges.
type Workflow struct {
	ID          string
	Name        string
	Description string
	Status      WorkflowStatus
	Version     int
	Nodes       []NodeInstance
	Edges       []Edge
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WorkflowVersion is a snapshot of a workflow at deploy time.
type WorkflowVersion struct {
	ID         string
	WorkflowID string
	Version    int
	Definition []byte // JSON snapshot
	DeployedAt time.Time
	DeployedBy string
	CreatedAt  time.Time
}

// WorkflowSummary is a lightweight projection for listing.
type WorkflowSummary struct {
	ID        string
	Name      string
	Status    WorkflowStatus
	Version   int
	UpdatedAt time.Time
}

// DeployPayload is the data sent to the webhook when deploying.
type DeployPayload struct {
	APIVersion      string         `json:"apiVersion"`
	Event           string         `json:"event"`
	Timestamp       time.Time      `json:"timestamp"`
	Deployment      DeploymentInfo `json:"deployment"`
	Workflow        WorkflowInfo   `json:"workflow"`
	PreviousVersion int            `json:"previousVersion"`
	Checksum        string         `json:"checksum"`
}

// DeploymentInfo describes the deploy action.
type DeploymentInfo struct {
	ID          string       `json:"id"`
	Target      string       `json:"target"`
	TriggeredBy TriggerUser  `json:"triggeredBy"`
}

// TriggerUser identifies who triggered a deploy.
type TriggerUser struct {
	UserID   string `json:"userID"`
	Username string `json:"username"`
}

// WorkflowInfo is the workflow data included in a deploy payload.
type WorkflowInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Version    int    `json:"version"`
	Definition any    `json:"definition"`
}

// DeployResult is the outcome of a deploy operation.
type DeployResult struct {
	Success          bool
	RunID            string
	Message          string
	ValidationErrors []ValidationError
}

// DeployVerification represents the status of a deployment verification check.
type DeployVerification string

const (
	DeployVerificationUnknown  DeployVerification = "unknown"  // Engine doesn't support status checks
	DeployVerificationVerified DeployVerification = "verified" // Engine confirms workflow is deployed
	DeployVerificationMissing  DeployVerification = "missing"  // Engine says workflow is NOT deployed
	DeployVerificationError    DeployVerification = "error"    // Could not reach engine
)

// DeployStatusResult is the outcome of a deploy status check.
type DeployStatusResult struct {
	WorkflowID   string
	Version      int
	Verification DeployVerification
	Message      string
	CheckedAt    time.Time
}

var (
	ErrNodeNotFound      = errors.New("node not found")
	ErrEdgeNotFound      = errors.New("edge not found")
	ErrPortNotFound      = errors.New("port not found")
	ErrPortTypeMismatch  = errors.New("port types do not match")
	ErrMaxConnections    = errors.New("port has reached maximum connections")
	ErrCycleDetected     = errors.New("adding this edge would create a cycle")
	ErrDuplicateEdge     = errors.New("edge already exists between these ports")
	ErrSelfReference     = errors.New("cannot connect a node to itself")
	ErrConnectionRule    = errors.New("connection not allowed by node definition rules")
	ErrDuplicateNodeID   = errors.New("duplicate node ID")
)

// AddNode creates a new node instance on this workflow at the given position.
func (w *Workflow) AddNode(def *NodeDefinition, x, y float64, instanceID string) *NodeInstance {
	attrs := make(map[string]any)
	for _, a := range def.Attributes {
		if a.Default != nil {
			attrs[a.ID] = a.Default
		}
	}

	node := NodeInstance{
		ID:              instanceID,
		DefinitionID:    def.ID,
		Label:           def.Name,
		X:               x,
		Y:               y,
		AttributeValues: attrs,
		Definition:      def,
	}
	w.Nodes = append(w.Nodes, node)
	return &w.Nodes[len(w.Nodes)-1]
}

// RemoveNode removes a node and all its connected edges.
// Returns the removed node and edges for undo.
func (w *Workflow) RemoveNode(nodeID string) (*NodeInstance, []Edge, error) {
	idx := -1
	for i := range w.Nodes {
		if w.Nodes[i].ID == nodeID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, nil, ErrNodeNotFound
	}

	removed := w.Nodes[idx]
	w.Nodes = append(w.Nodes[:idx], w.Nodes[idx+1:]...)

	// Remove all connected edges
	var removedEdges []Edge
	remaining := make([]Edge, 0, len(w.Edges))
	for _, e := range w.Edges {
		if e.SourceNodeID == nodeID || e.TargetNodeID == nodeID {
			removedEdges = append(removedEdges, e)
		} else {
			remaining = append(remaining, e)
		}
	}
	w.Edges = remaining

	return &removed, removedEdges, nil
}

// AddEdge connects two ports after validating compatibility.
func (w *Workflow) AddEdge(sourceNodeID, sourcePortID, targetNodeID, targetPortID, edgeID string) error {
	if sourceNodeID == targetNodeID {
		return ErrSelfReference
	}

	srcNode := w.FindNode(sourceNodeID)
	if srcNode == nil {
		return fmt.Errorf("source %w: %s", ErrNodeNotFound, sourceNodeID)
	}
	tgtNode := w.FindNode(targetNodeID)
	if tgtNode == nil {
		return fmt.Errorf("target %w: %s", ErrNodeNotFound, targetNodeID)
	}

	srcPort := findOutputPort(srcNode, sourcePortID)
	if srcPort == nil {
		return fmt.Errorf("source %w: %s", ErrPortNotFound, sourcePortID)
	}
	tgtPort := findInputPort(tgtNode, targetPortID)
	if tgtPort == nil {
		return fmt.Errorf("target %w: %s", ErrPortNotFound, targetPortID)
	}

	// Port type matching
	if srcPort.Type != tgtPort.Type {
		return fmt.Errorf("%w: %s -> %s", ErrPortTypeMismatch, srcPort.Type, tgtPort.Type)
	}

	// Duplicate check
	for _, e := range w.Edges {
		if e.SourceNodeID == sourceNodeID && e.SourcePortID == sourcePortID &&
			e.TargetNodeID == targetNodeID && e.TargetPortID == targetPortID {
			return ErrDuplicateEdge
		}
	}

	// Max connections check on target port
	if tgtPort.MaxConnections > 0 {
		count := 0
		for _, e := range w.Edges {
			if e.TargetNodeID == targetNodeID && e.TargetPortID == targetPortID {
				count++
			}
		}
		if count >= tgtPort.MaxConnections {
			return ErrMaxConnections
		}
	}

	// Max connections check on source port
	if srcPort.MaxConnections > 0 {
		count := 0
		for _, e := range w.Edges {
			if e.SourceNodeID == sourceNodeID && e.SourcePortID == sourcePortID {
				count++
			}
		}
		if count >= srcPort.MaxConnections {
			return ErrMaxConnections
		}
	}

	// Connection rules from source node definition
	if srcNode.Definition != nil {
		for _, rule := range srcNode.Definition.Validation.ConnectionRules {
			if rule.OutputPort == sourcePortID {
				if err := checkConnectionRule(rule, tgtNode, tgtPort); err != nil {
					return err
				}
			}
		}
	}

	// Cycle detection: temporarily add edge and check
	w.Edges = append(w.Edges, Edge{
		ID:           edgeID,
		SourceNodeID: sourceNodeID,
		SourcePortID: sourcePortID,
		TargetNodeID: targetNodeID,
		TargetPortID: targetPortID,
	})

	if w.hasCycle() {
		w.Edges = w.Edges[:len(w.Edges)-1]
		return ErrCycleDetected
	}

	return nil
}

// RemoveEdge disconnects two ports.
func (w *Workflow) RemoveEdge(edgeID string) (*Edge, error) {
	for i, e := range w.Edges {
		if e.ID == edgeID {
			removed := w.Edges[i]
			w.Edges = append(w.Edges[:i], w.Edges[i+1:]...)
			return &removed, nil
		}
	}
	return nil, ErrEdgeNotFound
}

// FindNode returns the node with the given ID or nil.
func (w *Workflow) FindNode(nodeID string) *NodeInstance {
	for i := range w.Nodes {
		if w.Nodes[i].ID == nodeID {
			return &w.Nodes[i]
		}
	}
	return nil
}

// FindEdge returns the edge with the given ID or nil.
func (w *Workflow) FindEdge(edgeID string) *Edge {
	for i := range w.Edges {
		if w.Edges[i].ID == edgeID {
			return &w.Edges[i]
		}
	}
	return nil
}

// Validate checks all invariants and returns any errors.
func (w *Workflow) Validate() []ValidationError {
	var errs []ValidationError

	// Check for duplicate node IDs
	seen := make(map[string]bool)
	for _, n := range w.Nodes {
		if seen[n.ID] {
			errs = append(errs, ValidationError{
				NodeID:  n.ID,
				Message: "duplicate node ID",
			})
		}
		seen[n.ID] = true
	}

	// Check required attributes
	for _, n := range w.Nodes {
		if n.Definition == nil {
			continue
		}
		for _, attr := range n.Definition.Attributes {
			if !attr.Required {
				continue
			}
			val, ok := n.AttributeValues[attr.ID]
			if !ok || val == nil || val == "" {
				errs = append(errs, ValidationError{
					NodeID:  n.ID,
					Field:   attr.ID,
					Message: fmt.Sprintf("required attribute %q is missing", attr.Label),
				})
			}
		}
	}

	// Check edges reference valid nodes and ports
	for _, e := range w.Edges {
		srcNode := w.FindNode(e.SourceNodeID)
		if srcNode == nil {
			errs = append(errs, ValidationError{
				Field:   "edge:" + e.ID,
				Message: fmt.Sprintf("source node %s not found", e.SourceNodeID),
			})
			continue
		}
		tgtNode := w.FindNode(e.TargetNodeID)
		if tgtNode == nil {
			errs = append(errs, ValidationError{
				Field:   "edge:" + e.ID,
				Message: fmt.Sprintf("target node %s not found", e.TargetNodeID),
			})
			continue
		}

		srcPort := findOutputPort(srcNode, e.SourcePortID)
		tgtPort := findInputPort(tgtNode, e.TargetPortID)
		if srcPort != nil && tgtPort != nil && srcPort.Type != tgtPort.Type {
			errs = append(errs, ValidationError{
				Field:   "edge:" + e.ID,
				Message: fmt.Sprintf("port type mismatch: %s -> %s", srcPort.Type, tgtPort.Type),
			})
		}

		// Validate connection rules
		if srcNode.Definition != nil && srcPort != nil {
			for _, rule := range srcNode.Definition.Validation.ConnectionRules {
				if rule.OutputPort == e.SourcePortID {
					if err := checkConnectionRule(rule, tgtNode, tgtPort); err != nil {
						errs = append(errs, ValidationError{
							Field:   "edge:" + e.ID,
							Message: err.Error(),
						})
					}
				}
			}
		}
	}

	// Check for cycles
	if w.hasCycle() {
		errs = append(errs, ValidationError{
			Message: "workflow contains a cycle",
		})
	}

	return errs
}

// RestoreNode adds a node back to the workflow (used by undo).
func (w *Workflow) RestoreNode(node NodeInstance) {
	w.Nodes = append(w.Nodes, node)
}

// RestoreEdge adds an edge back to the workflow (used by undo).
func (w *Workflow) RestoreEdge(edge Edge) {
	w.Edges = append(w.Edges, edge)
}

// hasCycle uses Kahn's algorithm to detect cycles in the node graph.
func (w *Workflow) hasCycle() bool {
	if len(w.Nodes) == 0 {
		return false
	}

	// Build adjacency and in-degree
	inDegree := make(map[string]int)
	adj := make(map[string][]string)
	for _, n := range w.Nodes {
		inDegree[n.ID] = 0
	}
	for _, e := range w.Edges {
		adj[e.SourceNodeID] = append(adj[e.SourceNodeID], e.TargetNodeID)
		inDegree[e.TargetNodeID]++
	}

	// Seed queue with zero-degree nodes
	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	visited := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		visited++
		for _, neighbor := range adj[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	return visited != len(w.Nodes)
}

func findOutputPort(node *NodeInstance, portID string) *PortDefinition {
	if node.Definition == nil {
		return nil
	}
	for i := range node.Definition.Outputs {
		if node.Definition.Outputs[i].ID == portID {
			return &node.Definition.Outputs[i]
		}
	}
	return nil
}

func findInputPort(node *NodeInstance, portID string) *PortDefinition {
	if node.Definition == nil {
		return nil
	}
	for i := range node.Definition.Inputs {
		if node.Definition.Inputs[i].ID == portID {
			return &node.Definition.Inputs[i]
		}
	}
	return nil
}

func checkConnectionRule(rule ConnectionRule, tgtNode *NodeInstance, tgtPort *PortDefinition) error {
	if len(rule.AllowedTargetCategories) > 0 && tgtNode.Definition != nil {
		allowed := false
		for _, cat := range rule.AllowedTargetCategories {
			if tgtNode.Definition.Category.Group == cat {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("%w: target category %q not in allowed list %v",
				ErrConnectionRule, tgtNode.Definition.Category.Group, rule.AllowedTargetCategories)
		}
	}

	if len(rule.AllowedTargetPorts) > 0 && tgtPort != nil {
		allowed := false
		for _, pt := range rule.AllowedTargetPorts {
			if string(tgtPort.Type) == pt {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("%w: target port type %q not in allowed list %v",
				ErrConnectionRule, tgtPort.Type, rule.AllowedTargetPorts)
		}
	}

	return nil
}
