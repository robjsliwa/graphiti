interface Workflow {
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
type WorkflowStatus = 'draft' | 'deployed' | 'archived';
interface WorkflowSummary {
    ID: string;
    Name: string;
    Status: WorkflowStatus;
    Version: number;
    UpdatedAt: string;
}
interface NodeInstance {
    ID: string;
    DefinitionID: string;
    Label: string;
    X: number;
    Y: number;
    AttributeValues: Record<string, unknown>;
    Definition?: NodeDefinition;
}
interface NodeDefinition {
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
interface ShapeDefinition {
    Type: ShapeType;
    Width: number;
    HeaderColor: string;
    HeaderBackground: string;
}
type ShapeType = 'rounded-rect' | 'pill' | 'diamond' | 'hexagon' | 'sub-workflow' | 'custom';
interface CategoryDefinition {
    Group: string;
}
interface PortDefinition {
    ID: string;
    Label: string;
    Type: PortType;
    Position: string;
    MaxConnections: number;
}
type PortType = 'data' | 'control' | 'error';
interface AttributeDefinition {
    ID: string;
    Label: string;
    Type: AttributeType;
    Display: string;
}
type AttributeType = 'string' | 'text' | 'number' | 'boolean' | 'enum' | 'secret' | 'json' | 'expression' | 'port-mapping' | 'workflow-reference';
interface Edge {
    ID: string;
    SourceNodeID: string;
    SourcePortID: string;
    TargetNodeID: string;
    TargetPortID: string;
}
interface WorkflowState {
    nodes: NodeState[];
    edges: EdgeState[];
}
interface NodeState {
    id: string;
    definitionId: string;
    label: string;
    x: number;
    y: number;
    attributes: Record<string, unknown>;
    definition?: NodeDefJSON;
}
interface NodeDefJSON {
    icon: string;
    shape: {
        type: string;
        width: number;
        headerColor: string;
        headerBackground: string;
    };
    category: {
        group: string;
    };
    inputs: PortDefJSON[];
    outputs: PortDefJSON[];
    attributes: AttrDefJSON[];
}
interface PortDefJSON {
    id: string;
    label: string;
    type: string;
    position: string;
    maxConnections: number;
}
interface AttrDefJSON {
    id: string;
    label: string;
    type: string;
    display: string;
}
interface EdgeState {
    id: string;
    sourceNodeId: string;
    sourcePortId: string;
    targetNodeId: string;
    targetPortId: string;
}
interface CommandRequest {
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
type CommandType = 'add_node' | 'remove_node' | 'move_node' | 'move_nodes' | 'add_edge' | 'remove_edge' | 'update_attribute' | 'rename_node';
interface NodePosition {
    nodeId: string;
    x: number;
    y: number;
}
interface CommandResponse {
    ok: boolean;
    canUndo: boolean;
    canRedo: boolean;
    affectedNodeIds?: string[];
    affectedEdgeIds?: string[];
    error?: string;
    workflow?: WorkflowState;
}
interface ClipboardPayload {
    version: number;
    source: string;
    nodes: ClipboardNode[];
    edges: ClipboardEdge[];
}
interface ClipboardNode {
    originalId: string;
    definitionId: string;
    label: string;
    relativeX: number;
    relativeY: number;
    attributes: Record<string, unknown>;
}
interface ClipboardEdge {
    sourceOriginalId: string;
    sourcePortId: string;
    targetOriginalId: string;
    targetPortId: string;
}
interface ClipboardCopyResponse {
    ok: boolean;
    payload?: ClipboardPayload;
    error?: string;
}
interface ClipboardPasteRequest {
    payload: ClipboardPayload;
    x: number;
    y: number;
    relative?: boolean;
}
interface ValidationResponse {
    valid: boolean;
    results: ValidationResult[];
    summary: ValidationSummary;
}
interface ValidationResult {
    severity: 'error' | 'warning' | 'info';
    category: string;
    code: string;
    message: string;
    nodeId: string;
    edgeId: string;
    field: string;
}
interface ValidationSummary {
    errors: number;
    warnings: number;
    info: number;
}
interface DeployRequest {
    target?: string;
}
interface DeployResponse {
    success: boolean;
    runId?: string;
    message: string;
    validationErrors?: LegacyValidationError[];
    validationResults?: ValidationResult[];
    validationSummary?: ValidationSummary;
}
interface LegacyValidationError {
    nodeId: string;
    field: string;
    message: string;
}
interface DeployStatusResponse {
    workflowId: string;
    version: number;
    verification: 'unknown' | 'verified' | 'missing' | 'error';
    message: string;
    checkedAt: string;
}
interface WorkflowVersion {
    id: string;
    version: number;
    deployedAt: string;
    deployedBy: string;
}
interface ExecutionRunSummary {
    id: string;
    workflowId: string;
    status: ExecutionStatus;
    startedAt: string;
    completedAt: string;
    triggerType: string;
}
interface ExecutionRun {
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
type ExecutionStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
interface NodeExecutionStatus {
    nodeId: string;
    status: NodeExecStatus;
    startedAt: string;
    completedAt: string;
    inputData: Record<string, unknown>;
    outputData: Record<string, unknown>;
    errorMessage: string;
    logs: LogEntry[];
}
type NodeExecStatus = 'pending' | 'running' | 'completed' | 'failed' | 'skipped';
interface LogEntry {
    timestamp: string;
    level: string;
    message: string;
}
interface NodeStatusEvent {
    type: 'node_status';
    runID: string;
    nodeID: string;
    status: string;
    startedAt?: string;
    completedAt?: string;
    duration?: string;
}
interface ApiError {
    error: {
        code: number;
        message: string;
        detail?: string;
    };
}
interface NodeConfigResponse {
    node: NodeState;
    definition: NodeDefJSON;
}
interface ListWorkflowsResponse {
    items: WorkflowSummary[];
}
interface CreateWorkflowResponse {
    workflow: Workflow;
}
interface GetWorkflowResponse {
    workflow: Workflow;
    nodeDefinitions: NodeDefinition[];
}
interface ListNodeDefinitionsResponse {
    items: NodeDefinition[];
}
interface SearchNodesResponse {
    items: NodeDefinition[];
}
interface UpdateAttributesResponse {
    node: NodeState;
}

interface GraphitiClientOptions {
    /** Base URL of the Graphiti server (e.g. "http://localhost:8080"). */
    baseUrl: string;
    /** Bearer token for API authentication. */
    token: string;
    /** Optional custom fetch implementation (for testing). */
    fetch?: typeof globalThis.fetch;
}
declare class GraphitiClient {
    private readonly baseUrl;
    private readonly token;
    private readonly fetch;
    constructor(options: GraphitiClientOptions);
    listWorkflows(): Promise<ListWorkflowsResponse>;
    createWorkflow(body?: {
        name?: string;
        description?: string;
    }): Promise<CreateWorkflowResponse>;
    getWorkflow(id: string): Promise<GetWorkflowResponse>;
    deleteWorkflow(id: string): Promise<void>;
    renameWorkflow(id: string, name: string): Promise<void>;
    getWorkflowState(id: string): Promise<WorkflowState>;
    executeCommand(workflowId: string, cmd: CommandRequest): Promise<CommandResponse>;
    undo(workflowId: string): Promise<CommandResponse>;
    redo(workflowId: string): Promise<CommandResponse>;
    getNodeConfig(workflowId: string, nodeId: string): Promise<NodeConfigResponse>;
    updateAttributes(workflowId: string, nodeId: string, attributes: Record<string, unknown>): Promise<UpdateAttributesResponse>;
    searchNodes(query: string): Promise<SearchNodesResponse>;
    listNodeDefinitions(): Promise<ListNodeDefinitionsResponse>;
    clipboardCopy(workflowId: string, nodeIds: string[]): Promise<ClipboardCopyResponse>;
    clipboardPaste(workflowId: string, req: ClipboardPasteRequest): Promise<CommandResponse>;
    validateWorkflow(id: string): Promise<ValidationResponse>;
    deployWorkflow(id: string, target?: string): Promise<DeployResponse>;
    getDeployStatus(id: string): Promise<DeployStatusResponse>;
    exportWorkflow(id: string, format?: 'json' | 'yaml'): Promise<ArrayBuffer>;
    getVersionHistory(id: string): Promise<WorkflowVersion[]>;
    listRuns(workflowId: string): Promise<ExecutionRunSummary[]>;
    getRun(runId: string): Promise<ExecutionRun>;
    private get;
    private post;
    private requestNoContent;
    private request;
    private rawRequest;
}

interface GraphitiWSOptions {
    /** Base URL of the Graphiti server (http/https -- converted to ws/wss). */
    baseUrl: string;
    /**
     * Bearer token for authentication.
     *
     * Note: The token is sent as a query parameter in the WebSocket URL because
     * the browser WebSocket API does not support custom headers. This means the
     * token may appear in server access logs and proxy logs. For production use,
     * consider using short-lived tokens or a ticket-exchange mechanism.
     */
    token: string;
    /** Workflow ID to subscribe to. */
    workflowId: string;
    /** Auto-reconnect on disconnect. Default: true. */
    autoReconnect?: boolean;
    /** Max reconnect attempts. Default: 10. */
    maxReconnectAttempts?: number;
    /** Custom WebSocket constructor (for testing). */
    WebSocket?: typeof globalThis.WebSocket;
}
type WSEventMap = {
    node_status: NodeStatusEvent;
    connected: void;
    disconnected: {
        code: number;
        reason: string;
    };
    error: Error;
};
type Listener<K extends keyof WSEventMap> = (data: WSEventMap[K]) => void;
declare class GraphitiWS {
    private readonly baseUrl;
    private readonly token;
    private readonly workflowId;
    private readonly autoReconnect;
    private readonly maxReconnectAttempts;
    private readonly WS;
    private ws;
    private reconnectAttempts;
    private reconnectTimer;
    private intentionalClose;
    private listeners;
    constructor(options: GraphitiWSOptions);
    on<K extends keyof WSEventMap>(event: K, callback: Listener<K>): void;
    off<K extends keyof WSEventMap>(event: K, callback: Listener<K>): void;
    connect(): void;
    disconnect(): void;
    private createConnection;
    private scheduleReconnect;
    private emit;
}

/** Thrown when the Graphiti API returns a non-2xx response. */
declare class GraphitiApiError extends Error {
    /** HTTP status code. */
    readonly status: number;
    /** Error code from the JSON envelope (mirrors status). */
    readonly code: number;
    /** Optional detail string from the API. */
    readonly detail?: string;
    constructor(status: number, code: number, message: string, detail?: string);
}
/** Thrown when a network error prevents the request from completing. */
declare class GraphitiNetworkError extends Error {
    readonly cause?: Error;
    constructor(message: string, cause?: Error);
}

export { type ApiError, type AttrDefJSON, type AttributeDefinition, type AttributeType, type CategoryDefinition, type ClipboardCopyResponse, type ClipboardEdge, type ClipboardNode, type ClipboardPasteRequest, type ClipboardPayload, type CommandRequest, type CommandResponse, type CommandType, type CreateWorkflowResponse, type DeployRequest, type DeployResponse, type DeployStatusResponse, type Edge, type EdgeState, type ExecutionRun, type ExecutionRunSummary, type ExecutionStatus, type GetWorkflowResponse, GraphitiApiError, GraphitiClient, type GraphitiClientOptions, GraphitiNetworkError, GraphitiWS, type GraphitiWSOptions, type LegacyValidationError, type ListNodeDefinitionsResponse, type ListWorkflowsResponse, type LogEntry, type NodeConfigResponse, type NodeDefJSON, type NodeDefinition, type NodeExecStatus, type NodeExecutionStatus, type NodeInstance, type NodePosition, type NodeState, type NodeStatusEvent, type PortDefJSON, type PortDefinition, type PortType, type SearchNodesResponse, type ShapeDefinition, type ShapeType, type UpdateAttributesResponse, type ValidationResponse, type ValidationResult, type ValidationSummary, type WSEventMap, type Workflow, type WorkflowState, type WorkflowStatus, type WorkflowSummary, type WorkflowVersion };
