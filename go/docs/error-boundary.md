# Error Boundary Design

PondLive will expose an error boundary builder so components can catch panics in
a subtree and render fallback UI without taking down the entire session. This
document outlines the API and runtime behaviour.

## 1. Goals

1. Let developers wrap unstable regions with a small, declarative pattern.
2. Keep the API consistent with existing HTML builders (e.g., `h.Div`,
   `h.Fragment`).
3. Ensure diagnostics still flow through the standard pipeline (`docs/diagnostics.md`).
4. Provide a reset mechanism so boundaries can re-attempt rendering when inputs
   change (e.g., when a `key` changes).

## 2. API (Builder Pattern)

We introduce a builder `h.ErrorBoundary` with the familiar signature:

```go
func ErrorBoundary(ctx live.Ctx, props ...h.Prop) h.Node
```

It accepts children (variadic `h.Node` like `h.Div`) and props such as
`h.OnError`. The builder has full access to hooks via `ctx`, so it can manage
error state internally.

```go
type ErrorInfo struct {
    Diagnostic live.Diagnostic
}

func Dashboard(ctx live.Ctx) h.Node {
    return h.ErrorBoundary(
        ctx,
        h.OnError(func(ctx live.Ctx, info ErrorInfo) h.Node {
            return h.Div(
                h.Class("error-panel"),
                h.H3(h.Text("Dashboard Error")),
                h.P(h.Text(info.Diagnostic.Message)),
                h.Button(
                    h.On("click", func(h.Event) h.Updates {
                        return live.ResetBoundary(ctx)
                    }),
                    h.Text("Retry"),
                ),
            )
        }),
        UserProfile(ctx),
        StatsWidget(ctx),
        ActivityFeed(ctx),
    )
}
```

- `h.ErrorBoundary` takes variadic children like other builders.
- `h.OnError` attaches the fallback function (`func(live.Ctx, ErrorInfo) h.Node`).
- Additional props (e.g., `h.Key("user:"+id)`) can reset the boundary when they
  change.

## 3. Runtime Behaviour

- Internally `h.ErrorBoundary` renders children inside a localized recovery
  scope (similar to `withRecovery`). When a child panics, it:
  1. Captures the `Diagnostic`.
  2. Calls the fallback handler so the boundary can render fallback UI.
  3. Reports the diagnostic via `ReportDiagnostic` (so overlays/logs update).
  4. Caches the `ErrorInfo` in boundary state so future renders use the fallback
     until reset.
- Boundaries can reset when:
  - Their `key` changes (same semantics as keyed components).
  - The fallback invokes `live.ResetBoundary()` (helper that clears the cached
    error and re-renders the child).

## 4. API Details

- Props:
  - `h.OnError(func(live.Ctx, ErrorInfo) h.Node)` – required, defines fallback UI.
  - `h.Key(string)` – optional; changing the key resets the boundary.
  - Future props could include `h.OnReset`, `h.ResetOn` helper, etc.
- `live.ErrorInfo` exposes the captured diagnostic so fallbacks can display
  component names, messages, suggestions, etc.
- `live.ResetBoundary(ctx)` (helper) schedules the boundary to retry its children.

## 5. Diagnostics Integration

- When a boundary catches a panic, it still reports the diagnostic through
  `ReportDiagnostic` so dev mode overlays/logs show the failure.
- The overlay can detect when a boundary reset is available (e.g., show a
  “Retry” button that calls `live.ResetBoundary()`).

## 6. Implementation Notes

- Extend the renderer with a localized recovery helper (e.g., `renderWithBoundary`)
  that wraps child renders in `withRecovery` but stops the panic from bubbling
  up to the session.
- Boundary state lives in a hook cell (similar to `UseState`) tracking the
  current `ErrorInfo`.
- Ensure boundaries integrate with keyed renders so resetting via `h.Key` works
  identically to keyed components.

## 7. Future Enhancements

- Allow boundaries to report custom metadata (e.g., `boundaryName`).
- Support hierarchical boundaries (outer boundary sees fallback of inner boundary).
- Provide helpers to reset all boundaries under a subtree (for global “retry”).
