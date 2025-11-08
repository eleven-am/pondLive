import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest';
import {
  registerHandlers,
  unregisterHandlers,
  syncEventListeners,
  setupEventDelegation,
  teardownEventDelegation,
  clearHandlers,
} from '../src/events';
import { registerRefs, clearRefs, getRegistrySnapshot } from '../src/refs';

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

describe('ref metadata payload capture', () => {
  let send: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    clearHandlers();
    clearRefs();
    teardownEventDelegation();
    document.body.innerHTML = '';
    send = vi.fn();
    setupEventDelegation(send);
  });

  afterEach(() => {
    teardownEventDelegation();
    clearHandlers();
    clearRefs();
    document.body.innerHTML = '';
  });

  it('collects ref props for handler events', () => {
    document.body.innerHTML = `
      <div data-live-ref="r1" data-onclick="h1" data-state="ready">
        <button id="btn">Click</button>
      </div>
    `;

    registerRefs({
      r1: {
        tag: 'div',
        events: {
          click: { props: ['element.dataset.state', 'target.id'] },
        },
      },
    });

    registerHandlers({ h1: { event: 'click' } });
    syncEventListeners();

    const button = document.getElementById('btn');
    expect(button).not.toBeNull();

    button!.dispatchEvent(new MouseEvent('click', { bubbles: true, clientX: 12, clientY: 8 }));

    expect(send).toHaveBeenCalledTimes(1);
    expect(send).toHaveBeenCalledWith({
      hid: 'h1',
      payload: expect.objectContaining({
        type: 'click',
        'element.dataset.state': 'ready',
        'target.id': 'btn',
      }),
    });

    const record = getRegistrySnapshot().get('r1');
    expect(record?.lastPayloads?.click).toEqual(
      expect.objectContaining({
        type: 'click',
        'element.dataset.state': 'ready',
        'target.id': 'btn',
      }),
    );
  });
});
