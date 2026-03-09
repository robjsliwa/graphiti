package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
	"graphiti/internal/ports/driving"
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
	return s.repo.GetByID(ctx, id)
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
		CanUndo: session.history.CanUndo(),
		CanRedo: session.history.CanRedo(),
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
		CanUndo: session.history.CanUndo(),
		CanRedo: session.history.CanRedo(),
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
		CanUndo: session.history.CanUndo(),
		CanRedo: session.history.CanRedo(),
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
			Success: false,
			Message: fmt.Sprintf("validation failed with %d errors", len(errs)),
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

	data, err := json.MarshalIndent(session.workflow, "", "  ")
	if err != nil {
		return nil, err
	}
	return data, nil
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

	session = &workflowSession{
		workflow: wf,
		history:  domain.NewCommandHistory(s.maxUndo),
	}

	s.mu.Lock()
	s.sessions[workflowID] = session
	s.mu.Unlock()

	return session, nil
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
