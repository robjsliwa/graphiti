import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import type { ReactNode } from 'react';
import { GraphitiClient } from '@graphiti/client';
import {
  GraphitiProvider,
  useGraphitiClient,
  useWorkflows,
  useNodeDefinitions,
  useExecutionStatus,
} from '../src/index.js';

function createMockClient(overrides: Partial<GraphitiClient> = {}): GraphitiClient {
  const mockFetch = vi.fn().mockResolvedValue(new Response('{}', { status: 200 }));
  const client = new GraphitiClient({
    baseUrl: 'http://localhost:8080',
    token: 'test-token',
    fetch: mockFetch,
  });

  // Override methods with mocks
  for (const [key, value] of Object.entries(overrides)) {
    (client as any)[key] = value;
  }

  return client;
}

function createWrapper(client: GraphitiClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <GraphitiProvider client={client}>{children}</GraphitiProvider>;
  };
}

describe('useGraphitiClient', () => {
  it('returns the client from context', () => {
    const client = createMockClient();
    const { result } = renderHook(() => useGraphitiClient(), {
      wrapper: createWrapper(client),
    });
    expect(result.current).toBe(client);
  });

  it('throws when used outside provider', () => {
    expect(() => {
      renderHook(() => useGraphitiClient());
    }).toThrow('useGraphitiClient must be used within a GraphitiProvider');
  });
});

describe('useWorkflows', () => {
  it('returns loading state initially', () => {
    const client = createMockClient({
      listWorkflows: vi.fn().mockResolvedValue({ items: [] }),
    } as any);

    const { result } = renderHook(() => useWorkflows(), {
      wrapper: createWrapper(client),
    });

    expect(result.current.loading).toBe(true);
    expect(result.current.workflows).toBeUndefined();
    expect(result.current.error).toBeUndefined();
  });

  it('returns workflows after loading', async () => {
    const mockWorkflows = [
      { ID: 'wf-1', Name: 'Test', Status: 'draft', Version: 1, UpdatedAt: '' },
    ];
    const client = createMockClient({
      listWorkflows: vi.fn().mockResolvedValue({ items: mockWorkflows }),
    } as any);

    const { result } = renderHook(() => useWorkflows(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.workflows).toEqual(mockWorkflows);
    expect(result.current.error).toBeUndefined();
  });

  it('returns error on failure', async () => {
    const client = createMockClient({
      listWorkflows: vi.fn().mockRejectedValue(new Error('Network error')),
    } as any);

    const { result } = renderHook(() => useWorkflows(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.workflows).toBeUndefined();
    expect(result.current.error?.message).toBe('Network error');
  });

  it('refetch reloads data', async () => {
    const listWorkflows = vi
      .fn()
      .mockResolvedValueOnce({ items: [{ ID: 'wf-1', Name: 'First', Status: 'draft', Version: 1, UpdatedAt: '' }] })
      .mockResolvedValueOnce({ items: [{ ID: 'wf-1', Name: 'Updated', Status: 'draft', Version: 1, UpdatedAt: '' }] });

    const client = createMockClient({ listWorkflows } as any);

    const { result } = renderHook(() => useWorkflows(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.workflows?.[0]).toMatchObject({ Name: 'First' });

    await act(async () => {
      result.current.refetch();
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.workflows?.[0]).toMatchObject({ Name: 'Updated' });
    expect(listWorkflows).toHaveBeenCalledTimes(2);
  });
});

describe('useNodeDefinitions', () => {
  it('returns loading state initially', () => {
    const client = createMockClient({
      listNodeDefinitions: vi.fn().mockResolvedValue({ items: [] }),
    } as any);

    const { result } = renderHook(() => useNodeDefinitions(), {
      wrapper: createWrapper(client),
    });

    expect(result.current.loading).toBe(true);
    expect(result.current.definitions).toBeUndefined();
  });

  it('returns definitions after loading', async () => {
    const mockDefs = [{ ID: 'api-gateway', Name: 'API Gateway' }];
    const client = createMockClient({
      listNodeDefinitions: vi.fn().mockResolvedValue({ items: mockDefs }),
    } as any);

    const { result } = renderHook(() => useNodeDefinitions(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.definitions).toEqual(mockDefs);
  });

  it('returns error on failure', async () => {
    const client = createMockClient({
      listNodeDefinitions: vi.fn().mockRejectedValue(new Error('fail')),
    } as any);

    const { result } = renderHook(() => useNodeDefinitions(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error?.message).toBe('fail');
  });
});

// Mock WebSocket for useExecutionStatus tests
class MockWebSocket {
  static instances: MockWebSocket[] = [];
  url: string;
  onopen: ((ev: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  close = vi.fn().mockImplementation(function (this: MockWebSocket) {
    // Simulate the close event when close() is called
    setTimeout(() => {
      this.onclose?.({ code: 1000, reason: 'client disconnect' } as CloseEvent);
    }, 0);
  });

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
    // Defer open to allow event handler attachment
    setTimeout(() => {
      this.onopen?.(new Event('open'));
    }, 0);
  }

  simulateMessage(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) } as MessageEvent);
  }
}

// Save original WebSocket
const OriginalWebSocket = globalThis.WebSocket;

describe('useExecutionStatus', () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    // Replace WebSocket globally BEFORE hooks run
    (globalThis as any).WebSocket = MockWebSocket;
  });

  afterEach(() => {
    // Restore original
    (globalThis as any).WebSocket = OriginalWebSocket;
  });

  it('starts disconnected with empty statuses', () => {
    const { result } = renderHook(() =>
      useExecutionStatus('http://localhost:8080', 'token', 'wf-1'),
    );
    // Before async open fires
    expect(result.current.connected).toBe(false);
    expect(result.current.statuses.size).toBe(0);
  });

  it('sets connected to true after WebSocket opens', async () => {
    const { result } = renderHook(() =>
      useExecutionStatus('http://localhost:8080', 'token', 'wf-1'),
    );

    await waitFor(() => expect(result.current.connected).toBe(true));
    expect(MockWebSocket.instances.length).toBeGreaterThanOrEqual(1);
  });

  it('updates statuses on node_status messages', async () => {
    const { result } = renderHook(() =>
      useExecutionStatus('http://localhost:8080', 'token', 'wf-1'),
    );

    await waitFor(() => expect(result.current.connected).toBe(true));

    const ws = MockWebSocket.instances[MockWebSocket.instances.length - 1];
    await act(async () => {
      ws.simulateMessage({
        type: 'node_status',
        runID: 'run-1',
        nodeID: 'node-1',
        status: 'running',
      });
    });

    expect(result.current.statuses.size).toBe(1);
    expect(result.current.statuses.get('node-1')).toMatchObject({
      nodeID: 'node-1',
      status: 'running',
    });
  });

  it('disconnects WebSocket on unmount', async () => {
    const { result, unmount } = renderHook(() =>
      useExecutionStatus('http://localhost:8080', 'token', 'wf-1'),
    );

    await waitFor(() => expect(result.current.connected).toBe(true));
    const ws = MockWebSocket.instances[MockWebSocket.instances.length - 1];

    unmount();

    expect(ws.close).toHaveBeenCalled();
  });

  it('does not connect when params are empty', () => {
    renderHook(() => useExecutionStatus('', '', ''));
    expect(MockWebSocket.instances.length).toBe(0);
  });
});

