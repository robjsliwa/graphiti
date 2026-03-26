package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driving"
)

type copyRequest struct {
	NodeIDs []string `json:"nodeIds"`
}

type copyResponse struct {
	OK      bool `json:"ok"`
	Payload any  `json:"payload,omitempty"`
	Error   string `json:"error,omitempty"`
}

type pasteRequest struct {
	Payload  json.RawMessage `json:"payload"`
	X        float64         `json:"x"`
	Y        float64         `json:"y"`
	Relative bool            `json:"relative,omitempty"`
}

type attributeUpdateRequest struct {
	AttrID   string `json:"attrId"`
	NewValue any    `json:"newValue"`
}

func handleClipboardCopy(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")

		var req copyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, copyResponse{Error: "invalid request body"})
			return
		}

		payload, err := svc.CopyNodes(r.Context(), workflowID, req.NodeIDs)
		if err != nil {
			slog.Error("copy failed", "error", err)
			writeJSON(w, http.StatusUnprocessableEntity, copyResponse{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, copyResponse{OK: true, Payload: payload})
	}
}

func handleClipboardPaste(svc driving.WorkflowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")

		var req pasteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, commandResponse{Error: "invalid request body"})
			return
		}

		// Deserialize the clipboard payload
		var payload struct {
			Version int `json:"version"`
			Source  string `json:"source"`
			Nodes   []struct {
				OriginalID   string         `json:"originalId"`
				DefinitionID string         `json:"definitionId"`
				Label        string         `json:"label"`
				RelativeX    float64        `json:"relativeX"`
				RelativeY    float64        `json:"relativeY"`
				Attributes   map[string]any `json:"attributes"`
			} `json:"nodes"`
			Edges []struct {
				SourceOriginalID string `json:"sourceOriginalId"`
				SourcePortID     string `json:"sourcePortId"`
				TargetOriginalID string `json:"targetOriginalId"`
				TargetPortID     string `json:"targetPortId"`
			} `json:"edges"`
		}
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, commandResponse{Error: "invalid clipboard payload"})
			return
		}

		// Rebuild domain clipboard payload
		domainPayload := &domain.ClipboardPayload{
			Version: payload.Version,
			Source:  payload.Source,
		}
		for _, n := range payload.Nodes {
			domainPayload.Nodes = append(domainPayload.Nodes, domain.ClipboardNode{
				OriginalID: n.OriginalID, DefinitionID: n.DefinitionID,
				Label: n.Label, RelativeX: n.RelativeX, RelativeY: n.RelativeY,
				Attributes: n.Attributes,
			})
		}
		for _, e := range payload.Edges {
			domainPayload.Edges = append(domainPayload.Edges, domain.ClipboardEdge{
				SourceOriginalID: e.SourceOriginalID, SourcePortID: e.SourcePortID,
				TargetOriginalID: e.TargetOriginalID, TargetPortID: e.TargetPortID,
			})
		}

		// Calculate paste position
		pasteX, pasteY := req.X, req.Y
		if req.Relative && len(payload.Nodes) > 0 {
			// Find first node's position and offset
			first, err := svc.GetWorkflow(r.Context(), workflowID)
			if err == nil && first != nil {
				for _, n := range first.Nodes {
					for _, cn := range payload.Nodes {
						if n.ID == cn.OriginalID {
							pasteX = n.X + req.X
							pasteY = n.Y + req.Y
							goto found
						}
					}
				}
			found:
			}
		}

		result, err := svc.PasteNodes(r.Context(), workflowID, domainPayload, pasteX, pasteY)
		if err != nil {
			slog.Error("paste failed", "error", err)
			writeJSON(w, http.StatusUnprocessableEntity, commandResponse{Error: err.Error()})
			return
		}

		resp := commandResponse{
			OK:      true,
			CanUndo: result.CanUndo,
			CanRedo: result.CanRedo,
		}
		if result.Workflow != nil {
			resp.Workflow = buildWorkflowState(result.Workflow)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func handleAttributeUpdate(svc driving.WorkflowService, registry driving.NodeRegistryService) http.HandlerFunc {
	resolver := &nodeResolver{registry: registry}
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")
		nodeID := r.PathValue("nodeId")

		// Get current node to find old values
		wf, err := svc.GetWorkflow(r.Context(), workflowID)
		if err != nil {
			if wantsJSON(r) {
				writeJSONError(w, http.StatusNotFound, "workflow not found")
			} else {
				http.Error(w, "workflow not found", http.StatusNotFound)
			}
			return
		}
		node := wf.FindNode(nodeID)
		if node == nil {
			if wantsJSON(r) {
				writeJSONError(w, http.StatusNotFound, "node not found")
			} else {
				http.Error(w, "node not found", http.StatusNotFound)
			}
			return
		}

		// Determine if this is a JSON request or form-encoded
		isJSON := strings.Contains(r.Header.Get("Content-Type"), "application/json")

		if isJSON {
			// JSON body: {"attributes": {"key": "value", ...}}
			var body struct {
				Attributes map[string]any `json:"attributes"`
			}
			if !decodeJSONBody(w, r, &body) {
				return
			}

			for attrID, newValue := range body.Attributes {
				oldValue := node.AttributeValues[attrID]
				cmd, err := buildCommand(commandRequest{
					Type:     "update_attribute",
					NodeID:   nodeID,
					AttrID:   attrID,
					OldValue: oldValue,
					NewValue: newValue,
				}, resolver)
				if err != nil {
					writeJSONError(w, http.StatusBadRequest, "invalid attribute", attrID)
					return
				}
				if _, err := svc.ExecuteCommand(r.Context(), workflowID, cmd); err != nil {
					slog.Error("attribute update failed", "error", err, "nodeId", nodeID, "attrId", attrID)
					writeJSONError(w, http.StatusUnprocessableEntity, "attribute update failed", err.Error())
					return
				}
			}

			// Re-fetch and return updated node as JSON
			wf, err = svc.GetWorkflow(r.Context(), workflowID)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "internal error")
				return
			}
			updatedNode := wf.FindNode(nodeID)
			if updatedNode == nil {
				writeJSONError(w, http.StatusInternalServerError, "node not found after update")
				return
			}

			// Mask secret attributes in response
			attrs := make(map[string]any)
			for k, v := range updatedNode.AttributeValues {
				attrs[k] = v
			}
			maskSecretAttributes(attrs, updatedNode.Definition)

			writeJSON(w, http.StatusOK, map[string]any{
				"node": map[string]any{
					"id":           updatedNode.ID,
					"definitionId": updatedNode.DefinitionID,
					"label":        updatedNode.Label,
					"attributes":   attrs,
				},
			})
			return
		}

		// Form-encoded (HTMX path)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		// Process each form field as an attribute update
		for attrID, values := range r.Form {
			if len(values) == 0 {
				continue
			}
			newValue := values[0]
			oldValue := node.AttributeValues[attrID]

			cmd, err := buildCommand(commandRequest{
				Type:     "update_attribute",
				NodeID:   nodeID,
				AttrID:   attrID,
				OldValue: oldValue,
				NewValue: newValue,
			}, resolver)
			if err != nil {
				continue
			}

			if _, err := svc.ExecuteCommand(r.Context(), workflowID, cmd); err != nil {
				slog.Error("attribute update failed", "error", err, "nodeId", nodeID, "attrId", attrID)
			}
		}

		// Re-fetch workflow to get updated node with new attribute values
		wf, err = svc.GetWorkflow(r.Context(), workflowID)
		if err != nil {
			http.Error(w, "workflow not found", http.StatusInternalServerError)
			return
		}
		updatedNode := wf.FindNode(nodeID)
		if updatedNode == nil {
			http.Error(w, "node not found", http.StatusInternalServerError)
			return
		}

		// Build the attribute display data for the node body
		// Only include attributes visible on the node (node-body or both)
		type bodyAttr struct {
			Label string `json:"label"`
			Value string `json:"value"`
		}
		var attrs []bodyAttr
		if updatedNode.Definition != nil {
			for _, a := range updatedNode.Definition.Attributes {
				if a.Display == domain.DisplayNodeBody || a.Display == domain.DisplayBoth {
					val := ""
					if v, ok := updatedNode.AttributeValues[a.ID]; ok && v != nil {
						if s, ok := v.(string); ok {
							val = s
						}
					}
					attrs = append(attrs, bodyAttr{Label: a.Label, Value: val})
				}
			}
		}

		// Send HX-Trigger with updated node data so client JS can update the SVG
		triggerData := map[string]any{
			"nodeUpdated": map[string]any{
				"nodeId":     nodeID,
				"label":      updatedNode.Label,
				"attributes": attrs,
			},
		}
		triggerJSON, _ := json.Marshal(triggerData)
		w.Header().Set("HX-Trigger", string(triggerJSON))
		w.WriteHeader(http.StatusOK)
	}
}
