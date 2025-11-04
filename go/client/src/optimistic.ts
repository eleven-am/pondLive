/**
 * Optimistic Updates
 *
 * Manages optimistic DOM updates with rollback support.
 */

import { applyOps, computeInverseOps } from './patcher';
import type { DiffOp, OptimisticUpdate } from './types';

export class OptimisticUpdateManager {
  private updates = new Map<string, OptimisticUpdate>();
  private nextId: number = 0;
  private readonly onRollback?: (id: string, patches: DiffOp[]) => void;
  private readonly onError?: (error: Error, context: string) => void;
  private readonly debug: boolean = false;

  constructor(options?: {
    onRollback?: (id: string, patches: DiffOp[]) => void;
    onError?: (error: Error, context: string) => void;
    debug?: boolean;
  }) {
    this.onRollback = options?.onRollback;
    this.onError = options?.onError;
    this.debug = options?.debug || false;
  }

  /**
   * Apply optimistic update
   */
  apply(patches: DiffOp[]): string {
    const id = `opt_${this.nextId++}`;

    // Compute inverse operations BEFORE applying patches
    // This captures the current DOM state for rollback
    const inverseOps = computeInverseOps(patches);

    const update: OptimisticUpdate = {
      id,
      patches,
      inverseOps,
      timestamp: Date.now(),
    };

    this.updates.set(id, update);
    applyOps(patches);

    if (this.debug) {
      console.log('[optimistic] Applied update:', id, 'patches:', patches.length);
    }

    return id;
  }

  /**
   * Commit optimistic update (server confirmed)
   */
  commit(id: string): void {
    if (this.updates.delete(id) && this.debug) {
      console.log('[optimistic] Committed update:', id);
    }
  }

  /**
   * Rollback optimistic update (server rejected)
   */
  rollback(id: string): void {
    const update = this.updates.get(id);
    if (!update) return;

    if (this.debug) {
      console.log('[optimistic] Rolling back update:', id, 'inverse ops:', update.inverseOps.length);
    }

    try {
      // Apply inverse operations to restore previous DOM state
      applyOps(update.inverseOps);
      this.onRollback?.(id, update.patches);
    } catch (error) {
      console.error('[optimistic] Rollback error:', error);
      this.onError?.(error as Error, 'rollback');
    }

    this.updates.delete(id);
  }

  /**
   * Get pending update count
   */
  getPendingCount(): number {
    return this.updates.size;
  }

  /**
   * Clear all pending updates
   */
  clear(): void {
    this.updates.clear();
  }
}
