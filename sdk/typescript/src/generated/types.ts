// Auto-derived from Graphiti Go API handlers.
// When Story 1.7 (OpenAPI/Huma) lands, regenerate from the spec.
//
// CASING NOTE: Two conventions exist in this file:
// - PascalCase fields (e.g. Workflow.ID, Workflow.Name) match Go's default JSON
//   serialization for domain types returned by REST endpoints.
// - camelCase fields (e.g. NodeState.id, EdgeState.sourceNodeId) match explicit
//   JSON tags used by the canvas state / command response structs.
// Both mirror the server's actual JSON output.

// --- Workflow ---

export interface Workflow {
  ID: string;
  Name: string;
  Description: string;
  Status: WorkflowStatus;
  Version: number;
  Nodes: NodeInstance[];
  Edges: Edge[];
  CreatedBy: string;
  CreatedAt: string;
  UpdatedAt: string;
}

export type WorkflowStatus = 'draft' | 'deployed' | 'archived';

export interface WorkflowSummary {
  ID: string;
  Name: string;
  Status: WorkflowStatus;
  Version: number;
  UpdatedAt: string;
}

// --- Nodes ---

export interface NodeInstance {
  ID: string;
  DefinitionID: string;
  Label: string;
  X: number;
  Y: number;
  AttributeValues: Record<string, unknown>;
  Definition?: NodeDefinition;
}

export interface NodeDefinition {
  ID: string;
  Name: string;
  Description: string;
  Icon: string;
  Shape: ShapeDefinition;
  Category: CategoryDefinition;
  Inputs: PortDefinition[];
  Outputs: PortDefinition[];
  Attributes: AttributeDefinition[];
}

export interface ShapeDefinition {
  Type: ShapeType;
  Width: number;
  HeaderColor: string;
  HeaderBackground: string;
}

export type ShapeType =
  | 'rounded-rect'
  | 'pill'
  | 'diamond'
  | 'hexagon'
  | 'sub-workflow'
  | 'custom';

export interface CategoryDefinition {
  Group: string;
}

export interface PortDefinition {
  ID: string;
  Label: string;
  Type: PortType;
  Position: string;
  MaxConnections: number;
}

export type PortType = 'data' | 'control' | 'error';

export interface AttributeDefinition {
  ID: string;
  Label: string;
  Type: AttributeType;
  Display: string;
}

export type AttributeType =
  | 'string'
  | 'text'
  | 'number'
  | 'boolean'
  | 'enum'
  | 'secret'
  | 'json'
  | 'expression'
  | 'port-mapping'
  | 'workflow-reference';

// --- Edges ---

export interface Edge {
  ID: string;
  SourceNodeID: string;
  SourcePortID: string;
  TargetNodeID: string;
  TargetPortID: string;
}

// --- Workflow State (canvas sync format) ---

export interface WorkflowState {
  nodes: NodeState[];
  edges: EdgeState[];
}

export interface NodeState {
  id: string;
  definitionId: string;
  label: string;
  x: number;
  y: number;
  attributes: Record<string, unknown>;
  definition?: NodeDefJSON;
}

export interface NodeDefJSON {
  icon: string;
  shape: {
    type: string;
    width: number;
    headerColor: string;
    headerBackground: string;
  };
  category: { group: string };
  inputs: PortDefJSON[];
  outputs: PortDefJSON[];
  attributes: AttrDefJSON[];
}

export interface PortDefJSON {
  id: string;
  label: string;
  type: string;
  position: string;
  maxConnections: number;
}

export interface AttrDefJSON {
  id: string;
  label: string;
  type: string;
  display: string;
}

export interface EdgeState {
  id: string;
  sourceNodeId: string;
  sourcePortId: string;
  targetNodeId: string;
  targetPortId: string;
}

// --- Commands ---

export interface CommandRequest {
  type: CommandType;
  definitionId?: string;
  instanceId?: string;
  label?: string;
  nodeId?: string;
  x?: number;
  y?: number;
  fromX?: number;
  fromY?: number;
  toX?: number;
  toY?: number;
  edgeId?: string;
  sourceNodeId?: string;
  sourcePortId?: string;
  targetNodeId?: string;
  targetPortId?: string;
  attrId?: string;
  oldValue?: unknown;
  newValue?: unknown;
  oldLabel?: string;
  newLabel?: string;
  from?: NodePosition[];
  to?: NodePosition[];
}

export type CommandType =
  | 'add_node'
  | 'remove_node'
  | 'move_node'
  | 'move_nodes'
  | 'add_edge'
  | 'remove_edge'
  | 'update_attribute'
  | 'rename_node';

export interface NodePosition {
  nodeId: string;
  x: number;
  y: number;
}

export interface CommandResponse {
  ok: boolean;
  canUndo: boolean;
  canRedo: boolean;
  affectedNodeIds?: string[];
  affectedEdgeIds?: string[];
  error?: string;
  workflow?: WorkflowState;
}

// --- Clipboard ---

export interface ClipboardPayload {
  version: number;
  source: string;
  nodes: ClipboardNode[];
  edges: ClipboardEdge[];
}

export interface ClipboardNode {
  originalId: string;
  definitionId: string;
  label: string;
  relativeX: number;
  relativeY: number;
  attributes: Record<string, unknown>;
}

export interface ClipboardEdge {
  sourceOriginalId: string;
  sourcePortId: string;
  targetOriginalId: string;
  targetPortId: string;
}

export interface ClipboardCopyResponse {
  ok: boolean;
  payload?: ClipboardPayload;
  error?: string;
}

export interface ClipboardPasteRequest {
  payload: ClipboardPayload;
  x: number;
  y: number;
  relative?: boolean;
}

// --- Validation ---

export interface ValidationResponse {
  valid: boolean;
  results: ValidationResult[];
  summary: ValidationSummary;
}

export interface ValidationResult {
  severity: 'error' | 'warning' | 'info';
  category: string;
  code: string;
  message: string;
  nodeId: string;
  edgeId: string;
  field: string;
}

export interface ValidationSummary {
  errors: number;
  warnings: number;
  info: number;
}

// --- Deploy ---

export interface DeployRequest {
  target?: string;
}

export interface DeployResponse {
  success: boolean;
  runId?: string;
  message: string;
  validationErrors?: LegacyValidationError[];
  validationResults?: ValidationResult[];
  validationSummary?: ValidationSummary;
}

export interface LegacyValidationError {
  nodeId: string;
  field: string;
  message: string;
}

export interface DeployStatusResponse {
  workflowId: string;
  version: number;
  verification: 'unknown' | 'verified' | 'missing' | 'error';
  message: string;
  checkedAt: string;
}

// --- Version History ---

export interface WorkflowVersion {
  id: string;
  version: number;
  deployedAt: string;
  deployedBy: string;
}

// --- Execution ---

export interface ExecutionRunSummary {
  id: string;
  workflowId: string;
  status: ExecutionStatus;
  startedAt: string;
  completedAt: string;
  triggerType: string;
}

export interface ExecutionRun {
  id: string;
  workflowId: string;
  workflowVersion: number;
  status: ExecutionStatus;
  startedAt: string;
  completedAt: string;
  triggerType: string;
  triggerData: Record<string, unknown>;
  nodeStatuses: Record<string, NodeExecutionStatus>;
}

export type ExecutionStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled';

export interface NodeExecutionStatus {
  nodeId: string;
  status: NodeExecStatus;
  startedAt: string;
  completedAt: string;
  inputData: Record<string, unknown>;
  outputData: Record<string, unknown>;
  errorMessage: string;
  logs: LogEntry[];
}

export type NodeExecStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'failed'
  | 'skipped';

export interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

// --- WebSocket Events ---

export interface NodeStatusEvent {
  type: 'node_status';
  runID: string;
  nodeID: string;
  status: string;
  startedAt?: string;
  completedAt?: string;
  duration?: string;
}

// --- API Error ---

export interface ApiError {
  error: {
    code: number;
    message: string;
    detail?: string;
  };
}

// --- Node Config ---

export interface NodeConfigResponse {
  node: NodeState;
  definition: NodeDefJSON;
}

// --- API Response Wrappers ---

export interface ListWorkflowsResponse {
  items: WorkflowSummary[];
}

export interface CreateWorkflowResponse {
  workflow: Workflow;
}

export interface GetWorkflowResponse {
  workflow: Workflow;
  nodeDefinitions: NodeDefinition[];
}

export interface ListNodeDefinitionsResponse {
  items: NodeDefinition[];
}

export interface SearchNodesResponse {
  items: NodeDefinition[];
}

export interface UpdateAttributesResponse {
  node: NodeState;
}
