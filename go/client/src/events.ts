import type { SlotBinding, BindingTable } from './types';
import { Logger } from './logger';

const slotTable = new Map<number, SlotBinding[]>();
const eventSlots = new Map<string, Set<number>>();
const eventObservers = new Set<(event: string) => void>();

export function registerSlotTable(table?: BindingTable | null): void {
  slotTable.clear();
  eventSlots.clear();
  if (!table) {
    Logger.debug('[Events]', 'cleared slot table');
    return;
  }
  for (const [key, value] of Object.entries(table)) {
    const slotId = Number(key);
    if (Number.isNaN(slotId)) {
      continue;
    }
    const bindings = cloneBindings(value);
    slotTable.set(slotId, bindings);
    addEventIndex(slotId, bindings);
  }
  Logger.debug('[Events]', 'registered slot table', { slots: slotTable.size });
}

export function registerBindingsForSlot(slotId: number, specs: SlotBinding[] | null | undefined): void {
  if (!Number.isFinite(slotId)) {
    return;
  }
  slotTable.delete(slotId);
  eventSlots.forEach((set) => set.delete(slotId));
  if (!Array.isArray(specs)) {
    return;
  }
  const bindings = cloneBindings(specs);
  slotTable.set(slotId, bindings);
  addEventIndex(slotId, bindings);
  Logger.debug('[Events]', 'registered slot bindings', { slotId, count: bindings.length });
}

export function getSlotBindings(slotId: number): SlotBinding[] | undefined {
  const bindings = slotTable.get(slotId);
  return bindings ? cloneBindings(bindings) : undefined;
}

export function forEachSlotBinding(callback: (slotId: number, bindings: SlotBinding[]) => void): void {
  slotTable.forEach((bindings, slotId) => {
    callback(slotId, cloneBindings(bindings));
  });
}

export function getSlotsForEvent(event: string): number[] {
  const set = eventSlots.get(event);
  return set ? Array.from(set) : [];
}

export function getRegisteredSlotEvents(): string[] {
  return Array.from(eventSlots.keys());
}

export function observeSlotEvents(observer: (event: string) => void): () => void {
  eventObservers.add(observer);
  return () => eventObservers.delete(observer);
}

function addEventIndex(slotId: number, bindings: SlotBinding[]): void {
  bindings.forEach((binding) => {
    const event = binding.event;
    if (!event) {
      return;
    }
    const set = eventSlots.get(event) ?? new Set<number>();
    const hadEvent = set.size > 0;
    set.add(slotId);
    eventSlots.set(event, set);
    if (!hadEvent && set.size === 1) {
      notifyObservers([event]);
    }
    Logger.debug('[Events]', 'indexed event', { event, slotId, totalSlots: set.size });
  });
}

function notifyObservers(events: string[]): void {
  if (!Array.isArray(events)) {
    return;
  }
  events.forEach((event) => {
    if (!event) {
      return;
    }
    eventObservers.forEach((observer) => {
      try {
        observer(event);
      } catch {
        /* ignore */
      }
    });
  });
}

function cloneBindings(specs: SlotBinding[] | undefined): SlotBinding[] {
  if (!Array.isArray(specs)) {
    return [];
  }
  return specs.map((spec) => ({
    event: spec?.event ?? '',
    handler: spec?.handler ?? '',
    listen: Array.isArray(spec?.listen) ? [...spec.listen] : undefined,
    props: Array.isArray(spec?.props) ? [...spec.props] : undefined,
  }));
}
