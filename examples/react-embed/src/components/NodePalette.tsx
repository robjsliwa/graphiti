import { type RefObject } from 'react';
import { useNodeDefinitions, type GraphitiCanvasRef, type NodeDefinition } from '@graphiti/react';

interface NodePaletteProps {
  canvasRef: RefObject<GraphitiCanvasRef | null>;
}

export function NodePalette({ canvasRef }: NodePaletteProps) {
  const { definitions, loading, error } = useNodeDefinitions();

  if (loading) return <p data-testid="palette-loading">Loading nodes...</p>;
  if (error) return <p data-testid="palette-error">Failed to load nodes</p>;

  // Group by category
  const groups = new Map<string, NodeDefinition[]>();
  for (const def of definitions || []) {
    const group = def.Category?.Group || 'other';
    if (!groups.has(group)) groups.set(group, []);
    groups.get(group)!.push(def);
  }

  async function handleAddNode(definitionId: string) {
    canvasRef.current?.executeCommand({
      type: 'add_node',
      definitionId,
      instanceId: crypto.randomUUID(),
      x: 200 + Math.random() * 200,
      y: 100 + Math.random() * 200,
    });
  }

  return (
    <div className="node-palette" data-testid="node-palette">
      <h3>Node Palette</h3>
      {Array.from(groups.entries()).map(([group, defs]) => (
        <div key={group} className="palette-group">
          <h4>{group}</h4>
          <ul>
            {defs.map((def) => (
              <li key={def.ID}>
                <button
                  className="palette-node"
                  data-testid={`palette-node-${def.ID}`}
                  onClick={() => handleAddNode(def.ID)}
                >
                  {def.Name || def.ID}
                </button>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}
