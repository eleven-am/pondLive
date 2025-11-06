/**
 * Boot Payload Handler
 *
 * Handles SSR boot payload detection, parsing, and initial DOM registration.
 */

import * as dom from './dom-index';
import { clearHandlers, registerHandlers, syncEventListeners } from './events';
import type { BootPayload, DynamicSlot, Location } from './types';

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
    const anchors = this.collectSlotAnchors();

    // Register individual slots
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

    // Initialize list slots
    const listSlots = this.collectListSlotIndexes(boot.d);
    if (listSlots.length > 0) {
      dom.initLists(listSlots);
    }
  }

  /**
   * Collect list slot indexes from dynamic slots
   */
  private collectListSlotIndexes(dynamics?: DynamicSlot[]): number[] {
    if (!Array.isArray(dynamics)) {
      return [];
    }
    const result: number[] = [];
    dynamics.forEach((dyn, index) => {
      if (dyn && dyn.kind === 'list') {
        result.push(index);
      }
    });
    return result;
  }

  private collectSlotAnchors(): Map<number, Node> {
    const anchors = new Map<number, Node>();
    if (typeof document === 'undefined') {
      return anchors;
    }

    const elements = document.querySelectorAll('[data-slot-index]');
    elements.forEach((element) => {
      const raw = element.getAttribute('data-slot-index');
      if (!raw) return;

      raw
        .split(/\s+/)
        .map(token => token.trim())
        .filter(token => token.length > 0)
        .forEach((token) => {
          const [slotPart, childPart] = token.split('@');
          const slotId = Number(slotPart);
          if (Number.isNaN(slotId) || anchors.has(slotId)) {
            return;
          }

          let node: Node = element;
          if (childPart !== undefined) {
            const childIndex = Number(childPart);
            if (!Number.isNaN(childIndex)) {
              const child = element.childNodes.item(childIndex);
              if (child) {
                node = child;
              }
            }
          }

          anchors.set(slotId, node);
        });
    });

    return anchors;
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
