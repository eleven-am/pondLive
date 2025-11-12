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
const elementRefMap = new WeakMap<Element, Set<string>>();

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

  const id = getPrimaryRefId(refElement);
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
  if (record.element && record.element !== element) {
    removeRefFromElement(record.element, id);
  }
  record.element = element;
  if (!record.meta.tag) {
    record.meta = { ...record.meta, tag: element.tagName.toLowerCase() };
  }
  addRefToElement(element, id);
}

function detachRef(id: string, element: Element | null): void {
  const record = registry.get(id);
  if (!record) {
    return;
  }
  if (!element || record.element === element) {
    if (record.element) {
      removeRefFromElement(record.element, id);
    }
    record.element = null;
  }
}

function addRefToElement(element: Element, id: string): void {
  let ids = elementRefMap.get(element);
  if (!ids) {
    ids = new Set<string>();
    elementRefMap.set(element, ids);
  }
  ids.add(id);
}

function removeRefFromElement(element: Element, id: string): void {
  const ids = elementRefMap.get(element);
  if (!ids) {
    return;
  }
  ids.delete(id);
  if (ids.size === 0) {
    elementRefMap.delete(element);
  }
}

function getPrimaryRefId(element: Element | null): string | null {
  if (!element) {
    return null;
  }
  const ids = elementRefMap.get(element);
  if (ids && ids.size > 0) {
    const first = ids.values().next();
    if (!first.done) {
      return first.value ?? null;
    }
  }
  const attr = element.getAttribute?.("data-live-ref") ?? null;
  return attr && attr.length > 0 ? attr : null;
}

export function clearRefs(): void {
  registry.clear();
  elementRefMap.clear();
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

export function unbindRefsInTree(
  root: ParentNode | Element | DocumentFragment | Document | null | undefined,
): void {
  if (!root || typeof document === "undefined") {
    return;
  }
  const walker = document.createTreeWalker(root as Node, NodeFilter.SHOW_ELEMENT);
  let current: Node | null = root as Node;
  while (current) {
    if (current instanceof Element) {
      const ids = elementRefMap.get(current);
      if (ids && ids.size > 0) {
        for (const id of Array.from(ids)) {
          detachRef(id, current);
        }
      }
      const attrId = current.getAttribute?.("data-live-ref");
      if (attrId) {
        detachRef(attrId, current);
      }
    }
    current = walker.nextNode();
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

export function attachRefToElement(
  refId: string,
  element: Element | null,
): void {
  if (!refId) {
    return;
  }
  if (!element) {
    detachRef(refId, null);
    return;
  }
  attachRef(refId, element);
}

function findClosestRefElement(element: Element | null): Element | null {
  let current: Element | null = element;
  while (current) {
    if (getPrimaryRefId(current)) {
      return current;
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
