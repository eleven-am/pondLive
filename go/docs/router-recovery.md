# Router Recovery & Reset Flow

When router hydration fails (e.g., missing DOM range, binding mismatch) we want
to reset just the affected subtree instead of reconnecting the entire session.
This document outlines the diagnostic-triggered router reset handshake.

## 1. When to Reset

Router hydration diagnostics (see `docs/diagnostics.md`) include the component
ID/path that failed. Typical scenarios:

- Component boot template couldn’t find the DOM range (`componentId` missing).
- Router binding `{componentId,path}` didn’t resolve to a DOM node.
- Slot/list anchor mismatch inside a router outlet.

In dev mode the client overlay can offer a “Reset route”/“Retry component”
action when these diagnostics appear.

## 2. Control Messages

### 2.1 Client → Server: Router Reset Request

```jsonc
{
  "t": "routerReset",
  "sid": "session-id",
  "componentId": "c123"
}
```

- `componentId` identifies the router subtree to reset (usually the outlet
  component ID that failed to hydrate).
- The client sends this when the developer clicks “Reset component” or
  automatically when repeated hydration attempts fail.

### 2.2 Server → Client: Template Replacement

- Server responds with a `t:"template"` frame scoped to `componentId`:
  ```jsonc
  {
    "t": "template",
    "sid": "session-id",
    "scope": {
      "componentId": "c123",
      "parentId": "c045",
      "parentPath": [1, 0]
    },
    "templateHash": "sha256:deadbeef",
    "s": [...],
    "d": [...],
    "bindings": { ... }
  }
  ```
- Client applies the template per the hydration flow (splice DOM, rehydrate
  bindings). If successful, the diagnostic resolves; if not, the overlay can
  escalate (suggest full reload).

### 2.3 Failure Responses

- If the server no longer knows the component (e.g., session expired), it can
  send a `diagnostic` frame (`code: "router_reset_failed"`) instructing the
  client to reload.
- Optionally the server can acknowledge the reset request with a `routerResetAck`
  containing status (`pending`, `failed`, etc.). For a first implementation the
  template frame alone is sufficient.

## 3. Client Behaviour

1. Diagnostic logged (component/path). Overlay shows “Reset component” button.
2. Button sends `{t:"routerReset", componentId}`. While waiting, the overlay can
   show “Resetting…” state.
3. When the template frame arrives, hydrate it. On success, clear the diagnostic
   entry. On failure, update the diagnostic (e.g., “Reset failed; reload page.”).
4. Rate-limit automatic resets to avoid loops.

## 4. Server Behaviour

1. Validate the component ID (ensure it belongs to a router subtree).
2. Trigger a component boot for that component (similar to router navigation) to
   generate a fresh template.
3. Send the template frame to the client (and optionally a `diagnostic` frame if
   reset fails).

## 5. Future Enhancements

- Support resetting multiple components at once (e.g., entire router outlet
  stack).
- Integrate with `live.ErrorBoundary` so a boundary can automatically request a
  router reset when it catches a panic.
- Add telemetry for resets (how often they happen, success rate).
