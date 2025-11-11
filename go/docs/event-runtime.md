# Event Runtime Redesign

This document describes the planned overhaul of PondLive’s event pipeline so
that:

1. Server-rendered HTML stays clean (no `data-on*` or `data-live-ref` attrs).
2. Every binding—`h.On`, `h.OnWith`, router helpers, ref listeners—hydrates via
   the same metadata channel.
3. The browser-side runtime no longer distinguishes between “element handlers”
   and “ref handlers”; they share one delegation path.

It builds on the structured DOM and clean handler hydration plans.

## 1. Current State (for context)

1. Go code registers handlers through `h.On` / `h.OnWith`. Refs add listeners
   through `ref.AddListener`.
2. During render, `FinalizeWithHandlers` writes `data-on<Event>` attributes for
   every binding and `data-live-ref` for refs.
3. Client hydration scans every element, parses/strips the attributes, and
   caches handler IDs inside `handlerBindings`.
4. Event delegation:
   - Finds handler IDs for `h.On` via the cache.
   - If none match, tries ref-only metadata (`refs.ts`) and sends events with
     `hid=refId`.
   - Router metadata piggybacks on `data-router-*` attrs.

Problems: polluted HTML, duplicated logic, two code paths (handlers vs refs),
attribute parsing cost.

## 2. Target Model

### 2.1 Metadata-Driven Bindings

Boot/init payloads (and component boots) ship:

```jsonc
"handlers": {
  "slots": {
    "12": [
      { "event": "click", "handler": "h5", "listen": ["pointerdown"], "props": ["target.value"] }
    ]
  },
  "statics": [
    { "componentId": "root", "path": [3], "bindings": [{ "event": "click", "handler": "ref:2", "props": ["target.files"] }] }
  ],
  "router": [
    { "componentId": "root", "path": [1], "meta": { "path": "/foo", "replace": "true" } }
  ]
}
```

- `slots` covers dynamic attribute slots (stable IDs).
- `statics` handles the rare cases where no slot exists.
- Router metadata becomes part of the same structure; no more `data-router-*`.
- **Ref listeners are indistinguishable** from normal handlers: their
  `handler` field is simply the ref ID (e.g., `ref:3`), and `props` lists any
  selectors registered via `ref.AddListener`.

### 2.2 Unified Client Hydration

1. Resolve slot/component paths to DOM nodes (per structured DOM doc).
2. Populate `handlerBindings` (WeakMap) for every node:
   ```ts
   handlerBindings.set(element, new Map([
     ["click", { id: "h5", props: ["target.value"] }],
     ["change", { id: "ref:2", props: ["target.files"] }]
   ]));
   ```
3. Store handler metadata (`handlers` map) keyed by ID:
   ```ts
   handlers.set("h5", { event: "click", listen: ["pointerdown"], props: ["target.value"] });
   handlers.set("ref:2", { event: "change", props: ["target.files"] });
   ```
4. Router metadata uses a parallel WeakMap (`routerBindings`), hydrated from
   the `router` section above.

### 2.3 Event Dispatch

Single flow inside `handleEvent`:

```ts
const info = findHandlerInfo(target, eventType); // walks handlerBindings
if (!info) {
  tryRouterClickFallback(...); // only for plain anchors
  return;
}
const handlerId = info.id;
const meta = handlers.get(handlerId);
const refRecord = refRegistry.get(handlerId); // only set for ref:* IDs
const propsList = merge(meta?.props, refRecord?.meta.events[eventType]?.props);
const payload = extractEventPayload(e, target, propsList, info.element, refRecord?.element);
if (refRecord) refRecord.notify(payload);
sendEvent({ hid: handlerId, payload });
```

Key points:
- No attribute reads or fallback scanning.
- Ref IDs are treated exactly like normal handler IDs; the only extra step is
  `refRegistry.notify` so `UseElement` hooks can cache payloads.
- Router metadata (path/query/hash) is merged using the same `routerBindings`
  map without touching attributes.

## 3. Server Responsibilities

1. **Stop emitting metadata in HTML.** `FinalizeWithHandlers` should populate
   `HandlerAssignments` / `RefID`s for use by structured rendering, but the
   HTML itself stays untouched.
2. **Handler metadata extraction.**
   - For every `HandlerAssignment`, emit a `HandlerBinding` entry keyed by the
     slot or component path that owns it.
   - Include `Listen`/`Props` arrays as-is.
   - Ref bindings are not special: they already have IDs (`ref:*`) assigned by
     the ref hook; just include them.
3. **Router metadata.**
   - When a component sets router props (e.g., `router.To`, `router.Replace`),
     store them in the structured payload similar to handler bindings (same
     addressing scheme).
4. **Ref metadata.**
   - Continue emitting `protocol.RefMeta` for DOMCall/DOMGet. Event selectors
     live in the handler metadata instead.

## 4. Client Responsibilities

1. Hydrate bindings from metadata, not DOM attributes.
2. Maintain `handlerBindings`, `routerBindings`, and `refRegistry` (for DOM APIs).
3. Dispatch events through the unified path above.
4. Error handling: if a binding path fails to resolve, log in dev mode and
   request a component reboot/reconnect (same recovery plan as structured DOM).

## 5. Props & Selectors

- Server merges selector lists when building bindings (`binding.Props` already
  combines defaults, `h.OnWith` additions, and ref `AddListener` props).
- Client collects selectors by merging:
  1. `HandlerMeta.props` (per handler ID).
  2. Optional ref-specific selectors recorded in `RefMeta.events[event].props`
     (only relevant for DOMCall/DOMGet caching).
- `extractEventPayload` already accepts a selector list; no changes needed.

## 6. Router Integration

- Router helpers (e.g., `router.To`, link intercept metadata) hydrate through
  the same binding mechanism: router props live in the `router` block keyed by
  `(componentId,path)`.
- When an event fires, we copy router metadata into the payload if present,
  matching current behavior but without DOM attributes.

## 7. Migration Plan

1. **Phase 1** – Server emits both legacy attributes and new metadata. Client
   consumes metadata when a feature flag is on; otherwise falls back to legacy
   parsing.
2. **Phase 2** – Once telemetry shows 100% coverage, remove attribute emission
   and the old client path.
3. **Phase 3** – Clean up remaining code paths (ref fallback, router attribute
   parsing).

## 8. Open Questions

| Question | Notes |
| --- | --- |
| Slot ID stability | Yes—slot IDs remain constant for the life of a component instance. |
| Static handler frequency | Rare but supported via `handlers.statics`. Instrument to confirm before optimizing away. |
| Ref vs element handlers | Fully unified. The only difference is handler ID prefix, which the server uses for routing. |
| Performance | WeakMap lookups vs attribute reads need benchmarking, but expected to be equal or faster. |
| Error recovery | Missing bindings trigger component reboot or full reconnect, same as structured DOM errors today. |

With this design, events become metadata-only: clean HTML at every stage, one
delegation path, and no distinction between refs and regular handlers on the
client. This sets the foundation for the clean handler hydration work and
reduces complexity throughout the runtime.
