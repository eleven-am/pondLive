/**
 * Events
 *
 * Handles event delegation and forwards events to the server via the
 * WebSocket connection.
 */

import type {
  BindingTable,
  EventPayload,
  HandlerMap,
  HandlerMeta,
  SlotBinding,
} from "./types";
import { domGetSync } from "./dom";
import { resolveRefEventContext } from "./refs";

const handlers = new Map<string, HandlerMeta>();
const eventUsageCounts = new Map<string, number>();
const installedListeners = new Map<string, (e: Event) => void>();
const handlerBindings = new WeakMap<Element, Map<string, string>>();
const slotBindingSpecs = new Map<number, SlotBinding[]>();
const slotElements = new Map<number, Element>();
const routerBindings = new WeakMap<Element, RouterMeta>();

export const DATA_ROUTER_ATTR_PREFIX = "data-router-";

const ALWAYS_ACTIVE_EVENTS = ["click", "input", "change", "submit"];
const CAPTURE_EVENTS = new Set([
  "blur",
  "focus",
  "mouseenter",
  "mouseleave",
  "pointerenter",
  "pointerleave",
]);

interface RouterMeta {
  path?: string;
  query?: string;
  hash?: string;
  replace?: string;
}

function setRouterMetaValue(
  element: Element,
  field: keyof RouterMeta,
  value: string | null | undefined,
): void {
  let meta = routerBindings.get(element);
  if (value === null || value === undefined || value === "") {
    if (!meta) {
      return;
    }
    delete meta[field];
    if (!meta.path && !meta.query && !meta.hash && !meta.replace) {
      routerBindings.delete(element);
    }
    return;
  }
  if (!meta) {
    meta = {};
    routerBindings.set(element, meta);
  }
  meta[field] = value;
}

export function applyRouterAttribute(
  element: Element | null | undefined,
  key: string,
  value: string | null | undefined,
): void {
  if (!element || typeof key !== "string" || key.length === 0) {
    return;
  }
  switch (key) {
    case "path":
    case "query":
    case "hash":
    case "replace":
      setRouterMetaValue(element, key, value);
      break;
    default:
      break;
  }
}

export function clearRouterAttributes(
  element: Element | null | undefined,
): void {
  if (!element) {
    return;
  }
  routerBindings.delete(element);
}

function cloneSlotBindingList(specs: SlotBinding[] | null | undefined): SlotBinding[] {
  if (!Array.isArray(specs)) {
    return [];
  }
  return specs.map((spec) => {
    const clone: SlotBinding = {
      event: spec?.event || "",
      handler: spec?.handler || "",
    };
    if (Array.isArray(spec?.listen) && spec.listen.length > 0) {
      clone.listen = [...spec.listen];
    }
    if (Array.isArray(spec?.props) && spec.props.length > 0) {
      clone.props = [...spec.props];
    }
    return clone;
  });
}

function applySlotBindings(slotId: number): void {
  const element = slotElements.get(slotId);
  if (!element) {
    return;
  }

  const specs = slotBindingSpecs.get(slotId) ?? [];
  if (!Array.isArray(specs) || specs.length === 0) {
    handlerBindings.delete(element);
    return;
  }

  const map = new Map<string, string>();
  for (const spec of specs) {
    if (!spec || typeof spec.event !== "string" || typeof spec.handler !== "string") {
      continue;
    }
    const eventName = spec.event.trim();
    const handlerId = spec.handler.trim();
    if (eventName.length === 0 || handlerId.length === 0) {
      continue;
    }
    map.set(eventName, handlerId);
  }

  if (map.size === 0) {
    handlerBindings.delete(element);
    return;
  }

  handlerBindings.set(element, map);
}

export function primeSlotBindings(table: BindingTable | null | undefined): void {
  slotBindingSpecs.clear();
  if (table && typeof table === "object") {
    for (const [key, value] of Object.entries(table)) {
      const slotId = Number(key);
      if (Number.isNaN(slotId)) {
        continue;
      }
      slotBindingSpecs.set(slotId, cloneSlotBindingList(value));
    }
  }
  slotElements.forEach((_element, slotId) => {
    applySlotBindings(slotId);
  });
}

export function registerBindingsForSlot(
  slotId: number,
  specs: SlotBinding[] | null | undefined,
): void {
  if (!Number.isFinite(slotId)) {
    return;
  }
  if (!Array.isArray(specs)) {
    slotBindingSpecs.set(slotId, []);
  } else {
    slotBindingSpecs.set(slotId, cloneSlotBindingList(specs));
  }
  applySlotBindings(slotId);
}

export function getRegisteredSlotBindings(
  slotId: number,
): ReadonlyArray<SlotBinding> | undefined {
  const specs = slotBindingSpecs.get(slotId);
  if (!specs) {
    return undefined;
  }
  return cloneSlotBindingList(specs);
}

export function onSlotRegistered(slotId: number, node: Node | null | undefined): void {
  if (!Number.isFinite(slotId) || !node) {
    return;
  }
  if (node instanceof Element) {
    slotElements.set(slotId, node);
    applySlotBindings(slotId);
    return;
  }
  slotElements.delete(slotId);
}

export function onSlotUnregistered(slotId: number): void {
  const element = slotElements.get(slotId);
  if (element) {
    handlerBindings.delete(element);
    clearRouterAttributes(element);
  }
  slotElements.delete(slotId);
}

function mergeSelectorLists(
  primary?: string[] | null,
  secondary?: string[] | null,
): string[] | undefined {
  const merged: string[] = [];
  const seen = new Set<string>();

  const add = (list?: string[] | null) => {
    if (!Array.isArray(list)) {
      return;
    }
    for (const value of list) {
      if (typeof value !== "string" || value.length === 0) {
        continue;
      }
      if (seen.has(value)) {
        continue;
      }
      seen.add(value);
      merged.push(value);
    }
  };

  add(primary);
  add(secondary);

  return merged.length > 0 ? merged : undefined;
}

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

/** @internal - exposed for tests to introspect cached handler bindings */
export function getHandlerBindingSnapshot(
  element: Element | null | undefined,
): ReadonlyMap<string, string> | undefined {
  if (!element) {
    return undefined;
  }
  const bindings = handlerBindings.get(element);
  return bindings ? new Map(bindings) : undefined;
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
function shouldUseCapture(eventType: string): boolean {
  return CAPTURE_EVENTS.has(eventType);
}

function installListener(eventType: string): void {
  if (installedListeners.has(eventType)) return;

  const useCapture = shouldUseCapture(eventType);
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

  const useCapture = shouldUseCapture(eventType);
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

  // Find the handler ID and element from the element or its parents FIRST
  // This ensures Link components with handlers take precedence over navigation interception
  const handlerInfo = findHandlerInfo(target, eventType);

  // If there's a LiveUI handler, let it run (Link components use this)
  if (handlerInfo) {
    const handler = handlers.get(handlerInfo.id);
    if (handler && handlerSupportsEvent(handler, eventType)) {
      // Extract event payload based on event type
      // Pass the handler element so that "currentTarget" in selectors refers to the element with the handler
      const refContext = resolveRefEventContext(handlerInfo.element, eventType);
      const combinedProps = mergeSelectorLists(handler.props, refContext?.props);
      const payload = extractEventPayload(
        e,
        target,
        combinedProps,
        handlerInfo.element,
        refContext?.element ?? null,
      );

      const routerMeta = routerBindings.get(handlerInfo.element);
      if (routerMeta) {
        if (
          routerMeta.path !== undefined &&
          payload["currentTarget.dataset.routerPath"] === undefined
        ) {
          payload["currentTarget.dataset.routerPath"] = routerMeta.path;
        }
        if (
          routerMeta.query !== undefined &&
          payload["currentTarget.dataset.routerQuery"] === undefined
        ) {
          payload["currentTarget.dataset.routerQuery"] = routerMeta.query;
        }
        if (
          routerMeta.hash !== undefined &&
          payload["currentTarget.dataset.routerHash"] === undefined
        ) {
          payload["currentTarget.dataset.routerHash"] = routerMeta.hash;
        }
        if (
          routerMeta.replace !== undefined &&
          payload["currentTarget.dataset.routerReplace"] === undefined
        ) {
          payload["currentTarget.dataset.routerReplace"] = routerMeta.replace;
        }
      }

      if (refContext) {
        refContext.notify(payload);
      }

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
        hid: handlerInfo.id,
        payload: payload,
      });
      return;
    }
  }

  // No h.On handler found, but check for ref handlers
  const refContext = resolveRefEventContext(target, eventType);
  if (refContext) {
    // Extract payload for ref-only events
    const payload = extractEventPayload(
      e,
      target,
      refContext.props,
      refContext.element,
      refContext.element,
    );

    refContext.notify(payload);

    // Send ref-only event to server
    sendEvent({
      hid: refContext.id, // Use ref ID as handler ID for ref-only events
      payload: payload,
    });
    return;
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

function findHandlerInfo(
  element: Element,
  eventType: string,
): { id: string; element: Element } | null {
  let current: Element | null = element;

  while (current && current !== document.documentElement) {
    const binding = handlerBindings.get(current);

    if (binding && binding.size > 0) {
      const directId = binding.get(eventType);
      if (directId) {
        const meta = handlers.get(directId);
        if (!meta || handlerSupportsEvent(meta, eventType)) {
          return { id: directId, element: current };
        }
      }

      for (const [registeredEvent, handlerId] of binding.entries()) {
        if (registeredEvent === eventType) {
          continue;
        }
        const meta = handlers.get(handlerId);
        if (meta && handlerSupportsEvent(meta, eventType)) {
          return { id: handlerId, element: current };
        }
      }
    }

    current = current.parentElement;
  }

  return null;
}

/**
 * Auto-capture all serializable properties from an event object.
 * Skips functions, DOM nodes, and circular references.
 */
function autoCaptureEventProperties(e: Event): Record<string, any> {
  const captured: Record<string, any> = {};

  for (const key in e) {
    try {
      const value = (e as any)[key];

      // Skip non-serializable types
      if (
        typeof value === "function" ||
        value instanceof Node ||
        value instanceof Window ||
        key === "target" || // Too large, causes circular refs
        key === "currentTarget" ||
        key === "srcElement"
      ) {
        continue;
      }

      // Serialize primitives and plain objects
      if (value !== null && typeof value === "object") {
        // Try to serialize, skip if circular/complex
        try {
          JSON.stringify(value);
          captured[key] = value;
        } catch {
          continue; // Skip circular refs
        }
      } else {
        captured[key] = value;
      }
    } catch {
      continue; // Skip if property access throws
    }
  }

  return captured;
}

function extractEventPayload(
  e: Event,
  target: Element,
  props?: string[] | null,
  handlerElement?: Element,
  refElement?: Element | null,
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
    if (props.includes("*")) {
      // Auto-capture all event properties
      const captured = autoCaptureEventProperties(e);
      Object.assign(payload, captured);
    } else {
      // Normal mode: use specified props
      const domValues = domGetSync(props, {
        event: e,
        target,
        handlerElement: handlerElement ?? null,
        refElement: refElement ?? null,
      });
      if (domValues) {
        Object.assign(payload, domValues);
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
