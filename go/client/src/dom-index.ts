/**
 * DOM Index
 *
 * Keeps track of dynamic slot anchors and keyed list containers so
 * list operations can insert, remove, and reorder DOM regions without
 * full re-renders.
 */

import type {ListRecord} from './types';

const slotMap = new Map<number, Node>();
const listMap = new Map<number, ListRecord>();

function collectRows(container: Element | null): Map<string, Element> {
    const rows = new Map<string, Element>();
    if (!container) return rows;

    const elements = container.querySelectorAll('[data-row-key]');
    elements.forEach((el) => {
        const key = el.getAttribute('data-row-key');
        if (!key) return;
        rows.set(key, el as Element);
    });

    return rows;
}

export function registerSlot(index: number, node: Node): void {
    if (node) {
        slotMap.set(index, node);
    }
}

export function getSlot(index: number): Node | null {
    return slotMap.get(index) || null;
}

export function unregisterSlot(index: number): void {
    slotMap.delete(index);
}

export function reset(): void {
    slotMap.clear();
    listMap.clear();
}

/**
 * Initialize all list containers upfront to avoid querySelector on first access
 * Call this during initialization with all known list slot indexes
 */
export function initLists(slotIndexes: number[]): void {
    if (!Array.isArray(slotIndexes)) return;
    if (typeof document === 'undefined') return;

    // Register all list containers in one pass
    for (const slotIndex of slotIndexes) {
        if (!listMap.has(slotIndex)) {
            const container = document.querySelector(`[data-list-slot="${slotIndex}"]`);
            if (container) {
                listMap.set(slotIndex, {container, rows: collectRows(container)});
            }
        }
    }
}

export function ensureList(slotIndex: number): ListRecord {
    // Fast path: list already registered
    if (listMap.has(slotIndex)) {
        return listMap.get(slotIndex)!;
    }

    // Slow path: fallback to querySelector (only happens if initLists wasn't called)
    const container = document.querySelector(`[data-list-slot="${slotIndex}"]`);
    if (!container) {
        throw new Error(`liveui: list slot ${slotIndex} not registered`);
    }
    const record = {container, rows: collectRows(container)};
    listMap.set(slotIndex, record);
    return record;
}

export function registerList(slotIndex: number, container: Element, rows?: Map<string, Element>): void {
    if (!container) return;
    listMap.set(slotIndex, {container, rows: rows ?? collectRows(container)});
}

export function setRow(slotIndex: number, key: string, root: Element): void {
    const list = ensureList(slotIndex);
    list.rows.set(key, root);
}

export function getRow(slotIndex: number, key: string): Element | null {
    const list = ensureList(slotIndex);
    return list.rows.get(key) || null;
}

export function deleteRow(slotIndex: number, key: string): void {
    const list = ensureList(slotIndex);
    list.rows.delete(key);
}
