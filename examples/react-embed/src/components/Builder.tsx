import { useRef, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { GraphitiCanvas, type GraphitiCanvasRef } from '@graphiti/react';
import { NodePalette } from './NodePalette.js';
import { ConfigPanel } from './ConfigPanel.js';
import { Toolbar } from './Toolbar.js';

const GRAPHITI_URL = import.meta.env.VITE_GRAPHITI_URL || 'http://localhost:8080';
const GRAPHITI_TOKEN = import.meta.env.VITE_GRAPHITI_TOKEN || 'dev-token';

export function Builder() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const canvasRef = useRef<GraphitiCanvasRef>(null);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [lastError, setLastError] = useState<string | null>(null);

  if (!id) return <p>Missing workflow ID</p>;

  return (
    <div className="builder" data-testid="builder">
      <header className="builder-header">
        <button data-testid="back-btn" onClick={() => navigate('/')}>
          Back
        </button>
        <Toolbar canvasRef={canvasRef} workflowId={id} />
      </header>

      <div className="builder-layout">
        <aside className="builder-sidebar" data-testid="node-palette-sidebar">
          <NodePalette canvasRef={canvasRef} />
        </aside>

        <main className="builder-canvas" data-testid="canvas-area">
          <GraphitiCanvas
            ref={canvasRef}
            apiUrl={GRAPHITI_URL}
            workflowId={id}
            token={GRAPHITI_TOKEN}
            theme="light"
            className="canvas-element"
            style={{ width: '100%', height: '100%' }}
            onNodeSelected={(e) => setSelectedNodeId(e.nodeId)}
            onNodeDeselected={() => setSelectedNodeId(null)}
            onError={(e) => setLastError(e.message)}
          />
          {lastError && (
            <div className="error-toast" data-testid="error-toast">
              {lastError}
              <button onClick={() => setLastError(null)}>Dismiss</button>
            </div>
          )}
        </main>

        <aside className="builder-config" data-testid="config-sidebar">
          {selectedNodeId ? (
            <ConfigPanel workflowId={id} nodeId={selectedNodeId} />
          ) : (
            <p className="config-placeholder">Select a node to configure it</p>
          )}
        </aside>
      </div>
    </div>
  );
}
