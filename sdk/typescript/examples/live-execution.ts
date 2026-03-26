/**
 * Subscribe to live execution status updates via WebSocket.
 *
 * Usage:
 *   GRAPHITI_URL=http://localhost:8080 GRAPHITI_TOKEN=dev-token WORKFLOW_ID=wf-123 \
 *     npx tsx examples/live-execution.ts
 */
import { GraphitiClient, GraphitiWS, GraphitiApiError } from '../src/index.js';

const baseUrl = process.env.GRAPHITI_URL || 'http://localhost:8080';
const token = process.env.GRAPHITI_TOKEN || 'dev-token';
const workflowId = process.env.WORKFLOW_ID;

if (!workflowId) {
  console.error('Set WORKFLOW_ID environment variable to the workflow to monitor.');
  process.exit(1);
}

const client = new GraphitiClient({ baseUrl, token });

// Verify workflow exists
try {
  const { workflow } = await client.getWorkflow(workflowId);
  console.log(`Monitoring workflow: ${workflow.Name} (${workflow.ID})`);
} catch (err) {
  if (err instanceof GraphitiApiError) {
    console.error(`Cannot find workflow: ${err.message}`);
    process.exit(1);
  }
  throw err;
}

// Connect WebSocket
const ws = new GraphitiWS({
  baseUrl,
  token,
  workflowId,
});

ws.on('connected', () => {
  console.log('WebSocket connected. Waiting for execution events...\n');
});

ws.on('node_status', (event) => {
  const duration = event.duration ? ` (${event.duration})` : '';
  console.log(`  [${event.status.toUpperCase()}] Node ${event.nodeID}${duration}`);
});

ws.on('disconnected', ({ code, reason }) => {
  console.log(`\nDisconnected: ${code} ${reason}`);
});

ws.on('error', (err) => {
  console.error('WebSocket error:', err.message);
});

ws.connect();

// Graceful shutdown
process.on('SIGINT', () => {
  console.log('\nShutting down...');
  ws.disconnect();
  process.exit(0);
});
