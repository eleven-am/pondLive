# State & Hook Lifecycle

This note describes how PondLive’s hooks interact with the component session:
hook ordering, dirty marking, effect scheduling, refs/streams/contexts, and
where diagnostics come from. Use it as a guide when modifying hook internals or
adding new hook flavours.

## 1. Hook Frames & Ordering

- Every component owns a `hookFrame` (`[]any cells + idx`). Before each render
  `component.beginRender` resets `frame.idx = 0` and calls `resetAttachment()`
  on cells that implement it (e.g., element refs).
- Each hook (`UseState`, `UseMemo`, `UseEffect`, …) reads/writes `frame.cells`
  at the current index and increments `frame.idx`. If the render path changes
  (e.g., calling a hook inside a conditional), `panicHookMismatch` fires with a
  detailed diagnostic (component name, expected hook, index, actual cell type).
- After rendering, `component.endRender` unmounts any child component that
  wasn’t rendered in the current epoch.

## 2. UseState & Dirty Marking

- `UseState` allocates a `stateCell` on first render (`val`, `eq`, owning
  component). Get/set closures capture the cell.
- `set(next)` compares against the stored value using `eq` (customisable via
  `WithEqual`). When the value changes, it updates the cell and calls
  `sess.markDirty(cell.owner)` → marks the component (and ancestors) dirty and
  schedules a flush unless we’re already flushing/handling an event.
- `WithoutRender` increments `sess.suspend` while executing a callback so
  multiple state updates can batch before scheduling a flush. `NoRender` clears
  the dirty flag for the current component if the user handled DOM updates
  manually.

## 3. Memo, Ref, Element Ref

- `UseMemo` stores `{val,deps}`. On dependency change it recomputes; otherwise
  returns the cached value.
- `UseRef` stores mutable `Ref[T]{Cur}` cells that do not trigger renders.
- `UseElement` allocates a `dom.ElementRef` with a stable `ref:*` ID:
  - Captures a `state` slot inside `elementRefCell` so ref-local data can mark
    the owning component dirty when it changes.
  - Registers the ref with the session (`registerElementRef`) so metadata and
    DOM actions can target it later.
  - Implements `resetAttachment()` so each render can attach the ref to at most
    one element; reusing it without calling `ResetAttachment` panics.

## 4. UseEffect Lifecycle

- `UseEffect` records dependencies in an `effectCell`. On initial render it
  enqueues `effectTask{comp,index,setup}` via `sess.enqueueEffect`.
- After flush completes, `runEffects` executes each task:
  1. Calls `setup()`.
  2. Stores the returned `Cleanup` on the corresponding `effectCell`.
- When deps change, we enqueue a cleanup (`enqueueCleanup`) followed by a new
  effect. If deps unchanged but no cleanup exists (first render), we enqueue the
  setup again.
- Cleanups run before the next effect or during unmount.

## 5. UseStream

- `UseStream` composes hooks:
  - State: `UseState([]StreamItem)` stores the current collection.
  - Ref: `UseRef(map[key]int)` tracks the index of each key for fast lookup.
- `StreamHandle` methods clone/update the slice, enforce unique keys, rebuild
  the index, and call the state setter. Mutations therefore mark the component
  dirty through normal `UseState` semantics.
- Rendering: `renderStreamFragment` produces a fragment of row nodes, ensuring
  each row’s `*html.Element` has `Key = item.Key` so diffing can emit list ops.

## 6. Context Providers & Consumers

- `Context[T]` assigns a unique `contextID`. Providers (`Provide`,
  `ProvideFunc`) ensure a `providerEntry` exists for the current component:
  `{get,set,assign,owner,eq,last}`.
- Provider entries track whether the current component is the owner and whether
  they’re derived (i.e., computed via `ProvideFunc`).
- `Use` walks up the component tree to find the nearest entry; `UsePair` returns
  a getter/setter pair that can shadow local state when the consumer is also a
  provider.
- `UseSelect` combines contexts with `UseRef` to memoize derived slices and
  avoid rerenders when an equality check passes.

## 7. Dirty Flags, Batching, Diagnostics

- `sess.markDirty(component)`:
  1. Adds the component to `sess.dirty`.
  2. Marks `sess.dirtyRoot = true`, `sess.pendingFlush = true`.
  3. Calls `component.markDirtyChain()` to set `dirty=true` on the component and
     its ancestors.
  4. If not already flushing or suspended, asks the owning `LiveSession` to
     `flushAsync`.
- `sess.Flush()` re-renders dirty components (respecting hook order), diffs the
  DOM, and runs pending effects/cleanups. Diagnostics captured during render or
  effect setup bubble up via `withRecovery`.
- Hook misuse (calling outside render, mismatched order) panics with
  `hookMismatchError` that includes component name, hook index, and suggestion.

## 8. Attachment API (Refs & Uploads)

- `h.Attach` becomes a generic wrapper around an `Attachment` interface:
  ```go
  type Attachment interface {
      AttachTo(el *h.Element)
  }
  ```
  Hooks implement this interface to declare how they want to bind to a DOM node.
- `UseElement` exposes `ref.Attachment()` so components call
  `h.Attach(myRef.Attachment())`. Internally it still calls
  `dom.AttachElementRef`, enforces descriptors, and registers the ref ID for
  bindings.
- `UseUpload` exposes `upload.Attachment()` so components can write
  `h.Attach(upload.Attachment())` instead of spreading props. During render the
  attachment registers upload slot metadata (slot ID, accept, multiple, maxSize)
  with the renderer so it ends up in `bindings.uploads`.
- `BindInput` remains as a backward-compatible helper that calls
  `h.Attach(upload.Attachment())` under the hood, but new code should prefer the
  uniform attachment API.

Attachments let hooks plug into the renderer without mutating HTML (no
`data-*`); the renderer records bindings (handlers, uploads, refs) inside the
structured template payload.

## 9. Interaction Summary

| Hook | Backing cell | Dirty trigger | Notes |
| --- | --- | --- | --- |
| `UseState` | `stateCell` | Setter marks owning component dirty | `WithEqual` customises comparisons |
| `UseMemo` | `memoCell` | No direct dirty trigger; depends on consumer state | Uses `reflect.DeepEqual` on deps |
| `UseRef` | `*Ref[T]` | Manual mutations only affect component when user does so | Used for mutable values and context memo |
| `UseElement` | `elementRefCell` | Ref state setter marks component dirty | Allocates `ref:*` IDs, registers with session, implements the attachment interface so you can call `h.Attach(ref)`. |
| `UseEffect` | `effectCell` | Effects run post-flush; cleanup enqueued on dep change | Cleanups stored back on the cell |
| `UseStream` | `UseState` + `UseRef` | Handle methods call setter | Enforces unique keys, wraps row elements |
| `UseUpload` | `uploadSlot` | Progress updates call `markDirty` | Implements the attachment interface so you can call `h.Attach(upload)` to bind inputs. |
| `Context` | `providerEntry` per component | `Provide` updates session store & owner entry | `UsePair` supports local shadowing |

## 9. Potential Improvements

- **Better diagnostics** – expose hook state in dev tooling (e.g., log hook
  indices, render stack on panic) and include component IDs in errors.
- **Concurrent safety** – today hooks assume single-threaded renders. If we ever
  add concurrent rendering, we’ll need per-render copies of frames and stricter
  locking around provider entries.
- **Effect tracing** – capture timing per effect (we already record totals in
  `FrameMetrics`), but we could flag individual slow effects with component/id
  for developer dashboards.
- **Stream diffing** – `UseStream` currently clones slices on every mutation.
  Consider structural sharing or pooling for large lists.
- **Context cleanup** – provider entries stick around even if a component stops
  providing. We might prune them (or reuse entries) to reduce allocation churn.

Understanding these hooks—and how they touch `ComponentSession`—is essential
when touching runtime internals. Changes to hook ordering, dirty marking, or
effect scheduling ripple through rendering, diffing, and diagnostics.
