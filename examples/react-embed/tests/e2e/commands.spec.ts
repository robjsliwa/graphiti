import { test, expect } from '@playwright/test';

const API_URL = 'http://localhost:8080';
const AUTH_HEADERS = {
  Authorization: 'Bearer dev-token',
  'Content-Type': 'application/json',
  Accept: 'application/json',
};

test.describe('Commands', () => {
  let workflowId: string;

  test.beforeEach(async ({ request }) => {
    const res = await request.post(`${API_URL}/api/workflows`, {
      headers: AUTH_HEADERS,
      data: { name: 'E2E Command Test' },
    });
    const body = await res.json();
    workflowId = body.workflow.ID;
  });

  test.afterEach(async ({ request }) => {
    if (workflowId) {
      await request.delete(`${API_URL}/api/workflows/${workflowId}`, {
        headers: AUTH_HEADERS,
      });
    }
  });

  test('add a node via API and verify state', async ({ request }) => {
    // Add a node via command API
    const cmdRes = await request.post(
      `${API_URL}/api/workflows/${workflowId}/commands`,
      {
        headers: AUTH_HEADERS,
        data: {
          type: 'add_node',
          definitionId: 'api-gateway',
          instanceId: crypto.randomUUID(),
          x: 200,
          y: 150,
        },
      },
    );
    const cmdBody = await cmdRes.json();
    expect(cmdBody.ok).toBe(true);

    // Verify state
    const stateRes = await request.get(
      `${API_URL}/api/workflows/${workflowId}/state`,
      { headers: AUTH_HEADERS },
    );
    const state = await stateRes.json();
    expect(state.nodes.length).toBe(1);
    expect(state.nodes[0].definitionId).toBe('api-gateway');
  });

  test('undo removes the last added node', async ({ request }) => {
    // Add a node
    await request.post(
      `${API_URL}/api/workflows/${workflowId}/commands`,
      {
        headers: AUTH_HEADERS,
        data: {
          type: 'add_node',
          definitionId: 'api-gateway',
          instanceId: crypto.randomUUID(),
          x: 200,
          y: 150,
        },
      },
    );

    // Undo
    const undoRes = await request.post(
      `${API_URL}/api/workflows/${workflowId}/undo`,
      { headers: AUTH_HEADERS, data: {} },
    );
    const undoBody = await undoRes.json();
    expect(undoBody.ok).toBe(true);

    // Verify state is empty
    const stateRes = await request.get(
      `${API_URL}/api/workflows/${workflowId}/state`,
      { headers: AUTH_HEADERS },
    );
    const state = await stateRes.json();
    expect(state.nodes.length).toBe(0);
  });

  test('redo restores undone node', async ({ request }) => {
    // Add a node
    await request.post(
      `${API_URL}/api/workflows/${workflowId}/commands`,
      {
        headers: AUTH_HEADERS,
        data: {
          type: 'add_node',
          definitionId: 'api-gateway',
          instanceId: crypto.randomUUID(),
          x: 200,
          y: 150,
        },
      },
    );

    // Undo
    await request.post(
      `${API_URL}/api/workflows/${workflowId}/undo`,
      { headers: AUTH_HEADERS, data: {} },
    );

    // Redo
    const redoRes = await request.post(
      `${API_URL}/api/workflows/${workflowId}/redo`,
      { headers: AUTH_HEADERS, data: {} },
    );
    const redoBody = await redoRes.json();
    expect(redoBody.ok).toBe(true);

    // Verify node is back
    const stateRes = await request.get(
      `${API_URL}/api/workflows/${workflowId}/state`,
      { headers: AUTH_HEADERS },
    );
    const state = await stateRes.json();
    expect(state.nodes.length).toBe(1);
  });

  test('connect two nodes via commands', async ({ request }) => {
    // Add two nodes
    const nodeAId = crypto.randomUUID();
    const nodeBId = crypto.randomUUID();

    await request.post(
      `${API_URL}/api/workflows/${workflowId}/commands`,
      {
        headers: AUTH_HEADERS,
        data: { type: 'add_node', definitionId: 'api-gateway', instanceId: nodeAId, x: 100, y: 100 },
      },
    );

    await request.post(
      `${API_URL}/api/workflows/${workflowId}/commands`,
      {
        headers: AUTH_HEADERS,
        data: { type: 'add_node', definitionId: 'api-gateway', instanceId: nodeBId, x: 400, y: 100 },
      },
    );

    // Connect them
    const connectRes = await request.post(
      `${API_URL}/api/workflows/${workflowId}/commands`,
      {
        headers: AUTH_HEADERS,
        data: {
          type: 'add_edge',
          sourceNodeId: nodeAId,
          sourcePortId: 'output',
          targetNodeId: nodeBId,
          targetPortId: 'input',
        },
      },
    );
    const connectBody = await connectRes.json();
    expect(connectBody.ok).toBe(true);

    // Verify state has edge
    const stateRes = await request.get(
      `${API_URL}/api/workflows/${workflowId}/state`,
      { headers: AUTH_HEADERS },
    );
    const state = await stateRes.json();
    expect(state.edges.length).toBe(1);
    expect(state.edges[0].sourceNodeId).toBe(nodeAId);
    expect(state.edges[0].targetNodeId).toBe(nodeBId);
  });
});
