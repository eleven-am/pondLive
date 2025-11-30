# PondLive

PondLive is a server-driven UI toolkit for Go. Inspired by React and Phoenix LiveView, it keeps component logic on the server while streaming HTML diffs to the browser over PondSocket-powered WebSockets. The PondSocket transport also exposes handler-level routing primitives similar to Phoenix, so you can build interactive pages and navigation flows without duplicating logic in the browser.

## Installation

```bash
go get github.com/eleven-am/pondlive/go
```

## Minimal Application

Every Living UI starts with a root component and a `server.App`. The snippet below wires a counter into the default HTTP stack and serves both SSR HTML and the PondSocket websocket endpoint.

```go
package main

import (
    "context"
    "log"
    "net/http"

    ui "github.com/eleven-am/pondlive/go/pkg/live"
    h "github.com/eleven-am/pondlive/go/pkg/live/pkg"
    "github.com/eleven-am/pondlive/go/pkg/live/router"
    "github.com/eleven-am/pondlive/go/pkg/live/server"
)

func Counter(ctx ui.Ctx) h.Node {
    count, setCount := ui.UseState(ctx, 0)

    return router.Router(ctx,
        h.Div(
            h.H1(h.Text("Counter")),
            h.Button(
                h.On("click", func(h.Event) h.Updates {
                    setCount(count() + 1)
                    return nil
                }),
                h.Textf("Count: %d", count()),
            ),
        ),
    )
}

func main() {
    app, err := server.NewApp(context.Background(), Counter)
    if err != nil {
        log.Fatal(err)
    }

    log.Println("listening on http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", app.Handler()))
}
```

When a browser requests the page it receives server-rendered HTML. As soon as the embedded `/pondlive.js` boot script connects over PondSocket, the server streams DOM patches for state changes instead of shipping full page reloads.

## Component Model

Live components are just Go functions; the runtime tracks hook usage and diffable HTML to keep the API familiar if you already know React.

- Components are plain functions `func(ctx ui.Ctx) h.Node` (or `ui.Component[P]` for typed props).
- Hooks (`UseState`, `UseEffect`, `UseMemo`, `UseRef`, `UseSelect`) mirror React semantics: call them in a consistent order per render. Setters schedule rerenders for the owning component only.
- `ui.Render` renders child components manually; wrap with `ui.WithKey` when rendering collections.
- `ui.WithoutRender` batches setter calls; `ui.NoRender` clears pending rerenders if you handled updates manually.

```go
func TodoList(ctx ui.Ctx) h.Node {
    todos, setTodos := ui.UseState(ctx, []string{})

    onSubmit := func(ev h.Event) h.Updates {
        text := ev.Form["item"]
        if text != "" {
            setTodos(append(todos(), text))
        }
        return nil
    }

    return h.Form(
        h.On("submit", onSubmit),
        h.Ul(h.Map(todos(), func(item string) h.Node {
            return h.Li(h.Text(item))
        })),
        h.Input(h.Name("item")),
        h.Button(h.Attr("type", "submit"), h.Text("Add")),
    )
}
```

`h.Event` exposes `Value`, `Form`, `Payload`, and modifier state. Returning `nil` lets state changes decide rerenders; return `h.Rerender()` when you mutate external state (files, caches) and still need a diff pushed to the client.

## HTML Builder

`pkg/live/html` provides constructors for HTML nodes:

- Elements are typed values, so you can compose markup without string templates and still rely on the runtime to diff efficiently.
- Tags like `h.Div(...)`, `h.Button(...)`, `h.Input(...)` generate `*html.Element` values.
- Text helpers (`h.Text`, `h.Textf`) and `h.Fragment` build static content.
- Attribute helpers (`h.Attr`, `h.Class`, `h.Style`, `h.Key`, `h.UnsafeHTML`) and event binding via `h.On`.
- `h.Map` and `h.MapIdx` expand slices into node lists; pair with `h.Key` or `ui.WithKey` for deterministic diffing.
- Conditional helpers `h.If` and `h.IfFn` include optional items without cluttering render logic.

## File Uploads

`live.UseUpload(ctx)` registers an upload slot for the component and returns a handle you attach to an `<input type="file">`.

```go
upload := live.UseUpload(ctx)
upload.OnComplete(func(file live.UploadedFile) h.Updates {
    // move file.TempPath to permanent storage
    return h.Rerender()
})

inputProps := upload.BindInput(h.Type("file"))
progress := upload.Progress()
```

Optional callbacks such as `OnChange` (receives `live.FileMeta`) and `OnError` let you react to selection or failures. Use
`Accept`, `AllowMultiple`, or `MaxSize` to mirror input attributes on the client while enforcing limits server-side. The
`Progress()` snapshot exposes bytes loaded, total size, and `Status` values like `UploadStatusUploading` or
`UploadStatusComplete` for rendering progress indicators.

## Context and Derived State

Use typed contexts to share state across the tree; providers export setters so you can build scoped stores without global variables:

```go
var ThemeCtx = ui.NewContext("light")

func ThemeProvider(ctx ui.Ctx, mode string, render func() h.Node) h.Node {
    return ThemeCtx.Provide(ctx, mode, render)
}

func ThemeToggle(ctx ui.Ctx) h.Node {
    mode := ThemeCtx.Use(ctx)
    return h.Div(h.Text("theme: " + mode))
}
```

Pass a render closure so the runtime activates the provider before descendants render. `Context.UsePair` exposes a getter/setter, and `ui.UseSelect` lets you project fields while controlling equality checks. Combine these with hooks to centralize cross-cutting data such as auth or feature flags.

## Routing

`router.Router` keeps the URL in sync with component state and exposes LiveView-style navigation helpers. Wrap your root once and describe route trees with `router.Routes`/`router.Route`:

```go
func App(ctx ui.Ctx) h.Node {
    return router.Router(ctx,
        router.Routes(ctx,
            router.Route(ctx, router.RouteProps{Path: "/", Component: Home}),
            router.Route(ctx, router.RouteProps{Path: "/users/:id", Component: User}),
        ),
    )
}

func User(ctx ui.Ctx, match router.Match) h.Node {
    id := match.Params["id"]
    return h.Div(h.Text("user " + id))
}
```

Key APIs:

- `router.Match` delivers `Params`, `Query`, and the matched `Path` to components.
- `router.Outlet` renders nested child routes.
- `router.Link` renders a client-side navigation anchor. `router.Navigate` / `router.Replace` push history imperatively; `NavigateWithSearch` and `ReplaceWithSearch` mutate query strings in place.
- `router.UseLocation`, `router.UseParams`, and `router.UseSearch` expose the current URL. `router.UseSearchParam` returns getter/setter functions for a single query key.
- `router.Redirect` triggers a history replace inside a render pass.
- `router.UseMetadata` merges `ui.Meta` into the document head on navigation.

All router hooks require `router.Router` in the tree; otherwise they panic with `router.ErrMissingRouter`.

## Document Metadata

`ui.Meta` captures `<title>`, description, and head tags. Combine helpers as needed to keep layouts declarative:

```go
func Layout(ctx ui.Ctx, body h.Node) h.Node {
    base := &ui.Meta{
        Title: "PondLive app",
        Links: h.LinkTags(h.LinkTag{Rel: "stylesheet", Href: "/styles.css"}),
    }
    return ui.WithMetadata(body, base)
}
```

Nested components can merge metadata with `ui.MergeMeta` or call `router.UseMetadata` to update the session’s current head state.

## Pub/Sub and Multi-Session Updates

`ui.NewPubsub` couples `UsePubsub` with a publisher so you can add multi-session state without wiring Redis or other transports first:

```go
type Message struct{ Text string }

var chatTopic = ui.NewPubsub[Message]("chat", nil)

func Chat(ctx ui.Ctx) h.Node {
    handle := chatTopic.Use(ctx)

    send := func(text string) {
        _ = chatTopic.Publish(context.Background(), Message{Text: text}, nil)
    }

    latest, ok := handle.Latest()
    status := "waiting"
    if ok {
        status = latest.Payload.Text
    }

    return h.Div(
        h.P(h.Text("latest: " + status)),
        h.Button(h.On("click", func(h.Event) h.Updates { send("hi"); return nil }), h.Text("Send")),
    )
}
```

When you mount the app through `server.NewApp`, the PondSocket endpoint provides a default pub/sub backend. Supply a custom backend with `ui.WithPubsubProvider` or adapt external systems via `ui.WrapPubsubProvider` when you scale beyond a single process.

`PubsubHandle` exposes `Latest`, `Messages`, `Publish`, and `Connected` for diagnostics.

## Testing Components

`pkg/live/test` exposes an in-memory harness. It renders components through the same scheduler used in production so you can reason about emitted ops and HTML without spinning up a server:

```go
harness := test.NewHarness()
harness.Mount(func() html.Node { return Counter(ctx) })
harness.Click("button#increment", nil)
harness.Flush()
if got := harness.HTML(); !strings.Contains(got, "Count: 1") { t.Fatal(got) }
```

The harness lets you inspect server-rendered HTML, diff ops, and simulate navigation or event payloads without a browser. It is handy for asserting routing flows or hook-driven side effects.

## Server Integration Details

- `server.NewApp` wires SSR, session lifecycle, PondSocket transport, and PondSocket-powered handler-level routing. `App.Handler()` returns a ready-to-use `http.Handler` serving boot HTML, the websocket endpoint, and the embedded `/pondlive.js` client asset.
- Each browser tab receives an isolated `ui.Session`. Access it via `ctx.Session()` for advanced scenarios (e.g., per-session storage, metadata overrides, or pub/sub diagnostics).
- Static files are not served automatically—mount your own file server alongside the Live handler, as shown in `examples/counter`. The pattern mirrors Phoenix: treat Live routes as handlers and plug in whatever HTTP mux you prefer around them.

## Development Workflow

- Build client assets: `cd client && npm run build:prod`
- Run the Tailwind counter example: `cd examples/counter && go run .`
- Execute tests: `go test ./...` (the `Makefile` includes common targets such as `make test` and `make build`).

## Requirements

- Go 1.24 or newer
- Node.js 20+ (only for rebuilding the client bundle)
- A WebSocket-capable environment (PondSocket is embedded by default)

This README focuses on practical usage. Refer to the source if you need deeper runtime behaviour or transport details.
