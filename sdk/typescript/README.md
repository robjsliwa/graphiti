# @graphiti/client

TypeScript SDK for the Graphiti workflow builder API.

## Installation

```bash
npm install @graphiti/client
```

## Quick Start

```typescript
import { GraphitiClient } from '@graphiti/client';

const client = new GraphitiClient({
  baseUrl: 'http://localhost:8080',
  token: 'your-bearer-token',
});

// List all workflows
const { items } = await client.listWorkflows();
console.log(items);

// Create a workflow
const { workflow } = await client.createWorkflow({ name: 'My Flow' });

// Add a node
const result = await client.executeCommand(workflow.ID, {
  type: 'add_node',
  definitionId: 'api-gateway',
  x: 200,
  y: 100,
});
console.log('Added node:', result.affectedNodeIds);
```

## API Reference

### `GraphitiClient`

#### Constructor

```typescript
new GraphitiClient({
  baseUrl: string;      // Graphiti server URL
  token: string;        // Bearer auth token
  fetch?: typeof fetch; // Custom fetch (for testing)
})
```

#### Workflow CRUD

| Method | Description |
|--------|-------------|
| `listWorkflows()` | List all workflows for the authenticated user |
| `createWorkflow({ name?, description? })` | Create a new workflow |
| `getWorkflow(id)` | Get workflow with node definitions |
| `deleteWorkflow(id)` | Delete a workflow |
| `renameWorkflow(id, name)` | Rename a workflow |

#### Canvas State

| Method | Description |
|--------|-------------|
| `getWorkflowState(id)` | Get full canvas state (nodes, edges, definitions) |

#### Commands

| Method | Description |
|--------|-------------|
| `executeCommand(workflowId, cmd)` | Execute a canvas command |
| `undo(workflowId)` | Undo the last command |
| `redo(workflowId)` | Redo the last undone command |

Command types: `add_node`, `remove_node`, `move_node`, `move_nodes`, `add_edge`, `remove_edge`, `update_attribute`, `rename_node`.

#### Node Configuration

| Method | Description |
|--------|-------------|
| `getNodeConfig(workflowId, nodeId)` | Get node config and definition |
| `updateAttributes(workflowId, nodeId, attrs)` | Update node attributes |
| `searchNodes(query)` | Search node definitions |
| `listNodeDefinitions()` | List all node definitions |

#### Clipboard

| Method | Description |
|--------|-------------|
| `clipboardCopy(workflowId, nodeIds)` | Copy nodes to clipboard |
| `clipboardPaste(workflowId, request)` | Paste nodes from clipboard |

#### Validation & Deploy

| Method | Description |
|--------|-------------|
| `validateWorkflow(id)` | Validate workflow configuration |
| `deployWorkflow(id, target?)` | Deploy workflow to target |
| `getDeployStatus(id)` | Check deployment status |
| `exportWorkflow(id, format?)` | Export as JSON or YAML blob |
| `getVersionHistory(id)` | Get deployment version history |

#### Execution

| Method | Description |
|--------|-------------|
| `listRuns(workflowId)` | List execution runs |
| `getRun(runId)` | Get execution run details |

### `GraphitiWS`

Real-time WebSocket client for execution status updates.

```typescript
import { GraphitiWS } from '@graphiti/client';

const ws = new GraphitiWS({
  baseUrl: 'http://localhost:8080',
  token: 'your-bearer-token',
  workflowId: 'wf-123',
  autoReconnect: true,       // default: true
  maxReconnectAttempts: 10,  // default: 10
});

ws.on('connected', () => console.log('Connected'));
ws.on('node_status', (event) => {
  console.log(`Node ${event.nodeID}: ${event.status}`);
});
ws.on('disconnected', ({ code, reason }) => {
  console.log(`Disconnected: ${code} ${reason}`);
});
ws.on('error', (err) => console.error(err));

ws.connect();

// Later...
ws.disconnect();
```

#### Events

| Event | Payload | Description |
|-------|---------|-------------|
| `connected` | `void` | WebSocket connection established |
| `node_status` | `NodeStatusEvent` | Node execution status update |
| `disconnected` | `{ code, reason }` | Connection closed |
| `error` | `Error` | Connection or protocol error |

### Error Handling

```typescript
import { GraphitiApiError, GraphitiNetworkError } from '@graphiti/client';

try {
  await client.getWorkflow('nonexistent');
} catch (err) {
  if (err instanceof GraphitiApiError) {
    console.log(err.status);  // 404
    console.log(err.code);    // 404
    console.log(err.message); // "workflow not found"
    console.log(err.detail);  // optional detail string
  }
  if (err instanceof GraphitiNetworkError) {
    console.log(err.message); // "Network error: ..."
    console.log(err.cause);   // original Error
  }
}
```

### Exported Types

All API types are re-exported from the package:

`Workflow`, `WorkflowSummary`, `WorkflowState`, `NodeInstance`, `NodeDefinition`, `Edge`, `NodeState`, `EdgeState`, `CommandRequest`, `CommandResponse`, `CommandType`, `ValidationResponse`, `DeployResponse`, `DeployStatusResponse`, `ExecutionRun`, `ExecutionRunSummary`, `NodeStatusEvent`, `ApiError`, `ClipboardPayload`, `NodeConfigResponse`, `WorkflowVersion`, and more.

## Examples

See the [`examples/`](./examples/) directory for runnable scripts:

- `list-workflows.ts` — List all workflows
- `create-and-build.ts` — Create a workflow, add nodes, connect them
- `live-execution.ts` — Subscribe to live execution updates via WebSocket

Run with:

```bash
npx tsx examples/list-workflows.ts
```

## Development

```bash
npm install
npm run typecheck   # Type check
npm test            # Run tests
npm run build       # Build ESM + CJS + declarations
```

## License

MIT
