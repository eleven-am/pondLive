# Router Architecture – Current State & Improvements

This note captures how the PondLive router works today (server + client) and
highlights opportunities to simplify navigation, metadata, and component boot
behaviour. It ties into the structured DOM / clean handler work already in
progress.

## 1. Runtime Flow (Server)

1. **Router root** (`Router`) wraps the tree with a `routerState` containing
   `getLoc`/`setLoc` closures backed by `UseState`. Each session stores a
   `routerSessionState` entry (navigation history, handlers, route params,
   render depth).
2. **Placeholders** – During SSR, `RouterLink`, `Routes`, etc. emit placeholder
   fragments (link/routing metadata is registered on the session). When the
   router renders on the server, `normalizeRouterNode` replaces placeholders
   with live elements by consulting the placeholder maps.
3. **Navigation handlers** – `routerState.setLoc` calls `storeSessionLocation`,
   updating session navigation state and notifying the owning `LiveSession`
   (used for diagnostics and HTTP seed extraction). `recordNavigation` keeps a
   history for diffing and sending pending nav updates to the client.
4. **Component boot** – When navigation changes the active route tree, the
   session either:
   - Emits pure diffs (if routing only toggled text/attrs), or
   - Produces a `componentBoot` payload for the subtree (HTML + statics/dynamics
     + bindings), or
   - Falls back to a full template reset (rare; e.g., router errors).
5. **Router links** – `renderLink` builds `<a>` tags with `data-router-*`
   attributes (path/query/hash/replace) and attaches an `h.OnWith("click", …)`
   handler that calls `performLocationUpdate`. During SSR it also outputs those
   attributes so the client can intercept before LiveUI hydrates.

## 2. Runtime Flow (Client)

1. **Event delegation** – Global click handlers examine `data-router-*`
   attributes. If a Live handler exists, the router payload is merged into the
   event payload; otherwise `registerNavigationHandler` intercepts plain `<a>`
   clicks and calls `sendNavigation`.
2. **Optimistic navigation** – `sendNavigation` pushes a new history entry
   immediately (tracking `lastOptimisticNavTime` to avoid duplicate pushes) and
   sends `{t:"nav", path,q,hash}` to the server. Router metadata in handler
   payloads allows Live handlers to piggyback navigation on regular events.
3. **Popstate** – Browser back/forward triggers `send pop` messages so the
   server can reconcile location state.
4. **Component boot** – When the server responds with a component boot,
   `Client.applyEffects` replaces the affected DOM range and re-hydrates slots/
   handlers/refs for that subtree. Router navigations typically produce one of
   these payloads.

## 3. Structured Metadata Usage

- Router currently relies on `data-router-*` attributes in the DOM to pass path
  info to the client (both for SSR fallback and for handler payloads). This
  conflicts with the clean HTML goal.
- Component boot payloads contain enough data to hydrate new router subtrees,
  but the format differs from initial boot: router metadata lives in ad-hoc
  fields instead of sharing the same structured template schema.

## 4. Pain Points / Risks

1. **HTML pollution** – Router embeds `data-router-*` attributes and relies on
   them for both event handlers and the fallback navigation handler.
2. **Duplicated hydration logic** – Boot vs. component boot payloads are shaped
   differently; router-specific code has to special-case component boot effects.
3. **Ordering ambiguity** – Component boots are processed inside `effects`
   after patches, so diffs targeting the old DOM can race with router replacements.
4. **Coarse-grained resets** – Navigations often trigger full component boot
   even when a diff/promotion would suffice, shipping more HTML than necessary.
5. **Error recovery** – If a router subtree fails to hydrate (slot path mismatch)
   the only recovery path is a full session reset/reconnect.

## 5. Router v2 Plan (Clean Metadata + Unified Templates)

With the new protocol (see `docs/protocol.md`) we can redesign the router off
DOM attributes while reusing the unified template pipeline.

### 5.1 Template Bindings

- Router links emit binding entries alongside handlers. Each entry specifies
  `{componentId, path, meta}` where `meta = { path, query, hash, replace }`.
- No `data-router-*` attributes appear in HTML—`RouterLink` renders plain `<a>`
  elements with standard `href` values for fallback.
- Component boot templates (`t:"template"`) include router bindings for their
  subtree so the client hydrates newly mounted routes identically to boot/init.

### 5.2 Client Hydration

- When a template frame (boot/init/component boot) arrives, the hydrator
  resolves every router binding entry into a DOM element (using component ranges
  + child paths) and stores metadata in a `routerBindings` `WeakMap<Element,
  RouterMeta>`.
- On subtree replacement the hydrator clears router metadata for DOM nodes being
  removed before applying new bindings; no attribute parsing required.

### 5.3 Navigation Flow

- Event delegation consults `routerBindings` instead of `data-router-*`. If the
  click target (or ancestor) has router metadata and the click isn’t modified,
  it prevents default, sends a `nav` message with `{path, query, hash}`, and
  updates `history` (`push` or `replace` based on metadata).
- Anchors without router metadata fall back to the existing interception path
  (plain `<a>` detection), so non-Live links still work.
- Server-initiated navigations still emit `NavDelta` entries; the client applies
  them to keep browser history in sync with server state.

### 5.4 Template Sequencing & Diffing

- `t:"template"` frames are applied before any diff ops in the same tick. That
  guarantees router metadata exists for new DOM nodes before events fire.
- Structured metadata + promotion tracking let more navigations be expressed as
  diffs (keyed outlet regions) instead of full component boots, reducing HTML
  churn.

### 5.5 Recovery & Diagnostics

- If resolving a router binding fails (no DOM node at `{componentId,path}`),
  dev-mode clients log a warning, fire a `diagnostic` event, and may send a
  targeted router reset request (future message, e.g., `{t:"routerReset", componentId}`)
  so the server can resend a template for that subtree. This avoids full session
  reconnects.
- Diagnostics surfaced via `docs/diagnostics.md` include router context
  (component name, path) so the overlay can highlight the failing outlet and
  offer a “reset subtree” action.

### 5.6 Summary

| Area | Change | Impact |
| --- | --- | --- |
| Metadata | Router props live in `bindings.router` (`componentId + path`). | Clean HTML, single hydrator code path. |
| Templates | Boot/init/component boot share schema (`t:"boot"`, `t:"init"`, `t:"template"`). | Predictable sequencing + reuse. |
| Event handling | Delegation inspects `routerBindings` instead of DOM attributes. | No attribute parsing, faster dispatch. |
| Recovery | Targeted router resets/logging when hydration fails. | Better resilience without full reconnects. |

## 6. Next Steps

1. Implement router binding export on the server (during placeholder
   normalization) and include them in template payloads.
2. Update the client hydrator to populate `routerBindings` from template frames
   and route click events through the new metadata.
3. Enforce template-before-patch ordering so router bindings are always current
   before diffs run.
4. Add diagnostics/logging around router hydration failures and consider a
   router reset control frame for targeted recovery.

With these changes, the router keeps the “author HTML == rendered HTML”
guarantee and shares infrastructure with clean handlers, bindings, and unified
template payloads. The result is simpler hydration, fewer surprises after
navigations, and better tooling for diagnosing routing issues.
