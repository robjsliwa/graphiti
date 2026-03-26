import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { GraphitiWS } from '../src/ws.js';

// Minimal mock WebSocket that simulates the browser WebSocket API.
class MockWebSocket {
  static instances: MockWebSocket[] = [];

  url: string;
  onopen: ((ev: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  readyState = 0; // CONNECTING

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  close(code?: number, reason?: string) {
    this.readyState = 3; // CLOSED
    if (this.onclose) {
      this.onclose({ code: code ?? 1000, reason: reason ?? '' } as CloseEvent);
    }
  }

  // Test helpers
  simulateOpen() {
    this.readyState = 1; // OPEN
    if (this.onopen) this.onopen(new Event('open'));
  }

  simulateMessage(data: unknown) {
    if (this.onmessage) {
      this.onmessage({ data: JSON.stringify(data) } as MessageEvent);
    }
  }

  simulateClose(code = 1006, reason = 'abnormal') {
    this.readyState = 3;
    if (this.onclose) {
      this.onclose({ code, reason } as CloseEvent);
    }
  }

  simulateError() {
    if (this.onerror) this.onerror(new Event('error'));
  }
}

function createWS(opts: Partial<ConstructorParameters<typeof GraphitiWS>[0]> = {}): GraphitiWS {
  return new GraphitiWS({
    baseUrl: 'http://localhost:8080',
    token: 'test-token',
    workflowId: 'wf-1',
    WebSocket: MockWebSocket as unknown as typeof WebSocket,
    ...opts,
  });
}

describe('GraphitiWS', () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('connect', () => {
    it('creates WebSocket with correct URL', () => {
      const ws = createWS();
      ws.connect();
      expect(MockWebSocket.instances).toHaveLength(1);
      expect(MockWebSocket.instances[0].url).toBe(
        'ws://localhost:8080/api/ws/workflows/wf-1?token=test-token',
      );
    });

    it('converts https to wss', () => {
      const ws = createWS({ baseUrl: 'https://graphiti.example.com' });
      ws.connect();
      expect(MockWebSocket.instances[0].url.startsWith('wss://')).toBe(true);
    });

    it('fires connected event on open', () => {
      const ws = createWS();
      const handler = vi.fn();
      ws.on('connected', handler);
      ws.connect();
      MockWebSocket.instances[0].simulateOpen();
      expect(handler).toHaveBeenCalledOnce();
    });
  });

  describe('node_status events', () => {
    it('dispatches typed node_status events', () => {
      const ws = createWS();
      const handler = vi.fn();
      ws.on('node_status', handler);
      ws.connect();
      MockWebSocket.instances[0].simulateOpen();

      MockWebSocket.instances[0].simulateMessage({
        type: 'node_status',
        runID: 'run-1',
        nodeID: 'n-1',
        status: 'running',
        startedAt: '2026-01-01T00:00:00Z',
      });

      expect(handler).toHaveBeenCalledOnce();
      expect(handler).toHaveBeenCalledWith(expect.objectContaining({
        type: 'node_status',
        runID: 'run-1',
        nodeID: 'n-1',
        status: 'running',
      }));
    });

    it('ignores non-node_status messages', () => {
      const ws = createWS();
      const handler = vi.fn();
      ws.on('node_status', handler);
      ws.connect();
      MockWebSocket.instances[0].simulateMessage({ type: 'unknown_event' });
      expect(handler).not.toHaveBeenCalled();
    });

    it('ignores malformed JSON', () => {
      const ws = createWS();
      const handler = vi.fn();
      ws.on('node_status', handler);
      ws.connect();
      // Directly send bad data
      const mock = MockWebSocket.instances[0];
      if (mock.onmessage) {
        mock.onmessage({ data: 'not json{' } as MessageEvent);
      }
      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe('disconnect', () => {
    it('fires disconnected event', () => {
      const ws = createWS();
      const handler = vi.fn();
      ws.on('disconnected', handler);
      ws.connect();
      MockWebSocket.instances[0].simulateOpen();
      ws.disconnect();
      expect(handler).toHaveBeenCalledWith({ code: 1000, reason: 'client disconnect' });
    });

    it('stops auto-reconnect', () => {
      const ws = createWS();
      ws.connect();
      MockWebSocket.instances[0].simulateOpen();
      ws.disconnect();

      // Advance timers - should not create new connections
      vi.advanceTimersByTime(60000);
      expect(MockWebSocket.instances).toHaveLength(1);
    });
  });

  describe('auto-reconnect', () => {
    it('reconnects with exponential backoff', () => {
      const ws = createWS();
      ws.connect();
      MockWebSocket.instances[0].simulateOpen();

      // Simulate abnormal close (not intentional)
      MockWebSocket.instances[0].simulateClose(1006, 'abnormal');
      expect(MockWebSocket.instances).toHaveLength(1); // no reconnect yet

      // First retry: 1s
      vi.advanceTimersByTime(1000);
      expect(MockWebSocket.instances).toHaveLength(2);

      // Second abnormal close
      MockWebSocket.instances[1].simulateClose(1006, 'abnormal');

      // Second retry: 2s
      vi.advanceTimersByTime(1500);
      expect(MockWebSocket.instances).toHaveLength(2); // not yet
      vi.advanceTimersByTime(500);
      expect(MockWebSocket.instances).toHaveLength(3);
    });

    it('resets attempt counter on successful connect', () => {
      const ws = createWS();
      ws.connect();

      // Close -> reconnect -> open -> close -> should retry from 1s again
      MockWebSocket.instances[0].simulateClose(1006, 'fail');
      vi.advanceTimersByTime(1000);
      expect(MockWebSocket.instances).toHaveLength(2);

      MockWebSocket.instances[1].simulateOpen(); // successful
      MockWebSocket.instances[1].simulateClose(1006, 'fail');

      // Should be 1s again (not 2s), since counter reset
      vi.advanceTimersByTime(1000);
      expect(MockWebSocket.instances).toHaveLength(3);
    });

    it('stops after maxReconnectAttempts', () => {
      const ws = createWS({ maxReconnectAttempts: 2 });
      const errorHandler = vi.fn();
      ws.on('error', errorHandler);
      ws.connect();

      // Close -> attempt 1
      MockWebSocket.instances[0].simulateClose(1006, 'fail');
      vi.advanceTimersByTime(1000);
      expect(MockWebSocket.instances).toHaveLength(2);

      // Close -> attempt 2
      MockWebSocket.instances[1].simulateClose(1006, 'fail');
      vi.advanceTimersByTime(2000);
      expect(MockWebSocket.instances).toHaveLength(3);

      // Close -> no more attempts
      MockWebSocket.instances[2].simulateClose(1006, 'fail');
      vi.advanceTimersByTime(30000);
      expect(MockWebSocket.instances).toHaveLength(3); // no new connections
      expect(errorHandler).toHaveBeenCalledWith(
        expect.objectContaining({ message: expect.stringContaining('Max reconnect') }),
      );
    });

    it('does not reconnect when autoReconnect is false', () => {
      const ws = createWS({ autoReconnect: false });
      ws.connect();
      MockWebSocket.instances[0].simulateClose(1006, 'fail');
      vi.advanceTimersByTime(30000);
      expect(MockWebSocket.instances).toHaveLength(1);
    });
  });

  describe('event listeners', () => {
    it('supports multiple listeners on the same event', () => {
      const ws = createWS();
      const h1 = vi.fn();
      const h2 = vi.fn();
      ws.on('connected', h1);
      ws.on('connected', h2);
      ws.connect();
      MockWebSocket.instances[0].simulateOpen();
      expect(h1).toHaveBeenCalledOnce();
      expect(h2).toHaveBeenCalledOnce();
    });

    it('off removes a specific listener', () => {
      const ws = createWS();
      const h1 = vi.fn();
      const h2 = vi.fn();
      ws.on('connected', h1);
      ws.on('connected', h2);
      ws.off('connected', h1);
      ws.connect();
      MockWebSocket.instances[0].simulateOpen();
      expect(h1).not.toHaveBeenCalled();
      expect(h2).toHaveBeenCalledOnce();
    });
  });

  describe('error event', () => {
    it('fires error event on WebSocket error', () => {
      const ws = createWS();
      const handler = vi.fn();
      ws.on('error', handler);
      ws.connect();
      MockWebSocket.instances[0].simulateError();
      expect(handler).toHaveBeenCalledWith(expect.any(Error));
    });
  });

  describe('listener exception safety', () => {
    it('a throwing listener does not prevent other listeners from firing', () => {
      const ws = createWS();
      const h1 = vi.fn(() => { throw new Error('bad listener'); });
      const h2 = vi.fn();
      ws.on('connected', h1);
      ws.on('connected', h2);
      ws.connect();
      MockWebSocket.instances[0].simulateOpen();
      expect(h1).toHaveBeenCalledOnce();
      expect(h2).toHaveBeenCalledOnce();
    });
  });
});
