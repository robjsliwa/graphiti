import { GraphitiApiError, GraphitiNetworkError } from './errors.js';
import type {
  ClipboardCopyResponse,
  ClipboardPasteRequest,
  CommandRequest,
  CommandResponse,
  CreateWorkflowResponse,
  DeployRequest,
  DeployResponse,
  DeployStatusResponse,
  ExecutionRun,
  ExecutionRunSummary,
  GetWorkflowResponse,
  ListNodeDefinitionsResponse,
  ListWorkflowsResponse,
  NodeConfigResponse,
  SearchNodesResponse,
  UpdateAttributesResponse,
  ValidationResponse,
  Workflow,
  WorkflowState,
  WorkflowVersion,
} from './generated/types.js';

export interface GraphitiClientOptions {
  /** Base URL of the Graphiti server (e.g. "http://localhost:8080"). */
  baseUrl: string;
  /** Bearer token for API authentication. */
  token: string;
  /** Optional custom fetch implementation (for testing). */
  fetch?: typeof globalThis.fetch;
}

export class GraphitiClient {
  private readonly baseUrl: string;
  private readonly token: string;
  private readonly fetch: typeof globalThis.fetch;

  constructor(options: GraphitiClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, '');
    this.token = options.token;
    this.fetch = options.fetch ?? globalThis.fetch.bind(globalThis);
  }

  // --- Workflow CRUD ---

  async listWorkflows(): Promise<ListWorkflowsResponse> {
    return this.get<ListWorkflowsResponse>('/api/workflows');
  }

  async createWorkflow(body: { name?: string; description?: string } = {}): Promise<CreateWorkflowResponse> {
    return this.post<CreateWorkflowResponse>('/api/workflows', body);
  }

  async getWorkflow(id: string): Promise<GetWorkflowResponse> {
    return this.get<GetWorkflowResponse>(`/api/workflows/${enc(id)}/overview`);
  }

  async deleteWorkflow(id: string): Promise<void> {
    await this.requestNoContent('DELETE', `/api/workflows/${enc(id)}`);
  }

  async renameWorkflow(id: string, name: string): Promise<void> {
    await this.request('PATCH', `/api/workflows/${enc(id)}/name`, { name });
  }

  // --- Canvas State ---

  async getWorkflowState(id: string): Promise<WorkflowState> {
    return this.get<WorkflowState>(`/api/workflows/${enc(id)}/state`);
  }

  // --- Commands ---

  async executeCommand(workflowId: string, cmd: CommandRequest): Promise<CommandResponse> {
    return this.post<CommandResponse>(`/api/workflows/${enc(workflowId)}/commands`, cmd);
  }

  async undo(workflowId: string): Promise<CommandResponse> {
    return this.post<CommandResponse>(`/api/workflows/${enc(workflowId)}/undo`, {});
  }

  async redo(workflowId: string): Promise<CommandResponse> {
    return this.post<CommandResponse>(`/api/workflows/${enc(workflowId)}/redo`, {});
  }

  // --- Node Config ---

  async getNodeConfig(workflowId: string, nodeId: string): Promise<NodeConfigResponse> {
    return this.get<NodeConfigResponse>(
      `/api/workflows/${enc(workflowId)}/nodes/${enc(nodeId)}/config`,
    );
  }

  async updateAttributes(
    workflowId: string,
    nodeId: string,
    attributes: Record<string, unknown>,
  ): Promise<UpdateAttributesResponse> {
    return this.request<UpdateAttributesResponse>(
      'PATCH',
      `/api/workflows/${enc(workflowId)}/nodes/${enc(nodeId)}/attributes`,
      { attributes },
    );
  }

  // --- Search ---

  async searchNodes(query: string): Promise<SearchNodesResponse> {
    return this.get<SearchNodesResponse>(`/api/nodes/search?q=${encodeURIComponent(query)}`);
  }

  async listNodeDefinitions(): Promise<ListNodeDefinitionsResponse> {
    return this.get<ListNodeDefinitionsResponse>('/api/nodes');
  }

  // --- Clipboard ---

  async clipboardCopy(workflowId: string, nodeIds: string[]): Promise<ClipboardCopyResponse> {
    return this.post<ClipboardCopyResponse>(
      `/api/workflows/${enc(workflowId)}/clipboard/copy`,
      { nodeIds },
    );
  }

  async clipboardPaste(workflowId: string, req: ClipboardPasteRequest): Promise<CommandResponse> {
    return this.post<CommandResponse>(
      `/api/workflows/${enc(workflowId)}/clipboard/paste`,
      req,
    );
  }

  // --- Validation, Deploy, Export ---

  async validateWorkflow(id: string): Promise<ValidationResponse> {
    return this.post<ValidationResponse>(`/api/workflows/${enc(id)}/validate`, {});
  }

  async deployWorkflow(id: string, target?: string): Promise<DeployResponse> {
    const body: DeployRequest = target ? { target } : {};
    return this.post<DeployResponse>(`/api/workflows/${enc(id)}/deploy`, body);
  }

  async getDeployStatus(id: string): Promise<DeployStatusResponse> {
    return this.get<DeployStatusResponse>(`/api/workflows/${enc(id)}/deploy/status`);
  }

  async exportWorkflow(id: string, format: 'json' | 'yaml' = 'json'): Promise<ArrayBuffer> {
    const res = await this.rawRequest('GET', `/api/workflows/${enc(id)}/export?format=${format}`);
    return res.arrayBuffer();
  }

  async getVersionHistory(id: string): Promise<WorkflowVersion[]> {
    return this.get<WorkflowVersion[]>(`/api/workflows/${enc(id)}/versions`);
  }

  // --- Execution ---

  async listRuns(workflowId: string): Promise<ExecutionRunSummary[]> {
    return this.get<ExecutionRunSummary[]>(`/api/workflows/${enc(workflowId)}/runs`);
  }

  async getRun(runId: string): Promise<ExecutionRun> {
    return this.get<ExecutionRun>(`/api/runs/${enc(runId)}`);
  }

  // --- Internal helpers ---

  private async get<T>(path: string): Promise<T> {
    return this.request<T>('GET', path);
  }

  private async post<T>(path: string, body: unknown): Promise<T> {
    return this.request<T>('POST', path, body);
  }

  private async requestNoContent(method: string, path: string): Promise<void> {
    await this.rawRequest(method, path);
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
  ): Promise<T> {
    const res = await this.rawRequest(method, path, body);
    const json = await res.json();
    return json as T;
  }

  private async rawRequest(
    method: string,
    path: string,
    body?: unknown,
  ): Promise<Response> {
    const url = `${this.baseUrl}${path}`;
    const headers: Record<string, string> = {
      Accept: 'application/json',
      Authorization: `Bearer ${this.token}`,
    };

    let reqBody: string | undefined;
    if (body !== undefined) {
      headers['Content-Type'] = 'application/json';
      reqBody = JSON.stringify(body);
    }

    let res: Response;
    try {
      res = await this.fetch(url, { method, headers, body: reqBody });
    } catch (err) {
      throw new GraphitiNetworkError(
        `Network error: ${err instanceof Error ? err.message : String(err)}`,
        err instanceof Error ? err : undefined,
      );
    }

    if (!res.ok) {
      let code = res.status;
      let message = res.statusText;
      let detail: string | undefined;

      try {
        const errBody = await res.json();
        if (errBody?.error) {
          code = errBody.error.code ?? code;
          message = errBody.error.message ?? message;
          detail = errBody.error.detail;
        }
      } catch {
        // ignore parse errors on error responses
      }

      throw new GraphitiApiError(res.status, code, message, detail);
    }

    return res;
  }
}

function enc(s: string): string {
  if (!s) throw new Error('ID must not be empty');
  return encodeURIComponent(s);
}
