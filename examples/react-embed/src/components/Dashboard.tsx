import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useWorkflows, useGraphitiClient } from '@graphiti/react';

export function Dashboard() {
  const { workflows, loading, error, refetch } = useWorkflows();
  const client = useGraphitiClient();
  const navigate = useNavigate();
  const [creating, setCreating] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  async function handleCreate() {
    setCreating(true);
    setActionError(null);
    try {
      const res = await client.createWorkflow({ name: 'New Workflow' });
      refetch();
      navigate(`/workflows/${res.workflow.ID}`);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to create workflow');
    } finally {
      setCreating(false);
    }
  }

  async function handleDelete(id: string) {
    setActionError(null);
    try {
      await client.deleteWorkflow(id);
      refetch();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to delete workflow');
    }
  }

  return (
    <div className="dashboard" data-testid="dashboard">
      <header className="dashboard-header">
        <h1>Graphiti Workflows</h1>
        <button
          data-testid="create-workflow-btn"
          onClick={handleCreate}
          disabled={creating}
        >
          {creating ? 'Creating...' : 'New Workflow'}
        </button>
      </header>

      {loading && <p data-testid="loading">Loading workflows...</p>}
      {error && <p className="error" data-testid="error">Error: {error.message}</p>}
      {actionError && <p className="error" data-testid="action-error">Error: {actionError}</p>}

      {workflows && (
        <ul className="workflow-list" data-testid="workflow-list">
          {workflows.length === 0 && (
            <li className="empty" data-testid="empty-state">No workflows yet. Create one to get started.</li>
          )}
          {workflows.map((wf: any) => (
            <li key={wf.ID} className="workflow-item" data-testid={`workflow-${wf.ID}`}>
              <button
                className="workflow-link"
                data-testid={`workflow-link-${wf.ID}`}
                onClick={() => navigate(`/workflows/${wf.ID}`)}
              >
                <span className="workflow-name">{wf.Name}</span>
                <span className="workflow-status">{wf.Status}</span>
              </button>
              <button
                className="delete-btn"
                data-testid={`delete-workflow-${wf.ID}`}
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(wf.ID);
                }}
              >
                Delete
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
