# PondLive · Go Runtime

PondLive is a server‑driven UI runtime for Go. It renders components on the
server, streams DOM diffs to the browser over PondSocket, and keeps client and
server state in sync without client-side Virtual DOMs or hydration. This
repository contains the Go runtime, the embedded router, a WebSocket transport,
and the TypeScript client bundle that replays patches in the browser.

## Highlights

- **Server-rendered components** written in idiomatic Go functions.
- **Zero-hydration diffing:** the runtime computes granular DOM patches and
  applies them on the client.
- **First-class router** with SSR, navigation history, and metadata helpers.
- **Pub/Sub integration:** component hooks publish and receive messages across
  sessions via PondSocket or custom backends.
- **Single bundled client** (`/pondlive.js`) embedded in the server for easy
  deployment.

## Repository layout

```
client/           # TypeScript source and build scripts for pondlive.js
examples/         # End-to-end examples (counter, Tailwind, etc.)
internal/         # Runtime, renderer, server adapters, and supporting packages
pkg/live/         # Public API surface (hooks, contexts, HTML DSL)
pkg/live/server/  # Application bootstrap, HTTP manager, embedded assets
```

## Requirements

- Go 1.24+
- Node.js 20+ (for rebuilding the browser client)

## Quick start

1. **Build the client bundle** (once, or whenever you change `client/src`):

   ```bash
   cd client
   npm install
   npm run build:prod   # writes pkg/live/server/static/pondlive.js
   ```

2. **Run the Tailwind counter example:**

   ```bash
   cd examples/counter
   go run .
   ```

   Visit <http://localhost:8080>; the page is server-rendered and stays live via
   PondSocket diff streaming.

## Building your own app

```go
package main

import (
    "context"
    "log"
    "net/http"

    live "github.com/eleven-am/go/pondlive/pkg/live"
    "github.com/eleven-am/go/pondlive/pkg/live/server"
    h "github.com/eleven-am/go/pondlive/pkg/live/html"
)

func Counter(ctx live.Ctx, _ struct{}) h.Node {
    count, setCount := live.UseState(ctx, 0)
    return h.Div(
        h.Button(
            h.Text("Clicked "),
            h.Textf("%d", count()),
            h.Text(" times"),
            h.On("click", func() { setCount(count() + 1) }),
        ),
    )
}

func main() {
    app, err := server.NewApp(context.Background(), Counter)
    if err != nil {
        log.Fatal(err)
    }
    log.Fatal(http.ListenAndServe(":8080", app.Handler()))
}
```

The server automatically serves the bundled client at `/pondlive.js` and mounts
the PondSocket endpoint that keeps sessions synchronized.

## Pub/Sub helpers

- `live.UsePubsub` subscribes a component to a topic. Publishing from inside a
  component uses the same hook handle.
- `live.NewPubsub(topic, nil, ...)` creates a helper outside a component. When
  the app boots it registers the transport-backed publisher as the default, so
  `Publish` works even when you omit a custom backend. Provide your own
  `live.PubsubPublisher` to integrate Redis, NATS, etc.

## Development tasks

- Run tests: `go test ./...`
- Rebuild client bundle: `cd client && npm run build:prod`
- Run example smoke tests: `go test ./examples/...`

## Related packages

- `pkg/live/router`: declarative routing, navigation effects, metadata tools.
- `pkg/live/html`: a minimal HTML DSL used by components.
- `internal/server/http`: SSR HTTP manager that builds the boot payload.
- `internal/server/pondsocket`: PondSocket endpoint integration and transports.

## Contributing

Issues and pull requests are welcome. Please run `go test ./...` and rebuild the
client bundle before submitting changes.

---

PondLive is maintained by Eleven AM. Refer to the repository root for licensing
details and additional documentation.

