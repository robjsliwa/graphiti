"use strict";
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

// src/index.ts
var index_exports = {};
__export(index_exports, {
  GraphitiApiError: () => GraphitiApiError,
  GraphitiClient: () => GraphitiClient,
  GraphitiNetworkError: () => GraphitiNetworkError,
  GraphitiWS: () => GraphitiWS
});
module.exports = __toCommonJS(index_exports);

// src/errors.ts
var GraphitiApiError = class extends Error {
  /** HTTP status code. */
  status;
  /** Error code from the JSON envelope (mirrors status). */
  code;
  /** Optional detail string from the API. */
  detail;
  constructor(status, code, message, detail) {
    super(message);
    this.name = "GraphitiApiError";
    this.status = status;
    this.code = code;
    this.detail = detail;
  }
};
var GraphitiNetworkError = class extends Error {
  cause;
  constructor(message, cause) {
    super(message);
    this.name = "GraphitiNetworkError";
    this.cause = cause;
  }
};

// src/client.ts
var GraphitiClient = class {
  baseUrl;
  token;
  fetch;
  constructor(options) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, "");
    this.token = options.token;
    this.fetch = options.fetch ?? globalThis.fetch.bind(globalThis);
  }
  // --- Workflow CRUD ---
  async listWorkflows() {
    return this.get("/api/workflows");
  }
  async createWorkflow(body = {}) {
    return this.post("/api/workflows", body);
  }
  async getWorkflow(id) {
    return this.get(`/api/workflows/${enc(id)}/overview`);
  }
  async deleteWorkflow(id) {
    await this.requestNoContent("DELETE", `/api/workflows/${enc(id)}`);
  }
  async renameWorkflow(id, name) {
    await this.request("PATCH", `/api/workflows/${enc(id)}/name`, { name });
  }
  // --- Canvas State ---
  async getWorkflowState(id) {
    return this.get(`/api/workflows/${enc(id)}/state`);
  }
  // --- Commands ---
  async executeCommand(workflowId, cmd) {
    return this.post(`/api/workflows/${enc(workflowId)}/commands`, cmd);
  }
  async undo(workflowId) {
    return this.post(`/api/workflows/${enc(workflowId)}/undo`, {});
  }
  async redo(workflowId) {
    return this.post(`/api/workflows/${enc(workflowId)}/redo`, {});
  }
  // --- Node Config ---
  async getNodeConfig(workflowId, nodeId) {
    return this.get(
      `/api/workflows/${enc(workflowId)}/nodes/${enc(nodeId)}/config`
    );
  }
  async updateAttributes(workflowId, nodeId, attributes) {
    return this.request(
      "PATCH",
      `/api/workflows/${enc(workflowId)}/nodes/${enc(nodeId)}/attributes`,
      { attributes }
    );
  }
  // --- Search ---
  async searchNodes(query) {
    return this.get(`/api/nodes/search?q=${encodeURIComponent(query)}`);
  }
  async listNodeDefinitions() {
    return this.get("/api/nodes");
  }
  // --- Clipboard ---
  async clipboardCopy(workflowId, nodeIds) {
    return this.post(
      `/api/workflows/${enc(workflowId)}/clipboard/copy`,
      { nodeIds }
    );
  }
  async clipboardPaste(workflowId, req) {
    return this.post(
      `/api/workflows/${enc(workflowId)}/clipboard/paste`,
      req
    );
  }
  // --- Validation, Deploy, Export ---
  async validateWorkflow(id) {
    return this.post(`/api/workflows/${enc(id)}/validate`, {});
  }
  async deployWorkflow(id, target) {
    const body = target ? { target } : {};
    return this.post(`/api/workflows/${enc(id)}/deploy`, body);
  }
  async getDeployStatus(id) {
    return this.get(`/api/workflows/${enc(id)}/deploy/status`);
  }
  async exportWorkflow(id, format = "json") {
    const res = await this.rawRequest("GET", `/api/workflows/${enc(id)}/export?format=${format}`);
    return res.arrayBuffer();
  }
  async getVersionHistory(id) {
    return this.get(`/api/workflows/${enc(id)}/versions`);
  }
  // --- Execution ---
  async listRuns(workflowId) {
    return this.get(`/api/workflows/${enc(workflowId)}/runs`);
  }
  async getRun(runId) {
    return this.get(`/api/runs/${enc(runId)}`);
  }
  // --- Internal helpers ---
  async get(path) {
    return this.request("GET", path);
  }
  async post(path, body) {
    return this.request("POST", path, body);
  }
  async requestNoContent(method, path) {
    await this.rawRequest(method, path);
  }
  async request(method, path, body) {
    const res = await this.rawRequest(method, path, body);
    const json = await res.json();
    return json;
  }
  async rawRequest(method, path, body) {
    const url = `${this.baseUrl}${path}`;
    const headers = {
      Accept: "application/json",
      Authorization: `Bearer ${this.token}`
    };
    let reqBody;
    if (body !== void 0) {
      headers["Content-Type"] = "application/json";
      reqBody = JSON.stringify(body);
    }
    let res;
    try {
      res = await this.fetch(url, { method, headers, body: reqBody });
    } catch (err) {
      throw new GraphitiNetworkError(
        `Network error: ${err instanceof Error ? err.message : String(err)}`,
        err instanceof Error ? err : void 0
      );
    }
    if (!res.ok) {
      let code = res.status;
      let message = res.statusText;
      let detail;
      try {
        const errBody = await res.json();
        if (errBody?.error) {
          code = errBody.error.code ?? code;
          message = errBody.error.message ?? message;
          detail = errBody.error.detail;
        }
      } catch {
      }
      throw new GraphitiApiError(res.status, code, message, detail);
    }
    return res;
  }
};
function enc(s) {
  if (!s) throw new Error("ID must not be empty");
  return encodeURIComponent(s);
}

// src/ws.ts
var GraphitiWS = class {
  baseUrl;
  token;
  workflowId;
  autoReconnect;
  maxReconnectAttempts;
  WS;
  ws = null;
  reconnectAttempts = 0;
  reconnectTimer = null;
  intentionalClose = false;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  listeners = /* @__PURE__ */ new Map();
  constructor(options) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, "");
    this.token = options.token;
    this.workflowId = options.workflowId;
    this.autoReconnect = options.autoReconnect ?? true;
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 10;
    this.WS = options.WebSocket ?? globalThis.WebSocket;
  }
  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, /* @__PURE__ */ new Set());
    }
    this.listeners.get(event).add(callback);
  }
  off(event, callback) {
    this.listeners.get(event)?.delete(callback);
  }
  connect() {
    this.intentionalClose = false;
    this.reconnectAttempts = 0;
    this.createConnection();
  }
  disconnect() {
    this.intentionalClose = true;
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close(1e3, "client disconnect");
      this.ws = null;
    }
  }
  createConnection() {
    const wsUrl = this.baseUrl.replace(/^http/, "ws");
    const url = `${wsUrl}/api/ws/workflows/${encodeURIComponent(this.workflowId)}?token=${encodeURIComponent(this.token)}`;
    this.ws = new this.WS(url);
    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.emit("connected", void 0);
    };
    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(String(event.data));
        if (data.type === "node_status") {
          this.emit("node_status", data);
        }
      } catch {
      }
    };
    this.ws.onclose = (event) => {
      this.emit("disconnected", { code: event.code, reason: event.reason });
      this.ws = null;
      if (!this.intentionalClose && this.autoReconnect) {
        this.scheduleReconnect();
      }
    };
    this.ws.onerror = () => {
      this.emit("error", new Error("WebSocket error"));
    };
  }
  scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.emit("error", new Error(`Max reconnect attempts (${this.maxReconnectAttempts}) exceeded`));
      return;
    }
    const delay = Math.min(1e3 * Math.pow(2, this.reconnectAttempts), 3e4);
    this.reconnectAttempts++;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (this.intentionalClose) return;
      this.createConnection();
    }, delay);
  }
  emit(event, data) {
    const set = this.listeners.get(event);
    if (set) {
      for (const fn of set) {
        try {
          fn(data);
        } catch {
        }
      }
    }
  }
};
// Annotate the CommonJS export names for ESM import in node:
0 && (module.exports = {
  GraphitiApiError,
  GraphitiClient,
  GraphitiNetworkError,
  GraphitiWS
});
//# sourceMappingURL=index.cjs.map