package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"graphiti/internal/domain"
	"graphiti/internal/domain/commands"
	"graphiti/internal/ports/driving"
)

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
}

type commandResponse struct {
	OK              bool     `json:"ok"`
	CanUndo         bool     `json:"canUndo"`
	CanRedo         bool     `json:"canRedo"`
	AffectedNodeIDs []string `json:"affectedNodeIds,omitempty"`
	AffectedEdgeIDs []string `json:"affectedEdgeIds,omitempty"`
	Error           string   `json:"error,omitempty"`
}

// nodeResolver wraps the driving.NodeRegistryService to satisfy commands.NodeDefinitionResolver.
type nodeResolver struct {
	registry driving.NodeRegistryService
}

func (r *nodeResolver) GetByID(id string) (*domain.NodeDefinition, error) {
	return r.registry.GetByID(context.TODO(), id)
}

func handleExecuteCommand(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")

		var req commandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, commandResponse{Error: "invalid request body"})
			return
		}

		cmd, err := buildCommand(req)
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

		writeJSON(w, http.StatusOK, commandResponse{
			OK:              true,
			CanUndo:         result.CanUndo,
			CanRedo:         result.CanRedo,
			AffectedNodeIDs: result.AffectedNodeIDs,
			AffectedEdgeIDs: result.AffectedEdgeIDs,
		})
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
		writeJSON(w, http.StatusOK, commandResponse{
			OK:      true,
			CanUndo: result.CanUndo,
			CanRedo: result.CanRedo,
		})
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
		writeJSON(w, http.StatusOK, commandResponse{
			OK:      true,
			CanUndo: result.CanUndo,
			CanRedo: result.CanRedo,
		})
	}
}

func buildCommand(req commandRequest) (domain.Command, error) {
	switch req.Type {
	case "add_node":
		return commands.NewAddNodeCommand(req.DefinitionID, req.InstanceID, req.Label, req.X, req.Y, nil), nil
	case "move_node":
		return &commands.MoveNodeCommand{
			NodeID: req.NodeID,
			FromX:  req.FromX,
			FromY:  req.FromY,
			ToX:    req.ToX,
			ToY:    req.ToY,
		}, nil
	case "add_edge":
		return &commands.AddEdgeCommand{
			EdgeID:       req.EdgeID,
			SourceNodeID: req.SourceNodeID,
			SourcePortID: req.SourcePortID,
			TargetNodeID: req.TargetNodeID,
			TargetPortID: req.TargetPortID,
		}, nil
	case "remove_node":
		return &commands.RemoveNodeCommand{NodeID: req.NodeID}, nil
	case "remove_edge":
		return &commands.RemoveEdgeCommand{EdgeID: req.EdgeID}, nil
	case "update_attribute":
		return &commands.UpdateAttributeCommand{
			NodeID:   req.NodeID,
			AttrID:   req.AttrID,
			OldValue: req.OldValue,
			NewValue: req.NewValue,
		}, nil
	case "rename_node":
		return &commands.RenameNodeCommand{
			NodeID:   req.NodeID,
			OldLabel: req.OldLabel,
			NewLabel: req.NewLabel,
		}, nil
	default:
		return nil, domain.ErrNodeNotFound
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
