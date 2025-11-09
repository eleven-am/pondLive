/**
 * Boot Payload Handler
 *
 * Handles SSR boot payload detection, parsing, and initial DOM registration.
 */

import * as dom from './dom-index';
import { clearHandlers, primeSlotBindings, registerHandlers, syncEventListeners } from './events';
import { getRenderableChild, initializeComponentMarkers, resolveParentNode } from './componentMarkers';
import { bindRefsInTree, clearRefs, registerRefs } from './refs';
import type { BootPayload, Location, SlotMeta } from './types';

export class BootHandler {
  private boot: BootPayload | null = null;
  private readonly debug: boolean = false;

  constructor(options?: { debug?: boolean }) {
    this.debug = options?.debug || false;
  }

  /**
   * Load boot payload from explicit source or auto-detect
   */
  load(explicit?: BootPayload | null): BootPayload | null {
    const candidate = explicit ?? this.detect();
    if (!candidate || typeof candidate.sid !== 'string' || candidate.sid.length === 0) {
      this.log('No boot payload detected or payload invalid');
      return null;
    }
    this.apply(candidate);
    return candidate;
  }

  /**
   * Detect boot payload from window or DOM
   */
  private detect(): BootPayload | null {
    // Try window global first
    if (typeof window !== 'undefined') {
      const globalBoot = (window as any).__LIVEUI_BOOT__;
      if (globalBoot && typeof globalBoot === 'object' && typeof globalBoot.sid === 'string') {
        return globalBoot as BootPayload;
      }
    }

    // Try script tag in DOM
    if (typeof document !== 'undefined') {
      const script = document.getElementById('live-boot');
      const payload = script?.textContent;
      if (payload) {
        try {
          return JSON.parse(payload) as BootPayload;
        } catch (error) {
          this.log('Failed to parse boot payload from DOM', error);
        }
      }
    }

    return null;
  }

  /**
   * Apply boot payload: register handlers, slots, and sync location
   */
  private apply(boot: BootPayload): void {
    this.boot = boot;

    // Register event handlers
    if (boot.handlers) {
      clearHandlers();
      registerHandlers(boot.handlers);
      syncEventListeners();
    }

    primeSlotBindings(boot.bindings);

    // Register element refs and index current DOM
    clearRefs();
    registerRefs(boot.refs);
    if (typeof document !== 'undefined') {
      bindRefsInTree(document);
      initializeComponentMarkers(boot.markers ?? null, document);
    }

    // Register DOM slots
    this.registerInitialDom(boot);

    // Sync browser location with boot location
    if (typeof window !== 'undefined' && boot.location) {
      const queryPart = boot.location.q ? `?${boot.location.q}` : '';
      const hashPart = boot.location.hash ? `#${boot.location.hash}` : '';
      const target = `${boot.location.path}${queryPart}${hashPart}`;
      const current = `${window.location.pathname}${window.location.search}${window.location.hash}`;
      if (target && current !== target) {
        window.history.replaceState({}, '', target);
      }
    }
  }

  /**
   * Register initial DOM slots from boot payload
   */
  private registerInitialDom(boot: BootPayload): void {
    if (typeof document === 'undefined') {
      return;
    }

    dom.reset();
    const anchors = this.resolveSlotAnchors(boot.slots);
    if (Array.isArray(boot.slots)) {
      for (const slot of boot.slots) {
        if (!slot || typeof slot.anchorId !== 'number') continue;
        const node = anchors.get(slot.anchorId);
        if (node) {
          dom.registerSlot(slot.anchorId, node);
        } else if (this.debug) {
          console.warn(`liveui: slot ${slot.anchorId} not registered during boot`);
        }
      }
    }

    this.registerInitialLists(boot, anchors);
  }

  private registerInitialLists(boot: BootPayload, anchors: Map<number, Node>): void {
    if (!Array.isArray(boot.d)) {
      return;
    }

    boot.d.forEach((dyn, index) => {
      if (!dyn || dyn.kind !== 'list') {
        return;
      }

      const container = anchors.get(index);
      if (!(container instanceof Element)) {
        if (this.debug) {
          console.warn(`liveui: list slot ${index} missing container anchor during boot`);
        }
        return;
      }

      const rows = new Map<string, Element>();
      const listRows = Array.isArray(dyn.list) ? dyn.list : [];
      let position = 0;
      for (const row of listRows) {
        if (!row || typeof row.key !== 'string') {
          continue;
        }
        const child = getRenderableChild(container, position);
        position += 1;
        if (!(child instanceof Element)) {
          if (this.debug) {
            console.warn(`liveui: list slot ${index} row ${row.key} missing element at position ${position - 1}`);
          }
          continue;
        }
        rows.set(row.key, child);
        if (Array.isArray(row.slotMeta) && row.slotMeta.length > 0) {
          const rowAnchors = this.resolveSlotAnchors(row.slotMeta, child);
          for (const [slotId, node] of rowAnchors.entries()) {
            dom.registerSlot(slotId, node);
          }
        }
      }

      dom.registerList(index, container, rows);
    });
  }

  resolveSlotAnchors(slots?: SlotMeta[], root?: ParentNode | null): Map<number, Node> {
    const anchors = new Map<number, Node>();
    const hasDocument = typeof document !== 'undefined';
    const targetRoot: ParentNode | null = root ?? (hasDocument ? document : null);
    if (!targetRoot || !Array.isArray(slots)) {
      return anchors;
    }

    for (const slot of slots) {
      if (!slot || typeof slot.anchorId !== 'number') continue;
      const node = this.resolveAnchorFromMeta(slot, targetRoot);
      if (node) {
        anchors.set(slot.anchorId, node);
      }
    }
    return anchors;
  }

  private resolveAnchorFromMeta(slot: SlotMeta, root: ParentNode): Node | null {
    if (!slot || typeof slot.anchorId !== 'number') {
      return null;
    }

    let parentPath: number[] | undefined;
    if (Array.isArray(slot.parentPath)) {
      parentPath = [];
      for (const value of slot.parentPath) {
        const index = Number(value);
        if (!Number.isInteger(index) || index < 0) {
          parentPath = undefined;
          break;
        }
        parentPath.push(index);
      }
    }

    const parent = resolveParentNode(root, parentPath);
    if (!parent) {
      return null;
    }

    if (slot.childIndex === undefined || slot.childIndex === null) {
      return parent;
    }

    const childIndex = Number(slot.childIndex);
    if (!Number.isInteger(childIndex) || childIndex < 0) {
      return null;
    }

    const child = getRenderableChild(parent, childIndex);
    return child ?? null;
  }

  private resolveAnchorFromMeta(slot: SlotMeta, root: ParentNode): Node | null {
    if (!slot || typeof slot.anchorId !== 'number') {
      return null;
    }

    let parentPath: number[] | undefined;
    if (Array.isArray(slot.parentPath)) {
      parentPath = [];
      for (const value of slot.parentPath) {
        const index = Number(value);
        if (!Number.isInteger(index) || index < 0) {
          parentPath = undefined;
          break;
        }
        parentPath.push(index);
      }
    }

    const parent = resolveParentNode(root, parentPath);
    if (!parent) {
      return null;
    }

    if (slot.childIndex === undefined || slot.childIndex === null) {
      return parent;
    }

    const childIndex = Number(slot.childIndex);
    if (!Number.isInteger(childIndex) || childIndex < 0) {
      return null;
    }

    const child = getRenderableChild(parent, childIndex);
    return child ?? null;
  }

  /**
   * Get current boot payload
   */
  get(): BootPayload | null {
    return this.boot;
  }

  /**
   * Ensure boot payload exists, throw if missing
   */
  ensure(): BootPayload {
    if (!this.boot || !this.boot.sid) {
      throw new Error('LiveUI: boot payload is required before connecting');
    }
    return this.boot;
  }

  /**
   * Get join location from browser or boot fallback
   */
  getJoinLocation(): Location {
    const fallback = this.boot?.location ?? { path: '/', q: '', hash: '' };
    if (typeof window === 'undefined') {
      return fallback;
    }
    const path = window.location.pathname || fallback.path || '/';
    const rawQuery = window.location.search ?? '';
    const query = rawQuery.startsWith('?') ? rawQuery.substring(1) : rawQuery;
    const rawHash = window.location.hash ?? '';
    const hash = rawHash.startsWith('#') ? rawHash.substring(1) : rawHash;
    return {
      path,
      q: query,
      hash,
    };
  }

  private log(...args: any[]): void {
    if (this.debug) {
      console.log('[boot]', ...args);
    }
  }
}
