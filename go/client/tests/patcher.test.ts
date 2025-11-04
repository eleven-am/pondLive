import { describe, it, expect, beforeEach, vi } from 'vitest';
import { computeInverseOps } from '../src/patcher';
import * as dom from '../src/dom-index';
import type { DiffOp } from '../src/types';

vi.mock('../src/dom-index');

describe('Patcher - computeInverseOps', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('setText inverse', () => {
    it('should capture current text before change', () => {
      const element = document.createElement('div');
      element.textContent = 'original text';

      vi.mocked(dom.getSlot).mockReturnValue(element);

      const patches: DiffOp[] = [['setText', 1, 'new text']];
      const inverse = computeInverseOps(patches);

      expect(inverse).toEqual([['setText', 1, 'original text']]);
    });

    it('should handle empty text', () => {
      const element = document.createElement('div');
      element.textContent = '';

      vi.mocked(dom.getSlot).mockReturnValue(element);

      const patches: DiffOp[] = [['setText', 1, 'new text']];
      const inverse = computeInverseOps(patches);

      expect(inverse).toEqual([['setText', 1, '']]);
    });

    it('should handle missing element', () => {
      vi.mocked(dom.getSlot).mockReturnValue(null);

      const patches: DiffOp[] = [['setText', 1, 'new text']];
      const inverse = computeInverseOps(patches);

      expect(inverse).toEqual([]);
    });
  });

  describe('setAttrs inverse', () => {
    it('should capture current attributes', () => {
      const element = document.createElement('div');
      element.setAttribute('class', 'old-class');
      element.setAttribute('id', 'old-id');

      vi.mocked(dom.getSlot).mockReturnValue(element);

      const patches: DiffOp[] = [
        ['setAttrs', 1, { class: 'new-class', id: 'new-id' }, []],
      ];
      const inverse = computeInverseOps(patches);

      expect(inverse).toEqual([
        ['setAttrs', 1, { class: 'old-class', id: 'old-id' }, []],
      ]);
    });

    it('should mark new attributes for removal in inverse', () => {
      const element = document.createElement('div');
      element.setAttribute('existing', 'value');

      vi.mocked(dom.getSlot).mockReturnValue(element);

      const patches: DiffOp[] = [
        ['setAttrs', 1, { existing: 'changed', newAttr: 'value' }, []],
      ];
      const inverse = computeInverseOps(patches);

      expect(inverse).toEqual([
        ['setAttrs', 1, { existing: 'value' }, ['newAttr']],
      ]);
    });

    it('should restore removed attributes', () => {
      const element = document.createElement('div');
      element.setAttribute('toRemove', 'value');

      vi.mocked(dom.getSlot).mockReturnValue(element);

      const patches: DiffOp[] = [['setAttrs', 1, {}, ['toRemove']]];
      const inverse = computeInverseOps(patches);

      expect(inverse).toEqual([['setAttrs', 1, { toRemove: 'value' }, []]]);
    });

    it('should handle missing element', () => {
      vi.mocked(dom.getSlot).mockReturnValue(null);

      const patches: DiffOp[] = [['setAttrs', 1, { class: 'new' }, []]];
      const inverse = computeInverseOps(patches);

      expect(inverse).toEqual([]);
    });

    it('should handle non-Element nodes', () => {
      const textNode = document.createTextNode('text');
      vi.mocked(dom.getSlot).mockReturnValue(textNode);

      const patches: DiffOp[] = [['setAttrs', 1, { class: 'new' }, []]];
      const inverse = computeInverseOps(patches);

      expect(inverse).toEqual([]);
    });
  });

  describe('list operations inverse', () => {
    it('should create delete for insert', () => {
      const patches: DiffOp[] = [
        ['list', 1, ['ins', 0, { key: 'item-1', html: '<div>Item</div>' }]],
      ];
      const inverse = computeInverseOps(patches);

      expect(inverse).toEqual([['list', 1, ['del', 'item-1']]]);
    });

    it('should create reverse move for move', () => {
      const patches: DiffOp[] = [['list', 1, ['mov', 2, 5]]];
      const inverse = computeInverseOps(patches);

      expect(inverse).toEqual([['list', 1, ['mov', 5, 2]]]);
    });

    it('should skip delete operations (no inverse)', () => {
      const patches: DiffOp[] = [['list', 1, ['del', 'item-1']]];
      const inverse = computeInverseOps(patches);

      // Delete operations are skipped as they'd require full HTML capture
      expect(inverse).toEqual([]);
    });

    it('should handle multiple list operations in reverse order', () => {
      const patches: DiffOp[] = [
        [
          'list',
          1,
          ['ins', 0, { key: 'a', html: '<div>A</div>' }],
          ['mov', 1, 2],
          ['ins', 3, { key: 'b', html: '<div>B</div>' }],
        ],
      ];
      const inverse = computeInverseOps(patches);

      // Operations should be reversed
      expect(inverse).toEqual([
        ['list', 1, ['del', 'b'], ['mov', 2, 1], ['del', 'a']],
      ]);
    });
  });

  describe('multiple operations', () => {
    it('should reverse order of operations', () => {
      const element1 = document.createElement('div');
      element1.textContent = 'text1';
      const element2 = document.createElement('div');
      element2.textContent = 'text2';

      vi.mocked(dom.getSlot)
        .mockReturnValueOnce(element1)
        .mockReturnValueOnce(element2);

      const patches: DiffOp[] = [
        ['setText', 1, 'new1'],
        ['setText', 2, 'new2'],
      ];
      const inverse = computeInverseOps(patches);

      // Operations should be in reverse order
      expect(inverse).toEqual([
        ['setText', 2, 'text2'],
        ['setText', 1, 'text1'],
      ]);
    });

    it('should handle mixed operation types', () => {
      const element1 = document.createElement('div');
      element1.textContent = 'old text';
      const element2 = document.createElement('div');
      element2.setAttribute('class', 'old');

      vi.mocked(dom.getSlot)
        .mockReturnValueOnce(element1)
        .mockReturnValueOnce(element2);

      const patches: DiffOp[] = [
        ['setText', 1, 'new text'],
        ['setAttrs', 2, { class: 'new' }, []],
        ['list', 3, ['ins', 0, { key: 'item', html: '<div></div>' }]],
      ];
      const inverse = computeInverseOps(patches);

      expect(inverse).toEqual([
        ['list', 3, ['del', 'item']],
        ['setAttrs', 2, { class: 'old' }, []],
        ['setText', 1, 'old text'],
      ]);
    });
  });
});
