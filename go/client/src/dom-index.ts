/**
 * DOM Index
 *
 * Keeps track of dynamic slot anchors and keyed list containers so
 * list operations can insert, remove, and reorder DOM regions without
 * full re-renders.
 */

import type { ListRecord } from "./types";
import { onSlotRegistered, onSlotUnregistered } from "./events";

const slotMap = new Map<number, Node>();
const listMap = new Map<number, ListRecord>();

function collectRows(container: Element | null): Map<string, Element> {
  const rows = new Map<string, Element>();
  if (!container) return rows;

  const elements = container.querySelectorAll("[data-row-key]");
  elements.forEach((el) => {
    const key = el.getAttribute("data-row-key");
    if (!key) return;
    rows.set(key, el as Element);
  });

  return rows;
}

export function registerSlot(index: number, node: Node): void {
  if (!node) return;
  slotMap.set(index, node);
  onSlotRegistered(index, node);
}

export function getSlot(index: number): Node | null {
  return slotMap.get(index) ?? null;
}

export function unregisterSlot(index: number): void {
  onSlotUnregistered(index);
  slotMap.delete(index);
}

export function unregisterList(slotIndex: number): void {
  listMap.delete(slotIndex);
}

export function reset(): void {
  slotMap.forEach((_node, index) => {
    onSlotUnregistered(index);
  });
  slotMap.clear();
  listMap.clear();
}

export function ensureList(slotIndex: number): ListRecord {
  if (listMap.has(slotIndex)) {
    return listMap.get(slotIndex)!;
  }
  throw new Error(`liveui: list slot ${slotIndex} not registered`);
}

export function registerList(
  slotIndex: number,
  container: Element,
  rows?: Map<string, Element>,
): void {
  if (!container) return;
  listMap.set(slotIndex, { container, rows: rows ?? collectRows(container) });
}

export function setRow(slotIndex: number, key: string, root: Element): void {
  const list = ensureList(slotIndex);
  list.rows.set(key, root);
}

export function getRow(slotIndex: number, key: string): Element | null {
  const list = ensureList(slotIndex);
  return list.rows.get(key) ?? null;
}

export function deleteRow(slotIndex: number, key: string): void {
  const list = ensureList(slotIndex);
  list.rows.delete(key);
}
