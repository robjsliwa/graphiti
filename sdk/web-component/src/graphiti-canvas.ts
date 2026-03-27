import { GraphitiClient, GraphitiWS } from '@graphiti/client';
import type {
  CommandRequest,
  CommandResponse,
  WorkflowState,
} from '@graphiti/client';
import { SVGRenderer } from './renderer/svg-renderer.js';
import { SelectionManager } from './interactions/selection.js';
import { ConnectionManager } from './interactions/connection.js';
import { KeyboardManager } from './interactions/keyboard.js';
import { canvasStyles } from './styles/canvas.css.js';

export class GraphitiCanvasElement extends HTMLElement {
  static observedAttributes = ['api-url', 'workflow-id', 'token', 'theme', 'read-only'];

  private client: GraphitiClient | null = null;
  private clientBaseUrl = '';
  private clientToken = '';
  private ws: GraphitiWS | null = null;
  private renderer: SVGRenderer | null = null;
  private selection: SelectionManager | null = null;
  private connection: ConnectionManager | null = null;
  private keyboard: KeyboardManager | null = null;
  private shadowInitialized = false;

  // --- Reflected properties ---

  get apiUrl(): string {
    return this.getAttribute('api-url') || '';
  }
  set apiUrl(value: string) {
    this.setAttribute('api-url', value);
  }

  get workflowId(): string {
    return this.getAttribute('workflow-id') || '';
  }
  set workflowId(value: string) {
    this.setAttribute('workflow-id', value);
  }

  get token(): string {
    return this.getAttribute('token') || '';
  }
  set token(value: string) {
    this.setAttribute('token', value);
  }

  get theme(): 'light' | 'dark' {
    return (this.getAttribute('theme') as 'light' | 'dark') || 'light';
  }
  set theme(value: 'light' | 'dark') {
    this.setAttribute('theme', value);
  }

  get readOnly(): boolean {
    return this.hasAttribute('read-only');
  }
  set readOnly(value: boolean) {
    if (value) this.setAttribute('read-only', '');
    else this.removeAttribute('read-only');
  }

  // --- Lifecycle ---

  connectedCallback(): void {
    if (!this.shadowInitialized) {
      const shadow = this.attachShadow({ mode: 'open' });
      const style = document.createElement('style');
      style.textContent = canvasStyles;
      shadow.appendChild(style);

      const root = document.createElement('div');
      root.setAttribute('class', 'canvas-root');
      root.setAttribute('tabindex', '0');
      shadow.appendChild(root);

      this.renderer = new SVGRenderer();
      root.appendChild(this.renderer.svg);

      this.setupInteractions(shadow);
      this.shadowInitialized = true;
    } else {
      // Re-attach interactions after being moved in the DOM
      this.reattachInteractions();
    }

    if (this.workflowId && this.apiUrl) {
      this.loadWorkflow();
    }
  }

  disconnectedCallback(): void {
    this.teardownInteractions();
    if (this.ws) {
      this.ws.disconnect();
      this.ws = null;
    }
    if (this.renderer) {
      this.renderer.destroy();
    }
  }

  attributeChangedCallback(name: string, oldValue: string | null, newValue: string | null): void {
    if (oldValue === newValue) return;

    if (name === 'workflow-id' && this.shadowInitialized && newValue) {
      this.loadWorkflow();
    }
  }

  // --- Public methods ---

  async loadWorkflow(): Promise<void> {
    if (!this.apiUrl || !this.workflowId) return;

    this.ensureClient();

    try {
      const state = await this.client!.getWorkflowState(this.workflowId);
      this.renderer!.renderState(state);
      if (this.renderer!.svg.querySelectorAll('.node').length > 0) {
        this.renderer!.zoomToFit();
      }
      this.emit('graphiti:workflow-changed', { workflow: state });
    } catch (err) {
      this.emit('graphiti:error', {
        message: err instanceof Error ? err.message : 'Failed to load workflow',
      });
    }
  }

  async executeCommand(cmd: CommandRequest): Promise<CommandResponse | null> {
    if (this.readOnly) return null;
    this.ensureClient();

    try {
      const result = await this.client!.executeCommand(this.workflowId, cmd);
      if (result.ok && result.workflow) {
        this.renderer!.renderState(result.workflow);
        this.emit('graphiti:workflow-changed', { workflow: result.workflow });
      }
      this.emit('graphiti:command-executed', { type: cmd.type, ok: result.ok });
      return result;
    } catch (err) {
      this.emit('graphiti:error', {
        message: err instanceof Error ? err.message : 'Command failed',
      });
      return null;
    }
  }

  async undo(): Promise<void> {
    if (this.readOnly) return;
    this.ensureClient();

    try {
      const result = await this.client!.undo(this.workflowId);
      if (result.ok && result.workflow) {
        this.renderer!.renderState(result.workflow);
        this.selection?.clearSelection();
        this.emit('graphiti:workflow-changed', { workflow: result.workflow });
      }
    } catch (err) {
      this.emit('graphiti:error', {
        message: err instanceof Error ? err.message : 'Undo failed',
      });
    }
  }

  async redo(): Promise<void> {
    if (this.readOnly) return;
    this.ensureClient();

    try {
      const result = await this.client!.redo(this.workflowId);
      if (result.ok && result.workflow) {
        this.renderer!.renderState(result.workflow);
        this.selection?.clearSelection();
        this.emit('graphiti:workflow-changed', { workflow: result.workflow });
      }
    } catch (err) {
      this.emit('graphiti:error', {
        message: err instanceof Error ? err.message : 'Redo failed',
      });
    }
  }

  async exportJSON(): Promise<WorkflowState | null> {
    this.ensureClient();
    try {
      return await this.client!.getWorkflowState(this.workflowId);
    } catch (err) {
      this.emit('graphiti:error', {
        message: err instanceof Error ? err.message : 'Export failed',
      });
      return null;
    }
  }

  zoomToFit(): void {
    this.renderer?.zoomToFit();
  }

  // --- Private helpers ---

  private ensureClient(): void {
    if (!this.client || this.clientBaseUrl !== this.apiUrl || this.clientToken !== this.token) {
      this.clientBaseUrl = this.apiUrl;
      this.clientToken = this.token;
      this.client = new GraphitiClient({
        baseUrl: this.apiUrl,
        token: this.token,
      });
    }
  }

  private setupInteractions(shadow: ShadowRoot): void {
    if (!this.renderer) return;

    this.selection = new SelectionManager(this.renderer, {
      onSelect: (nodeIds) => {
        if (nodeIds.length === 1) {
          this.emit('graphiti:node-selected', { nodeId: nodeIds[0] });
        }
      },
      onDeselect: () => {
        this.emit('graphiti:node-deselected', {});
      },
      onDeleteNodes: async (nodeIds) => {
        for (const id of nodeIds) {
          await this.executeCommand({ type: 'remove_node', nodeId: id });
        }
      },
      onDeleteEdges: async (edgeIds) => {
        for (const id of edgeIds) {
          await this.executeCommand({ type: 'remove_edge', edgeId: id });
        }
      },
      onDragEnd: (nodeId, x, y) => {
        const el = this.renderer!.svg.querySelector(`[data-node-id="${nodeId}"]`);
        if (!el) return;
        const orig = this.renderer!.getNodePosition(el);
        this.executeCommand({
          type: 'move_node',
          nodeId,
          fromX: orig.x,
          fromY: orig.y,
          toX: x,
          toY: y,
        });
      },
      onMultiDragEnd: (moves) => {
        // For multi-drag we need the from positions which are the current
        // positions before the command
        const from = moves.map((m) => {
          const el = this.renderer!.svg.querySelector(`[data-node-id="${m.nodeId}"]`);
          const pos = el ? this.renderer!.getNodePosition(el) : { x: m.x, y: m.y };
          return { nodeId: m.nodeId, x: pos.x, y: pos.y };
        });
        this.executeCommand({
          type: 'move_nodes',
          from,
          to: moves,
        });
      },
    });

    this.connection = new ConnectionManager(this.renderer, {
      onConnect: (source, target) => {
        this.executeCommand({
          type: 'add_edge',
          sourceNodeId: source.nodeId,
          sourcePortId: source.portId,
          targetNodeId: target.nodeId,
          targetPortId: target.portId,
        });
      },
    });

    this.keyboard = new KeyboardManager({
      onUndo: () => this.undo(),
      onRedo: () => this.redo(),
      onZoomIn: () => {
        const rect = this.renderer!.svg.getBoundingClientRect();
        this.renderer!.viewport.zoom(1, rect.left + rect.width / 2, rect.top + rect.height / 2, rect);
      },
      onZoomOut: () => {
        const rect = this.renderer!.svg.getBoundingClientRect();
        this.renderer!.viewport.zoom(-1, rect.left + rect.width / 2, rect.top + rect.height / 2, rect);
      },
      onZoomToFit: () => this.zoomToFit(),
    });

    this.selection.attach(shadow);
    this.connection.attach();
    this.keyboard.attach(shadow);
  }

  private reattachInteractions(): void {
    const shadow = this.shadowRoot;
    if (!shadow || !this.renderer) return;
    // Re-bind renderer events
    this.renderer = new SVGRenderer();
    const root = shadow.querySelector('.canvas-root');
    if (root) {
      const oldSvg = root.querySelector('svg');
      if (oldSvg) oldSvg.remove();
      root.appendChild(this.renderer.svg);
    }
    this.setupInteractions(shadow);
  }

  private teardownInteractions(): void {
    const shadow = this.shadowRoot;
    if (!shadow) return;
    this.selection?.detach(shadow);
    this.connection?.detach();
    this.keyboard?.detach(shadow);
  }

  private emit(name: string, detail: Record<string, unknown>): void {
    this.dispatchEvent(
      new CustomEvent(name, {
        detail,
        bubbles: true,
        composed: true,
      }),
    );
  }
}

// Register the custom element
if (typeof customElements !== 'undefined' && !customElements.get('graphiti-canvas')) {
  customElements.define('graphiti-canvas', GraphitiCanvasElement);
}
