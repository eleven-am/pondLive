import { describe, it, expect, beforeEach, vi } from 'vitest';
import { HydrationManager } from '../src/hydration';
import { LiveRuntime } from '../src/runtime';
import type { FrameMessage, DOMActionEffect } from '../src/types';

describe('DOMActionEffect', () => {
  let manager: HydrationManager;
  let runtime: LiveRuntime;
  let testElement: HTMLElement;

  beforeEach(() => {
    document.body.innerHTML = '';
    runtime = new LiveRuntime({ autoConnect: false });
    manager = new HydrationManager(runtime);
    testElement = document.createElement('div');
    testElement.id = 'test-element';
    document.body.appendChild(testElement);
  });

  const createFrameWithEffect = (effect: DOMActionEffect): FrameMessage => ({
    t: 'frame',
    sid: 'test',
    ver: 1,
    seq: 1,
    effects: [effect],
  });

  describe('dom.call', () => {
    it('should call method on element without args', () => {
      testElement.focus = vi.fn();
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('test-ref', { tag: 'div' });
      getRegistry().bindings.set('test-ref', { element: testElement, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.call',
        ref: 'test-ref',
        method: 'focus',
      });

      (manager as any).applyFrame(frame);
      expect(testElement.focus).toHaveBeenCalled();
    });

    it('should call method with args', () => {
      const scrollTo = vi.fn();
      testElement.scrollTo = scrollTo;
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('test-ref', { tag: 'div' });
      getRegistry().bindings.set('test-ref', { element: testElement, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.call',
        ref: 'test-ref',
        method: 'scrollTo',
        args: [100, 200],
      });

      (manager as any).applyFrame(frame);
      expect(scrollTo).toHaveBeenCalledWith(100, 200);
    });

    it('should skip if method does not exist', () => {
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('test-ref', { tag: 'div' });
      getRegistry().bindings.set('test-ref', { element: testElement, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.call',
        ref: 'test-ref',
        method: 'nonexistentMethod',
      });

      expect(() => (manager as any).applyFrame(frame)).not.toThrow();
    });

    it('should skip if element not found', () => {
      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.call',
        ref: 'missing-ref',
        method: 'focus',
      });

      expect(() => (manager as any).applyFrame(frame)).not.toThrow();
    });
  });

  describe('dom.set', () => {
    it('should set property value', () => {
      const input = document.createElement('input');
      input.type = 'text';
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('input-ref', { tag: 'input' });
      getRegistry().bindings.set('input-ref', { element: input, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.set',
        ref: 'input-ref',
        prop: 'value',
        value: 'test value',
      });

      (manager as any).applyFrame(frame);
      expect(input.value).toBe('test value');
    });

    it('should set numeric property', () => {
      const input = document.createElement('input');
      input.type = 'number';
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('input-ref', { tag: 'input' });
      getRegistry().bindings.set('input-ref', { element: input, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.set',
        ref: 'input-ref',
        prop: 'valueAsNumber',
        value: 42,
      });

      (manager as any).applyFrame(frame);
      expect(input.valueAsNumber).toBe(42);
    });

    it('should set custom property', () => {
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('test-ref', { tag: 'div' });
      getRegistry().bindings.set('test-ref', { element: testElement, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.set',
        ref: 'test-ref',
        prop: 'customProp',
        value: { foo: 'bar' },
      });

      (manager as any).applyFrame(frame);
      expect((testElement as any).customProp).toEqual({ foo: 'bar' });
    });
  });

  describe('dom.toggle', () => {
    it('should toggle boolean property to true', () => {
      const button = document.createElement('button');
      button.disabled = false;
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('btn-ref', { tag: 'button' });
      getRegistry().bindings.set('btn-ref', { element: button, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.toggle',
        ref: 'btn-ref',
        prop: 'disabled',
        value: true,
      });

      (manager as any).applyFrame(frame);
      expect(button.disabled).toBe(true);
    });

    it('should toggle boolean property to false', () => {
      const input = document.createElement('input');
      input.readOnly = true;
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('input-ref', { tag: 'input' });
      getRegistry().bindings.set('input-ref', { element: input, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.toggle',
        ref: 'input-ref',
        prop: 'readOnly',
        value: false,
      });

      (manager as any).applyFrame(frame);
      expect(input.readOnly).toBe(false);
    });
  });

  describe('dom.class', () => {
    it('should add class when on=true', () => {
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('test-ref', { tag: 'div' });
      getRegistry().bindings.set('test-ref', { element: testElement, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.class',
        ref: 'test-ref',
        class: 'active',
        on: true,
      });

      (manager as any).applyFrame(frame);
      expect(testElement.classList.contains('active')).toBe(true);
    });

    it('should remove class when on=false', () => {
      testElement.classList.add('active');
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('test-ref', { tag: 'div' });
      getRegistry().bindings.set('test-ref', { element: testElement, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.class',
        ref: 'test-ref',
        class: 'active',
        on: false,
      });

      (manager as any).applyFrame(frame);
      expect(testElement.classList.contains('active')).toBe(false);
    });

    it('should handle multiple class toggles', () => {
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('test-ref', { tag: 'div' });
      getRegistry().bindings.set('test-ref', { element: testElement, listeners: new Map() });

      const frame: FrameMessage = {
        t: 'frame',
        sid: 'test',
        ver: 1,
        seq: 1,
        effects: [
          { type: 'dom', kind: 'dom.class', ref: 'test-ref', class: 'foo', on: true },
          { type: 'dom', kind: 'dom.class', ref: 'test-ref', class: 'bar', on: true },
          { type: 'dom', kind: 'dom.class', ref: 'test-ref', class: 'baz', on: false },
        ],
      };

      (manager as any).applyFrame(frame);
      expect(testElement.classList.contains('foo')).toBe(true);
      expect(testElement.classList.contains('bar')).toBe(true);
      expect(testElement.classList.contains('baz')).toBe(false);
    });
  });

  describe('dom.scroll', () => {
    it('should call scrollIntoView with options', () => {
      const scrollIntoView = vi.fn();
      testElement.scrollIntoView = scrollIntoView;
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('test-ref', { tag: 'div' });
      getRegistry().bindings.set('test-ref', { element: testElement, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.scroll',
        ref: 'test-ref',
        behavior: 'smooth',
        block: 'center',
        inline: 'nearest',
      });

      (manager as any).applyFrame(frame);
      expect(scrollIntoView).toHaveBeenCalledWith({
        behavior: 'smooth',
        block: 'center',
        inline: 'nearest',
      });
    });

    it('should call scrollIntoView with partial options', () => {
      const scrollIntoView = vi.fn();
      testElement.scrollIntoView = scrollIntoView;
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('test-ref', { tag: 'div' });
      getRegistry().bindings.set('test-ref', { element: testElement, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.scroll',
        ref: 'test-ref',
        behavior: 'auto',
      });

      (manager as any).applyFrame(frame);
      expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'auto' });
    });

    it('should call scrollIntoView with empty options', () => {
      const scrollIntoView = vi.fn();
      testElement.scrollIntoView = scrollIntoView;
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('test-ref', { tag: 'div' });
      getRegistry().bindings.set('test-ref', { element: testElement, listeners: new Map() });

      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.scroll',
        ref: 'test-ref',
      });

      (manager as any).applyFrame(frame);
      expect(scrollIntoView).toHaveBeenCalledWith({});
    });
  });

  describe('combined effects', () => {
    it('should apply multiple DOM actions in sequence', () => {
      const input = document.createElement('input');
      input.focus = vi.fn();
      const getRegistry = () => (manager as any).refs;
      getRegistry().meta.set('input-ref', { tag: 'input' });
      getRegistry().bindings.set('input-ref', { element: input, listeners: new Map() });

      const frame: FrameMessage = {
        t: 'frame',
        sid: 'test',
        ver: 1,
        seq: 1,
        effects: [
          { type: 'dom', kind: 'dom.set', ref: 'input-ref', prop: 'value', value: 'hello' },
          { type: 'dom', kind: 'dom.class', ref: 'input-ref', class: 'active', on: true },
          { type: 'dom', kind: 'dom.call', ref: 'input-ref', method: 'focus' },
        ],
      };

      (manager as any).applyFrame(frame);
      expect(input.value).toBe('hello');
      expect(input.classList.contains('active')).toBe(true);
      expect(input.focus).toHaveBeenCalled();
    });
  });

  describe('error handling', () => {
    it('should not throw on invalid effect', () => {
      const frame: FrameMessage = {
        t: 'frame',
        sid: 'test',
        ver: 1,
        seq: 1,
        effects: [{ type: 'dom', kind: 'invalid.kind', ref: 'test-ref' }],
      };

      expect(() => (manager as any).applyFrame(frame)).not.toThrow();
    });

    it('should not throw when ref is missing', () => {
      const frame = createFrameWithEffect({
        type: 'dom',
        kind: 'dom.call',
        ref: '',
        method: 'focus',
      });

      expect(() => (manager as any).applyFrame(frame)).not.toThrow();
    });

    it('should not throw when kind is missing', () => {
      const frame: FrameMessage = {
        t: 'frame',
        sid: 'test',
        ver: 1,
        seq: 1,
        effects: [{ type: 'dom', kind: '', ref: 'test-ref' } as DOMActionEffect],
      };

      expect(() => (manager as any).applyFrame(frame)).not.toThrow();
    });
  });
});
