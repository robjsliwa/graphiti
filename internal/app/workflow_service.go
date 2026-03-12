package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
	"graphiti/internal/ports/driving"

	"gopkg.in/yaml.v3"
)

// workflowSession holds an in-memory workflow and its command history.
type workflowSession struct {
	workflow *domain.Workflow
	history  *domain.CommandHistory
}

// WorkflowService implements the driving.WorkflowService port.
type WorkflowService struct {
	repo         driven.WorkflowRepository
	nodeRegistry *NodeRegistry
	deployTarget driven.DeployTarget

	mu       sync.RWMutex
	sessions map[string]*workflowSession // workflowID -> session
	maxUndo  int
}

// NewWorkflowService creates a new workflow application service.
func NewWorkflowService(repo driven.WorkflowRepository, registry *NodeRegistry, maxUndo int) *WorkflowService {
	if maxUndo <= 0 {
		maxUndo = 100
	}
	return &WorkflowService{
		repo:         repo,
		nodeRegistry: registry,
		sessions:     make(map[string]*workflowSession),
		maxUndo:      maxUndo,
	}
}

// SetDeployTarget sets the deploy target adapter.
func (s *WorkflowService) SetDeployTarget(target driven.DeployTarget) {
	s.deployTarget = target
}

func (s *WorkflowService) CreateWorkflow(ctx context.Context, name, description, userID string) (*domain.Workflow, error) {
	wf := &domain.Workflow{
		ID:          generateID(),
		Name:        name,
		Description: description,
		Status:      domain.WorkflowStatusDraft,
		Version:     1,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.repo.Create(ctx, wf); err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *WorkflowService) GetWorkflow(ctx context.Context, id string) (*domain.Workflow, error) {
	wf, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.resolveDefinitions(ctx, wf)
	return wf, nil
}

func (s *WorkflowService) ListWorkflows(ctx context.Context, userID string) ([]*domain.WorkflowSummary, error) {
	return s.repo.List(ctx, driven.WorkflowFilter{UserID: userID})
}

func (s *WorkflowService) SaveWorkflow(ctx context.Context, wf *domain.Workflow) error {
	wf.UpdatedAt = time.Now()
	return s.repo.Update(ctx, wf)
}

func (s *WorkflowService) DeleteWorkflow(ctx context.Context, id string) error {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
	return s.repo.Delete(ctx, id)
}

func (s *WorkflowService) ExecuteCommand(ctx context.Context, workflowID string, cmd domain.Command) (*driving.CommandResult, error) {
	session, err := s.getOrLoadSession(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if err := session.history.Execute(session.workflow, cmd); err != nil {
		return nil, err
	}

	session.workflow.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, session.workflow); err != nil {
		return nil, fmt.Errorf("failed to save after command: %w", err)
	}

	return &driving.CommandResult{
		CanUndo:  session.history.CanUndo(),
		CanRedo:  session.history.CanRedo(),
		Workflow: session.workflow,
	}, nil
}

func (s *WorkflowService) Undo(ctx context.Context, workflowID string) (*driving.CommandResult, error) {
	session, err := s.getOrLoadSession(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if err := session.history.Undo(session.workflow); err != nil {
		return nil, err
	}

	session.workflow.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, session.workflow); err != nil {
		return nil, fmt.Errorf("failed to save after undo: %w", err)
	}

	return &driving.CommandResult{
		CanUndo:  session.history.CanUndo(),
		CanRedo:  session.history.CanRedo(),
		Workflow: session.workflow,
	}, nil
}

func (s *WorkflowService) Redo(ctx context.Context, workflowID string) (*driving.CommandResult, error) {
	session, err := s.getOrLoadSession(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if err := session.history.Redo(session.workflow); err != nil {
		return nil, err
	}

	session.workflow.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, session.workflow); err != nil {
		return nil, fmt.Errorf("failed to save after redo: %w", err)
	}

	return &driving.CommandResult{
		CanUndo:  session.history.CanUndo(),
		CanRedo:  session.history.CanRedo(),
		Workflow: session.workflow,
	}, nil
}

func (s *WorkflowService) CopyNodes(ctx context.Context, workflowID string, nodeIDs []string) (*domain.ClipboardPayload, error) {
	session, err := s.getOrLoadSession(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	payload := domain.BuildClipboardPayload(session.workflow, nodeIDs, workflowID)
	return payload, nil
}

func (s *WorkflowService) PasteNodes(ctx context.Context, workflowID string, payload *domain.ClipboardPayload, x, y float64) (*driving.CommandResult, error) {
	// Build nodes with new IDs and absolute positions
	idMapping := make(map[string]string, len(payload.Nodes))
	var nodes []domain.NodeInstance
	for _, cn := range payload.Nodes {
		newID := generateID()
		idMapping[cn.OriginalID] = newID

		def, _ := s.nodeRegistry.GetByID(ctx, cn.DefinitionID)
		nodes = append(nodes, domain.NodeInstance{
			ID:              newID,
			DefinitionID:    cn.DefinitionID,
			Label:           cn.Label,
			X:               x + cn.RelativeX,
			Y:               y + cn.RelativeY,
			AttributeValues: cn.Attributes,
			Definition:      def,
		})
	}

	// Remap edges
	var edges []domain.Edge
	for _, ce := range payload.Edges {
		srcID, srcOK := idMapping[ce.SourceOriginalID]
		tgtID, tgtOK := idMapping[ce.TargetOriginalID]
		if srcOK && tgtOK {
			edges = append(edges, domain.Edge{
				ID:           generateID(),
				SourceNodeID: srcID,
				SourcePortID: ce.SourcePortID,
				TargetNodeID: tgtID,
				TargetPortID: ce.TargetPortID,
			})
		}
	}

	cmd := &pasteCommand{nodes: nodes, edges: edges}
	return s.ExecuteCommand(ctx, workflowID, cmd)
}

func (s *WorkflowService) DeployWorkflow(ctx context.Context, workflowID, target, userID string) (*domain.DeployResult, error) {
	session, err := s.getOrLoadSession(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	// Validate first
	errs := session.workflow.Validate()
	if len(errs) > 0 {
		return &domain.DeployResult{
			Success:          false,
			Message:          fmt.Sprintf("validation failed with %d errors", len(errs)),
			ValidationErrors: errs,
		}, nil
	}

	if s.deployTarget == nil {
		return nil, fmt.Errorf("no deploy target configured")
	}

	defJSON, _ := json.Marshal(struct {
		Nodes []domain.NodeInstance `json:"nodes"`
		Edges []domain.Edge        `json:"edges"`
	}{Nodes: session.workflow.Nodes, Edges: session.workflow.Edges})

	var defAny any
	json.Unmarshal(defJSON, &defAny)

	payload := domain.DeployPayload{
		APIVersion: "graphiti/v1",
		Event:      "workflow.deployed",
		Timestamp:  time.Now(),
		Deployment: domain.DeploymentInfo{
			ID:     generateID(),
			Target: target,
			TriggeredBy: domain.TriggerUser{
				UserID:   userID,
				Username: userID,
			},
		},
		Workflow: domain.WorkflowInfo{
			ID:         session.workflow.ID,
			Name:       session.workflow.Name,
			Version:    session.workflow.Version,
			Definition: defAny,
		},
		PreviousVersion: session.workflow.Version - 1,
	}

	result, err := s.deployTarget.Deploy(ctx, payload)
	if err != nil {
		return nil, err
	}

	// Create version snapshot before incrementing
	versionRecord := &domain.WorkflowVersion{
		ID:         generateID(),
		WorkflowID: session.workflow.ID,
		Version:    session.workflow.Version,
		Definition: defJSON,
		DeployedAt: time.Now(),
		DeployedBy: userID,
		CreatedAt:  time.Now(),
	}
	if err := s.repo.CreateVersion(ctx, versionRecord); err != nil {
		// Log but don't fail the deploy
		_ = err
	}

	// Increment version on successful deploy
	session.workflow.Version++
	session.workflow.Status = domain.WorkflowStatusDeployed
	s.repo.Update(ctx, session.workflow)

	return result, nil
}

func (s *WorkflowService) ExportWorkflow(ctx context.Context, workflowID, format string) ([]byte, error) {
	session, err := s.getOrLoadSession(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	export := exportData{
		APIVersion: "graphiti/v1",
		Kind:       "Workflow",
		Metadata: exportMetadata{
			ID:          session.workflow.ID,
			Name:        session.workflow.Name,
			Description: session.workflow.Description,
			Version:     session.workflow.Version,
			Status:      string(session.workflow.Status),
		},
		Nodes: make([]exportNode, 0, len(session.workflow.Nodes)),
		Edges: make([]exportEdge, 0, len(session.workflow.Edges)),
	}
	for _, n := range session.workflow.Nodes {
		export.Nodes = append(export.Nodes, exportNode{
			ID:           n.ID,
			DefinitionID: n.DefinitionID,
			Label:        n.Label,
			X:            n.X,
			Y:            n.Y,
			Attributes:   n.AttributeValues,
		})
	}
	for _, e := range session.workflow.Edges {
		export.Edges = append(export.Edges, exportEdge{
			ID:           e.ID,
			SourceNodeID: e.SourceNodeID,
			SourcePortID: e.SourcePortID,
			TargetNodeID: e.TargetNodeID,
			TargetPortID: e.TargetPortID,
		})
	}

	switch format {
	case "yaml":
		return yaml.Marshal(export)
	default:
		return json.MarshalIndent(export, "", "  ")
	}
}

// GetVersionHistory returns all version snapshots for a workflow.
func (s *WorkflowService) GetVersionHistory(ctx context.Context, workflowID string) ([]*domain.WorkflowVersion, error) {
	return s.repo.GetVersionHistory(ctx, workflowID)
}

// CheckDeployStatus verifies whether a workflow is still deployed on the engine.
// It uses getOrLoadSession to ensure a single source of truth for the workflow state,
// avoiding split-brain between the repo copy and the in-memory session.
func (s *WorkflowService) CheckDeployStatus(ctx context.Context, workflowID string) (*domain.DeployStatusResult, error) {
	session, err := s.getOrLoadSession(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	wf := session.workflow

	unknown := &domain.DeployStatusResult{
		WorkflowID:   workflowID,
		Version:      wf.Version,
		Verification: domain.DeployVerificationUnknown,
		Message:      "status checking not available",
		CheckedAt:    time.Now(),
	}

	if wf.Status != domain.WorkflowStatusDeployed {
		return unknown, nil
	}

	if s.deployTarget == nil {
		return unknown, nil
	}

	checker, ok := s.deployTarget.(driven.DeployStatusChecker)
	if !ok {
		return unknown, nil
	}

	result, err := checker.CheckDeployStatus(ctx, workflowID, wf.Version)
	if err != nil {
		return &domain.DeployStatusResult{
			WorkflowID:   workflowID,
			Version:      wf.Version,
			Verification: domain.DeployVerificationError,
			Message:      "status check failed",
			CheckedAt:    time.Now(),
		}, nil
	}

	// Reconcile: if engine says missing, revert workflow to draft
	if result.Verification == domain.DeployVerificationMissing {
		now := time.Now()
		s.mu.Lock()
		wf.Status = domain.WorkflowStatusDraft
		wf.UpdatedAt = now
		s.mu.Unlock()

		if err := s.repo.Update(ctx, wf); err != nil {
			slog.Error("failed to reconcile workflow status", "error", err, "workflowID", workflowID)
		}
	}

	return result, nil
}

// exportData is the structure for workflow export.
type exportData struct {
	APIVersion string         `json:"apiVersion" yaml:"apiVersion"`
	Kind       string         `json:"kind" yaml:"kind"`
	Metadata   exportMetadata `json:"metadata" yaml:"metadata"`
	Nodes      []exportNode   `json:"nodes" yaml:"nodes"`
	Edges      []exportEdge   `json:"edges" yaml:"edges"`
}

type exportMetadata struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     int    `json:"version" yaml:"version"`
	Status      string `json:"status" yaml:"status"`
}

type exportNode struct {
	ID           string         `json:"id" yaml:"id"`
	DefinitionID string         `json:"definitionId" yaml:"definitionId"`
	Label        string         `json:"label" yaml:"label"`
	X            float64        `json:"x" yaml:"x"`
	Y            float64        `json:"y" yaml:"y"`
	Attributes   map[string]any `json:"attributes,omitempty" yaml:"attributes,omitempty"`
}

type exportEdge struct {
	ID           string `json:"id" yaml:"id"`
	SourceNodeID string `json:"sourceNodeId" yaml:"sourceNodeId"`
	SourcePortID string `json:"sourcePortId" yaml:"sourcePortId"`
	TargetNodeID string `json:"targetNodeId" yaml:"targetNodeId"`
	TargetPortID string `json:"targetPortId" yaml:"targetPortId"`
}

func (s *WorkflowService) getOrLoadSession(ctx context.Context, workflowID string) (*workflowSession, error) {
	s.mu.RLock()
	session, ok := s.sessions[workflowID]
	s.mu.RUnlock()
	if ok {
		return session, nil
	}

	wf, err := s.repo.GetByID(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	s.resolveDefinitions(ctx, wf)

	session = &workflowSession{
		workflow: wf,
		history:  domain.NewCommandHistory(s.maxUndo),
	}

	s.mu.Lock()
	s.sessions[workflowID] = session
	s.mu.Unlock()

	return session, nil
}

// resolveDefinitions populates Definition pointers on nodes from the registry.
func (s *WorkflowService) resolveDefinitions(ctx context.Context, wf *domain.Workflow) {
	for i := range wf.Nodes {
		if wf.Nodes[i].Definition == nil {
			def, _ := s.nodeRegistry.GetByID(ctx, wf.Nodes[i].DefinitionID)
			wf.Nodes[i].Definition = def
		}
	}
}

// pasteCommand is a thin wrapper used by PasteNodes in the service layer.
type pasteCommand struct {
	nodes []domain.NodeInstance
	edges []domain.Edge
}

func (c *pasteCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
	for _, n := range c.nodes {
		wf.RestoreNode(n)
	}
	for _, e := range c.edges {
		wf.RestoreEdge(e)
	}
	return &undoPasteCommand{nodes: c.nodes, edges: c.edges}, nil
}
func (c *pasteCommand) Type() string             { return "paste_nodes" }
func (c *pasteCommand) Serialize() ([]byte, error) { return json.Marshal(c) }

type undoPasteCommand struct {
	nodes []domain.NodeInstance
	edges []domain.Edge
}

func (c *undoPasteCommand) Execute(wf *domain.Workflow) (domain.Command, error) {
	for _, e := range c.edges {
		wf.RemoveEdge(e.ID)
	}
	for _, n := range c.nodes {
		wf.RemoveNode(n.ID)
	}
	return &pasteCommand{nodes: c.nodes, edges: c.edges}, nil
}
func (c *undoPasteCommand) Type() string             { return "undo_paste_nodes" }
func (c *undoPasteCommand) Serialize() ([]byte, error) { return json.Marshal(c) }
