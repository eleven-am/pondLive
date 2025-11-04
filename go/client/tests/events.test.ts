import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest';
import {
  registerHandlers,
  unregisterHandlers,
  syncEventListeners,
  setupEventDelegation,
  teardownEventDelegation,
  clearHandlers,
} from '../src/events';

describe('event delegation management', () => {
  let originalAdd: typeof document.addEventListener;
  let originalRemove: typeof document.removeEventListener;
  const added: string[] = [];
  const removed: string[] = [];

  beforeEach(() => {
    added.length = 0;
    removed.length = 0;

    originalAdd = document.addEventListener;
    originalRemove = document.removeEventListener;

    document.addEventListener = vi.fn(function (this: Document, type: any, listener: any, options: any) {
      added.push(type as string);
      return originalAdd.call(this, type, listener as EventListener, options);
    }) as typeof document.addEventListener;

    document.removeEventListener = vi.fn(function (this: Document, type: any, listener: any, options: any) {
      removed.push(type as string);
      return originalRemove.call(this, type, listener as EventListener, options);
    }) as typeof document.removeEventListener;

    clearHandlers();
    teardownEventDelegation();
    setupEventDelegation(() => undefined);
  });

  afterEach(() => {
    document.addEventListener = originalAdd;
    document.removeEventListener = originalRemove;
    clearHandlers();
    teardownEventDelegation();
  });

  it('installs and removes listeners when handlers change', () => {
    registerHandlers({ h1: { event: 'scroll' } });
    syncEventListeners();
    expect(added).toContain('scroll');

    added.length = 0;
    unregisterHandlers(['h1']);
    syncEventListeners();
    expect(removed).toContain('scroll');
  });

  it('updates listeners when handler type changes', () => {
    registerHandlers({ h1: { event: 'scroll' } });
    syncEventListeners();
    added.length = 0;
    removed.length = 0;

    registerHandlers({ h1: { event: 'focus' } });
    syncEventListeners();

    expect(removed).toContain('scroll');
    expect(added).toContain('focus');
  });

  it('teardown removes installed listeners', () => {
    registerHandlers({ h1: { event: 'input' } });
    syncEventListeners();
    removed.length = 0;

    teardownEventDelegation();
    expect(removed).toEqual(expect.arrayContaining(['click', 'input', 'change', 'submit']));
  });
});
