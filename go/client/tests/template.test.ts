import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import LiveUI from '../src/index';
import * as dom from '../src/dom-index';
import * as events from '../src/events';
import * as refs from '../src/refs';
import * as manifest from '../src/manifest';
import { registerComponentRange, resetComponentRanges } from '../src/componentRanges';
import type { FrameMessage, TemplateMessage } from '../src/types';

describe('LiveUI template hydration', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
    resetComponentRanges();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    document.body.innerHTML = '';
    resetComponentRanges();
  });

  it('applies root template payloads and registers anchors', () => {
    const live = new LiveUI({ autoConnect: false, debug: true });

    const resetSpy = vi.spyOn(dom, 'reset');
    const clearHandlersSpy = vi.spyOn(events, 'clearHandlers');
    const registerHandlersSpy = vi.spyOn(events, 'registerHandlers');
    const syncListenersSpy = vi.spyOn(events, 'syncEventListeners');
    const primeSlotsSpy = vi.spyOn(events, 'primeSlotBindings');
    const unregisterRefsSpy = vi.spyOn(refs, 'unregisterRefs');
    const registerRefsSpy = vi.spyOn(refs, 'registerRefs');
    const bindRefsSpy = vi.spyOn(refs, 'bindRefsInTree');
    const registerSlotSpy = vi.spyOn(dom, 'registerSlot');
    const applyComponentRangesSpy = vi
      .spyOn(manifest, 'applyComponentRanges')
      .mockReturnValue(new Map());

    const slotNode = document.createTextNode('');
    vi.spyOn(manifest, 'resolveSlotAnchors').mockReturnValue(new Map([[1, slotNode]]));
    vi.spyOn(manifest, 'resolveListContainers').mockReturnValue(new Map());

    const msg: TemplateMessage = {
      t: 'template',
      sid: 'root-session',
      ver: 2,
      html: '<main><span data-live-ref="foo">hi</span></main>',
      s: [],
      d: [],
      slots: [{ anchorId: 1 }],
      slotPaths: [],
      listPaths: [],
      componentPaths: [],
      handlers: { h1: { event: 'click' } },
      bindings: { slots: { 1: [{ event: 'click', handler: 'h1' }] } },
      refs: { add: { foo: { tag: 'span' } }, del: ['bar'] },
    };

    (live as any).handleTemplate(msg);

    expect(document.body.innerHTML).toContain('<main>');
    expect(resetSpy).toHaveBeenCalled();
    expect(clearHandlersSpy).toHaveBeenCalled();
    expect(registerHandlersSpy).toHaveBeenCalledWith(msg.handlers);
    expect(syncListenersSpy).toHaveBeenCalled();
    expect(primeSlotsSpy).toHaveBeenCalledWith(msg.bindings?.slots ?? null);
    expect(unregisterRefsSpy).toHaveBeenCalledWith(msg.refs?.del);
    expect(registerRefsSpy).toHaveBeenCalledWith(msg.refs?.add);
    expect(bindRefsSpy).toHaveBeenCalled();
    expect(registerSlotSpy).toHaveBeenCalledWith(1, slotNode);
    expect(applyComponentRangesSpy).toHaveBeenCalled();
  });

  it('applies component-scoped template payloads', () => {
    const live = new LiveUI({ autoConnect: false });

    const container = document.createElement('div');
    container.innerHTML = '<span>old</span>';
    document.body.appendChild(container);
    registerComponentRange('cmp', { container, startIndex: 0, endIndex: container.childNodes.length - 1 });

    const unregisterSlotSpy = vi.spyOn(dom, 'unregisterSlot');
    const registerSlotSpy = vi.spyOn(dom, 'registerSlot');
    const registerBindingsSpy = vi.spyOn(events, 'registerBindingsForSlot');
    const registerHandlersSpy = vi.spyOn(events, 'registerHandlers');
    const registerRefsSpy = vi.spyOn(refs, 'registerRefs');
    const bindRefsSpy = vi.spyOn(refs, 'bindRefsInTree');

    vi.spyOn(manifest, 'applyComponentRanges').mockReturnValue(new Map());
    vi.spyOn(manifest, 'resolveSlotAnchors').mockImplementation(() => {
      const node = container.querySelector('span')?.firstChild ?? null;
      const map = new Map<number, Node>();
      if (node) {
        map.set(7, node);
      }
      return map;
    });
    vi.spyOn(manifest, 'resolveListContainers').mockReturnValue(new Map());

    const msg: TemplateMessage = {
      t: 'template',
      sid: 'root-session',
      ver: 3,
      html: '<span data-live-ref="comp-ref">updated</span>',
      s: [],
      d: [],
      slots: [{ anchorId: 7 }],
      slotPaths: [],
      listPaths: [],
      componentPaths: [],
      handlers: { h2: { event: 'input' } },
      bindings: { slots: { 7: [{ event: 'input', handler: 'h2' }] } },
      refs: { add: { 'comp-ref': { tag: 'span' } } },
      scope: { componentId: 'cmp' },
    };

    (live as any).handleTemplate(msg);

    expect(container.textContent).toContain('updated');
    expect(unregisterSlotSpy).toHaveBeenCalledWith(7);
    expect(registerSlotSpy).toHaveBeenCalled();
    expect(registerBindingsSpy).toHaveBeenCalledWith(7, expect.any(Array));
    expect(registerHandlersSpy).toHaveBeenCalledWith(msg.handlers);
    expect(registerRefsSpy).toHaveBeenCalledWith(msg.refs?.add);
    expect(bindRefsSpy).toHaveBeenCalledWith(container);
  });

  it('reuses cached markup when template hash is provided for root payloads', () => {
    const live = new LiveUI({ autoConnect: false, debug: true });

    vi.spyOn(manifest, 'applyComponentRanges').mockReturnValue(new Map());
    vi.spyOn(manifest, 'resolveSlotAnchors').mockReturnValue(new Map());
    vi.spyOn(manifest, 'resolveListContainers').mockReturnValue(new Map());

    const initial: TemplateMessage = {
      t: 'template',
      sid: 'root-session',
      ver: 1,
      html: '<main><span>alpha</span></main>',
      templateHash: 'hash:root:alpha',
      s: [],
      d: [],
      slots: [],
      slotPaths: [],
      listPaths: [],
      componentPaths: [],
    };

    (live as any).handleTemplate(initial);
    expect(document.body.innerHTML).toContain('alpha');

    document.body.innerHTML = '<div>mutated</div>';

    const cachedOnly: TemplateMessage = {
      t: 'template',
      sid: 'root-session',
      ver: 2,
      templateHash: 'hash:root:alpha',
      s: [],
      d: [],
      slots: [],
      slotPaths: [],
      listPaths: [],
      componentPaths: [],
    };

    (live as any).handleTemplate(cachedOnly);

    expect(document.body.innerHTML).toContain('alpha');
  });

  it('reuses cached markup for component-scoped template payloads', () => {
    const live = new LiveUI({ autoConnect: false });

    const container = document.createElement('div');
    container.innerHTML = '<span>seed</span>';
    document.body.appendChild(container);
    registerComponentRange('cmp-cache', {
      container,
      startIndex: 0,
      endIndex: container.childNodes.length - 1,
    });

    vi.spyOn(manifest, 'applyComponentRanges').mockReturnValue(new Map());
    vi.spyOn(manifest, 'resolveListContainers').mockReturnValue(new Map());
    vi.spyOn(manifest, 'resolveSlotAnchors').mockImplementation(() => {
      const node = container.querySelector('span')?.firstChild ?? null;
      const map = new Map<number, Node>();
      if (node) {
        map.set(11, node);
      }
      return map;
    });

    const first: TemplateMessage = {
      t: 'template',
      sid: 'root-session',
      ver: 1,
      html: '<span>cached-value</span>',
      templateHash: 'hash:cmp:cached',
      s: [],
      d: [],
      slots: [{ anchorId: 11 }],
      slotPaths: [],
      listPaths: [],
      componentPaths: [],
      scope: { componentId: 'cmp-cache' },
    };

    (live as any).handleTemplate(first);
    expect(container.textContent).toContain('cached-value');

    container.textContent = 'mutated';

    const second: TemplateMessage = {
      t: 'template',
      sid: 'root-session',
      ver: 2,
      templateHash: 'hash:cmp:cached',
      s: [],
      d: [],
      slots: [{ anchorId: 11 }],
      slotPaths: [],
      listPaths: [],
      componentPaths: [],
      scope: { componentId: 'cmp-cache' },
    };

    (live as any).handleTemplate(second);

    expect(container.textContent).toContain('cached-value');
  });

  it('applies frame binding deltas', () => {
    const live = new LiveUI({ autoConnect: false });
    (live as any).sessionId.set('sid-1');

    const registerBindingsSpy = vi.spyOn(events, 'registerBindingsForSlot');

    const frame: FrameMessage = {
      t: 'frame',
      sid: 'sid-1',
      seq: 1,
      ver: 1,
      delta: { statics: false, slots: null },
      patch: [],
      effects: [],
      handlers: {},
      refs: {},
      bindings: { slots: { 9: [{ event: 'click', handler: 'abc' }] } },
      metrics: { renderMs: 0, ops: 0 },
    } as FrameMessage;

    (live as any).handleFrame(frame);

    expect(registerBindingsSpy).toHaveBeenCalledWith(9, expect.any(Array));
  });
});
