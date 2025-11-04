import { describe, it, expect, beforeEach, vi } from 'vitest';
import { OptimisticUpdateManager } from '../src/optimistic';
import * as patcher from '../src/patcher';
import type { DiffOp } from '../src/types';

vi.mock('../src/patcher', () => ({
  applyOps: vi.fn(),
  computeInverseOps: vi.fn((patches: DiffOp[]) => {
    // Mock inverse: reverse the patches
    return patches.map((p) => ['setText', p[1], 'original'] as DiffOp).reverse();
  }),
}));

describe('OptimisticUpdateManager', () => {
  let manager: OptimisticUpdateManager;
  let rollbackSpy: ReturnType<typeof vi.fn>;
  let errorSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    rollbackSpy = vi.fn();
    errorSpy = vi.fn();

    manager = new OptimisticUpdateManager({
      onRollback: rollbackSpy,
      onError: errorSpy,
      debug: false,
    });

    vi.clearAllMocks();
  });

  describe('apply', () => {
    it('should apply optimistic update and return id', () => {
      const patches: DiffOp[] = [
        ['setText', 1, 'new text'],
        ['setAttrs', 2, { class: 'active' }, []],
      ];

      const id = manager.apply(patches);

      expect(id).toMatch(/^opt_\d+$/);
      expect(patcher.computeInverseOps).toHaveBeenCalledWith(patches);
      expect(patcher.applyOps).toHaveBeenCalledWith(patches);
    });

    it('should generate unique ids', () => {
      const id1 = manager.apply([['setText', 1, 'test']]);
      const id2 = manager.apply([['setText', 2, 'test']]);

      expect(id1).not.toBe(id2);
    });

    it('should track pending updates', () => {
      manager.apply([['setText', 1, 'test']]);
      expect(manager.getPendingCount()).toBe(1);

      manager.apply([['setText', 2, 'test']]);
      expect(manager.getPendingCount()).toBe(2);
    });
  });

  describe('commit', () => {
    it('should remove committed update', () => {
      const id = manager.apply([['setText', 1, 'test']]);
      expect(manager.getPendingCount()).toBe(1);

      manager.commit(id);
      expect(manager.getPendingCount()).toBe(0);
    });

    it('should handle non-existent id gracefully', () => {
      expect(() => manager.commit('non-existent')).not.toThrow();
    });
  });

  describe('rollback', () => {
    it('should apply inverse operations on rollback', () => {
      const patches: DiffOp[] = [
        ['setText', 1, 'new'],
        ['setText', 2, 'text'],
      ];

      const id = manager.apply(patches);

      vi.clearAllMocks();
      manager.rollback(id);

      // Should apply inverse operations
      expect(patcher.applyOps).toHaveBeenCalledWith([
        ['setText', 2, 'original'],
        ['setText', 1, 'original'],
      ]);

      // Should emit rollback event
      expect(rollbackSpy).toHaveBeenCalledWith(id, patches);

      // Should remove from pending
      expect(manager.getPendingCount()).toBe(0);
    });

    it('should handle rollback errors', () => {
      const patches: DiffOp[] = [['setText', 1, 'test']];
      const id = manager.apply(patches);

      const error = new Error('Rollback failed');
      vi.mocked(patcher.applyOps).mockImplementationOnce(() => {
        throw error;
      });

      manager.rollback(id);

      expect(errorSpy).toHaveBeenCalledWith(error, 'rollback');
      expect(manager.getPendingCount()).toBe(0);
    });

    it('should handle non-existent id gracefully', () => {
      expect(() => manager.rollback('non-existent')).not.toThrow();
      expect(patcher.applyOps).not.toHaveBeenCalled();
    });
  });

  describe('clear', () => {
    it('should clear all pending updates', () => {
      manager.apply([['setText', 1, 'test1']]);
      manager.apply([['setText', 2, 'test2']]);
      manager.apply([['setText', 3, 'test3']]);

      expect(manager.getPendingCount()).toBe(3);

      manager.clear();
      expect(manager.getPendingCount()).toBe(0);
    });
  });

  describe('multiple operations', () => {
    it('should handle multiple concurrent optimistic updates', () => {
      const id1 = manager.apply([['setText', 1, 'update1']]);
      const id2 = manager.apply([['setText', 2, 'update2']]);
      const id3 = manager.apply([['setText', 3, 'update3']]);

      expect(manager.getPendingCount()).toBe(3);

      // Commit one
      manager.commit(id2);
      expect(manager.getPendingCount()).toBe(2);

      // Rollback one
      manager.rollback(id1);
      expect(manager.getPendingCount()).toBe(1);

      // Commit the last
      manager.commit(id3);
      expect(manager.getPendingCount()).toBe(0);
    });
  });
});
