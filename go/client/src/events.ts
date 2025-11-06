/**
 * Events
 *
 * Handles event delegation and forwards events to the server via the
 * WebSocket connection.
 */

import type { EventPayload, HandlerMap, HandlerMeta } from "./types";

const handlers = new Map<string, HandlerMeta>();
const eventUsageCounts = new Map<string, number>();
const installedListeners = new Map<string, (e: Event) => void>();

const ALWAYS_ACTIVE_EVENTS = ["click", "input", "change", "submit"];

function isLiveAnchor(
  element: Element,
): element is HTMLAnchorElement | SVGAElement {
  const hasHref =
    element.hasAttribute("href") || element.hasAttribute("xlink:href");
  if (!hasHref) {
    return false;
  }

  const isHtmlAnchor = element instanceof HTMLAnchorElement;
  const isSvgAnchor =
    element.namespaceURI === "http://www.w3.org/2000/svg" &&
    element.tagName.toLowerCase() === "a";

  return isHtmlAnchor || isSvgAnchor;
}

/**
 * Register event handlers from the server
 * @param handlerMap - Map of handler IDs to handler metadata
 */
export function registerHandlers(handlerMap: HandlerMap): void {
  if (!handlerMap) return;
  for (const [id, meta] of Object.entries(handlerMap)) {
    const previous = handlers.get(id);
    if (previous) {
      for (const eventName of collectEventTypes(previous)) {
        decrementEventUsage(eventName);
      }
    }

    handlers.set(id, meta);

    for (const eventName of collectEventTypes(meta)) {
      incrementEventUsage(eventName);
    }
  }
}

/**
 * Unregister event handlers
 * @param handlerIds - Array of handler IDs to remove
 */
export function unregisterHandlers(handlerIds: string[]): void {
  if (!Array.isArray(handlerIds)) return;
  for (const id of handlerIds) {
    const handler = handlers.get(id);
    handlers.delete(id);

    if (handler) {
      for (const eventName of collectEventTypes(handler)) {
        decrementEventUsage(eventName);
      }
    }
  }
}

/**
 * Clear all registered handlers
 */
export function clearHandlers(): void {
  handlers.clear();
  eventUsageCounts.clear();
  syncEventListeners();
}

/**
 * Event sender callback type
 */
export type EventSender = (event: {
  hid: string;
  payload: EventPayload;
}) => void;

/**
 * Navigation handler callback type
 */
export type NavigationHandler = (
  path: string,
  query: string,
  hash: string,
) => boolean;

let sendEventCallback: EventSender | null = null;
let navigationHandler: NavigationHandler | null = null;
let uploadDelegate: ((event: Event, eventType: string) => void) | null = null;

/**
 * Set up event delegation for the document
 * Only installs listeners for event types that are actually used
 * @param sendEvent - Callback to send events to the server
 */
export function setupEventDelegation(sendEvent: EventSender): void {
  sendEventCallback = sendEvent;

  // Start with common event types
  ALWAYS_ACTIVE_EVENTS.forEach((eventType) => {
    installListener(eventType);
  });
}

/**
 * Remove all installed event listeners
 */
export function teardownEventDelegation(): void {
  installedListeners.forEach((listener, eventType) => {
    const useCapture = eventType === "blur" || eventType === "focus";
    document.removeEventListener(eventType, listener, useCapture);
  });

  installedListeners.clear();
  sendEventCallback = null;
  uploadDelegate = null;
}

export function registerUploadDelegate(
  delegate: ((event: Event, eventType: string) => void) | null,
): void {
  uploadDelegate = delegate;
}

/**
 * Register a navigation handler for intercepting link clicks
 * @param handler - Callback to handle navigation
 */
export function registerNavigationHandler(handler: NavigationHandler): void {
  navigationHandler = handler;

  // Ensure click listener is installed for navigation interception to work
  // Even if setupEventDelegation hasn't been called yet
  if (!installedListeners.has("click")) {
    installListener("click");
  }
}

/**
 * Unregister the navigation handler
 */
export function unregisterNavigationHandler(): void {
  navigationHandler = null;

  // Note: We keep the click listener installed since it's used for other handlers too
  // syncEventListeners() will clean it up if no click handlers remain
}

/**
 * Install a listener for a specific event type (if not already installed)
 */
function installListener(eventType: string): void {
  if (installedListeners.has(eventType)) return;

  const useCapture = eventType === "blur" || eventType === "focus";
  const listener = (e: Event) => {
    if (sendEventCallback) {
      handleEvent(e, eventType, sendEventCallback);
    }
  };

  document.addEventListener(eventType, listener, useCapture);
  installedListeners.set(eventType, listener);
}

/**
 * Remove a listener for a specific event type
 */
function removeListener(eventType: string): void {
  const listener = installedListeners.get(eventType);
  if (!listener) return;

  const useCapture = eventType === "blur" || eventType === "focus";
  document.removeEventListener(eventType, listener, useCapture);
  installedListeners.delete(eventType);
}

/**
 * Sync event listeners based on active handler types
 * Call this after registering/unregistering handlers
 */
export function syncEventListeners(): void {
  // Install listeners for new event types
  eventUsageCounts.forEach((_, eventType) => {
    installListener(eventType);
  });

  // Remove listeners for event types that are no longer used
  installedListeners.forEach((_, eventType) => {
    if (
      !eventUsageCounts.has(eventType) &&
      !ALWAYS_ACTIVE_EVENTS.includes(eventType)
    ) {
      removeListener(eventType);
    }
  });
}

function handleEvent(
  e: Event,
  eventType: string,
  sendEvent: EventSender,
): void {
  const target = e.target;
  if (!target || !(target instanceof Element)) return;

  if (uploadDelegate) {
    try {
      uploadDelegate(e, eventType);
    } catch (error) {
      console.error("[LiveUI] upload delegate error", error);
    }
  }

  // Find the handler ID from the element or its parents FIRST
  // This ensures Link components with handlers take precedence over navigation interception
  const handlerId = findHandlerId(target, eventType);

  // If there's a LiveUI handler, let it run (Link components use this)
  if (handlerId) {
    const handler = handlers.get(handlerId);
    if (handler && handlerSupportsEvent(handler, eventType)) {
      // Extract event payload based on event type
      const payload = extractEventPayload(e, target, handler.props);

      // Prevent default for submit events
      if (eventType === "submit") {
        e.preventDefault();
      }

      // Prevent default for click events on anchors with LiveUI handlers
      if (eventType === "click") {
        const anchor = findAnchorElement(target);
        if (anchor && shouldInterceptNavigation(e as MouseEvent, anchor)) {
          e.preventDefault();
        }
      }

      // Send event to server - this will trigger runtime.InternalHandleNav for Link components
      sendEvent({
        hid: handlerId,
        payload: payload,
      });
      return;
    }
  }

  // No LiveUI handler found, try navigation interception for click events
  // This handles plain <a> tags without LiveUI handlers
  if (eventType === "click" && navigationHandler && e instanceof MouseEvent) {
    const anchor = findAnchorElement(target);
    if (anchor && shouldInterceptNavigation(e as MouseEvent, anchor)) {
      const href =
        anchor.getAttribute("href") ?? anchor.getAttribute("xlink:href");
      if (href) {
        try {
          const url = new URL(href, window.location.href);
          const path = url.pathname;
          const query = url.search.substring(1); // Remove leading '?'
          const hash = url.hash.startsWith("#")
            ? url.hash.substring(1)
            : url.hash;

          // Call navigation handler and prevent default if it returns true
          if (navigationHandler(path, query, hash)) {
            e.preventDefault();
            e.stopPropagation();
            return;
          }
        } catch (err) {
          // Invalid URL, let browser handle it
        }
      }
    }
  }
}

function findHandlerId(element: Element, eventType: string): string | null {
  let current: Element | null = element;

  while (current && current !== document.documentElement) {
    const directAttr = `data-on${eventType}`;
    if (typeof current.hasAttribute === "function" && current.hasAttribute(directAttr)) {
      const handlerId = current.getAttribute(directAttr);
      if (handlerId) {
        const meta = handlers.get(handlerId);
        if (!meta || handlerSupportsEvent(meta, eventType)) {
          return handlerId;
        }
      }
    }

    const attributeNames =
      typeof current.getAttributeNames === "function"
        ? current.getAttributeNames()
        : null;
    if (Array.isArray(attributeNames)) {
      for (const name of attributeNames) {
        if (!name.startsWith("data-on")) {
          continue;
        }
        const handlerId = current.getAttribute(name);
        if (!handlerId) {
          continue;
        }
        const meta = handlers.get(handlerId);
        if (meta && handlerSupportsEvent(meta, eventType)) {
          return handlerId;
        }
      }
    }

    current = current.parentElement;
  }

  return null;
}

function extractEventPayload(
  e: Event,
  target: Element,
  props?: string[] | null,
): EventPayload {
  const payload: EventPayload = {
    type: e.type,
  };

  // Add value for input elements
  if (
    target instanceof HTMLInputElement ||
    target instanceof HTMLTextAreaElement ||
    target instanceof HTMLSelectElement
  ) {
    payload.value = target.value;
    if (target instanceof HTMLInputElement) {
      if (target.type === "checkbox" || target.type === "radio") {
        payload.checked = target.checked;
      }
    }
  }

  // Add key information for keyboard events
  if (e instanceof KeyboardEvent) {
    payload.key = e.key;
    payload.keyCode = e.keyCode;
    payload.altKey = e.altKey;
    payload.ctrlKey = e.ctrlKey;
    payload.metaKey = e.metaKey;
    payload.shiftKey = e.shiftKey;
  }

  // Add mouse information for mouse events
  if (e instanceof MouseEvent) {
    payload.clientX = e.clientX;
    payload.clientY = e.clientY;
  }

  if (Array.isArray(props)) {
    for (const selector of props) {
      const value = resolvePropertySelector(selector, e, target);
      if (value === undefined) {
        continue;
      }
      const normalized = normalizePropertyValue(value);
      if (normalized !== undefined) {
        payload[selector] = normalized;
      }
    }
  }

  return payload;
}

/**
 * Walk up the DOM tree to find an anchor element
 */
function findAnchorElement(element: Element): HTMLAnchorElement | SVGAElement | null {
  let current: Element | null = element;

  while (current && current !== document.documentElement) {
    if (isLiveAnchor(current)) {
      return current;
    }
    current = current.parentElement;
  }

  return null;
}

/**
 * Determine if we should intercept this navigation
 * Returns true if all conditions are met for LiveUI to handle the navigation
 */
function shouldInterceptNavigation(
  e: MouseEvent,
  anchor: HTMLAnchorElement | SVGAElement,
): boolean {
  // Only intercept primary mouse button (left click)
  if (e.button !== 0) {
    return false;
  }

  // Don't intercept if modifier keys are pressed (allow open in new tab/window)
  if (e.ctrlKey || e.metaKey || e.altKey || e.shiftKey) {
    return false;
  }

  if (!isLiveAnchor(anchor)) {
    return false;
  }

  const href =
    anchor.getAttribute("href") ?? anchor.getAttribute("xlink:href") ?? undefined;
  if (!href) {
    return false;
  }

  // Don't intercept if target is specified (e.g., _blank, _top)
  const target = anchor.getAttribute("target");
  if (target && target !== "_self") {
    return false;
  }

  // Check if it's a same-origin URL
  try {
    const url = new URL(href, window.location.href);

    // Don't intercept different origin
    if (url.origin !== window.location.origin) {
      return false;
    }

    // Don't intercept hash-only links (same page navigation)
    if (
      url.pathname === window.location.pathname &&
      url.search === window.location.search &&
      url.hash
    ) {
      return false;
    }

    return true;
  } catch (err) {
    // Invalid URL, don't intercept
    return false;
  }
}

function incrementEventUsage(eventType: string): void {
  const current = eventUsageCounts.get(eventType) ?? 0;
  eventUsageCounts.set(eventType, current + 1);
}

function decrementEventUsage(eventType: string): void {
  const current = eventUsageCounts.get(eventType);
  if (!current) {
    return;
  }

  if (current <= 1) {
    eventUsageCounts.delete(eventType);
  } else {
    eventUsageCounts.set(eventType, current - 1);
  }
}

function collectEventTypes(meta?: HandlerMeta | null): string[] {
  if (!meta) {
    return [];
  }
  const seen = new Set<string>();
  const order: string[] = [];

  if (meta.event) {
    seen.add(meta.event);
    order.push(meta.event);
  }

  if (Array.isArray(meta.listen)) {
    for (const evt of meta.listen) {
      if (typeof evt !== "string" || evt.length === 0) {
        continue;
      }
      if (!seen.has(evt)) {
        seen.add(evt);
        order.push(evt);
      }
    }
  }

  return order;
}

function handlerSupportsEvent(meta: HandlerMeta, eventType: string): boolean {
  if (!meta) {
    return false;
  }
  if (meta.event === eventType) {
    return true;
  }
  if (Array.isArray(meta.listen)) {
    return meta.listen.includes(eventType);
  }
  return false;
}

function resolvePropertySelector(
  selector: string,
  event: Event,
  target: Element,
): any {
  if (typeof selector !== "string") {
    return undefined;
  }
  const trimmed = selector.trim();
  if (trimmed.length === 0) {
    return undefined;
  }

  const parts = trimmed.split(".");
  if (parts.length === 0) {
    return undefined;
  }

  const scope = parts.shift();
  let source: any;
  switch (scope) {
    case "event":
      source = event;
      break;
    case "target":
      source = target;
      break;
    case "currentTarget":
      source = event.currentTarget ?? undefined;
      break;
    default:
      return undefined;
  }

  let value = source;
  for (const part of parts) {
    if (!part) {
      continue;
    }
    if (value == null) {
      return undefined;
    }
    value = (value as any)[part];
  }

  return value;
}

function normalizePropertyValue(value: any): any {
  if (value === null) {
    return null;
  }
  const type = typeof value;
  if (type === "string" || type === "number" || type === "boolean") {
    return value;
  }
  if (value instanceof Date) {
    return value.toISOString();
  }
  if (typeof FileList !== "undefined" && value instanceof FileList) {
    return serializeFileList(value);
  }
  if (typeof File !== "undefined" && value instanceof File) {
    return { name: value.name, size: value.size, type: value.type };
  }
  if (typeof DOMTokenList !== "undefined" && value instanceof DOMTokenList) {
    return Array.from(value);
  }
  if (typeof TimeRanges !== "undefined" && value instanceof TimeRanges) {
    const ranges: Array<{ start: number; end: number }> = [];
    for (let i = 0; i < value.length; i++) {
      try {
        ranges.push({ start: value.start(i), end: value.end(i) });
      } catch (err) {
        // Ignore invalid ranges
      }
    }
    return ranges;
  }
  if (Array.isArray(value)) {
    return value
      .map((item) => normalizePropertyValue(item))
      .filter((item) => item !== undefined);
  }
  if (value && typeof value === "object") {
    try {
      return JSON.parse(JSON.stringify(value));
    } catch (err) {
      return undefined;
    }
  }
  return undefined;
}

function serializeFileList(list: FileList): Array<{
  name: string;
  size: number;
  type: string;
}> {
  const files: Array<{ name: string; size: number; type: string }> = [];
  for (let i = 0; i < list.length; i++) {
    const file = list.item(i);
    if (file) {
      files.push({ name: file.name, size: file.size, type: file.type });
    }
  }
  return files;
}
