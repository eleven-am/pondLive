# Hydration Flow – Template Frames & Bindings

This document describes how the PondLive client applies template frames
(`t:"boot"`, `t:"init"`, `t:"template"`) and hydrates structured metadata
(slots, bindings, refs, uploads, router) before processing diffs.

## 1. Overview

Hydration runs in two phases each tick:

1. **Template phase** – apply any template frames received before the diff frame
   (boot/init/component boot). This splices HTML into the DOM, registers slot
   anchors, and hydrates bindings (handlers/router/uploads/refs).
2. **Diff phase** – apply incremental patch ops (`diff.Op`), leveraging the slot
   and binding registries prepared in the template phase.

A hard constraint: *template frames must apply before the diff frame in the same
tick*. Otherwise bindings/slots would be stale when diffs run.

## 2. Template Application

Template payload fields (see `docs/protocol.md`):

- `scope`: `{componentId, parentId, parentPath}` for component boots. Omits for
  root boot/init.
- `html`/`s`/`d`: statics/dynamics for the template.
- `slots`, `slotPaths`, `listPaths`, `componentPaths`: structural metadata.
- `bindings`: `slots`, `statics`, `router`, `uploads`, `attachments`, etc.
- `templateHash`: optional cache key (hash of `s/d`) for fragment caching.

### 2.1 Locate DOM Range

1. Determine the parent container:
   - For root boot/init, use `document.body` (or root mount node).
   - For component boot, find the parent component’s DOM range using
     `componentPaths[parentId]`.
2. Walk `parentPath` (array of child indices) from the parent container to find
   where the component should live (even when fragments flatten children).
3. Use `componentPaths[componentId]` (if present) to remove the old DOM range
   (`firstChild`/`lastChild`). If missing, fall back to heuristics (e.g., remove
   `parentPath` range length).
4. If the range cannot be found, emit a diagnostic (“Hydration mismatch for
   component X”) with suggestions (reload page, reset component). In dev mode,
   offer a “Reset component” button that requests a fresh template for that
   component.

### 2.2 Insert Template & Register Slots

1. If `templateHash` matches a cached entry, clone the cached `DocumentFragment`.
   Otherwise, build the DOM from `html` or `s/d`, register slots, and stash the
   fragment in the cache (LRU).
2. Insert the fragment at the resolved DOM range.
3. Register slot anchors via `slotPaths` (same as today—use `dom-index` to map
   slot ID → DOM node).
4. Register list containers via `listPaths`.
5. Update `componentPaths` for the component and its descendants.

### 2.3 Hydrate Bindings

For each binding bucket:

- **Slots**: `bindings.slots[slotId]` attaches handler IDs to slot anchors (same
  as current handler system).
- **Statics (handlers)**: resolve `{componentId,path}` to an element, update
  `handlerBindings` (`WeakMap<Element, Map<event,id>>`). Remove stale entries
  for nodes replaced by the template.
- **Router**: resolve path, store metadata in `routerBindings` (`WeakMap`), so
  event delegation can intercept clicks without DOM attributes.
- **Uploads**: resolve path, store `{uploadId, accept, multiple, maxSize}` in an
  `uploadBindings` map so `UploadManager` finds inputs without `data-*`.
- **Attachments/Refs**: resolve path, bind `ref:*` IDs to DOM elements. Update
  `refs.ts` registry so `UseElement` refs point to the right nodes.

Binding resolution algorithm:

1. Use `componentPaths[componentId]` to get the component’s DOM range.
2. Walk the `path` indices to find the target element (taking fragments into
   account).
3. If found, update the relevant binding map. If not, emit a diagnostic (`phase:
   "template_hydrate"`) noting the component/path so devs can inspect the DOM.

### 2.4 Diagnostics & Recovery

- Missing DOM range → diagnostic with component name/key and suggestion to reset
  or reload. In dev mode overlay, show a “Reset component” button that requests
  a new template for that component only.
- Binding failures (router/upload/ref) → diagnostic with component/path and
  readable message (“Could not bind upload slot `u3` to `<input>`; check template”).
- Hydration diagnostics stream via `t:"diagnostic"` frames (see
  `docs/diagnostics.md`). Prod builds can log them server-side without overlay.

## 3. Diff Application

After templates are applied, the diff frame (`t:"frame"`) runs:

1. Update `handlers.add/del` → adjust handler metadata cache.
2. Apply `patch` ops using existing infrastructure (`slotMap`, `listMap`, etc.).
3. Apply `bindings` deltas (if present) the same way as template binding hydration.
4. Emit diagnostics if slot/list anchors referenced in ops can’t be found
   (“Hydration mismatch: slot 12 missing from component `AccountPanel`”).

## 4. Template Cache

- Maintain an LRU cache keyed by `(componentId, templateHash)`.
- Cache entries include:
  - `DocumentFragment` representing the static DOM (cloned for reuse).
  - `statics`/`dynamics` arrays.
  - Precomputed binding blueprints (optional).
- When a template payload arrives:
  - If entry exists, clone fragment + reuse metadata.
  - If not, build DOM, store entry.
- Evict oldest entries when exceeding size limit (configurable).

## 5. Dev Mode Overlay Hooks

- Hydration diagnostics include component names/paths, making overlay entries
  clickable (scroll to DOM node, offer “Reset component”).
- Boundaries (see `docs/error-boundary.md`) can trigger `live.ResetBoundary(ctx)`
  to retry rendering after a failure.
- Router-specific failures can offer “Reset route” (requests component boot for
  the router subtree).

## 6. Summary

- Template frames are the source of truth for DOM structure and bindings; they
  must apply before diffs.
- Metadata (`componentPaths`, `slotPaths`, binding paths) allows deterministic
  DOM traversal without DOM attributes.
- Diagnostics provide friendly messages when hydration fails, with dev-mode
  recovery hooks (reset component/route).
- Template caching reuses DOM fragments for repeated templates, improving
  performance for router transitions and list mounts.
