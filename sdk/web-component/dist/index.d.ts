import { CommandRequest, CommandResponse, WorkflowState, NodeState, EdgeState, NodeDefJSON } from '@graphiti/client';

declare class GraphitiCanvasElement extends HTMLElement {
    static observedAttributes: string[];
    private client;
    private clientBaseUrl;
    private clientToken;
    private ws;
    private renderer;
    private selection;
    private connection;
    private keyboard;
    private shadowInitialized;
    get apiUrl(): string;
    set apiUrl(value: string);
    get workflowId(): string;
    set workflowId(value: string);
    get token(): string;
    set token(value: string);
    get theme(): 'light' | 'dark';
    set theme(value: 'light' | 'dark');
    get readOnly(): boolean;
    set readOnly(value: boolean);
    connectedCallback(): void;
    disconnectedCallback(): void;
    attributeChangedCallback(name: string, oldValue: string | null, newValue: string | null): void;
    loadWorkflow(): Promise<void>;
    executeCommand(cmd: CommandRequest): Promise<CommandResponse | null>;
    undo(): Promise<void>;
    redo(): Promise<void>;
    exportJSON(): Promise<WorkflowState | null>;
    zoomToFit(): void;
    private ensureClient;
    private setupInteractions;
    private reattachInteractions;
    private teardownInteractions;
    private emit;
}

/** Viewport manages pan/zoom transforms and coordinate conversion. */
declare class Viewport {
    scale: number;
    panX: number;
    panY: number;
    readonly minScale = 0.25;
    readonly maxScale = 2;
    readonly gridSize: number;
    constructor(gridSize?: number);
    screenToCanvas(screenX: number, screenY: number, svgRect: DOMRect): {
        x: number;
        y: number;
    };
    canvasToScreen(canvasX: number, canvasY: number, svgRect: DOMRect): {
        x: number;
        y: number;
    };
    snapToGrid(x: number, y: number): {
        x: number;
        y: number;
    };
    zoom(delta: number, centerX: number, centerY: number, svgRect: DOMRect): void;
    pan(dx: number, dy: number): void;
    setViewport(x: number, y: number, zoom: number): void;
    getTransform(): string;
}

interface RendererOptions {
    gridSize?: number;
}
declare class SVGRenderer {
    readonly svg: SVGSVGElement;
    readonly viewport: Viewport;
    private contentGroup;
    private nodeLayer;
    private edgeLayer;
    private interactionLayer;
    private gridPattern;
    private isPanning;
    private lastX;
    private lastY;
    private cullPending;
    private nodeSizeCache;
    private currentNodes;
    private readonly boundMouseMove;
    private readonly boundMouseUp;
    private readonly boundWheel;
    private readonly boundMouseDown;
    constructor(options?: RendererOptions);
    private buildSVGStructure;
    /** Render the full workflow state. Replaces all existing nodes/edges. */
    renderState(state: WorkflowState): void;
    /** Add a single node to the canvas. */
    addNode(node: NodeState): SVGGElement;
    /** Remove a node by ID. */
    removeNode(nodeId: string): void;
    /** Add an edge to the canvas. */
    addEdge(edge: EdgeState): SVGGElement;
    /** Remove an edge by ID. */
    removeEdge(edgeId: string): void;
    /** Update a node's position without re-rendering. */
    updateNodePosition(nodeId: string, x: number, y: number): void;
    /** Update edges connected to a given node (during drag). */
    updateEdgesForNode(nodeId: string): void;
    /** Viewport controls. */
    setViewport(x: number, y: number, zoom: number): void;
    /** Zoom to fit all nodes in view. */
    zoomToFit(): void;
    /** Get the node layer (for interaction handlers to query). */
    getNodeLayer(): SVGGElement;
    /** Get the edge layer. */
    getEdgeLayer(): SVGGElement;
    /** Get the interaction layer (for previews/selection rect). */
    getInteractionLayer(): SVGGElement;
    /** Parse a node's position from its transform attribute. */
    getNodePosition(el: SVGElement | Element): {
        x: number;
        y: number;
    };
    /** Start a pan gesture (called by interaction handlers). */
    startPan(e: MouseEvent): void;
    /** Tear down event listeners. */
    destroy(): void;
    private domPortY;
    private applyTransform;
    private applyCulling;
    private bindEvents;
    private onWheel;
    private onMouseDown;
    private onMouseMove;
    private onMouseUp;
}

interface SelectionCallbacks {
    onSelect: (nodeIds: string[]) => void;
    onDeselect: () => void;
    onDeleteNodes: (nodeIds: string[]) => void;
    onDeleteEdges: (edgeIds: string[]) => void;
    onDragEnd: (nodeId: string, x: number, y: number) => void;
    onMultiDragEnd: (moves: Array<{
        nodeId: string;
        x: number;
        y: number;
    }>) => void;
}
declare class SelectionManager {
    private renderer;
    private callbacks;
    private selectedNodes;
    private selectedEdges;
    private dragging;
    private dragStartX;
    private dragStartY;
    private nodeOrigPositions;
    private selRect;
    private selStartX;
    private selStartY;
    private boundMouseDown;
    private boundMouseMove;
    private boundMouseUp;
    private boundKeyDown;
    constructor(renderer: SVGRenderer, callbacks: SelectionCallbacks);
    get selectedIds(): string[];
    get selectedEdgeIds(): string[];
    attach(root: ShadowRoot | HTMLElement): void;
    detach(root: ShadowRoot | HTMLElement): void;
    select(ids: string[]): void;
    clearSelection(): void;
    private selectNode;
    private selectEdge;
    private toggleNode;
    private onMouseDown;
    private onMouseMove;
    private onMouseUp;
    private onKeyDown;
    private startDrag;
    private onDrag;
    private endDrag;
    private startRubberBand;
    private onRubberBand;
    private endRubberBand;
    private deleteSelection;
    private selectAll;
}

interface PortRef {
    nodeId: string;
    portId: string;
    portType: string;
}
interface ConnectionCallbacks {
    onConnect: (source: PortRef, target: PortRef) => void;
}
declare class ConnectionManager {
    private renderer;
    private callbacks;
    private drawing;
    private preview;
    private srcNodeId;
    private srcPortId;
    private srcPortType;
    private startX;
    private startY;
    private boundMouseDown;
    private boundMouseMove;
    private boundMouseUp;
    constructor(renderer: SVGRenderer, callbacks: ConnectionCallbacks);
    attach(): void;
    detach(): void;
    private onMouseDown;
    private onMouseMove;
    private onMouseUp;
    private createPreview;
    private updatePreview;
    private removePreview;
    private highlightTargets;
    private clearHighlights;
}

interface KeyboardCallbacks {
    onUndo: () => void;
    onRedo: () => void;
    onZoomIn: () => void;
    onZoomOut: () => void;
    onZoomToFit: () => void;
}
declare class KeyboardManager {
    private callbacks;
    private boundKeyDown;
    constructor(callbacks: KeyboardCallbacks);
    attach(root: ShadowRoot | HTMLElement): void;
    detach(root: ShadowRoot | HTMLElement): void;
    private onKeyDown;
}

declare function renderRoundedRect(width: number, height: number): SVGRectElement;
declare function renderPill(width: number, height: number): SVGRectElement;
declare function renderDiamond(width: number, height: number): SVGPolygonElement;
declare function renderHexagon(width: number, height: number): SVGPolygonElement;
declare function renderSubWorkflow(width: number, height: number): SVGGElement;
type ShapeType = 'rounded-rect' | 'pill' | 'diamond' | 'hexagon' | 'sub-workflow' | 'custom';
declare function renderShape(type: ShapeType | string, width: number, height: number): SVGElement;

/**
 * Layout calculations for nodes — ported from canvas.js and canvas.templ.
 * Mirrors the server-side Go layout logic exactly.
 */

declare function calcNodeWidth(def: NodeDefJSON | undefined): number;
declare function calcNodeHeight(def: NodeDefJSON | undefined): number;
declare function portY(index: number, count: number, nodeHeight: number): number;
declare function findPortY(def: NodeDefJSON | undefined, portId: string, isOutput: boolean): number;
declare function getBodyAttrs(def: NodeDefJSON, attributes: Record<string, unknown>): Array<{
    label: string;
    value: string;
    x: number;
    y: number;
}>;

export { type ConnectionCallbacks, ConnectionManager, GraphitiCanvasElement, type KeyboardCallbacks, KeyboardManager, type PortRef, type RendererOptions, SVGRenderer, type SelectionCallbacks, SelectionManager, type ShapeType, Viewport, calcNodeHeight, calcNodeWidth, findPortY, getBodyAttrs, portY, renderDiamond, renderHexagon, renderPill, renderRoundedRect, renderShape, renderSubWorkflow };
