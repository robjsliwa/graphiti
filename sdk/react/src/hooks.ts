import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  useCallback,
} from 'react';
import { GraphitiClient, GraphitiWS } from '@graphiti/client';
import type {
  WorkflowSummary,
  NodeDefinition,
  NodeStatusEvent,
} from '@graphiti/client';

// --- Context ---

export const GraphitiContext = createContext<GraphitiClient | null>(null);

export function useGraphitiClient(): GraphitiClient {
  const client = useContext(GraphitiContext);
  if (!client) {
    throw new Error('useGraphitiClient must be used within a GraphitiProvider');
  }
  return client;
}

// --- useWorkflows ---

export interface UseWorkflowsResult {
  workflows: WorkflowSummary[] | undefined;
  loading: boolean;
  error: Error | undefined;
  refetch: () => void;
}

export function useWorkflows(): UseWorkflowsResult {
  const client = useGraphitiClient();
  const [workflows, setWorkflows] = useState<WorkflowSummary[] | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | undefined>(undefined);

  const load = useCallback(async () => {
    let cancelled = false;
    setLoading(true);
    setError(undefined);
    try {
      const res = await client.listWorkflows();
      if (!cancelled) {
        setWorkflows(res.items);
      }
    } catch (err) {
      if (!cancelled) {
        setError(err instanceof Error ? err : new Error(String(err)));
      }
    } finally {
      if (!cancelled) {
        setLoading(false);
      }
    }
    return () => {
      cancelled = true;
    };
  }, [client]);

  useEffect(() => {
    load();
  }, [load]);

  return { workflows, loading, error, refetch: load };
}

// --- useNodeDefinitions ---

export interface UseNodeDefinitionsResult {
  definitions: NodeDefinition[] | undefined;
  loading: boolean;
  error: Error | undefined;
}

export function useNodeDefinitions(): UseNodeDefinitionsResult {
  const client = useGraphitiClient();
  const [definitions, setDefinitions] = useState<NodeDefinition[] | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | undefined>(undefined);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(undefined);
      try {
        const res = await client.listNodeDefinitions();
        if (!cancelled) {
          setDefinitions(res.items);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err : new Error(String(err)));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [client]);

  return { definitions, loading, error };
}

// --- useExecutionStatus ---

export interface UseExecutionStatusResult {
  statuses: Map<string, NodeStatusEvent>;
  connected: boolean;
}

export function useExecutionStatus(
  baseUrl: string,
  token: string,
  workflowId: string,
): UseExecutionStatusResult {
  const [statuses, setStatuses] = useState<Map<string, NodeStatusEvent>>(new Map());
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<GraphitiWS | null>(null);

  useEffect(() => {
    if (!baseUrl || !token || !workflowId) return;

    const ws = new GraphitiWS({
      baseUrl,
      token,
      workflowId,
      autoReconnect: true,
    });

    ws.on('connected', () => setConnected(true));
    ws.on('disconnected', () => setConnected(false));
    ws.on('node_status', (event) => {
      setStatuses((prev) => {
        const next = new Map(prev);
        next.set(event.nodeID, event);
        return next;
      });
    });

    ws.connect();
    wsRef.current = ws;

    return () => {
      ws.disconnect();
      wsRef.current = null;
      setStatuses(new Map());
      setConnected(false);
    };
  }, [baseUrl, token, workflowId]);

  return { statuses, connected };
}
