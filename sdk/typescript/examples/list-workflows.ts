/**
 * List all workflows for the authenticated user.
 *
 * Usage:
 *   GRAPHITI_URL=http://localhost:8080 GRAPHITI_TOKEN=dev-token npx tsx examples/list-workflows.ts
 */
import { GraphitiClient, GraphitiApiError } from '../src/index.js';

const client = new GraphitiClient({
  baseUrl: process.env.GRAPHITI_URL || 'http://localhost:8080',
  token: process.env.GRAPHITI_TOKEN || 'dev-token',
});

try {
  const { items } = await client.listWorkflows();

  if (items.length === 0) {
    console.log('No workflows found.');
  } else {
    console.log(`Found ${items.length} workflow(s):\n`);
    for (const wf of items) {
      console.log(`  [${wf.Status}] ${wf.Name} (${wf.ID}) — v${wf.Version}`);
    }
  }
} catch (err) {
  if (err instanceof GraphitiApiError) {
    console.error(`API error ${err.status}: ${err.message}`);
  } else {
    throw err;
  }
}
