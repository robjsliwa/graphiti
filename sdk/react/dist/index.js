// src/GraphitiCanvas.tsx
import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef
} from "react";
import "@graphiti/canvas";
import { jsx } from "react/jsx-runtime";
var GraphitiCanvas = forwardRef(function GraphitiCanvas2(props, ref) {
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
    onError
  } = props;
  const elementRef = useRef(null);
  useEffect(() => {
    const el = elementRef.current;
    if (!el) return;
    const entries = [];
    function listen(event, cb) {
      if (!cb) return;
      const handler = (e) => cb(e.detail);
      el.addEventListener(event, handler);
      entries.push({ event, handler });
    }
    listen("graphiti:node-selected", onNodeSelected);
    listen("graphiti:node-deselected", onNodeDeselected ? () => onNodeDeselected() : void 0);
    listen("graphiti:workflow-changed", onWorkflowChanged);
    listen("graphiti:command-executed", onCommandExecuted);
    listen("graphiti:error", onError);
    return () => {
      for (const { event, handler } of entries) {
        el.removeEventListener(event, handler);
      }
    };
  }, [onNodeSelected, onNodeDeselected, onWorkflowChanged, onCommandExecuted, onError]);
  useImperativeHandle(ref, () => ({
    loadWorkflow: () => call("loadWorkflow"),
    executeCommand: (cmd) => call("executeCommand", cmd),
    undo: () => call("undo"),
    redo: () => call("redo"),
    zoomToFit: () => {
      const el = elementRef.current;
      if (el && typeof el.zoomToFit === "function") {
        el.zoomToFit();
      }
    },
    exportJSON: () => call("exportJSON")
  }));
  function call(method, ...args) {
    const el = elementRef.current;
    if (el && typeof el[method] === "function") {
      return el[method](...args);
    }
    return Promise.resolve(null);
  }
  return /* @__PURE__ */ jsx(
    "graphiti-canvas",
    {
      ref: elementRef,
      "api-url": apiUrl,
      "workflow-id": workflowId,
      token,
      theme,
      "read-only": readOnly ? "" : void 0,
      className,
      style
    }
  );
});

// src/hooks.ts
import {
  createContext,
  useContext,
  useEffect as useEffect2,
  useRef as useRef2,
  useState,
  useCallback
} from "react";
import { GraphitiWS } from "@graphiti/client";
var GraphitiContext = createContext(null);
function useGraphitiClient() {
  const client = useContext(GraphitiContext);
  if (!client) {
    throw new Error("useGraphitiClient must be used within a GraphitiProvider");
  }
  return client;
}
function useWorkflows() {
  const client = useGraphitiClient();
  const [workflows, setWorkflows] = useState(void 0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(void 0);
  const load = useCallback(async () => {
    let cancelled = false;
    setLoading(true);
    setError(void 0);
    try {
      const res = await client.listWorkflows();
      if (!cancelled) {
        setWorkflows(res.items);
      }
    } catch (err) {
      if (!cancelled) {
        setError(err instanceof Error ? err : new Error(String(err)));
      }
    } finally {
      if (!cancelled) {
        setLoading(false);
      }
    }
    return () => {
      cancelled = true;
    };
  }, [client]);
  useEffect2(() => {
    load();
  }, [load]);
  return { workflows, loading, error, refetch: load };
}
function useNodeDefinitions() {
  const client = useGraphitiClient();
  const [definitions, setDefinitions] = useState(void 0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(void 0);
  useEffect2(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError(void 0);
      try {
        const res = await client.listNodeDefinitions();
        if (!cancelled) {
          setDefinitions(res.items);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err : new Error(String(err)));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [client]);
  return { definitions, loading, error };
}
function useExecutionStatus(baseUrl, token, workflowId) {
  const [statuses, setStatuses] = useState(/* @__PURE__ */ new Map());
  const [connected, setConnected] = useState(false);
  const wsRef = useRef2(null);
  useEffect2(() => {
    if (!baseUrl || !token || !workflowId) return;
    const ws = new GraphitiWS({
      baseUrl,
      token,
      workflowId,
      autoReconnect: true
    });
    ws.on("connected", () => setConnected(true));
    ws.on("disconnected", () => setConnected(false));
    ws.on("node_status", (event) => {
      setStatuses((prev) => {
        const next = new Map(prev);
        next.set(event.nodeID, event);
        return next;
      });
    });
    ws.connect();
    wsRef.current = ws;
    return () => {
      ws.disconnect();
      wsRef.current = null;
      setStatuses(/* @__PURE__ */ new Map());
      setConnected(false);
    };
  }, [baseUrl, token, workflowId]);
  return { statuses, connected };
}

// src/GraphitiProvider.tsx
import { jsx as jsx2 } from "react/jsx-runtime";
function GraphitiProvider({ client, children }) {
  return /* @__PURE__ */ jsx2(GraphitiContext.Provider, { value: client, children });
}
export {
  GraphitiCanvas,
  GraphitiContext,
  GraphitiProvider,
  useExecutionStatus,
  useGraphitiClient,
  useNodeDefinitions,
  useWorkflows
};
//# sourceMappingURL=index.js.map