import { describe, it, expect, vi, beforeEach } from 'vitest';
import { DomRegistry } from '../src/dom-registry';
import { EventDelegation } from '../src/event-delegation';
import { LiveRuntime } from '../src/runtime';
import { registerSlotTable } from '../src/events';
import { applyRouterBindings } from '../src/router-bindings';
import type { ComponentRange } from '../src/manifest';

describe('EventDelegation', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
  });

  it('sends runtime events for bound slots', () => {
    const slot = document.createElement('button');
    document.body.appendChild(slot);
    const dom = new DomRegistry();
    dom['slots'].set(1, slot);
    registerSlotTable({ 1: [{ event: 'click', handler: 'hid-1' }] });
    const runtime = new LiveRuntime({ autoConnect: false, reconnect: false });
    const sendSpy = vi.spyOn(runtime, 'sendEvent');
    const delegation = new EventDelegation(dom, runtime);
    delegation.setup();
    slot.click();
    expect(sendSpy).toHaveBeenCalledWith('hid-1', { name: 'click' });
    delegation.teardown();
  });
  it('navigates when router metadata is present', () => {
    const link = document.createElement('a');
    document.body.appendChild(link);
    const dom = new DomRegistry();
    dom['slots'].set(2, link);
    registerSlotTable(undefined);
    const runtime = new LiveRuntime({ autoConnect: false, reconnect: false });
    const navSpy = vi.spyOn(runtime, 'sendNavigation');
    const delegation = new EventDelegation(dom, runtime);
    delegation.setup();
    const range: ComponentRange = {
      container: document.body,
      startIndex: Array.from(document.body.childNodes).indexOf(link),
      endIndex: Array.from(document.body.childNodes).indexOf(link),
    };
    applyRouterBindings(
      [{ componentId: 'root', pathValue: '/router', path: [] }],
      new Map([['root', range]]),
    );
    link.click();
    expect(navSpy).toHaveBeenCalledWith('/router', '', '');
    delegation.teardown();
  });
});
