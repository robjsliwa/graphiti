import type { NodeStatusEvent } from './generated/types.js';

export interface GraphitiWSOptions {
  /** Base URL of the Graphiti server (http/https -- converted to ws/wss). */
  baseUrl: string;
  /**
   * Bearer token for authentication.
   *
   * Note: The token is sent as a query parameter in the WebSocket URL because
   * the browser WebSocket API does not support custom headers. This means the
   * token may appear in server access logs and proxy logs. For production use,
   * consider using short-lived tokens or a ticket-exchange mechanism.
   */
  token: string;
  /** Workflow ID to subscribe to. */
  workflowId: string;
  /** Auto-reconnect on disconnect. Default: true. */
  autoReconnect?: boolean;
  /** Max reconnect attempts. Default: 10. */
  maxReconnectAttempts?: number;
  /** Custom WebSocket constructor (for testing). */
  WebSocket?: typeof globalThis.WebSocket;
}

export type WSEventMap = {
  node_status: NodeStatusEvent;
  connected: void;
  disconnected: { code: number; reason: string };
  error: Error;
};

type Listener<K extends keyof WSEventMap> = (data: WSEventMap[K]) => void;

export class GraphitiWS {
  private readonly baseUrl: string;
  private readonly token: string;
  private readonly workflowId: string;
  private readonly autoReconnect: boolean;
  private readonly maxReconnectAttempts: number;
  private readonly WS: typeof globalThis.WebSocket;

  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private intentionalClose = false;

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private listeners: Map<keyof WSEventMap, Set<Listener<any>>> = new Map();

  constructor(options: GraphitiWSOptions) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, '');
    this.token = options.token;
    this.workflowId = options.workflowId;
    this.autoReconnect = options.autoReconnect ?? true;
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 10;
    this.WS = options.WebSocket ?? globalThis.WebSocket;
  }

  on<K extends keyof WSEventMap>(event: K, callback: Listener<K>): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(callback);
  }

  off<K extends keyof WSEventMap>(event: K, callback: Listener<K>): void {
    this.listeners.get(event)?.delete(callback);
  }

  connect(): void {
    this.intentionalClose = false;
    this.reconnectAttempts = 0;
    this.createConnection();
  }

  disconnect(): void {
    this.intentionalClose = true;
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close(1000, 'client disconnect');
      this.ws = null;
    }
  }

  private createConnection(): void {
    const wsUrl = this.baseUrl
      .replace(/^http/, 'ws');
    const url = `${wsUrl}/api/ws/workflows/${encodeURIComponent(this.workflowId)}?token=${encodeURIComponent(this.token)}`;

    this.ws = new this.WS(url);

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.emit('connected', undefined as void);
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(String(event.data));
        if (data.type === 'node_status') {
          this.emit('node_status', data as NodeStatusEvent);
        }
      } catch {
        // ignore malformed messages
      }
    };

    this.ws.onclose = (event) => {
      this.emit('disconnected', { code: event.code, reason: event.reason });
      this.ws = null;

      if (!this.intentionalClose && this.autoReconnect) {
        this.scheduleReconnect();
      }
    };

    this.ws.onerror = () => {
      this.emit('error', new Error('WebSocket error'));
    };
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.emit('error', new Error(`Max reconnect attempts (${this.maxReconnectAttempts}) exceeded`));
      return;
    }

    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts++;

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (this.intentionalClose) return;
      this.createConnection();
    }, delay);
  }

  private emit<K extends keyof WSEventMap>(event: K, data: WSEventMap[K]): void {
    const set = this.listeners.get(event);
    if (set) {
      for (const fn of set) {
        try {
          fn(data);
        } catch {
          // Prevent one bad listener from breaking event dispatch
        }
      }
    }
  }
}
