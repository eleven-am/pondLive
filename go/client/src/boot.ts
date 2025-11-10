/**
 * Boot Payload Handler
 *
 * Handles SSR boot payload detection, parsing, and initial DOM registration.
 */

import * as dom from './dom-index';
import { clearHandlers, primeSlotBindings, registerHandlers, syncEventListeners } from './events';
import { initializeComponentMarkers } from './componentMarkers';
import { bindRefsInTree, clearRefs, registerRefs } from './refs';
import type { BootPayload, Location } from './types';
import { resolveListContainer, resolveSlotTarget } from './slot-resolver';

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
    if (!Array.isArray(boot.slots)) {
      return;
    }

    for (const slot of boot.slots) {
      if (!slot || typeof slot.anchorId !== 'number') continue;
      const target = resolveSlotTarget(slot);
      if (target) {
        dom.registerSlot(slot.anchorId, target);
      } else if (this.debug) {
        console.warn(`liveui: slot ${slot.anchorId} not registered during boot`);
      }

      const listContainer = resolveListContainer(slot);
      if (listContainer) {
        dom.registerList(slot.anchorId, listContainer);
      } else if (slot.list && this.debug) {
        console.warn(`liveui: list slot ${slot.anchorId} container not found during boot`);
      }
    }
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
