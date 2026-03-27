import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
  type Ref,
} from 'react';
import type { CommandRequest, CommandResponse, WorkflowState } from '@graphiti/client';

// Ensure the custom element is registered when this module loads
import '@graphiti/canvas';

export interface GraphitiCanvasProps {
  apiUrl: string;
  workflowId: string;
  token: string;
  theme?: 'light' | 'dark';
  readOnly?: boolean;
  className?: string;
  style?: React.CSSProperties;
  onNodeSelected?: (detail: { nodeId: string }) => void;
  onNodeDeselected?: () => void;
  onWorkflowChanged?: (detail: { workflow: WorkflowState }) => void;
  onCommandExecuted?: (detail: { type: string; ok: boolean }) => void;
  onError?: (detail: { message: string }) => void;
}

export interface GraphitiCanvasRef {
  loadWorkflow(): Promise<void>;
  executeCommand(cmd: CommandRequest): Promise<CommandResponse | null>;
  undo(): Promise<void>;
  redo(): Promise<void>;
  zoomToFit(): void;
  exportJSON(): Promise<WorkflowState | null>;
}

// Declare the custom element for JSX (React 19 uses React.JSX)
declare module 'react' {
  namespace JSX {
    interface IntrinsicElements {
      'graphiti-canvas': React.DetailedHTMLProps<
        React.HTMLAttributes<HTMLElement> & {
          'api-url'?: string;
          'workflow-id'?: string;
          token?: string;
          theme?: string;
          'read-only'?: string;
        },
        HTMLElement
      >;
    }
  }
}

type EventEntry = {
  event: string;
  handler: (e: Event) => void;
};

export const GraphitiCanvas = forwardRef(function GraphitiCanvas(
  props: GraphitiCanvasProps,
  ref: Ref<GraphitiCanvasRef>,
) {
  const {
    apiUrl,
    workflowId,
    token,
    theme,
    readOnly,
    className,
    style,
    onNodeSelected,
    onNodeDeselected,
    onWorkflowChanged,
    onCommandExecuted,
    onError,
  } = props;

  const elementRef = useRef<HTMLElement>(null);

  // Bridge custom events to React callbacks
  useEffect(() => {
    const el = elementRef.current;
    if (!el) return;

    const entries: EventEntry[] = [];

    function listen(event: string, cb: ((detail: any) => void) | undefined) {
      if (!cb) return;
      const handler = (e: Event) => cb((e as CustomEvent).detail);
      el!.addEventListener(event, handler);
      entries.push({ event, handler });
    }

    listen('graphiti:node-selected', onNodeSelected);
    listen('graphiti:node-deselected', onNodeDeselected ? () => onNodeDeselected() : undefined);
    listen('graphiti:workflow-changed', onWorkflowChanged);
    listen('graphiti:command-executed', onCommandExecuted);
    listen('graphiti:error', onError);

    return () => {
      for (const { event, handler } of entries) {
        el.removeEventListener(event, handler);
      }
    };
  }, [onNodeSelected, onNodeDeselected, onWorkflowChanged, onCommandExecuted, onError]);

  // Expose imperative methods via ref
  useImperativeHandle(ref, () => ({
    loadWorkflow: () => call<void>('loadWorkflow'),
    executeCommand: (cmd: CommandRequest) => call<CommandResponse | null>('executeCommand', cmd),
    undo: () => call<void>('undo'),
    redo: () => call<void>('redo'),
    zoomToFit: () => {
      const el = elementRef.current as Record<string, unknown> | null;
      if (el && typeof el.zoomToFit === 'function') {
        (el.zoomToFit as () => void)();
      }
    },
    exportJSON: () => call<WorkflowState | null>('exportJSON'),
  }));

  function call<T>(method: string, ...args: unknown[]): Promise<T> {
    const el = elementRef.current as Record<string, unknown> | null;
    if (el && typeof el[method] === 'function') {
      return (el[method] as (...a: unknown[]) => Promise<T>)(...args);
    }
    return Promise.resolve(null as T);
  }

  return (
    <graphiti-canvas
      ref={elementRef}
      api-url={apiUrl}
      workflow-id={workflowId}
      token={token}
      theme={theme}
      read-only={readOnly ? '' : undefined}
      className={className}
      style={style}
    />
  );
});
