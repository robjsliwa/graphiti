import { describe, it, expect, vi, beforeEach } from 'vitest';
import { GraphitiClient } from '../src/client.js';
import { GraphitiApiError, GraphitiNetworkError } from '../src/errors.js';

function mockFetch(status: number, body: unknown, headers: Record<string, string> = {}): typeof globalThis.fetch {
  return vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? 'OK' : 'Error',
    headers: new Headers(headers),
    json: () => Promise.resolve(body),
    blob: () => Promise.resolve(new Blob([JSON.stringify(body)])),
  } as Response);
}

function errorFetch(status: number, message: string, detail?: string): typeof globalThis.fetch {
  return mockFetch(status, { error: { code: status, message, detail } });
}

function client(fetch: typeof globalThis.fetch): GraphitiClient {
  return new GraphitiClient({ baseUrl: 'http://localhost:8080', token: 'test-token', fetch });
}

describe('GraphitiClient', () => {
  describe('constructor', () => {
    it('strips trailing slashes from baseUrl', () => {
      const fetch = mockFetch(200, { items: [] });
      const c = new GraphitiClient({ baseUrl: 'http://localhost:8080/', token: 't', fetch });
      c.listWorkflows();
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/workflows',
        expect.anything(),
      );
    });
  });

  describe('request headers', () => {
    it('sets Authorization and Accept headers on all requests', async () => {
      const fetch = mockFetch(200, { items: [] });
      const c = client(fetch);
      await c.listWorkflows();

      expect(fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: 'Bearer test-token',
            Accept: 'application/json',
          }),
        }),
      );
    });

    it('sets Content-Type on POST requests', async () => {
      const fetch = mockFetch(201, { workflow: { ID: 'wf-1' } });
      const c = client(fetch);
      await c.createWorkflow({ name: 'Test' });

      expect(fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
          body: JSON.stringify({ name: 'Test' }),
        }),
      );
    });
  });

  describe('listWorkflows', () => {
    it('returns items array', async () => {
      const data = { items: [{ ID: 'wf-1', Name: 'Test', Status: 'draft', Version: 1, UpdatedAt: '2026-01-01T00:00:00Z' }] };
      const c = client(mockFetch(200, data));
      const result = await c.listWorkflows();
      expect(result.items).toHaveLength(1);
      expect(result.items[0].ID).toBe('wf-1');
    });
  });

  describe('createWorkflow', () => {
    it('returns created workflow with 201', async () => {
      const data = { workflow: { ID: 'wf-1', Name: 'New' } };
      const c = client(mockFetch(201, data));
      const result = await c.createWorkflow({ name: 'New' });
      expect(result.workflow.Name).toBe('New');
    });

    it('defaults name to empty body when not provided', async () => {
      const fetch = mockFetch(201, { workflow: { ID: 'wf-1' } });
      const c = client(fetch);
      await c.createWorkflow();
      expect(fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ body: '{}' }),
      );
    });
  });

  describe('getWorkflow', () => {
    it('returns workflow and nodeDefinitions', async () => {
      const data = { workflow: { ID: 'wf-1' }, nodeDefinitions: [{ ID: 'api-gw' }] };
      const c = client(mockFetch(200, data));
      const result = await c.getWorkflow('wf-1');
      expect(result.nodeDefinitions).toHaveLength(1);
    });

    it('throws GraphitiApiError on 404', async () => {
      const c = client(errorFetch(404, 'workflow not found'));
      await expect(c.getWorkflow('missing')).rejects.toThrow(GraphitiApiError);
      await expect(c.getWorkflow('missing')).rejects.toMatchObject({
        status: 404,
        code: 404,
        message: 'workflow not found',
      });
    });
  });

  describe('deleteWorkflow', () => {
    it('resolves on 204', async () => {
      const fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 204,
        json: () => Promise.resolve(undefined),
      } as unknown as Response);
      const c = client(fetch);
      await expect(c.deleteWorkflow('wf-1')).resolves.toBeUndefined();
    });

    it('throws on 404', async () => {
      const c = client(errorFetch(404, 'workflow not found'));
      await expect(c.deleteWorkflow('missing')).rejects.toThrow(GraphitiApiError);
    });
  });

  describe('getWorkflowState', () => {
    it('returns nodes and edges', async () => {
      const state = { nodes: [{ id: 'n-1' }], edges: [{ id: 'e-1' }] };
      const c = client(mockFetch(200, state));
      const result = await c.getWorkflowState('wf-1');
      expect(result.nodes).toHaveLength(1);
      expect(result.edges).toHaveLength(1);
    });
  });

  describe('executeCommand', () => {
    it('sends command and returns response', async () => {
      const resp = { ok: true, canUndo: true, canRedo: false, workflow: { nodes: [], edges: [] } };
      const c = client(mockFetch(200, resp));
      const result = await c.executeCommand('wf-1', {
        type: 'add_node',
        definitionId: 'api-gw',
        x: 100,
        y: 200,
      });
      expect(result.ok).toBe(true);
      expect(result.canUndo).toBe(true);
    });
  });

  describe('undo / redo', () => {
    it('undo sends POST and returns command response', async () => {
      const resp = { ok: true, canUndo: false, canRedo: true };
      const fetch = mockFetch(200, resp);
      const c = client(fetch);
      const result = await c.undo('wf-1');
      expect(result.canRedo).toBe(true);
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/workflows/wf-1/undo',
        expect.objectContaining({ method: 'POST' }),
      );
    });

    it('redo sends POST', async () => {
      const fetch = mockFetch(200, { ok: true, canUndo: true, canRedo: false });
      const c = client(fetch);
      await c.redo('wf-1');
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/workflows/wf-1/redo',
        expect.objectContaining({ method: 'POST' }),
      );
    });
  });

  describe('getNodeConfig', () => {
    it('returns node and definition', async () => {
      const data = {
        node: { id: 'n-1', definitionId: 'api-gw', label: 'GW', x: 0, y: 0, attributes: {} },
        definition: { icon: 'globe', shape: {}, category: {}, inputs: [], outputs: [], attributes: [] },
      };
      const c = client(mockFetch(200, data));
      const result = await c.getNodeConfig('wf-1', 'n-1');
      expect(result.node.id).toBe('n-1');
    });
  });

  describe('updateAttributes', () => {
    it('sends PATCH with attributes object', async () => {
      const fetch = mockFetch(200, { node: { id: 'n-1', attributes: { port: 9000 } } });
      const c = client(fetch);
      await c.updateAttributes('wf-1', 'n-1', { port: 9000 });
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/workflows/wf-1/nodes/n-1/attributes',
        expect.objectContaining({
          method: 'PATCH',
          body: JSON.stringify({ attributes: { port: 9000 } }),
        }),
      );
    });
  });

  describe('searchNodes', () => {
    it('encodes query parameter', async () => {
      const fetch = mockFetch(200, { items: [] });
      const c = client(fetch);
      await c.searchNodes('api gateway');
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/nodes/search?q=api%20gateway',
        expect.anything(),
      );
    });
  });

  describe('clipboard', () => {
    it('copy sends nodeIds and returns payload', async () => {
      const resp = { ok: true, payload: { version: 1, source: 'canvas', nodes: [], edges: [] } };
      const c = client(mockFetch(200, resp));
      const result = await c.clipboardCopy('wf-1', ['n-1', 'n-2']);
      expect(result.ok).toBe(true);
    });

    it('paste sends payload with position', async () => {
      const fetch = mockFetch(200, { ok: true, canUndo: true, canRedo: false });
      const c = client(fetch);
      await c.clipboardPaste('wf-1', {
        payload: { version: 1, source: 'canvas', nodes: [], edges: [] },
        x: 300,
        y: 400,
      });
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/clipboard/paste'),
        expect.anything(),
      );
    });
  });

  describe('validateWorkflow', () => {
    it('returns validation results', async () => {
      const data = { valid: true, results: [], summary: { errors: 0, warnings: 0, info: 0 } };
      const c = client(mockFetch(200, data));
      const result = await c.validateWorkflow('wf-1');
      expect(result.valid).toBe(true);
    });
  });

  describe('deployWorkflow', () => {
    it('sends deploy request with target', async () => {
      const fetch = mockFetch(200, { success: true, message: 'ok' });
      const c = client(fetch);
      await c.deployWorkflow('wf-1', 'production');
      expect(fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ body: JSON.stringify({ target: 'production' }) }),
      );
    });

    it('sends empty body when no target', async () => {
      const fetch = mockFetch(200, { success: true, message: 'ok' });
      const c = client(fetch);
      await c.deployWorkflow('wf-1');
      expect(fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ body: '{}' }),
      );
    });
  });

  describe('getDeployStatus', () => {
    it('returns verification status', async () => {
      const data = { workflowId: 'wf-1', version: 5, verification: 'verified', message: '', checkedAt: '' };
      const c = client(mockFetch(200, data));
      const result = await c.getDeployStatus('wf-1');
      expect(result.verification).toBe('verified');
    });
  });

  describe('getVersionHistory', () => {
    it('returns array of versions', async () => {
      const data = [{ id: 'v-1', version: 1, deployedAt: '', deployedBy: 'user-1' }];
      const c = client(mockFetch(200, data));
      const result = await c.getVersionHistory('wf-1');
      expect(result).toHaveLength(1);
    });
  });

  describe('listRuns', () => {
    it('returns execution run summaries', async () => {
      const data = [{ id: 'run-1', workflowId: 'wf-1', status: 'completed' }];
      const c = client(mockFetch(200, data));
      const result = await c.listRuns('wf-1');
      expect(result).toHaveLength(1);
    });
  });

  describe('getRun', () => {
    it('returns full execution run', async () => {
      const data = { id: 'run-1', status: 'completed', nodeStatuses: {} };
      const c = client(mockFetch(200, data));
      const result = await c.getRun('run-1');
      expect(result.id).toBe('run-1');
    });
  });

  describe('error handling', () => {
    it('throws GraphitiApiError with parsed envelope on non-2xx', async () => {
      const c = client(errorFetch(400, 'bad request', 'name is required'));
      try {
        await c.listWorkflows();
        expect.fail('should have thrown');
      } catch (err) {
        expect(err).toBeInstanceOf(GraphitiApiError);
        const apiErr = err as GraphitiApiError;
        expect(apiErr.status).toBe(400);
        expect(apiErr.code).toBe(400);
        expect(apiErr.message).toBe('bad request');
        expect(apiErr.detail).toBe('name is required');
      }
    });

    it('throws GraphitiApiError even when error body is unparseable', async () => {
      const fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: () => Promise.reject(new Error('not json')),
      } as unknown as Response);
      const c = client(fetch);
      await expect(c.listWorkflows()).rejects.toMatchObject({
        status: 500,
        message: 'Internal Server Error',
      });
    });

    it('throws GraphitiNetworkError on fetch rejection', async () => {
      const fetch = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'));
      const c = client(fetch);
      try {
        await c.listWorkflows();
        expect.fail('should have thrown');
      } catch (err) {
        expect(err).toBeInstanceOf(GraphitiNetworkError);
        expect((err as GraphitiNetworkError).message).toContain('Failed to fetch');
        expect((err as GraphitiNetworkError).cause).toBeInstanceOf(TypeError);
      }
    });

    it('throws GraphitiApiError for 401 unauthorized', async () => {
      const c = client(errorFetch(401, 'invalid token'));
      await expect(c.listWorkflows()).rejects.toMatchObject({
        status: 401,
        message: 'invalid token',
      });
    });
  });

  describe('renameWorkflow', () => {
    it('sends PATCH with name', async () => {
      const fetch = mockFetch(200, {});
      const c = client(fetch);
      await c.renameWorkflow('wf-1', 'New Name');
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/workflows/wf-1/name',
        expect.objectContaining({
          method: 'PATCH',
          body: JSON.stringify({ name: 'New Name' }),
        }),
      );
    });
  });

  describe('exportWorkflow', () => {
    it('returns ArrayBuffer for JSON format', async () => {
      const data = { nodes: [], edges: [] };
      const fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        arrayBuffer: () => Promise.resolve(new ArrayBuffer(8)),
      } as unknown as Response);
      const c = client(fetch);
      const result = await c.exportWorkflow('wf-1', 'json');
      expect(result).toBeInstanceOf(ArrayBuffer);
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/workflows/wf-1/export?format=json',
        expect.anything(),
      );
    });

    it('requests yaml format when specified', async () => {
      const fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        arrayBuffer: () => Promise.resolve(new ArrayBuffer(8)),
      } as unknown as Response);
      const c = client(fetch);
      await c.exportWorkflow('wf-1', 'yaml');
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/workflows/wf-1/export?format=yaml',
        expect.anything(),
      );
    });
  });

  describe('URL encoding', () => {
    it('encodes workflow IDs with special characters', async () => {
      const fetch = mockFetch(200, { nodes: [], edges: [] });
      const c = client(fetch);
      await c.getWorkflowState('wf/special&chars');
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/workflows/wf%2Fspecial%26chars/state',
        expect.anything(),
      );
    });
  });

  describe('empty ID validation', () => {
    it('throws on empty workflow ID', async () => {
      const c = client(mockFetch(200, {}));
      await expect(c.getWorkflow('')).rejects.toThrow('ID must not be empty');
    });

    it('throws on empty node ID', async () => {
      const c = client(mockFetch(200, {}));
      await expect(c.getNodeConfig('wf-1', '')).rejects.toThrow('ID must not be empty');
    });
  });
});
