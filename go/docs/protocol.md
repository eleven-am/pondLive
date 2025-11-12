# PondLive Protocol & Diff Reference

This document defines the wire protocol between PondLive servers and clients.
It now includes unified template payloads (`t:"boot"`, `t:"init"`, `t:"template"`)
for boot/init/component boot, incremental diff frames, binding metadata
(handlers/router/uploads/refs), diff ops, navigation, effects, metrics, and
auxiliary messages.

## 1. Message Types

| Type | Direction | Purpose |
| --- | --- | --- |
| `boot` (`protocol.Boot`) | Server → Client (HTTP) | Initial SSR payload (HTML + template metadata) embedded in the page before the socket connects. |
| `init` (`protocol.Init`) | Server → Client (socket) | Full template snapshot sent after join (or reconnect). |
| `template` (`protocol.TemplateFrame`) | Server → Client (socket) | Subtree template replacement (“component boot”) using the same schema as boot/init. |
| `frame` (`protocol.Frame`) | Server → Client | Incremental diff: patch ops, slot delta, binding/handler/ref changes, nav commands, effects, metrics. |
| `diagnostic` (`protocol.Diagnostic`) | Server → Client (dev) | Streams non-fatal diagnostics (warnings/info) so devtools/overlays can react in real time. |
| `resume` (`protocol.ResumeOK`) | Server → Client | Confirms buffered frames after reconnect (seq range). |
| `error` (`protocol.ServerError`) | Server → Client | Structured runtime errors (hook mismatch, render panic, etc.). |
| `pubsub`, `upload`, `domreq` | Server → Client | Subsystem-specific controls (pubsub join/leave, upload commands, DOM property fetches). |
| `join`, `resume`, `ack`, `event`, `nav`, `pop`, `domres` | Client → Server | Connection lifecycle, event dispatch, navigation, DOM responses. |

All socket messages are JSON objects. `sid` (session ID) and `ver` (session
version) identify the live session.

## 2. Template Payloads (Boot, Init, Component Boot)

Every template delivery uses the same shape. Fields marked *optional* apply only
in certain contexts (e.g., `scope` for component boot).

```jsonc
{
  "t": "boot" | "init" | "template",
  "sid": "session-id",
  "ver": 3,
  "seq": 12,                 // init only
  "scope": {                 // component boot/template frames only
    "componentId": "c123",
    "parentId": "c045",
    "parentPath": [0, 2]
  },
  "html": "<div>...</div>",  // boot usually includes HTML; init/template may omit
  "templateHash": "sha256:abc123", // optional cache key
  "s": ["<div>", "</div>"],
  "d": [{ "kind": "text", "text": "Count" }],
  "slots": [{ "anchorId": 0 }],
  "slotPaths": [...],
  "listPaths": [...],
  "componentPaths": [...],
  "handlers": { "h7": { "event": "click", "listen": ["pointerdown"], "props": ["target.value"] }, "ref:3": { ... } },
  "bindings": {
      "slots": {
        "12": [{ "event": "click", "handler": "h7", "listen": [], "props": [] }]
      },
    "statics": [
      {
        "componentId": "c123",
        "path": [3,0],
        "events": [{ "event": "submit", "handler": "h8" }]
      }
    ],
    "router": [
      {
        "componentId": "c123",
        "path": [1],
        "meta": { "path": "/settings/security", "query": "", "hash": "", "replace": false }
      }
    ],
    "uploads": [
      {
        "componentId": "c123",
        "path": [4,0],
        "uploadId": "u1",
        "accept": ["image/png"],
        "multiple": false,
        "maxSize": 5242880
      }
    ],
    "attachments": [
      {
        "componentId": "c123",
        "path": [2,0],
        "type": "ref",
        "refId": "ref:3"
      }
    ],
    "refs": [
      { "componentId": "c123", "path": [2,0], "refId": "ref:3" }
    ]
  },
  "refs": {
    "add": {
      "ref:3": {
        "tag": "video",
        "events": { "timeupdate": { "listen": ["play","pause"], "props": ["target.currentTime"] } }
      }
    }
  },
  "location": { "path": "/todos", "q": "", "hash": "" }, // boot/init only
  "client": { "endpoint": "...", "upload": "...", "debug": true },
  "errors": [] // optional pending diagnostics
}
```

- `scope` tells the client where to splice component boot templates (component
  ID plus parent info). Omit for root boot/init.
- `html` may be omitted for init/template frames if statics/dynamics alone are
  sufficient to reconstruct DOM. Boot typically includes HTML for initial SSR.
- `bindings` replaces all DOM `data-*` metadata:
  - `slots`: bindings for dynamic attr slots (slot ID → handlers).
  - `statics`: handlers bound to static elements addressed by
    `{componentId,path}`.
  - `router`: router metadata (path/query/hash/replace) for link interception.
  - `uploads`: upload slot IDs/config attached to inputs.
  - `attachments`: generic hook attachments (e.g., element refs) keyed by
    `{componentId,path}`. For refs this pairs with `refs.add`.
- `refs.add` supplies ref metadata (tag + event selectors). Combined with
  `bindings.attachments`, the client can link refs to DOM nodes without
  attributes.
- `handlers` map handler/ref IDs to metadata (event/listen/props). Handler IDs
  include both `h*` entries and `ref:*` entries for ref listeners.
- `refs.add` supplies ref metadata (tag + event selectors). Combined with
  `bindings.refs`, the client can link refs to DOM nodes without attributes.

## 3. Diff Operations (`diff.Op`)

Frame `patch` arrays contain operations serialized as tuples:

| Op | Array form | Meaning |
| --- | --- | --- |
| `SetText{Slot,Text}` | `["setText", slot, text]` | Replace the text content of a dynamic text slot. |
| `SetAttrs{Slot,Upsert,Remove}` | `["setAttrs", slot, {k:v}, ["attr"]]` | Upsert/remove attributes for a dynamic attr slot. |
| `List{Slot,Ops}` | `["list", slot, ...ops]` | Apply child mutations to a keyed list slot. |
| `Ins{Pos,Row}` | `["ins", pos, { "key": "...", "html": "...", "slots": [...], "bindings": {...} }]` | Insert a row at position with pre-rendered HTML and nested slot bindings. |
| `Del{Key}` | `["del", key]` | Remove row by key. |
| `Mov{From,To}` | `["mov", from, to]` | Reorder row by index. |
| `Set{Key,SubSlot,Value}` | `["set", key, slot, value]` | Update a nested slot inside a row (rare). |

Slots refer to indices in the `slots` array from the latest template payload.
Row payloads optionally include nested slot IDs and binding tables for keyed
components rendered inside lists.

## 4. Slot & Path Metadata

- **SlotMeta** (`slots[]`): enumerates dynamic anchors (`anchorId`).
- **SlotPath**: maps slot ID → `{componentId, elementPath, textChildIndex}`.
- **ListPath**: same for list containers.
- **ComponentPath**: describes component subtree ranges (`parentId`,
  `parentPath`, `firstChild`, `lastChild`), enabling DOM range mapping even when
  fragments flatten children.

Clients resolve these paths to DOM nodes before applying diffs.

## 5. Binding & Handler Deltas

Frame `handlers` field carries incremental handler changes:

```jsonc
"handlers": {
  "add": {
    "h7": { "event": "click", "listen": ["pointerdown"], "props": ["target.value"] }
  },
  "del": ["h2"]
}
```

- `add` supplies metadata for new handler IDs (including refs).
- `del` removes handler IDs no longer referenced.

Frames may also include incremental binding updates (optional):

```jsonc
"bindings": {
  "slots": { "12": [ ... ] },
  "statics": [ { "componentId": "c123", "path": [3], "events": [...] } ],
  "router": [ ... ],
  "uploads": [ ... ],
  "refs": [ ... ]
}
```

If omitted, bindings remain as last templated.

## 6. Ref Delta

`refs` field mirrors handler deltas:

```jsonc
"refs": {
  "add": {
    "ref:3": { "tag": "video", "events": { "timeupdate": { "listen": ["play"], "props": ["target.currentTime"] } } }
  },
  "del": ["ref:1"]
}
```

Clients update their ref registry accordingly, mapping ref IDs to DOM nodes via
`bindings.refs`.

## 7. Frame Delta & Statics

- `delta.statics` (bool) indicates whether statics changed; if true, the client
  should reload `s` (rare, used for template resets).
- `delta.slots` can be `null` (no change), a full replacement, or a sparse map
  describing slot changes (reserved for future incremental slot updates).

## 8. Navigation & History

- `nav` field includes `{ "push": "/path" }` or `{ "replace": "/path" }` when
  the server wants the client to update browser history.
- Clients send `nav`/`pop` messages when intercepting link clicks or handling
  `popstate`.
- Router metadata now lives in `bindings.router`, removing `data-router-*`
  attributes.

## 9. Effects

`effects` array carries post-render actions. Built-in kinds:

| Effect | Payload | Behaviour |
| --- | --- | --- |
| `Focus` | `{ "type": "focus", "selector": ... }` | Focus a DOM node. |
| `Toast`, `Push`, `Replace`, `ScrollTop` | ... | Miscellaneous UI/history commands. |
| `dom.*` actions | see `docs/dom-actions.md` | Structured DOM commands (`dom.call`, `dom.set`, `dom.class`, `dom.scroll`, etc.). |

Custom effect payloads may be queued as needed (client should ignore unknown
types).

## 10. Metrics

`metrics` summarizes server-side work per frame:

- `renderMs` – time spent rendering/diffing.
- `ops` – number of diff ops in the frame.
- `effectsMs`, `maxEffectMs`, `slowEffects` – effect execution timings.

Useful for dev tooling; clients can surface these values in dashboards.

## 11. Other Messages

- **Client Event (`t:"evt"`)** – includes handler ID `hid`, client seq `cseq`,
  and `payload` (DOM props, form data). Server replies with `ack`.
- **Client Ack (`t:"ack"`)** – acknowledges frames up to `seq`.
- **DOM Request / Response** – server requests DOM properties via `domreq`
  (used by `DOMGet`). Client responds with `domres`.
- **Pubsub Control** – instructs the client to join/leave transport topics.
- **Upload Control** – server-initiated upload cancellation (`{op:"cancel"}`).
- **Diagnostic (`t:"diagnostic"`)** – dev-mode frame streaming non-fatal issues:
  ```jsonc
  {
    "t": "diagnostic",
    "sid": "session-id",
    "code": "router_hydration_warning",
    "message": "Failed to hydrate router binding",
    "details": {
      "componentId": "c123",
      "componentName": "users.RouterOutlet",
      "phase": "template_hydrate",
      "metadata": {
        "path": [1, 0],
        "binding": "router",
        "componentScope": {
          "componentId": "c123",
          "parentId": "c045",
          "parentPath": [1, 0]
        }
      },
      "stack": "",
      "capturedAt": "2024-04-05T12:34:56Z"
    }
  }
  ```
  `metadata.componentScope` mirrors the template scope for the failing component
  so the client can highlight and reset only that subtree.

## 12. Compatibility Contracts

1. **Slot IDs & Paths** – Stable for the life of a component instance. Template
   payloads (boot/init/template) define the authoritative mapping.
2. **Handler IDs** – Strings (`h*`, `ref:*`). Valid until listed in `handlers.del`.
3. **Refs** – Each `ref:*` maps to exactly one DOM node at a time. Clients must
   update bindings when templates/diffs replace DOM nodes.
4. **Frame ordering** – Frames carry monotonically increasing `seq`. Apply in
   order; buffer/drop out-of-order frames.
5. **Reconnect** – After reconnect, server sends `init` or `resume` with the
   authoritative template + any pending diagnostics. Clients replace local
   state before applying new diffs.

## 13. Future Work

- Incremental binding deltas in frames (slots/statics/router/uploads/refs) to
  avoid resending full tables.
- Optional template streaming (statics first, bindings later) for very large
  trees.
- Protocol versioning for handler/ref metadata to ease backwards compatibility.

This reference should stay in sync with `internal/protocol`, diff encoding, and
client `types.ts`. Any changes to template/binding formats must update this
doc.
