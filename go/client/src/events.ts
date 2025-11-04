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

/**
 * Register event handlers from the server
 * @param handlerMap - Map of handler IDs to handler metadata
 */
export function registerHandlers(handlerMap: HandlerMap): void {
  if (!handlerMap) return;
  for (const [id, meta] of Object.entries(handlerMap)) {
    const previous = handlers.get(id);
    if (previous?.event) {
      decrementEventUsage(previous.event);
    }

    handlers.set(id, meta);

    if (meta.event) {
      incrementEventUsage(meta.event);
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

    if (handler?.event) {
      decrementEventUsage(handler.event);
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
  if (!target) return;

  if (uploadDelegate) {
    try {
      uploadDelegate(e, eventType);
    } catch (error) {
      console.error("[LiveUI] upload delegate error", error);
    }
  }

  // Find the handler ID from the element or its parents FIRST
  // This ensures Link components with handlers take precedence over navigation interception
  const handlerId = findHandlerId(target as Element, eventType);

  // If there's a LiveUI handler, let it run (Link components use this)
  if (handlerId) {
    const handler = handlers.get(handlerId);
    if (handler && handler.event === eventType) {
      // Extract event payload based on event type
      const payload = extractEventPayload(e, target as HTMLElement);

      // Prevent default for submit events
      if (eventType === "submit") {
        e.preventDefault();
      }

      // Send event to server - this will trigger router.InternalHandleNav for Link components
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
    const anchor = findAnchorElement(target as Element);
    if (anchor && shouldInterceptNavigation(e as MouseEvent, anchor)) {
      const href = anchor.getAttribute("href");
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
  const attrName = `data-on${eventType}`;

  while (current && current !== document.documentElement) {
    if (current.hasAttribute && current.hasAttribute(attrName)) {
      return current.getAttribute(attrName);
    }
    current = current.parentElement;
  }

  return null;
}

function extractEventPayload(e: Event, target: HTMLElement): EventPayload {
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

  return payload;
}

/**
 * Walk up the DOM tree to find an anchor element
 */
function findAnchorElement(element: Element): HTMLAnchorElement | null {
  let current: Element | null = element;

  while (current && current !== document.documentElement) {
    if (current instanceof HTMLAnchorElement && current.hasAttribute("href")) {
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
  anchor: HTMLAnchorElement,
): boolean {
  // Only intercept primary mouse button (left click)
  if (e.button !== 0) {
    return false;
  }

  // Don't intercept if modifier keys are pressed (allow open in new tab/window)
  if (e.ctrlKey || e.metaKey || e.altKey || e.shiftKey) {
    return false;
  }

  const href = anchor.getAttribute("href");
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
