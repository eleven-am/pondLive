import type { EventPayload, RefMap, RefMeta } from "./types";

export interface RefEventContext {
  id: string;
  element: Element;
  props: string[];
  notify(payload: EventPayload): void;
}

interface RefRecord {
  id: string;
  meta: RefMeta;
  element: Element | null;
  lastPayloads?: Record<string, EventPayload>;
}

const registry = new Map<string, RefRecord>();

function escapeAttributeValue(value: string): string {
  if (typeof value !== "string") {
    return "";
  }
  if (typeof CSS !== "undefined" && typeof CSS.escape === "function") {
    return CSS.escape(value);
  }
  return value.replace(/["]|\\/g, "\\$&");
}

function queryRefElement(id: string): Element | null {
  if (typeof document === "undefined" || !id) {
    return null;
  }
  const selector = `[data-live-ref="${escapeAttributeValue(id)}"]`;
  try {
    return document.querySelector(selector);
  } catch (_error) {
    return document.querySelector(`[data-live-ref="${id}"]`);
  }
}

function ensureRecord(id: string, meta?: RefMeta): RefRecord {
  let record = registry.get(id);
  if (!record) {
    record = {
      id,
      meta: meta ?? { tag: "" },
      element: queryRefElement(id),
    };
    registry.set(id, record);
  } else if (meta) {
    record.meta = meta;
    if (!record.element) {
      record.element = queryRefElement(id);
    }
  }
  return record;
}

function collectRefEventProps(meta: RefMeta | null | undefined, eventType: string): string[] {
  if (!meta || !meta.events) {
    return [];
  }

  const selectors: string[] = [];
  const seen = new Set<string>();

  for (const [primary, eventMeta] of Object.entries(meta.events)) {
    if (!eventMeta) {
      continue;
    }

    const listens = Array.isArray(eventMeta.listen) ? eventMeta.listen : [];
    if (primary !== eventType && !listens.includes(eventType)) {
      continue;
    }

    if (!Array.isArray(eventMeta.props)) {
      continue;
    }

    for (const prop of eventMeta.props) {
      if (typeof prop !== "string" || prop.length === 0) {
        continue;
      }
      if (seen.has(prop)) {
        continue;
      }
      seen.add(prop);
      selectors.push(prop);
    }
  }

  return selectors;
}

export function resolveRefEventContext(
  element: Element | null,
  eventType: string,
): RefEventContext | null {
  if (!element || !eventType) {
    return null;
  }

  const refElement = findClosestRefElement(element);
  if (!refElement) {
    return null;
  }

  const id = refElement.getAttribute("data-live-ref");
  if (!id) {
    return null;
  }

  const record = ensureRecord(id);
  record.element = refElement;

  if (!refSupportsEvent(record.meta, eventType)) {
    return null;
  }

  const props = collectRefEventProps(record.meta, eventType);

  return {
    id,
    element: refElement,
    props,
    notify(payload: EventPayload) {
      if (!record.lastPayloads) {
        record.lastPayloads = Object.create(null);
      }
      record.lastPayloads[eventType] = { ...payload };
    },
  };
}

function attachRef(id: string, element: Element): void {
  if (!id) {
    return;
  }
  const record = ensureRecord(id);
  record.element = element;
  if (!record.meta.tag) {
    record.meta = { ...record.meta, tag: element.tagName.toLowerCase() };
  }
}

function detachRef(id: string, element: Element | null): void {
  const record = registry.get(id);
  if (!record) {
    return;
  }
  if (!element || record.element === element) {
    record.element = null;
  }
}

function forEachRefElement(
  root: ParentNode | Element | DocumentFragment,
  visit: (el: Element) => void,
): void {
  if (root instanceof Element) {
    visit(root);
  }

  const selectorAll = (root as ParentNode).querySelectorAll?.bind(root as ParentNode);
  if (typeof selectorAll !== "function") {
    return;
  }

  const matches = selectorAll("[data-live-ref]");
  matches.forEach((node) => {
    if (node instanceof Element) {
      visit(node);
    }
  });
}

export function clearRefs(): void {
  registry.clear();
}

export function registerRefs(refs?: RefMap | null): void {
  if (!refs) {
    return;
  }
  for (const [id, meta] of Object.entries(refs)) {
    if (!id) {
      continue;
    }
    ensureRecord(id, meta);
  }
}

export function unregisterRefs(ids?: string[] | null): void {
  if (!Array.isArray(ids)) {
    return;
  }
  for (const id of ids) {
    if (!id) {
      continue;
    }
    registry.delete(id);
  }
}

export function bindRefsInTree(
  root: ParentNode | Element | DocumentFragment | Document | null | undefined,
): void {
  if (!root || typeof document === "undefined") {
    return;
  }
  forEachRefElement(root as ParentNode, (el) => {
    const id = el.getAttribute("data-live-ref");
    if (id) {
      attachRef(id, el);
    }
  });
}

export function unbindRefsInTree(
  root: ParentNode | Element | DocumentFragment | Document | null | undefined,
): void {
  if (!root || typeof document === "undefined") {
    return;
  }
  forEachRefElement(root as ParentNode, (el) => {
    const id = el.getAttribute("data-live-ref");
    if (id) {
      detachRef(id, el);
    }
  });
}

export function updateRefBinding(
  element: Element,
  previousId: string | null,
  nextId: string | null,
): void {
  if (previousId && (!nextId || previousId !== nextId)) {
    detachRef(previousId, element);
  }
  if (nextId) {
    attachRef(nextId, element);
  } else if (previousId && !nextId) {
    detachRef(previousId, element);
  }
}

export function getRefElement(id: string): Element | null {
  const record = registry.get(id);
  return record?.element ?? null;
}

export function getRefMeta(id: string): RefMeta | null {
  const record = registry.get(id);
  return record?.meta ?? null;
}

export function getRegistrySnapshot(): Map<string, RefRecord> {
  return new Map(registry);
}

function findClosestRefElement(element: Element | null): Element | null {
  let current: Element | null = element;
  while (current) {
    if (typeof current.getAttribute === "function") {
      const id = current.getAttribute("data-live-ref");
      if (id) {
        return current;
      }
    }
    current = current.parentElement;
  }
  return null;
}

function refSupportsEvent(meta: RefMeta | null | undefined, eventType: string): boolean {
  if (!meta || !meta.events) {
    return false;
  }
  for (const [primary, eventMeta] of Object.entries(meta.events)) {
    if (primary === eventType) {
      return true;
    }
    if (eventMeta?.listen && eventMeta.listen.includes(eventType)) {
      return true;
    }
  }
  return false;
}

export function notifyRefEvent(
  element: Element | null,
  eventType: string,
  payload: EventPayload,
): void {
  const context = resolveRefEventContext(element, eventType);
  if (context) {
    context.notify(payload);
  }
}
