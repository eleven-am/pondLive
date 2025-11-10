import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest';
import {
  registerHandlers,
  primeHandlerBindings,
  getHandlerBindingSnapshot,
  unregisterHandlers,
  syncEventListeners,
  setupEventDelegation,
  teardownEventDelegation,
  clearHandlers,
  applyRouterAttribute,
  clearRouterAttributes,
  registerBindingsForSlot,
  getRegisteredSlotBindings,
  primeSlotBindings,
  onSlotRegistered,
  onSlotUnregistered,
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
    primeHandlerBindings(document);

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

  it('removes handler annotations after priming', () => {
    document.body.innerHTML = `
      <button id="btn" data-onclick="h1" data-onclick-listen="focus"></button>
    `;

    registerHandlers({ h1: { event: 'click' } });
    syncEventListeners();
    primeHandlerBindings(document);

    const button = document.getElementById('btn');
    expect(button).not.toBeNull();
    const snapshot = getHandlerBindingSnapshot(button!);
    expect(snapshot?.get('click')).toBe('h1');
    expect(button!.hasAttribute('data-onclick')).toBe(false);
    if (typeof button!.getAttributeNames === 'function') {
      expect(button!.getAttributeNames().some((name) => name.startsWith('data-on'))).toBe(false);
    }

    button!.dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(send).toHaveBeenCalledWith({
      hid: 'h1',
      payload: expect.objectContaining({ type: 'click' }),
    });
  });
});

describe('router metadata propagation', () => {
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

  it('includes router metadata on dispatched payloads', () => {
    document.body.innerHTML = `
      <a id="link" data-onclick="nav" href="/start"></a>
    `;

    registerHandlers({
      nav: { event: 'click' },
    });
    syncEventListeners();
    primeHandlerBindings(document);

    const link = document.getElementById('link');
    expect(link).not.toBeNull();

    applyRouterAttribute(link, 'path', '/dest');
    applyRouterAttribute(link, 'query', 'page=1');
    applyRouterAttribute(link, 'hash', '#section');
    applyRouterAttribute(link, 'replace', 'true');
    applyRouterAttribute(link, 'ignored', 'nope');

    link!.dispatchEvent(new MouseEvent('click', { bubbles: true, button: 0 }));

    expect(send).toHaveBeenCalledWith({
      hid: 'nav',
      payload: expect.objectContaining({
        type: 'click',
        'currentTarget.dataset.routerPath': '/dest',
        'currentTarget.dataset.routerQuery': 'page=1',
        'currentTarget.dataset.routerHash': '#section',
        'currentTarget.dataset.routerReplace': 'true',
      }),
    });
  });

  it('clears router metadata when attributes are removed', () => {
    document.body.innerHTML = `
      <a id="link" data-onclick="nav" href="/start"></a>
    `;

    registerHandlers({ nav: { event: 'click' } });
    syncEventListeners();
    primeHandlerBindings(document);

    const link = document.getElementById('link');
    expect(link).not.toBeNull();

    applyRouterAttribute(link, 'path', '/dest');
    clearRouterAttributes(link);

    link!.dispatchEvent(new MouseEvent('click', { bubbles: true, button: 0 }));

    expect(send).toHaveBeenCalledWith({
      hid: 'nav',
      payload: expect.not.objectContaining({
        'currentTarget.dataset.routerPath': expect.anything(),
      }),
    });
  });

  it('ignores invalid router metadata updates', () => {
    document.body.innerHTML = `
      <a id="link" data-onclick="nav"></a>
    `;

    registerHandlers({ nav: { event: 'click' } });
    syncEventListeners();
    primeHandlerBindings(document);

    const link = document.getElementById('link');
    expect(link).not.toBeNull();

    applyRouterAttribute(null, 'path', '/ignored');
    applyRouterAttribute(link, '', '/ignored');
    applyRouterAttribute(link, 'noop', '/ignored');
    applyRouterAttribute(link, 'path', '/tmp');
    applyRouterAttribute(link, 'path', '');
    applyRouterAttribute(link, 'query', undefined);

    link!.dispatchEvent(new MouseEvent('click', { bubbles: true, button: 0 }));

    expect(send).toHaveBeenCalledWith({
      hid: 'nav',
      payload: expect.not.objectContaining({
        'currentTarget.dataset.routerPath': expect.anything(),
        'currentTarget.dataset.routerQuery': expect.anything(),
      }),
    });
  });
});

describe('slot binding replication', () => {
  beforeEach(() => {
    clearHandlers();
    clearRefs();
    teardownEventDelegation();
    document.body.innerHTML = '';
    setupEventDelegation(() => undefined);
  });

  afterEach(() => {
    teardownEventDelegation();
    clearHandlers();
    clearRefs();
    document.body.innerHTML = '';
  });

  it('primes slot binding tables and returns defensive copies', () => {
    primeSlotBindings({
      5: [
        {
          event: 'input',
          handler: 'h1',
          listen: ['change'],
          props: ['target.value'],
        },
      ],
    });

    const bindings = getRegisteredSlotBindings(5);
    expect(bindings).toEqual([
      {
        event: 'input',
        handler: 'h1',
        listen: ['change'],
        props: ['target.value'],
      },
    ]);

    (bindings as any)[0].listen.push('blur');

    const refreshed = getRegisteredSlotBindings(5);
    expect(refreshed?.[0].listen).toEqual(['change']);
  });

  it('registers bindings for live slots and updates handler cache', () => {
    const slot = document.createElement('div');
    onSlotRegistered(9, slot);

    registerBindingsForSlot(9, [
      {
        event: 'click',
        handler: 'nav',
        listen: ['focus'],
        props: ['target.id'],
      },
    ]);

    const snapshot = getHandlerBindingSnapshot(slot);
    expect(snapshot?.get('click')).toBe('nav');

    const stored = getRegisteredSlotBindings(9);
    expect(stored?.[0]).toEqual({
      event: 'click',
      handler: 'nav',
      listen: ['focus'],
      props: ['target.id'],
    });

    onSlotUnregistered(9);
    expect(getHandlerBindingSnapshot(slot)).toBeUndefined();
  });

  it('normalizes invalid slot specs to empty arrays', () => {
    registerBindingsForSlot(11, null);
    expect(getRegisteredSlotBindings(11)).toEqual([]);

    registerBindingsForSlot(NaN, [
      { event: 'click', handler: 'noop' },
    ]);
    expect(getRegisteredSlotBindings(NaN)).toBeUndefined();
  });

  it('ignores malformed slot binding specs when applying to slots', () => {
    const slot = document.createElement('div');
    onSlotRegistered(21, slot);

    registerBindingsForSlot(21, [
      null as any,
      { event: '  ', handler: 'h1' } as any,
      { event: 'click', handler: '' } as any,
      { event: 'submit', handler: ' h2 ' } as any,
    ]);

    const snapshot = getHandlerBindingSnapshot(slot);
    expect(snapshot?.size).toBe(1);
    expect(snapshot?.get('submit')).toBe('h2');

    onSlotUnregistered(21);
  });

  it('reapplies primed slot bindings to registered slots', () => {
    const slot = document.createElement('div');
    onSlotRegistered(31, slot);

    primeSlotBindings({
      31: [
        {
          event: 'input',
          handler: 'h1',
          listen: ['change'],
          props: ['target.value'],
        },
      ],
    });

    const snapshot = getHandlerBindingSnapshot(slot);
    expect(snapshot?.get('input')).toBe('h1');
    const stored = getRegisteredSlotBindings(31);
    expect(stored?.[0]).toEqual({
      event: 'input',
      handler: 'h1',
      listen: ['change'],
      props: ['target.value'],
    });

    onSlotUnregistered(31);
  });
});
