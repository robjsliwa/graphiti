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

// ExecutionRepository implements driven.ExecutionRepository using SQLite.
type ExecutionRepository struct {
	db *sql.DB
}

// NewExecutionRepository creates a new SQLite-backed execution repository.
func NewExecutionRepository(db *sql.DB) *ExecutionRepository {
	return &ExecutionRepository{db: db}
}

// compile-time interface check
var _ driven.ExecutionRepository = (*ExecutionRepository)(nil)

// Create inserts a new execution run.
func (r *ExecutionRepository) Create(ctx context.Context, run *domain.ExecutionRun) error {
	triggerJSON, _ := json.Marshal(run.TriggerData)

	var startedAt, completedAt *string
	if !run.StartedAt.IsZero() {
		s := run.StartedAt.UTC().Format(time.RFC3339)
		startedAt = &s
	}
	if !run.CompletedAt.IsZero() {
		s := run.CompletedAt.UTC().Format(time.RFC3339)
		completedAt = &s
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO execution_runs (id, workflow_id, workflow_version, status, started_at, completed_at, trigger_type, trigger_data)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.WorkflowID, run.WorkflowVersion, string(run.Status),
		startedAt, completedAt, run.TriggerType, string(triggerJSON),
	)
	if err != nil {
		return fmt.Errorf("insert execution run: %w", err)
	}
	return nil
}

// GetByID retrieves an execution run with all its node statuses.
func (r *ExecutionRepository) GetByID(ctx context.Context, id string) (*domain.ExecutionRun, error) {
	var run domain.ExecutionRun
	var status string
	var startedAt, completedAt sql.NullString
	var triggerData sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, workflow_id, workflow_version, status, started_at, completed_at, trigger_type, trigger_data
		 FROM execution_runs WHERE id = ?`, id,
	).Scan(&run.ID, &run.WorkflowID, &run.WorkflowVersion, &status,
		&startedAt, &completedAt, &run.TriggerType, &triggerData)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution run %s: not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query execution run: %w", err)
	}

	run.Status = domain.ExecutionStatus(status)
	run.StartedAt = parseNullTime(startedAt)
	run.CompletedAt = parseNullTime(completedAt)

	if triggerData.Valid && triggerData.String != "" {
		json.Unmarshal([]byte(triggerData.String), &run.TriggerData)
	}

	// Load node statuses
	run.NodeStatuses, err = r.loadNodeStatuses(ctx, id)
	if err != nil {
		return nil, err
	}

	return &run, nil
}

// ListByWorkflow returns execution run summaries for a workflow.
func (r *ExecutionRepository) ListByWorkflow(ctx context.Context, workflowID string, filter driven.ExecutionFilter) ([]*domain.ExecutionRunSummary, error) {
	query := "SELECT id, workflow_id, status, started_at, completed_at, trigger_type FROM execution_runs WHERE workflow_id = ?"
	args := []any{workflowID}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, string(filter.Status))
	}
	query += " ORDER BY started_at DESC, id DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list execution runs: %w", err)
	}
	defer rows.Close()

	var summaries []*domain.ExecutionRunSummary
	for rows.Next() {
		var s domain.ExecutionRunSummary
		var status string
		var startedAt, completedAt sql.NullString
		if err := rows.Scan(&s.ID, &s.WorkflowID, &status, &startedAt, &completedAt, &s.TriggerType); err != nil {
			return nil, fmt.Errorf("scan execution summary: %w", err)
		}
		s.Status = domain.ExecutionStatus(status)
		s.StartedAt = parseNullTime(startedAt)
		s.CompletedAt = parseNullTime(completedAt)
		summaries = append(summaries, &s)
	}
	return summaries, rows.Err()
}

// UpdateNodeStatus creates or updates a node's execution status within a run.
func (r *ExecutionRepository) UpdateNodeStatus(ctx context.Context, runID, nodeID string, status domain.NodeExecutionStatus) error {
	logsJSON, _ := json.Marshal(status.Logs)
	inputJSON, _ := json.Marshal(status.InputData)
	outputJSON, _ := json.Marshal(status.OutputData)

	var startedAt, completedAt *string
	if !status.StartedAt.IsZero() {
		s := status.StartedAt.UTC().Format(time.RFC3339)
		startedAt = &s
	}
	if !status.CompletedAt.IsZero() {
		s := status.CompletedAt.UTC().Format(time.RFC3339)
		completedAt = &s
	}

	// Try update first, then insert
	result, err := r.db.ExecContext(ctx,
		`UPDATE execution_node_statuses
		 SET status = ?, started_at = ?, completed_at = ?, input_data = ?, output_data = ?, error_message = ?, logs = ?
		 WHERE run_id = ? AND node_id = ?`,
		string(status.Status), startedAt, completedAt,
		string(inputJSON), string(outputJSON), status.ErrorMessage, string(logsJSON),
		runID, nodeID,
	)
	if err != nil {
		return fmt.Errorf("update node status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		// Insert new
		id := fmt.Sprintf("%s-%s", runID, nodeID)
		_, err = r.db.ExecContext(ctx,
			`INSERT INTO execution_node_statuses (id, run_id, node_id, status, started_at, completed_at, input_data, output_data, error_message, logs)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, runID, nodeID, string(status.Status),
			startedAt, completedAt,
			string(inputJSON), string(outputJSON), status.ErrorMessage, string(logsJSON),
		)
		if err != nil {
			return fmt.Errorf("insert node status: %w", err)
		}
	}
	return nil
}

// AppendNodeLog adds a log entry to a node's execution status.
func (r *ExecutionRepository) AppendNodeLog(ctx context.Context, runID, nodeID string, entry domain.LogEntry) error {
	// Read existing logs
	var logsStr sql.NullString
	err := r.db.QueryRowContext(ctx,
		"SELECT logs FROM execution_node_statuses WHERE run_id = ? AND node_id = ?",
		runID, nodeID,
	).Scan(&logsStr)
	if err != nil {
		return fmt.Errorf("read existing logs: %w", err)
	}

	var logs []domain.LogEntry
	if logsStr.Valid && logsStr.String != "" && logsStr.String != "null" {
		json.Unmarshal([]byte(logsStr.String), &logs)
	}
	logs = append(logs, entry)

	logsJSON, _ := json.Marshal(logs)
	_, err = r.db.ExecContext(ctx,
		"UPDATE execution_node_statuses SET logs = ? WHERE run_id = ? AND node_id = ?",
		string(logsJSON), runID, nodeID,
	)
	if err != nil {
		return fmt.Errorf("update logs: %w", err)
	}
	return nil
}

// UpdateRunStatus updates the overall status of an execution run.
func (r *ExecutionRepository) UpdateRunStatus(ctx context.Context, runID string, status domain.ExecutionStatus) error {
	var completedAt *string
	if status == domain.ExecStatusCompleted || status == domain.ExecStatusFailed || status == domain.ExecStatusCancelled {
		s := time.Now().UTC().Format(time.RFC3339)
		completedAt = &s
	}

	_, err := r.db.ExecContext(ctx,
		"UPDATE execution_runs SET status = ?, completed_at = ? WHERE id = ?",
		string(status), completedAt, runID,
	)
	if err != nil {
		return fmt.Errorf("update run status: %w", err)
	}
	return nil
}

// loadNodeStatuses loads all node execution statuses for a run.
func (r *ExecutionRepository) loadNodeStatuses(ctx context.Context, runID string) (map[string]*domain.NodeExecutionStatus, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT node_id, status, started_at, completed_at, input_data, output_data, error_message, logs
		 FROM execution_node_statuses WHERE run_id = ?`, runID,
	)
	if err != nil {
		return nil, fmt.Errorf("query node statuses: %w", err)
	}
	defer rows.Close()

	statuses := make(map[string]*domain.NodeExecutionStatus)
	for rows.Next() {
		var ns domain.NodeExecutionStatus
		var status string
		var startedAt, completedAt sql.NullString
		var inputData, outputData, logsStr sql.NullString

		if err := rows.Scan(&ns.NodeID, &status, &startedAt, &completedAt,
			&inputData, &outputData, &ns.ErrorMessage, &logsStr); err != nil {
			return nil, fmt.Errorf("scan node status: %w", err)
		}

		ns.Status = domain.NodeExecStatus(status)
		ns.StartedAt = parseNullTime(startedAt)
		ns.CompletedAt = parseNullTime(completedAt)

		if inputData.Valid && inputData.String != "" && inputData.String != "null" {
			json.Unmarshal([]byte(inputData.String), &ns.InputData)
		}
		if outputData.Valid && outputData.String != "" && outputData.String != "null" {
			json.Unmarshal([]byte(outputData.String), &ns.OutputData)
		}
		if logsStr.Valid && logsStr.String != "" && logsStr.String != "null" {
			json.Unmarshal([]byte(logsStr.String), &ns.Logs)
		}

		statuses[ns.NodeID] = &ns
	}
	return statuses, rows.Err()
}

// parseTimeStr tries multiple time formats that modernc.org/sqlite may produce.
func ParseTimeStr(s string) time.Time {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// parseNullTime parses a nullable time string.
func parseNullTime(ns sql.NullString) time.Time {
	if !ns.Valid || ns.String == "" {
		return time.Time{}
	}
	return ParseTimeStr(ns.String)
}
