import { getSlotBindings, getSlotsForEvent, getRegisteredSlotEvents, observeSlotEvents } from './events';
import { getRouterMeta } from './router-bindings';
import { DomRegistry } from './dom-registry';
import type { LiveRuntime } from './runtime';
import { extractEventDetail } from './event-detail';
import { Logger } from './logger';

export class EventDelegation {
  private handlers = new Map<string, EventListener>();
  private stopSlotObserver?: () => void;

  constructor(private dom: DomRegistry, private runtime: LiveRuntime) {}

  setup(): void {
    if (typeof document === 'undefined') {
      return;
    }
    this.registerEvents(['click']);
    this.registerEvents(getRegisteredSlotEvents());
    this.stopSlotObserver = observeSlotEvents((event) => this.bind(event));
    Logger.debug('[Delegation]', 'event delegation setup complete', {
      handlers: this.handlers.size,
    });
  }

  teardown(): void {
    if (typeof document === 'undefined') {
      return;
    }
    if (this.stopSlotObserver) {
      this.stopSlotObserver();
      this.stopSlotObserver = undefined;
    }
    this.handlers.forEach((listener, event) => {
      document.removeEventListener(event, listener, true);
    });
    this.handlers.clear();
    Logger.debug('[Delegation]', 'event delegation torn down');
  }

  registerEvents(events: string[]): void {
    events.forEach((event) => this.bind(event));
  }

  private bind(event: string): void {
    if (this.handlers.has(event) || typeof document === 'undefined') {
      return;
    }
    const listener = (e: Event) => this.handleEvent(event, e);
    document.addEventListener(event, listener, true);
    this.handlers.set(event, listener);
    Logger.debug('[Delegation]', 'bound event listener', { event });
  }

  private handleEvent(event: string, e: Event): void {
    const target = e.target;
    if (!(target instanceof Element)) {
      return;
    }

    const router = getRouterMeta(target);
    if (router && router.path && event === 'click') {
      Logger.debug('[Delegation]', 'router navigation triggered', {
        path: router.path,
        hash: router.hash,
      });
      this.runtime.sendNavigation(router.path, router.query ?? '', router.hash ?? '');
      e.preventDefault();
      return;
    }

    const slotIds = getSlotsForEvent(event);
    for (const slotId of slotIds) {
      const specs = getSlotBindings(slotId) ?? [];
      const node = this.dom.getSlot(slotId);
      if (node instanceof Element && (node === target || node.contains(target))) {
        const binding = specs.find((spec) => spec.event === event);
        if (binding) {
          const detail = extractEventDetail(e, binding.props);
          this.runtime.sendEvent(binding.handler, detail ? { name: event, detail } : { name: event });
          Logger.debug('[Delegation]', 'slot event dispatched', {
            slotId,
            handler: binding.handler,
            event,
          });
          break;
        }
      }
    }
  }
}
