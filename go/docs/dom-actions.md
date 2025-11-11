# DOM Actions Design

This document describes how PondLive will deliver imperative DOM commands (e.g.
`ref.Play()`, scrolling, property updates) from the server to the browser while
keeping HTML clean and avoiding ad-hoc effect types.

## 1. Goals

1. Allow components to invoke DOM APIs (methods, property sets, toggles) on refs
   after a render completes.
2. Keep the transport structured so the client can validate and log failures.
3. Preserve ordering relative to the diff/patch pipeline—actions should run
   after DOM mutations so refs point at the correct nodes.
4. Provide ergonomic Go helpers (`live.DOMCall`, `DOMSet`, …) without exposing
   transport details.

## 2. Current Mechanics

- `live.DOMCall(ctx, ref, method, args...)` enqueues a `domCall` effect with the
  ref ID, method, and arguments.
- Flush serializes these into the frame payload’s `effects` array.
- Client `applyEffects` handles `type == "domCall"` by resolving the ref to an
  element (`refs.getRefElement(refId)`) and invoking `element[method](...args)`.

Limitations: only method calls are supported; failure reporting is ad hoc; the
effect namespace grows if we add more DOM-side behaviors.

## 3. Proposed “DOM Actions” Schema

### 3.1 Effect Payload

Instead of a single `domCall` type, we introduce a generic `actions` block:

```jsonc
{
  "effects": [
    { "kind": "dom.call",    "ref": "ref:7", "method": "play", "args": [] },
    { "kind": "dom.set",     "ref": "ref:4", "prop": "currentTime", "value": 0 },
    { "kind": "dom.toggle",  "ref": "ref:5", "prop": "muted", "value": true },
    { "kind": "dom.class",   "ref": "ref:9", "class": "is-active", "on": true },
    { "kind": "dom.scroll",  "ref": "ref:2", "behavior": "smooth", "block": "center" }
  ]
}
```

- `kind` determines which handler runs on the client.
- Every action specifies the ref ID so the client can locate the element.
- Additional fields depend on `kind` (method name, property name, class token,
  etc.). We can expand the vocabulary over time.

### 3.2 Client Dispatcher & Implementation Plan

Implementation steps:

1. **Protocol struct** – Replace the legacy `domCall` effect with a descriptor:
   ```jsonc
   { "type": "dom", "kind": "dom.call", "ref": "ref:7", "method": "play" }
   ```
   Update `protocol.DOMEfffect` so `Kind` carries `dom.call`, `dom.set`, etc.
2. **Server helpers** – Add Go helpers (`live.DOMCall`, `live.DOMSet`,
   `live.DOMToggleClass`, `live.DOMScrollIntoView`, …) that enqueue these
   descriptors. Validate inputs (whitelist methods, type-check values) before
   enqueueing.
3. **Client dispatcher** – Update the diff effect handler to switch on `kind`:
   ```ts
   const domActionHandlers = {
     "dom.call":    invokeMethod,
     "dom.set":     setProperty,
     "dom.toggle":  setBooleanProperty,
     "dom.class":   toggleClass,
     "dom.scroll":  scrollIntoView,
   };
   for (const effect of effects ?? []) {
     if (effect.type !== "dom") continue;
     const handler = domActionHandlers[effect.kind];
     if (handler) handler(effect);
   }
   ```
   Each handler resolves the ref via `refs.getRefElement`, executes the action,
   and logs a diagnostic if it fails.
4. **Diagnostics** – On failure (missing ref, unsafe method), emit a dev-mode
   diagnostic (`code: "dom_action_failed"`) so developers see immediate
   feedback.
5. **Backward compatibility** – Treat legacy `domCall` payloads as `dom.call`
   during migration. Eventually remove the old schema once helpers switch over.

### 3.3 Ordering & Timing

- Actions remain inside the `effects` array processed **after** patch ops. That
  guarantees refs point at the final DOM nodes for this frame.
- Actions execute sequentially in the order they were enqueued server-side,
  preserving component intent.
- If an element is missing (component not yet hydrated), we can:
  - Log and drop the action (default).
  - Optionally buffer and retry next frame by keeping a small queue keyed by
    ref ID (future enhancement).

### 3.4 Server Helpers

Go-side helpers wrap action creation:

```go
live.DOMCall(ctx, ref, "play")
live.DOMSet(ctx, ref, "currentTime", 0)
live.DOMToggle(ctx, ref, "muted", true)
live.DOMToggleClass(ctx, ref, "is-active", true)
live.DOMScrollIntoView(ctx, ref, live.ScrollOptions{Behavior: "smooth"})
```

All helpers append a `domAction` struct to the component session, which the
runtime serializes as described above.

### 3.5 Validation & Diagnostics

- Since `kind` is explicit, the client can whitelist allowable methods or
  properties (e.g., restrict `dom.call` to known safe names).
- Arguments/values are serialized JSON; validate types on both ends.
- On failure, emit a diagnostic (`code: "dom_action_failed"`) so dev tools/overlay
  show the issue. Prod builds may log server-side instead of streaming.

## 4. Migration

1. **Introduce new schema alongside legacy `domCall`.** Client understands both;
   server emits both or toggles via feature flag.
2. **Update Go helpers** to emit the richer descriptors.
3. **Deprecate legacy `domCall`** once all clients understand the new format.

## 5. Future Extensions

- `dom.focus`, `dom.blur`, `dom.selectText`, `dom.dispatchEvent`.
- Batched actions targeted at the same ref (`actions: { ref, steps: [...] }`).
- Retry semantics for refs not yet mounted (with `retries` counter in payload).
- Diagnostics channel for better visibility in dev mode.

With DOM Actions, imperative behaviors like `ref.Play()` are first-class,
structured, and extensible, making it straightforward to add new capabilities
without proliferating effect types or polluting HTML.
