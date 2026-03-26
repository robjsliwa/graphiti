package http

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"graphiti/internal/domain"
	"graphiti/internal/domain/commands"
	"graphiti/internal/ports/driving"
)

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

type commandRequest struct {
	Type         string  `json:"type"`
	DefinitionID string  `json:"definitionId,omitempty"`
	InstanceID   string  `json:"instanceId,omitempty"`
	Label        string  `json:"label,omitempty"`
	NodeID       string  `json:"nodeId,omitempty"`
	X            float64 `json:"x,omitempty"`
	Y            float64 `json:"y,omitempty"`
	FromX        float64 `json:"fromX,omitempty"`
	FromY        float64 `json:"fromY,omitempty"`
	ToX          float64 `json:"toX,omitempty"`
	ToY          float64 `json:"toY,omitempty"`
	EdgeID       string  `json:"edgeId,omitempty"`
	SourceNodeID string  `json:"sourceNodeId,omitempty"`
	SourcePortID string  `json:"sourcePortId,omitempty"`
	TargetNodeID string  `json:"targetNodeId,omitempty"`
	TargetPortID string  `json:"targetPortId,omitempty"`
	AttrID       string  `json:"attrId,omitempty"`
	OldValue     any     `json:"oldValue,omitempty"`
	NewValue     any     `json:"newValue,omitempty"`
	OldLabel     string  `json:"oldLabel,omitempty"`
	NewLabel     string  `json:"newLabel,omitempty"`
	// move_nodes batch fields
	From []commands.NodePosition `json:"from,omitempty"`
	To   []commands.NodePosition `json:"to,omitempty"`
}

type commandResponse struct {
	OK              bool                   `json:"ok"`
	CanUndo         bool                   `json:"canUndo"`
	CanRedo         bool                   `json:"canRedo"`
	AffectedNodeIDs []string               `json:"affectedNodeIds,omitempty"`
	AffectedEdgeIDs []string               `json:"affectedEdgeIds,omitempty"`
	Error           string                 `json:"error,omitempty"`
	Workflow        *workflowStateResponse `json:"workflow,omitempty"`
}

// workflowStateResponse is the full workflow state sent to the client for canvas sync.
type workflowStateResponse struct {
	Nodes []nodeStateResponse `json:"nodes"`
	Edges []edgeStateResponse `json:"edges"`
}

type nodeStateResponse struct {
	ID           string         `json:"id"`
	DefinitionID string         `json:"definitionId"`
	Label        string         `json:"label"`
	X            float64        `json:"x"`
	Y            float64        `json:"y"`
	Attributes   map[string]any `json:"attributes"`
	Definition   *nodeDefJSON   `json:"definition,omitempty"`
}

type nodeDefJSON struct {
	Icon       string          `json:"icon"`
	Shape      shapeJSON       `json:"shape"`
	Category   categoryJSON    `json:"category"`
	Inputs     []portDefJSON   `json:"inputs"`
	Outputs    []portDefJSON   `json:"outputs"`
	Attributes []attrDefJSON   `json:"attributes"`
}

type shapeJSON struct {
	Type             string `json:"type"`
	Width            int    `json:"width"`
	HeaderColor      string `json:"headerColor"`
	HeaderBackground string `json:"headerBackground"`
}

type categoryJSON struct {
	Group string `json:"group"`
}

type portDefJSON struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Type           string `json:"type"`
	Position       string `json:"position"`
	MaxConnections int    `json:"maxConnections"`
}

type attrDefJSON struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Type    string `json:"type"`
	Display string `json:"display"`
}

type edgeStateResponse struct {
	ID           string `json:"id"`
	SourceNodeID string `json:"sourceNodeId"`
	SourcePortID string `json:"sourcePortId"`
	TargetNodeID string `json:"targetNodeId"`
	TargetPortID string `json:"targetPortId"`
}

// nodeResolver wraps the driving.NodeRegistryService to satisfy commands.NodeDefinitionResolver.
type nodeResolver struct {
	registry driving.NodeRegistryService
}

func (r *nodeResolver) GetByID(id string) (*domain.NodeDefinition, error) {
	return r.registry.GetByID(context.TODO(), id)
}

func handleExecuteCommand(svc driving.WorkflowService, registry driving.NodeRegistryService) http.HandlerFunc {
	resolver := &nodeResolver{registry: registry}
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")

		var req commandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, commandResponse{Error: "invalid request body"})
			return
		}

		// Generate IDs server-side when not provided by the client
		if req.Type == "add_node" && req.InstanceID == "" {
			req.InstanceID = generateID()
		}
		if req.Type == "add_edge" && req.EdgeID == "" {
			req.EdgeID = generateID()
		}

		cmd, err := buildCommand(req, resolver)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, commandResponse{Error: err.Error()})
			return
		}

		result, err := svc.ExecuteCommand(r.Context(), workflowID, cmd)
		if err != nil {
			slog.Error("command execution failed", "error", err, "type", req.Type)
			writeJSON(w, http.StatusUnprocessableEntity, commandResponse{Error: err.Error()})
			return
		}

		resp := commandResponse{
			OK:              true,
			CanUndo:         result.CanUndo,
			CanRedo:         result.CanRedo,
			AffectedNodeIDs: result.AffectedNodeIDs,
			AffectedEdgeIDs: result.AffectedEdgeIDs,
		}
		if result.Workflow != nil {
			resp.Workflow = buildWorkflowState(result.Workflow)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func handleUndo(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")
		result, err := svc.Undo(r.Context(), workflowID)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, commandResponse{Error: err.Error()})
			return
		}
		resp := commandResponse{OK: true, CanUndo: result.CanUndo, CanRedo: result.CanRedo}
		if result.Workflow != nil {
			resp.Workflow = buildWorkflowState(result.Workflow)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func handleRedo(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")
		result, err := svc.Redo(r.Context(), workflowID)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, commandResponse{Error: err.Error()})
			return
		}
		resp := commandResponse{OK: true, CanUndo: result.CanUndo, CanRedo: result.CanRedo}
		if result.Workflow != nil {
			resp.Workflow = buildWorkflowState(result.Workflow)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func buildCommand(req commandRequest, resolver commands.NodeDefinitionResolver) (domain.Command, error) {
	switch req.Type {
	case "add_node":
		return commands.NewAddNodeCommand(req.DefinitionID, req.InstanceID, req.Label, req.X, req.Y, resolver), nil
	case "move_node":
		return &commands.MoveNodeCommand{
			NodeID: req.NodeID, FromX: req.FromX, FromY: req.FromY, ToX: req.ToX, ToY: req.ToY,
		}, nil
	case "move_nodes":
		return &commands.MoveNodesCommand{From: req.From, To: req.To}, nil
	case "add_edge":
		return &commands.AddEdgeCommand{
			EdgeID: req.EdgeID, SourceNodeID: req.SourceNodeID, SourcePortID: req.SourcePortID,
			TargetNodeID: req.TargetNodeID, TargetPortID: req.TargetPortID,
		}, nil
	case "remove_node":
		return &commands.RemoveNodeCommand{NodeID: req.NodeID}, nil
	case "remove_edge":
		return &commands.RemoveEdgeCommand{EdgeID: req.EdgeID}, nil
	case "update_attribute":
		return &commands.UpdateAttributeCommand{
			NodeID: req.NodeID, AttrID: req.AttrID, OldValue: req.OldValue, NewValue: req.NewValue,
		}, nil
	case "rename_node":
		return &commands.RenameNodeCommand{
			NodeID: req.NodeID, OldLabel: req.OldLabel, NewLabel: req.NewLabel,
		}, nil
	default:
		return nil, domain.ErrNodeNotFound
	}
}

func buildWorkflowState(wf *domain.Workflow) *workflowStateResponse {
	state := &workflowStateResponse{
		Nodes: make([]nodeStateResponse, 0, len(wf.Nodes)),
		Edges: make([]edgeStateResponse, 0, len(wf.Edges)),
	}
	for _, n := range wf.Nodes {
		ns := nodeStateResponse{
			ID: n.ID, DefinitionID: n.DefinitionID, Label: n.Label,
			X: n.X, Y: n.Y, Attributes: n.AttributeValues,
		}
		if n.Definition != nil {
			def := &nodeDefJSON{
				Icon: n.Definition.Icon,
				Shape: shapeJSON{
					Type: string(n.Definition.Shape.Type), Width: n.Definition.Shape.Width,
					HeaderColor: n.Definition.Shape.HeaderColor, HeaderBackground: n.Definition.Shape.HeaderBackground,
				},
				Category:   categoryJSON{Group: n.Definition.Category.Group},
				Inputs:     make([]portDefJSON, 0, len(n.Definition.Inputs)),
				Outputs:    make([]portDefJSON, 0, len(n.Definition.Outputs)),
				Attributes: make([]attrDefJSON, 0, len(n.Definition.Attributes)),
			}
			for _, p := range n.Definition.Inputs {
				def.Inputs = append(def.Inputs, portDefJSON{
					ID: p.ID, Label: p.Label, Type: string(p.Type),
					Position: string(p.Position), MaxConnections: p.MaxConnections,
				})
			}
			for _, p := range n.Definition.Outputs {
				def.Outputs = append(def.Outputs, portDefJSON{
					ID: p.ID, Label: p.Label, Type: string(p.Type),
					Position: string(p.Position), MaxConnections: p.MaxConnections,
				})
			}
			for _, a := range n.Definition.Attributes {
				def.Attributes = append(def.Attributes, attrDefJSON{
					ID: a.ID, Label: a.Label, Type: string(a.Type), Display: string(a.Display),
				})
			}
			ns.Definition = def
		}
		state.Nodes = append(state.Nodes, ns)
	}
	for _, e := range wf.Edges {
		state.Edges = append(state.Edges, edgeStateResponse{
			ID: e.ID, SourceNodeID: e.SourceNodeID, SourcePortID: e.SourcePortID,
			TargetNodeID: e.TargetNodeID, TargetPortID: e.TargetPortID,
		})
	}
	return state
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}
