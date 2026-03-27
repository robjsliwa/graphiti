/**
 * Layout calculations for nodes — ported from canvas.js and canvas.templ.
 * Mirrors the server-side Go layout logic exactly.
 */
import type { NodeDefJSON, AttrDefJSON, PortDefJSON } from '@graphiti/client';

const INPUT_LABEL_COL_WIDTH = 56;
const OUTPUT_LABEL_COL_WIDTH = 56;
const COL_GAP = 8;
const MIN_CENTER_WIDTH = 100;
const HEADER_HEIGHT = 36;
const BOTTOM_PADDING = 12;
const PORT_SPACING = 24;

function hasInputLabels(def: NodeDefJSON): boolean {
  return (def.inputs ?? []).some((p) => p.label);
}

function hasOutputLabels(def: NodeDefJSON): boolean {
  return (def.outputs ?? []).some((p) => p.label);
}

function bodyAttrCount(def: NodeDefJSON): number {
  return (def.attributes ?? []).filter(
    (a) => a.display === 'node-body' || a.display === 'both',
  ).length;
}

function hasBodyAttrs(def: NodeDefJSON): boolean {
  return bodyAttrCount(def) > 0;
}

export function calcNodeWidth(def: NodeDefJSON | undefined): number {
  const baseWidth = def?.shape?.width || 200;
  if (!def) return baseWidth;
  const hasIn = hasInputLabels(def);
  const hasOut = hasOutputLabels(def);
  const hasBody = hasBodyAttrs(def);
  if (hasBody && (hasIn || hasOut)) {
    let needed = 12 + MIN_CENTER_WIDTH + 12;
    if (hasIn) needed += INPUT_LABEL_COL_WIDTH + COL_GAP;
    if (hasOut) needed += OUTPUT_LABEL_COL_WIDTH + COL_GAP;
    return Math.max(baseWidth, needed);
  }
  return baseWidth;
}

export function calcNodeHeight(def: NodeDefJSON | undefined): number {
  if (!def) return 80;
  const inputs = (def.inputs ?? []).length;
  const outputs = (def.outputs ?? []).length;
  const maxPorts = Math.max(inputs, outputs);
  const bac = bodyAttrCount(def);
  const bodyAttrHeight = bac * 20;
  const portsHeight = maxPorts * PORT_SPACING;
  const contentHeight = Math.max(bodyAttrHeight, portsHeight);
  return Math.max(60, HEADER_HEIGHT + contentHeight + BOTTOM_PADDING);
}

export function bodyAttrX(def: NodeDefJSON): number {
  if (hasInputLabels(def) && hasBodyAttrs(def)) {
    return 12 + INPUT_LABEL_COL_WIDTH + COL_GAP;
  }
  return 12;
}

export function portY(index: number, count: number, nodeHeight: number): number {
  if (count === 0) return HEADER_HEIGHT;
  const contentHeight = nodeHeight - HEADER_HEIGHT - BOTTOM_PADDING;
  const startY = HEADER_HEIGHT + (contentHeight - (count - 1) * PORT_SPACING) / 2;
  return startY + index * PORT_SPACING;
}

export function findPortY(
  def: NodeDefJSON | undefined,
  portId: string,
  isOutput: boolean,
): number {
  if (!def) return HEADER_HEIGHT;
  const ports = isOutput ? (def.outputs ?? []) : (def.inputs ?? []);
  const h = calcNodeHeight(def);
  const idx = ports.findIndex((p) => p.id === portId);
  if (idx < 0) return h / 2;
  return portY(idx, ports.length, h);
}

export function getBodyAttrs(
  def: NodeDefJSON,
  attributes: Record<string, unknown>,
): Array<{ label: string; value: string; x: number; y: number }> {
  const result: Array<{ label: string; value: string; x: number; y: number }> = [];
  const x = bodyAttrX(def);
  let idx = 0;
  for (const attr of def.attributes ?? []) {
    if (attr.display === 'node-body' || attr.display === 'both') {
      const val = attributes?.[attr.id];
      result.push({
        label: attr.label,
        value: val != null ? String(val) : '',
        x,
        y: 52 + idx * 20,
      });
      idx++;
    }
  }
  return result;
}
