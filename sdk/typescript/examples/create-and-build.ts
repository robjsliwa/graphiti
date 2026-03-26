/**
 * Create a workflow, add two nodes, and connect them.
 *
 * Usage:
 *   GRAPHITI_URL=http://localhost:8080 GRAPHITI_TOKEN=dev-token npx tsx examples/create-and-build.ts
 */
import { GraphitiClient, GraphitiApiError } from '../src/index.js';

const client = new GraphitiClient({
  baseUrl: process.env.GRAPHITI_URL || 'http://localhost:8080',
  token: process.env.GRAPHITI_TOKEN || 'dev-token',
});

try {
  // 1. Create a workflow
  const { workflow } = await client.createWorkflow({
    name: 'SDK Demo Workflow',
    description: 'Created via @graphiti/client SDK',
  });
  console.log(`Created workflow: ${workflow.ID}`);

  // 2. Add an API Gateway node
  const addGateway = await client.executeCommand(workflow.ID, {
    type: 'add_node',
    definitionId: 'api-gateway',
    x: 200,
    y: 100,
  });
  const gatewayId = addGateway.affectedNodeIds?.[0];
  console.log(`Added API Gateway node: ${gatewayId}`);

  // 3. Add a PostgreSQL node
  const addPostgres = await client.executeCommand(workflow.ID, {
    type: 'add_node',
    definitionId: 'postgresql',
    x: 200,
    y: 350,
  });
  const postgresId = addPostgres.affectedNodeIds?.[0];
  console.log(`Added PostgreSQL node: ${postgresId}`);

  // 4. Connect them
  if (gatewayId && postgresId) {
    const addEdge = await client.executeCommand(workflow.ID, {
      type: 'add_edge',
      sourceNodeId: gatewayId,
      sourcePortId: 'data-out',
      targetNodeId: postgresId,
      targetPortId: 'data-in',
    });
    console.log(`Connected nodes: ${addEdge.affectedEdgeIds?.[0]}`);
  }

  // 5. Get final state
  const state = await client.getWorkflowState(workflow.ID);
  console.log(`\nWorkflow has ${state.nodes.length} nodes and ${state.edges.length} edges.`);

  // 6. Undo the edge
  const undoResult = await client.undo(workflow.ID);
  console.log(`Undo: canUndo=${undoResult.canUndo}, canRedo=${undoResult.canRedo}`);

  // 7. Redo the edge
  const redoResult = await client.redo(workflow.ID);
  console.log(`Redo: canUndo=${redoResult.canUndo}, canRedo=${redoResult.canRedo}`);
} catch (err) {
  if (err instanceof GraphitiApiError) {
    console.error(`API error ${err.status}: ${err.message}`);
    if (err.detail) console.error(`  Detail: ${err.detail}`);
  } else {
    throw err;
  }
}
