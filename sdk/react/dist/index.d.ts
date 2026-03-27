import * as react from 'react';
import { ReactNode } from 'react';
import { WorkflowState, CommandRequest, CommandResponse, GraphitiClient, NodeStatusEvent, NodeDefinition, WorkflowSummary } from '@graphiti/client';
export { CommandRequest, CommandResponse, CommandType, DeployResponse, DeployStatusResponse, Edge, EdgeState, ExecutionRun, ExecutionRunSummary, GraphitiClientOptions, ListNodeDefinitionsResponse, ListWorkflowsResponse, NodeConfigResponse, NodeDefJSON, NodeDefinition, NodeInstance, NodeState, NodeStatusEvent, SearchNodesResponse, ValidationResponse, Workflow, WorkflowState, WorkflowSummary } from '@graphiti/client';
import * as react_jsx_runtime from 'react/jsx-runtime';

interface GraphitiCanvasProps {
    apiUrl: string;
    workflowId: string;
    token: string;
    theme?: 'light' | 'dark';
    readOnly?: boolean;
    className?: string;
    style?: React.CSSProperties;
    onNodeSelected?: (detail: {
        nodeId: string;
    }) => void;
    onNodeDeselected?: () => void;
    onWorkflowChanged?: (detail: {
        workflow: WorkflowState;
    }) => void;
    onCommandExecuted?: (detail: {
        type: string;
        ok: boolean;
    }) => void;
    onError?: (detail: {
        message: string;
    }) => void;
}
interface GraphitiCanvasRef {
    loadWorkflow(): Promise<void>;
    executeCommand(cmd: CommandRequest): Promise<CommandResponse | null>;
    undo(): Promise<void>;
    redo(): Promise<void>;
    zoomToFit(): void;
    exportJSON(): Promise<WorkflowState | null>;
}
declare module 'react' {
    namespace JSX {
        interface IntrinsicElements {
            'graphiti-canvas': React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement> & {
                'api-url'?: string;
                'workflow-id'?: string;
                token?: string;
                theme?: string;
                'read-only'?: string;
            }, HTMLElement>;
        }
    }
}
declare const GraphitiCanvas: react.ForwardRefExoticComponent<GraphitiCanvasProps & react.RefAttributes<GraphitiCanvasRef>>;

interface GraphitiProviderProps {
    client: GraphitiClient;
    children: ReactNode;
}
declare function GraphitiProvider({ client, children }: GraphitiProviderProps): react_jsx_runtime.JSX.Element;

declare const GraphitiContext: react.Context<GraphitiClient | null>;
declare function useGraphitiClient(): GraphitiClient;
interface UseWorkflowsResult {
    workflows: WorkflowSummary[] | undefined;
    loading: boolean;
    error: Error | undefined;
    refetch: () => void;
}
declare function useWorkflows(): UseWorkflowsResult;
interface UseNodeDefinitionsResult {
    definitions: NodeDefinition[] | undefined;
    loading: boolean;
    error: Error | undefined;
}
declare function useNodeDefinitions(): UseNodeDefinitionsResult;
interface UseExecutionStatusResult {
    statuses: Map<string, NodeStatusEvent>;
    connected: boolean;
}
declare function useExecutionStatus(baseUrl: string, token: string, workflowId: string): UseExecutionStatusResult;

export { GraphitiCanvas, type GraphitiCanvasProps, type GraphitiCanvasRef, GraphitiContext, GraphitiProvider, type GraphitiProviderProps, type UseExecutionStatusResult, type UseNodeDefinitionsResult, type UseWorkflowsResult, useExecutionStatus, useGraphitiClient, useNodeDefinitions, useWorkflows };
