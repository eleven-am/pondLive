# Error & Diagnostic Pipeline

This document explains how PondLive captures runtime errors, ships diagnostics
to clients, surfaces them in dev tools, and ties metrics/logging together. It
covers panic recovery, `DiagnosticReporter`, `ServerError` payloads, and the
client overlay.

## 1. Server-Side Stages

### 1.1 Panic Recovery (`ComponentSession.withRecovery`)

- Every render/flush/event-handling path runs inside `withRecovery(phase, fn)`.
- If `fn` panics:
  1. `handlePanic` builds a `Diagnostic` capturing phase, component ID/name,
     message, panic type, stack, hook info (for hook mismatch), metadata, and
     timestamp.
  2. Session state flips to `errored`; pending ops/effects/nav are cleared.
  3. `DiagnosticReporter` (if installed) receives the diagnostic (LiveSession
     implements this).
  4. `handlePanic` returns `DiagnosticError`, so the caller can propagate it.

### 1.2 Diagnostic Structure (`internal/runtime/diagnostics.go`)

```go
type Diagnostic struct {
    Code, Phase, ComponentID, ComponentName string
    Message, Hook, Suggestion               string
    HookIndex                               int
    Stack, Panic                            string
    Metadata                                map[string]any
    CapturedAt                              time.Time
}
```

- `normalizeDiagnosticCode` builds a slug from the phase (e.g., `render_panic`).
- `Metadata` can include extra context (e.g., panic type, hook mismatch info,
  stream key collisions).
- `Diagnostic.AsError()` wraps it as `DiagnosticError` so normal error paths can
  return diagnostics.

### 1.3 Reporting (`DiagnosticReporter`, `LiveSession.ReportDiagnostic`)

- `ComponentSession` exposes `SetDiagnosticReporter`. LiveSession installs
  itself to:
  1. Append diagnostics to a rolling buffer (`defaultDiagnosticHistory` = 32).
  2. When `devMode` is true, stream each diagnostic immediately via a
     `{"t":"diagnostic"}` frame so dev tools/overlays can react in real time.
     The runtime augments diagnostics with `metadata.componentScope`
     (`componentId`, `parentId`, `parentPath`, `firstChild`, `lastChild`) so
     clients know exactly which subtree failed and can offer scoped recovery
     actions (router reset, boundary retry, etc.).
  3. For fatal diagnostics, send a `ServerError` (`transport.SendServerError`)
     so existing clients handle them consistently.
  4. Include buffered diagnostics in future `init`/`resume` frames so clients
     catch up after reconnects.
- `LiveSession.flushAsync` captures unhandled errors during async flushes and
  reports them the same way.

### 1.4 Metrics

- `Frame.metrics` includes render/op/effect timings. Slow effects can hint at
  hooks causing issues even if they don’t panic. Diagnostics may include
  `Metadata["renderMs"]` or similar if desired (future enhancement).

## 2. Client-Side Handling (Dev Mode)

1. **`handleDiagnostic`** consumes `{t:"diagnostic"}` frames in dev builds:
   - Logs via `console.warn` with the human-readable message emitted by the
     server.
   - Emits a `"diagnostic"` event (`live.on("diagnostic", handler)`) that
     carries a normalized `DiagnosticMessage`, allowing overlays and devtools to
     respond without poking at internal state.
   - Calls `recordDiagnostic`.
2. **`handleError`** still processes fatal `ServerError` frames:
   - Logs via `console.error`.
   - Emits an `"error"` event.
   - Calls `recordDiagnostic`.
3. **Diagnostics buffer** – Up to 20 entries (deduped by code/message/component
   etc.). Each entry includes timestamp, severity (info/warn/error), message,
   and `ErrorDetails`.
4. **Overlay** – When `options.debug` is true, `recordDiagnostic` triggers
   `renderErrorOverlay`, which highlights diagnostics by severity and uses the
   streamed metadata (component IDs, scope, suggestion) to offer contextual
   actions:
   - View metadata/stack traces to understand the failure.
   - Click **Reset component** (visible when `componentId` is present) to send a
     `{t:"routerReset", componentId}` control message that refreshes only the
     affected outlet subtree.
   - Use **Retry render** / **Clear** / **Close** controls for broader recovery.
5. **Resume handshake** – On reconnect, buffered diagnostics from
   `defaultDiagnosticHistory` are replayed so developers see what happened
   while disconnected.

## 3. ErrorDetails Payload (Server → Client)

`Diagnostic.ToErrorDetails()` populates:

| Field | Description |
| --- | --- |
| `phase` | Phase string passed to `withRecovery` (e.g., `render`, `event:h7`). |
| `componentId`, `componentName` | Identify failing component. |
| `hook`, `hookIndex` | For hook mismatch diagnostics. |
| `suggestion` | Human-readable fix hint (e.g., “Ensure hooks run in same order”). |
| `stack` | Stack trace captured via `debug.Stack()`. |
| `panic` | Stringified panic value. |
| `capturedAt` | RFC3339 timestamp. |
| `metadata` | Arbitrary context (panic type, handler ID, etc.). |

`protocol.ServerError` wraps these details with `code`/`message`.

## 4. Additional Sources of Diagnostics

- **Hook misuse** – `panicHookMismatch` implements `hookCarrier`, so diagnostics
  include hook metadata and suggestions.
- **Stream hook misuse** – `UseStream` panics on duplicate keys or invalid row
  nodes; metadata includes the offending key.
- **Upload/pubsub/nav** – When these subsystems encounter unrecoverable errors,
  they can call `sess.ReportDiagnostic(...)` directly (e.g., invalid upload
  message).
- **Async tasks** – When `flushAsync` or DOM request handlers error, they use
  `ReportDiagnostic` to include context (phase `async_flush`, metadata with the
  error).

## 5. Developer Workflow

1. Enable `options.debug` (dev mode) to show the overlay, subscribe to
   `diagnostic` events, and keep diagnostic history.
2. Inspect console warnings/errors (warnings originate from `diagnostic`
   frames, errors from fatal `ServerError` frames).
3. Use the overlay to drill into component metadata (phase/component/hook) or
   click through to DOM highlights when available.
4. On the server, monitor diagnostic streams (`ReportDiagnostic` logs) or wire
   `DiagnosticReporter` into observability pipelines; the metadata now always
   includes the failing component scope for easier filtering.

## 6. Improvement Ideas

- **Filter/Throttle** – implement rate limiting or grouping so repeated
  diagnostics (e.g., noisy effects) don’t flood the overlay/logs.
- **Metrics + diagnostics integration** – surface per-effect samples with
  component IDs when `slowEffects` > 0.
- **Source maps** – attach hashed component/function names or file/line info to
  diagnostics (requires build-time metadata) for easier navigation.
- **Production routing** – forward diagnostics to server logs in prod while
  keeping client streaming dev-only.
- **Component error boundaries** – explore per-component boundaries so nested
  panics can render fallback UI locally while still emitting diagnostics.

By understanding this pipeline, you can safely modify panic handling, metrics,
or logging without breaking developer tooling. Any new error source should
either panic+recover through `withRecovery` or call `ReportDiagnostic` so the
client experience stays consistent.
