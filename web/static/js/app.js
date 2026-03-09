// App initialization: wires all modules together
import { CanvasEngine } from './canvas.js';
import { CommandDispatcher } from './commands.js';
import { DragManager } from './drag.js';
import { ConnectManager } from './connect.js';
import { SelectionManager } from './select.js';
import { ClipboardManager } from './clipboard.js';
import { toast } from './toast.js';

// Expose toast globally for use by any module
window.toast = toast;

const svg = document.getElementById('workflow-canvas');
const wfId = document.getElementById('graphiti-data')?.dataset.workflowId;
// Expose for modules that read window.GRAPHITI.workflowID
if (wfId) window.GRAPHITI = { workflowID: wfId };

if (svg && wfId) {
  // Canvas engine (pan, zoom, viewport)
  const canvas = new CanvasEngine(svg);
  window.canvasEngine = canvas;

  // Command dispatcher (sends mutations to server, undo/redo)
  const dispatcher = new CommandDispatcher(wfId);
  window.commandDispatcher = dispatcher;

  // Selection manager (click, multi-select, node drag, delete)
  const selection = new SelectionManager(canvas, dispatcher);
  window.selectionManager = selection;
  dispatcher.selection = selection;

  // Drag manager (palette to canvas)
  const drag = new DragManager(canvas, dispatcher);
  window.dragManager = drag;

  // Connect manager (port-to-port edge drawing)
  const connect = new ConnectManager(canvas, dispatcher);
  window.connectManager = connect;

  // Clipboard manager (copy, cut, paste, duplicate)
  const clipboard = new ClipboardManager(dispatcher);
  window.clipboardManager = clipboard;
}
