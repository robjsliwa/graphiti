import { type RefObject, useState } from 'react';
import { useGraphitiClient, type GraphitiCanvasRef } from '@graphiti/react';

interface ToolbarProps {
  canvasRef: RefObject<GraphitiCanvasRef | null>;
  workflowId: string;
}

export function Toolbar({ canvasRef, workflowId }: ToolbarProps) {
  const client = useGraphitiClient();
  const [deploying, setDeploying] = useState(false);
  const [deployMsg, setDeployMsg] = useState<string | null>(null);

  async function handleDeploy() {
    setDeploying(true);
    setDeployMsg(null);
    try {
      const res = await client.deployWorkflow(workflowId);
      setDeployMsg(res.success ? 'Deployed successfully' : res.message);
    } catch (err) {
      setDeployMsg(err instanceof Error ? err.message : 'Deploy failed');
    } finally {
      setDeploying(false);
    }
  }

  return (
    <div className="toolbar" data-testid="toolbar">
      <button
        data-testid="undo-btn"
        onClick={() => canvasRef.current?.undo()}
        title="Undo"
      >
        Undo
      </button>
      <button
        data-testid="redo-btn"
        onClick={() => canvasRef.current?.redo()}
        title="Redo"
      >
        Redo
      </button>
      <button
        data-testid="zoom-to-fit-btn"
        onClick={() => canvasRef.current?.zoomToFit()}
        title="Zoom to Fit"
      >
        Fit
      </button>
      <button
        data-testid="deploy-btn"
        onClick={handleDeploy}
        disabled={deploying}
      >
        {deploying ? 'Deploying...' : 'Deploy'}
      </button>
      {deployMsg && (
        <span className="deploy-msg" data-testid="deploy-msg">
          {deployMsg}
        </span>
      )}
    </div>
  );
}
