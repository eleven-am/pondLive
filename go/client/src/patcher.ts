/**
 * Advanced Patcher with Maximum Optimizations
 *
 * Applies diff operations to the DOM with:
 * - Dirty checking (skip unchanged updates)
 * - Smart attribute diffing
 * - Batched DOM operations (prevent layout thrashing)
 * - DocumentFragment pooling
 * - Keyed list optimization
 * - Morphdom-style reconciliation
 * - Patch memoization
 * - Virtual scrolling support
 */

import * as dom from './dom-index';
import type {DiffOp, ListChildOp} from './types';

// ============================================================================
// Configuration
// ============================================================================

interface PatcherConfig {
    enableDirtyChecking: boolean;
    enableBatching: boolean;
    enableFragmentPooling: boolean;
    enableMemoization: boolean;
    enableVirtualScrolling: boolean;
    virtualScrollThreshold: number;
    memoizationCacheSize: number;
}

const config: PatcherConfig = {
    enableDirtyChecking: true,
    enableBatching: true,
    enableFragmentPooling: true,
    enableMemoization: true,
    enableVirtualScrolling: true,
    virtualScrollThreshold: 100, // Start virtual scrolling after 100 items
    memoizationCacheSize: 1000,
};

// ============================================================================
// Fragment Pooling (for future use)
// ============================================================================

// class FragmentPool {
//   private pool: DocumentFragment[] = [];
//   private maxSize: number = 10;
//
//   acquire(): DocumentFragment {
//     return this.pool.pop() || document.createDocumentFragment();
//   }
//
//   release(fragment: DocumentFragment): void {
//     if (this.pool.length < this.maxSize) {
//       // Clear fragment before reusing
//       while (fragment.firstChild) {
//         fragment.removeChild(fragment.firstChild);
//       }
//       this.pool.push(fragment);
//     }
//   }
// }
//
// const fragmentPool = new FragmentPool();

// ============================================================================
// Template Caching
// ============================================================================

const templateCache = document.createElement('template');
const htmlCache = new Map<string, DocumentFragment>();

function createFragment(html: string): DocumentFragment {
    if (config.enableFragmentPooling && htmlCache.has(html)) {
        // Clone cached fragment for reuse
        return htmlCache.get(html)!.cloneNode(true) as DocumentFragment;
    }

    templateCache.innerHTML = html;
    const fragment = templateCache.content.cloneNode(true) as DocumentFragment;

    // Cache popular fragments (limit size)
    if (config.enableFragmentPooling && htmlCache.size < 50) {
        htmlCache.set(html, fragment.cloneNode(true) as DocumentFragment);
    }

    return fragment;
}

// ============================================================================
// Patch Memoization
// ============================================================================

class PatchMemoizer {
    private cache = new Map<string, any>();
    private maxSize: number;

    constructor(maxSize: number) {
        this.maxSize = maxSize;
    }

    key(op: string, slotIndex: number, value: any): string {
        return `${op}:${slotIndex}:${JSON.stringify(value)}`;
    }

    has(key: string): boolean {
        return this.cache.has(key);
    }

    get(key: string): any {
        return this.cache.get(key);
    }

    set(key: string, value: any): void {
        if (this.cache.size >= this.maxSize) {
            // Simple LRU: delete first entry
            const firstKey = this.cache.keys().next().value;
            if (firstKey !== undefined) {
                this.cache.delete(firstKey);
            }
        }
        this.cache.set(key, value);
    }

    clear(): void {
        this.cache.clear();
    }
}

const memoizer = new PatchMemoizer(config.memoizationCacheSize);

// ============================================================================
// Batched DOM Operations
// ============================================================================

interface DOMBatch {
    reads: Array<() => any>;
    writes: Array<() => void>;
}

class DOMBatcher {
    private batch: DOMBatch = {reads: [], writes: []};
    private scheduled = false;

    scheduleRead(fn: () => any): void {
        this.batch.reads.push(fn);
        this.schedule();
    }

    scheduleWrite(fn: () => void): void {
        this.batch.writes.push(fn);
        this.schedule();
    }

    private schedule(): void {
        if (this.scheduled) return;
        this.scheduled = true;

        requestAnimationFrame(() => {
            this.flush();
        });
    }

    flush(): void {
        // Execute all reads first (prevent layout thrashing)
        this.batch.reads.forEach(fn => fn());

        // Then execute all writes
        this.batch.writes.forEach(fn => fn());

        // Reset
        this.batch = {reads: [], writes: []};
        this.scheduled = false;
    }

    immediate(): void {
        if (this.scheduled) {
            this.flush();
        }
    }
}

const batcher = new DOMBatcher();

// ============================================================================
// Virtual Scrolling Support
// ============================================================================

interface VirtualScrollState {
    container: Element;
    totalItems: number;
    visibleRange: { start: number; end: number };
    itemHeight: number;
    observer?: IntersectionObserver;
}

const virtualScrollStates = new Map<number, VirtualScrollState>();

function initVirtualScroll(slotIndex: number, container: Element): void {
    if (!config.enableVirtualScrolling) return;

    const state: VirtualScrollState = {
        container,
        totalItems: 0,
        visibleRange: {start: 0, end: 50},
        itemHeight: 0,
    };

    // Set up intersection observer for visibility tracking
    state.observer = new IntersectionObserver(
        (entries) => {
            entries.forEach((entry) => {
                if (entry.isIntersecting) {
                    // Item became visible
                    const target = entry.target;
                    if (target instanceof HTMLElement || target instanceof SVGElement) {
                        target.style.display = '';
                    }
                }
            });
        },
        {root: container, threshold: 0.1}
    );

    virtualScrollStates.set(slotIndex, state);
}

function shouldVirtualize(_slotIndex: number, itemCount: number): boolean {
    return config.enableVirtualScrolling && itemCount > config.virtualScrollThreshold;
}

// ============================================================================
// Dirty Checking
// ============================================================================

function ensureTextNode(node: Node | null, slotIndex: number): Node {
    if (!node) {
        throw new Error(`liveui: slot ${slotIndex} not registered`);
    }
    return node;
}

function applySetText(slotIndex: number, text: string): void {
    const node = ensureTextNode(dom.getSlot(slotIndex), slotIndex);

    // Dirty checking: skip if unchanged
    if (config.enableDirtyChecking && node.textContent === text) {
        return;
    }

    if (config.enableBatching) {
        batcher.scheduleWrite(() => {
            node.textContent = text;
        });
    } else {
        node.textContent = text;
    }
}

// ============================================================================
// Smart Attribute Diffing
// ============================================================================

function applySetAttrs(
    slotIndex: number,
    upsert: Record<string, string>,
    remove: string[]
): void {
    const node = dom.getSlot(slotIndex);
    if (!(node instanceof Element)) {
        throw new Error(`liveui: slot ${slotIndex} is not an element`);
    }

    const applyAttrs = () => {
        if (upsert) {
            for (const [k, v] of Object.entries(upsert)) {
                if (v === undefined || v === null) continue;

                // Dirty checking: only update if changed
                if (config.enableDirtyChecking && node.getAttribute(k) === String(v)) {
                    continue;
                }

                node.setAttribute(k, String(v));
            }
        }

        if (remove) {
            for (const key of remove) {
                if (node.hasAttribute(key)) {
                    node.removeAttribute(key);
                }
            }
        }
    };

    if (config.enableBatching) {
        batcher.scheduleWrite(applyAttrs);
    } else {
        applyAttrs();
    }
}

// ============================================================================
// Morphdom-style DOM Reconciliation (exported for advanced use cases)
// ============================================================================

export function morphElement(fromEl: Element, toEl: Element): void {
    // Sync attributes
    const fromAttrs = fromEl.attributes;
    const toAttrs = toEl.attributes;

    // Remove old attributes
    for (let i = fromAttrs.length - 1; i >= 0; i--) {
        const attr = fromAttrs[i];
        if (!toEl.hasAttribute(attr.name)) {
            fromEl.removeAttribute(attr.name);
        }
    }

    // Add/update new attributes
    for (let i = 0; i < toAttrs.length; i++) {
        const attr = toAttrs[i];
        if (fromEl.getAttribute(attr.name) !== attr.value) {
            fromEl.setAttribute(attr.name, attr.value);
        }
    }

    // Sync text content for text nodes
    if (fromEl.childNodes.length === 1 && fromEl.firstChild?.nodeType === Node.TEXT_NODE) {
        if (toEl.childNodes.length === 1 && toEl.firstChild?.nodeType === Node.TEXT_NODE) {
            if (fromEl.firstChild.textContent !== toEl.firstChild.textContent) {
                fromEl.firstChild.textContent = toEl.firstChild.textContent;
            }
        }
    }
}

// ============================================================================
// Slot Registration (optimized)
// ============================================================================

function registerRowSlots(slotIndexes: number[], fragment: DocumentFragment | Element): void {
    if (!Array.isArray(slotIndexes) || slotIndexes.length === 0) {
        return;
    }

    const pending = new Set(slotIndexes);
    const walker = document.createTreeWalker(fragment, NodeFilter.SHOW_ELEMENT);
    let current = walker.nextNode() as Element | null;

    // Track if we found any elements at all to avoid spurious warnings
    const foundAnyElements = current !== null;

    while (current && pending.size > 0) {
        const attr = current.getAttribute('data-slot-index');
        if (attr !== null) {
            attr
                .split(/\s+/)
                .map(token => token.trim())
                .filter(token => token.length > 0)
                .forEach((token) => {
                    const [slotPart, childPart] = token.split('@');
                    const slotId = Number(slotPart);
                    if (Number.isNaN(slotId) || !pending.has(slotId)) {
                        return;
                    }

                    let target: Node = current;
                    if (childPart !== undefined) {
                        const childIndex = Number(childPart);
                        if (!Number.isNaN(childIndex)) {
                            const child = current.childNodes.item(childIndex);
                            if (child) {
                                target = child;
                            }
                        }
                    }

                    dom.registerSlot(slotId, target);
                    pending.delete(slotId);
                });
        }
        current = walker.nextNode() as Element | null;
    }

    // Only warn if we found elements but some slots weren't resolved
    // Skip warnings if fragment had no element nodes at all
    if (pending.size > 0 && foundAnyElements) {
        pending.forEach((idx) => {
            console.warn(`liveui: slot ${idx} not resolved in inserted row`);
        });
    }
}

// ============================================================================
// Keyed List Operations with Advanced Optimizations
// ============================================================================

function applyList(slotIndex: number, childOps: ListChildOp[]): void {
    if (!Array.isArray(childOps) || childOps.length === 0) return;

    const record = dom.ensureList(slotIndex);
    const container = record.container;

    // Batch DOM reads
    const children = config.enableBatching ? Array.from(container.children) : null;

    // Track keys for virtual scrolling
    let itemCount = record.rows.size;

    for (const op of childOps) {
        if (!op || !op.length) continue;
        const kind = op[0];

        switch (kind) {
            case 'del': {
                const key = op[1];
                const row = dom.getRow(slotIndex, key);

                if (row && row.parentNode === container) {
                    const removeNode = () => {
                        container.removeChild(row);
                    };

                    if (config.enableBatching) {
                        batcher.scheduleWrite(removeNode);
                    } else {
                        removeNode();
                    }
                }

                dom.deleteRow(slotIndex, key);
                itemCount--;
                break;
            }

            case 'ins': {
                const pos = op[1];
                const payload = op[2] || {key: '', html: ''};

                // Check memoization cache
                const memoKey = memoizer.key('ins', slotIndex, payload.html);
                let fragment: DocumentFragment;

                if (config.enableMemoization && memoizer.has(memoKey)) {
                    fragment = memoizer.get(memoKey).cloneNode(true) as DocumentFragment;
                } else {
                    fragment = createFragment(payload.html || '');
                    if (config.enableMemoization) {
                        memoizer.set(memoKey, fragment.cloneNode(true));
                    }
                }

                const nodes = Array.from(fragment.childNodes);
                if (nodes.length === 0) {
                    console.warn('live: insertion payload missing nodes for key', payload.key);
                    break;
                }

                const insertNode = () => {
                    // Use batched children array if available
                    const refNode = config.enableBatching
                        ? children![pos] || null
                        : container.children[pos] || null;

                    container.insertBefore(fragment, refNode);

                    const root = nodes[0];
                    if (root instanceof Element) {
                        dom.setRow(slotIndex, payload.key, root);
                        registerRowSlots(payload.slots || [], root);

                        // Virtual scrolling: hide items outside viewport
                        if (shouldVirtualize(slotIndex, itemCount)) {
                            const state = virtualScrollStates.get(slotIndex);
                            if (state && (pos < state.visibleRange.start || pos > state.visibleRange.end)) {
                                if (root instanceof HTMLElement || root instanceof SVGElement) {
                                    root.style.display = 'none';
                                }
                            }
                        }
                    } else {
                        console.warn('live: row root is not an element for key', payload.key);
                    }
                };

                if (config.enableBatching) {
                    batcher.scheduleWrite(insertNode);
                } else {
                    insertNode();
                }

                itemCount++;
                break;
            }

            case 'mov': {
                const from = op[1];
                const to = op[2];

                if (from === to) break;

                const moveNode = () => {
                    // Use batched children array if available
                    const childArray = config.enableBatching ? children! : Array.from(container.children);
                    const child = childArray[from];

                    if (child) {
                        const refNode = to < childArray.length ? childArray[to] : null;
                        container.insertBefore(child, refNode);
                    }
                };

                if (config.enableBatching) {
                    batcher.scheduleWrite(moveNode);
                } else {
                    moveNode();
                }
                break;
            }

            default:
                console.warn('live: unknown list child op', op);
        }
    }

    // Update virtual scroll state
    if (shouldVirtualize(slotIndex, itemCount)) {
        const state = virtualScrollStates.get(slotIndex);
        if (state) {
            state.totalItems = itemCount;
        } else {
            initVirtualScroll(slotIndex, container);
        }
    }
}

// ============================================================================
// Main Apply Function with Batching
// ============================================================================

/**
 * Apply an array of diff operations to the DOM with advanced optimizations
 * @param ops - Array of operations from the server
 */
export function applyOps(ops: DiffOp[]): void {
    if (!Array.isArray(ops)) return;

    for (const op of ops) {
        if (!op || op.length === 0) continue;
        const kind = op[0];

        switch (kind) {
            case 'setText':
                applySetText(op[1], op[2]);
                break;
            case 'setAttrs':
                applySetAttrs(op[1], op[2] || {}, op[3] || []);
                break;
            case 'list':
                applyList(op[1], op.slice(2) as ListChildOp[]);
                break;
            default:
                console.warn('live: unknown op', op);
        }
    }

    // Flush batched operations if batching is enabled
    if (config.enableBatching) {
        batcher.immediate();
    }
}

// ============================================================================
// Public API for Configuration
// ============================================================================

export function configurePatcher(options: Partial<PatcherConfig>): void {
    Object.assign(config, options);
}

export function getPatcherConfig(): Readonly<PatcherConfig> {
    return {...config};
}

export function clearPatcherCaches(): void {
    htmlCache.clear();
    memoizer.clear();
    virtualScrollStates.clear();
}

export function getPatcherStats() {
    return {
        htmlCacheSize: htmlCache.size,
        memoizerCacheSize: memoizer['cache'].size,
        virtualScrollCount: virtualScrollStates.size,
    };
}

// ============================================================================
// Inverse Operations (for Optimistic Rollback)
// ============================================================================

/**
 * Compute inverse operations for a set of patches.
 * These can be applied to rollback optimistic updates.
 */
export function computeInverseOps(patches: DiffOp[]): DiffOp[] {
    const inverseOps: DiffOp[] = [];

    for (const patch of patches) {
        const [opType, slotId, ...args] = patch;

        if (opType === 'setText') {
            // Capture current text before it changes
            const element = dom.getSlot(slotId);
            if (element) {
                const currentText = element.textContent || '';
                inverseOps.push(['setText', slotId, currentText]);
            }
        } else if (opType === 'setAttrs') {
            // Capture current attributes before they change
            const element = dom.getSlot(slotId);
            if (element && element instanceof Element) {
                const [newAttrs, removeKeys] = args as [Record<string, string>, string[]];

                // Store current values of attributes that will be set
                const oldAttrs: Record<string, string> = {};
                const keysToRemove: string[] = [];

                for (const key of Object.keys(newAttrs)) {
                    const currentValue = element.getAttribute(key);
                    if (currentValue !== null) {
                        oldAttrs[key] = currentValue;
                    } else {
                        // Attribute doesn't exist, so inverse should remove it
                        keysToRemove.push(key);
                    }
                }

                // Store attributes that will be removed (so we can restore them)
                for (const key of removeKeys) {
                    const currentValue = element.getAttribute(key);
                    if (currentValue !== null) {
                        oldAttrs[key] = currentValue;
                    }
                }

                inverseOps.push(['setAttrs', slotId, oldAttrs, keysToRemove]);
            }
        } else if (opType === 'list') {
            // For list operations, compute inverse for each child operation
            const childOps = args as ListChildOp[];
            const inverseChildOps: ListChildOp[] = [];

            for (const childOp of childOps) {
                const [childOpType, ...childArgs] = childOp;

                if (childOpType === 'ins') {
                    // Insert inverse is delete
                    const [, {key}] = childArgs as [number, { key: string; html: string; slots?: number[] }];
                    inverseChildOps.push(['del', key]);
                } else if (childOpType === 'del') {
                    // Delete inverse requires capturing the element before deletion
                    // This is complex - we'd need the HTML and position
                    // For now, mark as not supported (would need re-render)
                    // Skip creating inverse for delete operations

                } else if (childOpType === 'mov') {
                    // Move inverse is move back
                    const [fromIdx, toIdx] = childArgs as [number, number];
                    inverseChildOps.push(['mov', toIdx, fromIdx]);
                }
            }

            if (inverseChildOps.length > 0) {
                inverseOps.push(['list', slotId, ...inverseChildOps.reverse()]);
            }
        }
    }

    // Reverse the order of inverse operations
    return inverseOps.reverse();
}
