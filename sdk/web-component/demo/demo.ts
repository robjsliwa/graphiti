import '../src/index.js';
import type { GraphitiCanvasElement } from '../src/graphiti-canvas.js';

const canvas = document.getElementById('canvas') as GraphitiCanvasElement;
const apiUrlInput = document.getElementById('api-url') as HTMLInputElement;
const tokenInput = document.getElementById('token') as HTMLInputElement;
const workflowSelect = document.getElementById('workflow-select') as HTMLSelectElement;
const eventLog = document.getElementById('event-log') as HTMLDivElement;

function logEvent(name: string, detail: unknown): void {
  const entry = document.createElement('div');
  entry.className = 'event-entry';

  const timeSpan = document.createElement('span');
  timeSpan.className = 'event-time';
  timeSpan.textContent = new Date().toLocaleTimeString();

  const nameSpan = document.createElement('span');
  nameSpan.className = 'event-name';
  nameSpan.textContent = ` ${name}`;

  const detailText = document.createElement('div');
  detailText.textContent = JSON.stringify(detail, null, 0).slice(0, 200);

  entry.appendChild(timeSpan);
  entry.appendChild(nameSpan);
  entry.appendChild(detailText);
  eventLog.appendChild(entry);
  eventLog.scrollTop = eventLog.scrollHeight;
}

// Listen to canvas events
const events = [
  'graphiti:node-selected',
  'graphiti:node-deselected',
  'graphiti:workflow-changed',
  'graphiti:command-executed',
  'graphiti:error',
];
for (const name of events) {
  canvas.addEventListener(name, (e) => logEvent(name, (e as CustomEvent).detail));
}

// Load workflows
document.getElementById('load-btn')!.addEventListener('click', async () => {
  const apiUrl = apiUrlInput.value;
  const token = tokenInput.value;
  canvas.setAttribute('api-url', apiUrl);
  canvas.setAttribute('token', token);

  try {
    const resp = await fetch(`${apiUrl}/api/workflows`, {
      headers: { Authorization: `Bearer ${token}`, Accept: 'application/json' },
    });
    const data = await resp.json();
    // Clear existing options
    while (workflowSelect.options.length > 1) {
      workflowSelect.remove(1);
    }
    for (const wf of data.items || []) {
      const opt = document.createElement('option');
      opt.value = wf.ID;
      opt.textContent = `${wf.Name} (${wf.ID.slice(0, 8)})`;
      workflowSelect.appendChild(opt);
    }
  } catch (err) {
    logEvent('fetch-error', { message: (err as Error).message });
  }
});

// Select workflow
workflowSelect.addEventListener('change', () => {
  const wfId = workflowSelect.value;
  if (wfId) {
    canvas.setAttribute('workflow-id', wfId);
  }
});

// Theme toggle
document.getElementById('theme-btn')!.addEventListener('click', () => {
  const current = canvas.getAttribute('theme') || 'light';
  canvas.setAttribute('theme', current === 'light' ? 'dark' : 'light');
});

// Zoom to fit
document.getElementById('fit-btn')!.addEventListener('click', () => {
  canvas.zoomToFit();
});
