import { useEffect, useRef, useState, useCallback } from 'react';
import { useGraphitiClient, type NodeConfigResponse } from '@graphiti/react';

interface ConfigPanelProps {
  workflowId: string;
  nodeId: string;
}

const DEBOUNCE_MS = 400;

export function ConfigPanel({ workflowId, nodeId }: ConfigPanelProps) {
  const client = useGraphitiClient();
  const [config, setConfig] = useState<NodeConfigResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const debounceTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);
      try {
        const res = await client.getNodeConfig(workflowId, nodeId);
        if (!cancelled) setConfig(res);
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load config');
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [client, workflowId, nodeId]);

  // Cleanup debounce timers on unmount
  useEffect(() => {
    return () => {
      for (const timer of debounceTimers.current.values()) {
        clearTimeout(timer);
      }
    };
  }, []);

  const handleAttrChange = useCallback(
    (attrId: string, value: unknown) => {
      // Clear existing debounce for this attribute
      const existing = debounceTimers.current.get(attrId);
      if (existing) clearTimeout(existing);

      const timer = setTimeout(async () => {
        debounceTimers.current.delete(attrId);
        try {
          await client.updateAttributes(workflowId, nodeId, { [attrId]: value });
        } catch (err) {
          setError(err instanceof Error ? err.message : 'Failed to update attribute');
        }
      }, DEBOUNCE_MS);

      debounceTimers.current.set(attrId, timer);
    },
    [client, workflowId, nodeId],
  );

  const handleAttrChangeImmediate = useCallback(
    async (attrId: string, value: unknown) => {
      try {
        await client.updateAttributes(workflowId, nodeId, { [attrId]: value });
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to update attribute');
      }
    },
    [client, workflowId, nodeId],
  );

  if (loading) return <p data-testid="config-loading">Loading config...</p>;
  if (error) return <p data-testid="config-error">{error}</p>;
  if (!config) return null;

  const { node, definition } = config;

  return (
    <div className="config-panel" data-testid="config-panel">
      <h3>{node.label || definition.icon}</h3>
      <p className="config-def-id">Type: {node.definitionId}</p>

      {definition.attributes.length === 0 && (
        <p className="config-no-attrs">No configurable attributes</p>
      )}

      {definition.attributes.map((attr) => (
        <div key={attr.id} className="config-attr" data-testid={`config-attr-${attr.id}`}>
          <label htmlFor={`attr-${attr.id}`}>{attr.label}</label>
          {attr.type === 'boolean' ? (
            <input
              id={`attr-${attr.id}`}
              type="checkbox"
              checked={!!node.attributes[attr.id]}
              onChange={(e) => handleAttrChangeImmediate(attr.id, e.target.checked)}
            />
          ) : attr.type === 'number' ? (
            <input
              id={`attr-${attr.id}`}
              type="number"
              value={(node.attributes[attr.id] as number) ?? ''}
              onChange={(e) => handleAttrChange(attr.id, Number(e.target.value))}
            />
          ) : attr.type === 'text' || attr.type === 'json' ? (
            <textarea
              id={`attr-${attr.id}`}
              value={(node.attributes[attr.id] as string) ?? ''}
              onChange={(e) => handleAttrChange(attr.id, e.target.value)}
            />
          ) : attr.type === 'secret' ? (
            <input
              id={`attr-${attr.id}`}
              type="password"
              placeholder="••••••"
              onChange={(e) => handleAttrChange(attr.id, e.target.value)}
            />
          ) : (
            <input
              id={`attr-${attr.id}`}
              type="text"
              value={(node.attributes[attr.id] as string) ?? ''}
              onChange={(e) => handleAttrChange(attr.id, e.target.value)}
            />
          )}
        </div>
      ))}
    </div>
  );
}
