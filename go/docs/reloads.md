# Reload & Patch Mechanics

PondLive currently has three “reload” paths:

1. **Boot** – first payload sent after the HTTP response. Contains full HTML,
   statics/dynamics, handler/ref metadata, location, and client config.
2. **Patch** – steady-state diffs: DOM ops, slot updates, handler/ref deltas,
   nav/effects/metrics. Assumes the client already has the template.
3. **Component boot** – scoped template refresh for a subtree (router swaps,
   lazy components, error recovery). Ships HTML + metadata for just that
   component and applies it via an effect.

This document explains why each path exists and how we can improve them.

## 1. Current Flow

| Path | When used | Payload highlights | Client steps |
| --- | --- | --- | --- |
| Boot | Initial hydration | `html`, `s`, `d`, slot/list/component paths, `handlers`, `refs`, `location`, `client config` | Mount HTML, hydrate metadata, start socket |
| Patch | Regular updates | `patch` ops, `delta.slots`, `handlers.add/del`, `refs.add/del`, `effects`, `nav`, `metrics` | Apply DOM patch, run effects, update metadata |
| Component boot | Subtree reset | HTML + statics/dynamics for component, scoped bindings, slot/list/component paths | Remove old DOM range, insert new fragment, hydrate metadata for that subtree |

Component boots live inside the `effects` array today.

## 2. Limitations

1. **Duplicated payload formats** – boot and component boot carry similar
   information but via different shapes (top-level frame vs. effect). Hydration
   code has to handle both.
2. **Heavy-handed resets** – component boot replaces the entire subtree even if
   only small parts changed (e.g., route swap where most markup stays the same).
3. **Ordering ambiguity** – component boots processed after patches can race
   with diffs targeting the old DOM.
4. **Recovery bluntness** – when hydration desyncs, we typically need a full
   reconnect instead of patching just the broken subtree.
5. **Boot latency** – initial boot waits for the full template and metadata to
   arrive before the page becomes interactive.

## 3. Improvement Opportunities

### 3.1 Unify Template Payloads

Introduce a single “template payload” schema used for both boot and component
boot, containing:

- HTML fragment
- Statics/dynamics (`s`, `d`)
- Slot/list/component paths
- Handler/ref metadata scoped to the template

Whether it’s the root or a child component, the client hydrates via the same
code path. Component boot could become a dedicated frame type or a top-level
section rather than an effect payload.

### 3.2 Reduce Component Boots via Promotions

With structured DOM metadata, many subtree changes can become regular diffs if
we:

- Promote static slots to dynamic when needed (text/attr promotions already
  exist).
- Emit keyed list/list-path metadata so router swaps can reuse existing DOM
  nodes instead of wholesale replacements.

Fewer component boots → less HTML shipped and less hydration work.

### 3.3 Deterministic Sequencing

Ensure template payloads apply before any diffs referencing the new DOM:

- Either split component boots into their own frames (processed ahead of patch
  ops) or guarantee ordering within the same frame.
- Document the contract so the client can safely clear old ranges before
  applying patches.

### 3.4 Targeted Recovery

When hydration fails (slot path missing, ref lookup fails), give the client a
way to request a fresh template for the affected component instead of dropping
the whole session. For example:

1. Client detects mismatch → sends `componentReset(componentId)` control frame.
2. Server responds with a component boot payload for that subtree.

This keeps the rest of the UI live and improves resilience.

### 3.5 Progressive Boot

To improve first load latency:

- Stream HTML first (already part of HTTP response), then send metadata chunks
  over the socket so hydration can begin incrementally.
- Allow component boot payloads to stream similarly for large subtrees.

### 3.6 Template Caching (Future Idea)

If multiple routes/components share identical templates, cache component boot
payloads on the client and only ship diff metadata + data payloads. Requires
stable template hashes and cache invalidation strategy.

## 4. Summary

- Keep all three concepts (boot, patch, component boot) but unify their
  schemas and improve sequencing so hydration logic is consistent.
- Reduce reliance on component boot by using structured metadata to express
  more updates as diffs/promotions.
- Add recovery and streaming capabilities to improve robustness and perceived
  performance.

These changes build on the structured DOM and clean handler plans already in
progress, giving the runtime a cleaner, more deterministic “reload” story.
