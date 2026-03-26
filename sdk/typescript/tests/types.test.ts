import { describe, it, expect } from 'vitest';
import type {
  Workflow,
  WorkflowSummary,
  NodeInstance,
  Edge,
  NodeDefinition,
  WorkflowState,
  CommandRequest,
  CommandResponse,
  ApiError,
  ValidationResponse,
  DeployResponse,
  DeployStatusResponse,
  ExecutionRunSummary,
  NodeStatusEvent,
  ClipboardPayload,
} from '../src/generated/types.js';

describe('Generated types compile and match API fixtures', () => {
  it('WorkflowSummary matches list response shape', () => {
    const item: WorkflowSummary = {
      ID: 'wf-1',
      Name: 'Test',
      Status: 'draft',
      Version: 1,
      UpdatedAt: '2026-01-01T00:00:00Z',
    };
    expect(item.ID).toBe('wf-1');
    expect(item.Status).toBe('draft');
  });

  it('Workflow matches full object shape', () => {
    const wf: Workflow = {
      ID: 'wf-1',
      Name: 'Test',
      Description: 'A test workflow',
      Status: 'deployed',
      Version: 3,
      Nodes: [],
      Edges: [],
      CreatedBy: 'user-1',
      CreatedAt: '2026-01-01T00:00:00Z',
      UpdatedAt: '2026-01-01T00:00:00Z',
    };
    expect(wf.Status).toBe('deployed');
    expect(wf.Version).toBe(3);
  });

  it('NodeDefinition matches search response shape', () => {
    const def: NodeDefinition = {
      ID: 'api-gateway',
      Name: 'API Gateway',
      Description: 'HTTP entry point',
      Icon: 'globe',
      Shape: { Type: 'rounded-rect', Width: 180, HeaderColor: '#fff', HeaderBackground: '#5588dd' },
      Category: { Group: 'sources' },
      Inputs: [{ ID: 'control-in', Label: 'Control', Type: 'control', Position: 'top', MaxConnections: 1 }],
      Outputs: [{ ID: 'data-out', Label: 'Data', Type: 'data', Position: 'bottom', MaxConnections: 0 }],
      Attributes: [{ ID: 'port', Label: 'Port', Type: 'number', Display: 'node-body' }],
    };
    expect(def.Shape.Type).toBe('rounded-rect');
    expect(def.Inputs).toHaveLength(1);
  });

  it('WorkflowState matches canvas sync format', () => {
    const state: WorkflowState = {
      nodes: [{
        id: 'n-1',
        definitionId: 'api-gateway',
        label: 'Gateway',
        x: 100,
        y: 200,
        attributes: { port: 8080 },
        definition: {
          icon: 'globe',
          shape: { type: 'rounded-rect', width: 180, headerColor: '#fff', headerBackground: '#55d' },
          category: { group: 'sources' },
          inputs: [],
          outputs: [],
          attributes: [],
        },
      }],
      edges: [{
        id: 'e-1',
        sourceNodeId: 'n-1',
        sourcePortId: 'data-out',
        targetNodeId: 'n-2',
        targetPortId: 'data-in',
      }],
    };
    expect(state.nodes[0].id).toBe('n-1');
    expect(state.edges[0].sourceNodeId).toBe('n-1');
  });

  it('CommandRequest covers all command types', () => {
    const addNode: CommandRequest = {
      type: 'add_node',
      definitionId: 'api-gateway',
      x: 100,
      y: 200,
    };
    expect(addNode.type).toBe('add_node');

    const addEdge: CommandRequest = {
      type: 'add_edge',
      sourceNodeId: 'n-1',
      sourcePortId: 'out',
      targetNodeId: 'n-2',
      targetPortId: 'in',
    };
    expect(addEdge.type).toBe('add_edge');
  });

  it('CommandResponse matches server shape', () => {
    const resp: CommandResponse = {
      ok: true,
      canUndo: true,
      canRedo: false,
      affectedNodeIds: ['n-1'],
      workflow: { nodes: [], edges: [] },
    };
    expect(resp.ok).toBe(true);
  });

  it('ApiError matches error envelope', () => {
    const err: ApiError = {
      error: { code: 404, message: 'not found', detail: 'workflow wf-1 does not exist' },
    };
    expect(err.error.code).toBe(404);
  });

  it('ValidationResponse matches validate endpoint', () => {
    const v: ValidationResponse = {
      valid: false,
      results: [{
        severity: 'error',
        category: 'connectivity',
        code: 'UNCONNECTED_PORT',
        message: 'Input port not connected',
        nodeId: 'n-1',
        edgeId: '',
        field: '',
      }],
      summary: { errors: 1, warnings: 0, info: 0 },
    };
    expect(v.valid).toBe(false);
    expect(v.summary.errors).toBe(1);
  });

  it('DeployResponse matches deploy endpoint', () => {
    const d: DeployResponse = {
      success: true,
      runId: 'run-1',
      message: 'Deployment initiated',
    };
    expect(d.success).toBe(true);
  });

  it('DeployStatusResponse matches status endpoint', () => {
    const s: DeployStatusResponse = {
      workflowId: 'wf-1',
      version: 5,
      verification: 'verified',
      message: 'Active',
      checkedAt: '2026-03-26T14:00:00Z',
    };
    expect(s.verification).toBe('verified');
  });

  it('ExecutionRunSummary matches list runs response', () => {
    const run: ExecutionRunSummary = {
      id: 'run-1',
      workflowId: 'wf-1',
      status: 'completed',
      startedAt: '2026-01-01T00:00:00Z',
      completedAt: '2026-01-01T00:01:00Z',
      triggerType: 'manual',
    };
    expect(run.status).toBe('completed');
  });

  it('NodeStatusEvent matches WebSocket message', () => {
    const evt: NodeStatusEvent = {
      type: 'node_status',
      runID: 'run-1',
      nodeID: 'n-1',
      status: 'running',
      startedAt: '2026-01-01T00:00:00Z',
    };
    expect(evt.type).toBe('node_status');
  });

  it('ClipboardPayload matches copy response', () => {
    const clip: ClipboardPayload = {
      version: 1,
      source: 'canvas',
      nodes: [{
        originalId: 'n-1',
        definitionId: 'api-gateway',
        label: 'Gateway',
        relativeX: 0,
        relativeY: 0,
        attributes: {},
      }],
      edges: [],
    };
    expect(clip.version).toBe(1);
  });
});
