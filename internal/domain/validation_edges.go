package domain

import "fmt"

// validateEdges checks all edges for structural validity, port compatibility,
// connection limits, and node definition rules.
func (w *Workflow) validateEdges(registry NodeDefinitionRegistry) []ValidationResult {
	var results []ValidationResult

	for _, edge := range w.Edges {
		results = append(results, w.validateSingleEdge(edge, registry)...)
	}

	results = append(results, w.validateNoDuplicateEdges()...)
	results = append(results, w.validateConnectionLimits(registry)...)

	return results
}

// validateSingleEdge checks one edge for structural issues.
func (w *Workflow) validateSingleEdge(edge Edge, registry NodeDefinitionRegistry) []ValidationResult {
	var results []ValidationResult

	// Self-reference check
	if edge.SourceNodeID == edge.TargetNodeID {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryEdge,
			EdgeID:   edge.ID,
			Code:     "EDGE_SELF_REFERENCE",
			Message:  fmt.Sprintf("Edge '%s' connects a node to itself.", edge.ID),
		})
		return results
	}

	// Source node existence
	srcNode := w.FindNode(edge.SourceNodeID)
	if srcNode == nil {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryEdge,
			EdgeID:   edge.ID,
			Code:     "EDGE_MISSING_SOURCE",
			Message:  fmt.Sprintf("Edge '%s' references source node '%s' which does not exist.", edge.ID, edge.SourceNodeID),
		})
		return results
	}

	// Target node existence
	tgtNode := w.FindNode(edge.TargetNodeID)
	if tgtNode == nil {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryEdge,
			EdgeID:   edge.ID,
			Code:     "EDGE_MISSING_TARGET",
			Message:  fmt.Sprintf("Edge '%s' references target node '%s' which does not exist.", edge.ID, edge.TargetNodeID),
		})
		return results
	}

	// Resolve definitions if not already set
	srcDef := srcNode.Definition
	if srcDef == nil && registry != nil {
		srcDef = registry.GetByID(srcNode.DefinitionID)
	}
	tgtDef := tgtNode.Definition
	if tgtDef == nil && registry != nil {
		tgtDef = registry.GetByID(tgtNode.DefinitionID)
	}

	// Check direction: source port must be an output, target port must be an input
	srcPort := findOutputPortFromDef(srcDef, edge.SourcePortID)
	tgtPort := findInputPortFromDef(tgtDef, edge.TargetPortID)

	// If source port is not in outputs, check if it's mistakenly an input (wrong direction)
	if srcPort == nil && srcDef != nil {
		if isInputPort(srcDef, edge.SourcePortID) {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryEdge,
				EdgeID:   edge.ID,
				Code:     "EDGE_WRONG_DIRECTION",
				Message:  fmt.Sprintf("Edge '%s' uses input port '%s' on '%s' as a source. Edges must go from output ports to input ports.", edge.ID, edge.SourcePortID, srcNode.Label),
			})
			return results
		}
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryEdge,
			EdgeID:   edge.ID,
			Code:     "EDGE_MISSING_SOURCE_PORT",
			Message:  fmt.Sprintf("Edge '%s' references source port '%s' which does not exist on node '%s'.", edge.ID, edge.SourcePortID, srcNode.Label),
		})
		return results
	}

	// If target port is not in inputs, check if it's mistakenly an output (wrong direction)
	if tgtPort == nil && tgtDef != nil {
		if isOutputPort(tgtDef, edge.TargetPortID) {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryEdge,
				EdgeID:   edge.ID,
				Code:     "EDGE_WRONG_DIRECTION",
				Message:  fmt.Sprintf("Edge '%s' uses output port '%s' on '%s' as a target. Edges must go from output ports to input ports.", edge.ID, edge.TargetPortID, tgtNode.Label),
			})
			return results
		}
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryEdge,
			EdgeID:   edge.ID,
			Code:     "EDGE_MISSING_TARGET_PORT",
			Message:  fmt.Sprintf("Edge '%s' references target port '%s' which does not exist on node '%s'.", edge.ID, edge.TargetPortID, tgtNode.Label),
		})
		return results
	}

	// Port type matching
	if srcPort != nil && tgtPort != nil && srcPort.Type != tgtPort.Type {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Category: CategoryPort,
			EdgeID:   edge.ID,
			Code:     "PORT_TYPE_MISMATCH",
			Message:  fmt.Sprintf("Edge '%s' connects %s port '%s' on '%s' to %s port '%s' on '%s'. Port types must match.", edge.ID, srcPort.Type, srcPort.ID, srcNode.Label, tgtPort.Type, tgtPort.ID, tgtNode.Label),
		})
	}

	// Connection rules from source node definition
	if srcDef != nil && srcPort != nil {
		for _, rule := range srcDef.Validation.ConnectionRules {
			if rule.OutputPort != edge.SourcePortID {
				continue
			}
			// Check allowed target categories
			if len(rule.AllowedTargetCategories) > 0 && tgtDef != nil {
				allowed := false
				for _, cat := range rule.AllowedTargetCategories {
					if tgtDef.Category.Group == cat {
						allowed = true
						break
					}
				}
				if !allowed {
					results = append(results, ValidationResult{
						Severity: SeverityError,
						Category: CategoryEdge,
						EdgeID:   edge.ID,
						Code:     "CONNECTION_CATEGORY_DENIED",
						Message:  fmt.Sprintf("Node '%s' cannot connect to '%s' (category '%s'). Allowed categories: %v.", srcNode.Label, tgtNode.Label, tgtDef.Category.Group, rule.AllowedTargetCategories),
					})
				}
			}
			// Check allowed target port types
			if len(rule.AllowedTargetPorts) > 0 && tgtPort != nil {
				allowed := false
				for _, pt := range rule.AllowedTargetPorts {
					if string(tgtPort.Type) == pt {
						allowed = true
						break
					}
				}
				if !allowed {
					results = append(results, ValidationResult{
						Severity: SeverityError,
						Category: CategoryEdge,
						EdgeID:   edge.ID,
						Code:     "CONNECTION_PORT_TYPE_DENIED",
						Message:  fmt.Sprintf("Node '%s' output '%s' cannot connect to port type '%s'. Allowed port types: %v.", srcNode.Label, edge.SourcePortID, tgtPort.Type, rule.AllowedTargetPorts),
					})
				}
			}
		}
	}

	return results
}

// validateNoDuplicateEdges checks for edges with identical source and target ports.
func (w *Workflow) validateNoDuplicateEdges() []ValidationResult {
	var results []ValidationResult
	type edgeKey struct {
		srcNode, srcPort, tgtNode, tgtPort string
	}
	seen := make(map[edgeKey]string) // key -> first edge ID

	for _, edge := range w.Edges {
		key := edgeKey{edge.SourceNodeID, edge.SourcePortID, edge.TargetNodeID, edge.TargetPortID}
		if firstID, exists := seen[key]; exists {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryEdge,
				EdgeID:   edge.ID,
				Code:     "EDGE_DUPLICATE",
				Message:  fmt.Sprintf("Edge '%s' is a duplicate of edge '%s'. Two edges cannot connect the same source port to the same target port.", edge.ID, firstID),
			})
		} else {
			seen[key] = edge.ID
		}
	}

	return results
}

// validateConnectionLimits checks that no port exceeds its maxConnections.
func (w *Workflow) validateConnectionLimits(registry NodeDefinitionRegistry) []ValidationResult {
	var results []ValidationResult

	// Count connections per target port
	type portKey struct {
		nodeID, portID string
	}
	targetCounts := make(map[portKey]int)
	sourceCounts := make(map[portKey]int)

	for _, edge := range w.Edges {
		targetCounts[portKey{edge.TargetNodeID, edge.TargetPortID}]++
		sourceCounts[portKey{edge.SourceNodeID, edge.SourcePortID}]++
	}

	// Check target port limits
	for key, count := range targetCounts {
		node := w.FindNode(key.nodeID)
		if node == nil {
			continue
		}
		def := node.Definition
		if def == nil && registry != nil {
			def = registry.GetByID(node.DefinitionID)
		}
		if def == nil {
			continue
		}
		port := findInputPortFromDef(def, key.portID)
		if port == nil {
			continue
		}
		if port.MaxConnections > 0 && count > port.MaxConnections {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryPort,
				NodeID:   key.nodeID,
				Field:    key.portID,
				Code:     "PORT_MAX_CONNECTIONS",
				Message:  fmt.Sprintf("Input port '%s' on node '%s' has %d connections but allows at most %d.", port.Label, node.Label, count, port.MaxConnections),
			})
		}
	}

	// Check source port limits
	for key, count := range sourceCounts {
		node := w.FindNode(key.nodeID)
		if node == nil {
			continue
		}
		def := node.Definition
		if def == nil && registry != nil {
			def = registry.GetByID(node.DefinitionID)
		}
		if def == nil {
			continue
		}
		port := findOutputPortFromDef(def, key.portID)
		if port == nil {
			continue
		}
		if port.MaxConnections > 0 && count > port.MaxConnections {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Category: CategoryPort,
				NodeID:   key.nodeID,
				Field:    key.portID,
				Code:     "PORT_MAX_CONNECTIONS",
				Message:  fmt.Sprintf("Output port '%s' on node '%s' has %d connections but allows at most %d.", port.Label, node.Label, count, port.MaxConnections),
			})
		}
	}

	return results
}

// Helper functions for port lookups using definitions directly.

func findOutputPortFromDef(def *NodeDefinition, portID string) *PortDefinition {
	if def == nil {
		return nil
	}
	for i := range def.Outputs {
		if def.Outputs[i].ID == portID {
			return &def.Outputs[i]
		}
	}
	return nil
}

func findInputPortFromDef(def *NodeDefinition, portID string) *PortDefinition {
	if def == nil {
		return nil
	}
	for i := range def.Inputs {
		if def.Inputs[i].ID == portID {
			return &def.Inputs[i]
		}
	}
	return nil
}

func isInputPort(def *NodeDefinition, portID string) bool {
	for _, p := range def.Inputs {
		if p.ID == portID {
			return true
		}
	}
	return false
}

func isOutputPort(def *NodeDefinition, portID string) bool {
	for _, p := range def.Outputs {
		if p.ID == portID {
			return true
		}
	}
	return false
}
