// App initialization: wires canvas modules together
import { CanvasEngine } from './canvas.js';
import { CommandDispatcher } from './commands.js';
import { DragManager } from './drag.js';
import { ConnectManager } from './connect.js';
import { SelectionManager } from './select.js';
import { ClipboardManager } from './clipboard.js';
import { toast } from './toast.js';

// Expose toast globally for Alpine components and other modules
window.toast = toast;

const svg = document.getElementById('workflow-canvas');
const wfId = document.getElementById('graphiti-data')?.dataset.workflowId;
if (wfId) window.GRAPHITI = { workflowID: wfId };

if (svg && wfId) {
  const canvas = new CanvasEngine(svg);
  window.canvasEngine = canvas;

  const dispatcher = new CommandDispatcher(wfId);
  window.commandDispatcher = dispatcher;

  const selection = new SelectionManager(canvas, dispatcher);
  window.selectionManager = selection;
  dispatcher.selection = selection;

  new DragManager(canvas, dispatcher);
  new ConnectManager(canvas, dispatcher);

  const clipboard = new ClipboardManager(dispatcher);
  window.clipboardManager = clipboard;
}
