# Promotion Tracker & Dynamic Slot Strategy

Rendering uses structured statics/dynamics to keep patches small. This doc
covers the “promotion tracker” mechanism that decides when text/attr nodes
become dynamic slots, how slot IDs stay stable, and heuristics for minimizing
diff traffic.

## 1. Background

- `render.Structured` flattens HTML into:
  - `S` – static HTML chunks.
  - `D` – dynamic slots (`DynText`, `DynAttrs`, `DynList`).
  - Metadata (slot/list/component paths, handler bindings).
- By default only nodes marked mutable (e.g., `TextNode.Mutable`,
  `Element.MutableAttrs`) or nodes with handlers become dynamic.
- Promotion tracker lets the runtime upgrade static nodes to dynamic lazily,
  based on heuristics or runtime observation (e.g., state changes).

## 2. PromotionTracker Interface

```go
type PromotionTracker interface {
    ResolveTextPromotion(componentID string, path []int, value string, mutable bool) bool
    ResolveAttrPromotion(componentID string, path []int, attrs map[string]string, mutable map[string]bool) bool
}
```

- `componentID`: deterministic ID assigned to each component instance
  (`buildComponentID`). Anchors promotions to specific components.
- `path`: child indices from the component root to the target node; used to
  identify nodes across renders.
- `mutable`: flags set by builders (`TextNode.Mutable`, `Element.MutableAttrs`)
  or upstream logic.

`StructuredOptions{Promotions: tracker}` passes the tracker into the builder so
it can consult it while visiting nodes.

## 3. Default Heuristics

- **Text nodes**:
  - Dynamic if `TextNode.Mutable` is true.
  - Else, promotion tracker decides. Without a tracker, text stays static.
  - When dynamic, `Dyn{Kind: DynText, Text: value}` is appended; builder adds a
    slot anchor to the parent element.
- **Attr nodes**:
  - Dynamic if the element has handler assignments (event bindings) or if its
    `MutableAttrs` map contains `*` or specific attributes.
  - Else, tracker decides via `ResolveAttrPromotion`.
  - If dynamic, the entire start tag becomes a `DynAttrs` slot (`Attrs` map
    sorted/indexed).

This keeps most nodes static, reducing `SetText`/`SetAttrs` churn. Only nodes
that actually change become slots.

## 4. Slot Reuse & IDs

- Slots map to `SlotMeta{anchorId}` (sequential integers). Anchors are stable as
  long as the component structure stays the same.
- When a node is promoted later (e.g., first render static, later render
  dynamic), the structured output changes: statics shrink, a new dynamic entry
  is inserted, and corresponding `SlotPath` entries map the slot to
  `{componentId, elementPath}`.
- Promotion decisions must be deterministic between renders; otherwise slot IDs
  would thrash. Trackers should base decisions on stable component/path info,
  not ephemeral state.

## 5. Practical Use Cases

- **Text promotions** for stateful content that initially looked static (e.g.,
  route-dependent text, optional user data).
- **Attr promotions** when class/style/`data-*` changes are driven by state but
  not marked mutable at build time.
- **Performance tuning** – instrument the tracker to promote nodes only when a
  diff would otherwise require template reset.

## 6. Tracker Implementation Tips

- Keep an internal map keyed by `(componentId, pathHash)` storing whether a node
  should be dynamic. Example from tests:
  ```go
  type recordingTracker struct {
      dynamic map[string]bool
  }
  func (r *recordingTracker) ResolveTextPromotion(id string, path []int, value string, mutable bool) bool {
      key := fmt.Sprintf("%s/%v", id, path)
      return r.dynamic[key]
  }
  ```
- Use component metadata (`ComponentSpan`, `componentPaths`) to recalc paths if
  needed when components rerender with new children.
- Avoid promoting entire subtrees unless necessary; each dynamic slot adds diff
  work.

## 7. Heuristics to Keep Patches Small

1. **Prefer static** until proven dynamic. Most nodes never change; keeping them
   static reduces `SetText` noise.
2. **Promote only stable targets**. If a node appears in conditional branches,
   ensure the path is consistent before promoting; otherwise slot IDs may shift.
3. **Use keyed lists**. `UseStream`/`h.Map` should always supply keys so list
   diffs can reuse slots rather than re-render entire sections.
4. **Avoid global promotions**. Track per-component/per-path rather than entire
   component classes; components rendered multiple times with different content
   should promote independently.
5. **Monitor diff size**. Instrument promotion tracker to log when large
   `SetText` runs persist; selectively promote those nodes.

## 8. Future Enhancements

- **Auto-promotion based on runtime diffing** – track nodes that required manual
  DOM ops (e.g., repeated attribute rewrites) and promote them in subsequent
  renders.
- **Heuristic thresholds** – promote after N diffs or when diff payload exceeds
  a byte threshold.
- **Component-level overrides** – allow components to mark whole subtrees
  opt-in/out of promotions (`live.AutoPromote(ctx, fn)`).
- **Tooling** – add debug stats (which nodes are dynamic, how many promotions)
  to help developers tune performance.

Promotion tracking is the bridge between static templates and dynamic slots.
Keeping it deterministic and conservative ensures diff payloads stay minimal
while still supporting rich state updates.
