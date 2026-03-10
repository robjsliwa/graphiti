package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

// WorkflowRepository implements driven.WorkflowRepository using SQLite.
type WorkflowRepository struct {
	db *sql.DB
}

// NewWorkflowRepository creates a new SQLite-backed workflow repository.
func NewWorkflowRepository(db *sql.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

// nodeJSON is the JSON serialization format for a node instance.
type nodeJSON struct {
	ID              string         `json:"id"`
	DefinitionID    string         `json:"definitionId"`
	Label           string         `json:"label"`
	X               float64        `json:"x"`
	Y               float64        `json:"y"`
	AttributeValues map[string]any `json:"attributeValues,omitempty"`
}

// edgeJSON is the JSON serialization format for an edge.
type edgeJSON struct {
	ID           string `json:"id"`
	SourceNodeID string `json:"sourceNodeId"`
	SourcePortID string `json:"sourcePortId"`
	TargetNodeID string `json:"targetNodeId"`
	TargetPortID string `json:"targetPortId"`
}

// workflowDefinition is the JSON structure stored in the definition column.
type workflowDefinition struct {
	Nodes []nodeJSON `json:"nodes"`
	Edges []edgeJSON `json:"edges"`
}

func marshalDefinition(wf *domain.Workflow) ([]byte, error) {
	def := workflowDefinition{
		Nodes: make([]nodeJSON, len(wf.Nodes)),
		Edges: make([]edgeJSON, len(wf.Edges)),
	}
	for i, n := range wf.Nodes {
		def.Nodes[i] = nodeJSON{
			ID:              n.ID,
			DefinitionID:    n.DefinitionID,
			Label:           n.Label,
			X:               n.X,
			Y:               n.Y,
			AttributeValues: n.AttributeValues,
		}
	}
	for i, e := range wf.Edges {
		def.Edges[i] = edgeJSON{
			ID:           e.ID,
			SourceNodeID: e.SourceNodeID,
			SourcePortID: e.SourcePortID,
			TargetNodeID: e.TargetNodeID,
			TargetPortID: e.TargetPortID,
		}
	}
	return json.Marshal(def)
}

func unmarshalDefinition(data []byte) ([]domain.NodeInstance, []domain.Edge, error) {
	var def workflowDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, nil, fmt.Errorf("unmarshal definition: %w", err)
	}

	nodes := make([]domain.NodeInstance, len(def.Nodes))
	for i, n := range def.Nodes {
		nodes[i] = domain.NodeInstance{
			ID:              n.ID,
			DefinitionID:    n.DefinitionID,
			Label:           n.Label,
			X:               n.X,
			Y:               n.Y,
			AttributeValues: n.AttributeValues,
		}
	}

	edges := make([]domain.Edge, len(def.Edges))
	for i, e := range def.Edges {
		edges[i] = domain.Edge{
			ID:           e.ID,
			SourceNodeID: e.SourceNodeID,
			SourcePortID: e.SourcePortID,
			TargetNodeID: e.TargetNodeID,
			TargetPortID: e.TargetPortID,
		}
	}

	return nodes, edges, nil
}

// Create inserts a new workflow into the database.
func (r *WorkflowRepository) Create(ctx context.Context, wf *domain.Workflow) error {
	defJSON, err := marshalDefinition(wf)
	if err != nil {
		return fmt.Errorf("marshal definition: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO workflows (id, name, description, definition, status, version, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		wf.ID, wf.Name, wf.Description, string(defJSON), string(wf.Status),
		wf.Version, wf.CreatedBy, wf.CreatedAt.UTC(), wf.UpdatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert workflow: %w", err)
	}
	return nil
}

// GetByID retrieves a workflow by its ID.
func (r *WorkflowRepository) GetByID(ctx context.Context, id string) (*domain.Workflow, error) {
	var wf domain.Workflow
	var defJSON string
	var status string
	var description sql.NullString
	var createdBy sql.NullString
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, definition, status, version, created_by, created_at, updated_at
		 FROM workflows WHERE id = ?`, id,
	).Scan(&wf.ID, &wf.Name, &description, &defJSON, &status,
		&wf.Version, &createdBy, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow %s: not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query workflow: %w", err)
	}

	wf.Description = description.String
	wf.CreatedBy = createdBy.String
	wf.Status = domain.WorkflowStatus(status)

	wf.CreatedAt, _ = time.Parse("2006-01-02 15:04:05-07:00", createdAt)
	if wf.CreatedAt.IsZero() {
		wf.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	}
	wf.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05-07:00", updatedAt)
	if wf.UpdatedAt.IsZero() {
		wf.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	}

	nodes, edges, err := unmarshalDefinition([]byte(defJSON))
	if err != nil {
		return nil, err
	}
	wf.Nodes = nodes
	wf.Edges = edges

	return &wf, nil
}

// List returns workflow summaries matching the given filter.
func (r *WorkflowRepository) List(ctx context.Context, filter driven.WorkflowFilter) ([]*domain.WorkflowSummary, error) {
	query := "SELECT id, name, status, version, updated_at FROM workflows WHERE 1=1"
	var args []any

	if filter.UserID != "" {
		query += " AND created_by = ?"
		args = append(args, filter.UserID)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, string(filter.Status))
	}
	query += " ORDER BY updated_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	defer rows.Close()

	var summaries []*domain.WorkflowSummary
	for rows.Next() {
		var s domain.WorkflowSummary
		var status string
		var updatedAt string
		if err := rows.Scan(&s.ID, &s.Name, &status, &s.Version, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan workflow summary: %w", err)
		}
		s.Status = domain.WorkflowStatus(status)
		s.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05-07:00", updatedAt)
		if s.UpdatedAt.IsZero() {
			s.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
		}
		summaries = append(summaries, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow rows: %w", err)
	}

	return summaries, nil
}

// Update saves changes to an existing workflow.
func (r *WorkflowRepository) Update(ctx context.Context, wf *domain.Workflow) error {
	defJSON, err := marshalDefinition(wf)
	if err != nil {
		return fmt.Errorf("marshal definition: %w", err)
	}

	wf.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx,
		`UPDATE workflows SET name = ?, description = ?, definition = ?, status = ?, version = ?, updated_at = ?
		 WHERE id = ?`,
		wf.Name, wf.Description, string(defJSON), string(wf.Status),
		wf.Version, wf.UpdatedAt.UTC(), wf.ID,
	)
	if err != nil {
		return fmt.Errorf("update workflow: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workflow %s: not found", wf.ID)
	}
	return nil
}

// Delete removes a workflow and its versions (via CASCADE).
func (r *WorkflowRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM workflows WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete workflow: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workflow %s: not found", id)
	}
	return nil
}

// GetVersionHistory returns all version snapshots for a workflow.
func (r *WorkflowRepository) GetVersionHistory(ctx context.Context, id string) ([]*domain.WorkflowVersion, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, workflow_id, version, definition, deployed_at, deployed_by, created_at
		 FROM workflow_versions WHERE workflow_id = ? ORDER BY version DESC`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("query version history: %w", err)
	}
	defer rows.Close()

	var versions []*domain.WorkflowVersion
	for rows.Next() {
		var v domain.WorkflowVersion
		var deployedAt, createdAt sql.NullString
		var deployedBy sql.NullString

		if err := rows.Scan(&v.ID, &v.WorkflowID, &v.Version, &v.Definition,
			&deployedAt, &deployedBy, &createdAt); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}

		v.DeployedBy = deployedBy.String
		if deployedAt.Valid {
			v.DeployedAt, _ = time.Parse("2006-01-02 15:04:05-07:00", deployedAt.String)
			if v.DeployedAt.IsZero() {
				v.DeployedAt, _ = time.Parse("2006-01-02 15:04:05", deployedAt.String)
			}
		}
		if createdAt.Valid {
			v.CreatedAt, _ = time.Parse("2006-01-02 15:04:05-07:00", createdAt.String)
			if v.CreatedAt.IsZero() {
				v.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
			}
		}

		versions = append(versions, &v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate version rows: %w", err)
	}

	return versions, nil
}

// CreateVersion inserts a new workflow version snapshot.
func (r *WorkflowRepository) CreateVersion(ctx context.Context, v *domain.WorkflowVersion) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO workflow_versions (id, workflow_id, version, definition, deployed_at, deployed_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.WorkflowID, v.Version, string(v.Definition),
		v.DeployedAt.UTC(), v.DeployedBy, v.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert workflow version: %w", err)
	}
	return nil
}

// Ensure WorkflowRepository implements driven.WorkflowRepository.
var _ driven.WorkflowRepository = (*WorkflowRepository)(nil)
