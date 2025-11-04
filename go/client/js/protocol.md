# PondSocket LiveUI client protocol

This document summarizes the JSON messages exchanged between the LiveUI browser client and the Go server when using PondSocket.

## Outbound messages (client → server)

| Type | Shape | Notes |
| ---- | ----- | ----- |
| `evt` | `{ "t":"evt", "hid":"<handlerId>", "payload":{}, "seq": <number> }` | Fired for DOM events wired to server handlers. |
| `nav` | `{ "t":"nav", "path":"/users/42", "q":"tab=a", "seq": <number> }` | Declarative navigation that pushes history. |
| `pop` | `{ "t":"pop", "path":"/users/41", "q":"", "seq": <number> }` | Browser back/forward notification. |
| `ack` | `{ "t":"ack", "seq": <number> }` | Optional flow-control acknowledgement. |

`seq` monotonically increases per connection and allows the server to correlate responses.

## Inbound messages (server → client)

| Type | Shape | Notes |
| ---- | ----- | ----- |
| `init` | `{ "t":"init", "sid":"<sessionId>", "ver": <number>, "s":[...], "d":[...], "slots":[...], "handlers":{...}, "location":{...} }` | Hydration payload sent once after the websocket is established. |
| `frame` | `{ "t":"frame", "sid":"<sessionId>", "seq": <number>, "ver": <number>, "delta":{...}, "patch":[...], "effects":[...], "nav":null, "handlers":{...}, "metrics":{...} }` | Batched diff and side-effects applied after hydration. |

Application order on the client is **patch → effects → nav**, ensuring DOM updates precede push/replace calls.

## Sequence and acknowledgement

Servers may request acknowledgements by emitting frames with a `seq` value. Clients can respond with `ack` messages when work completes to support adaptive throttling.
