# PondLive

PondLive is a Go library for building interactive web interfaces entirely in Go. UI components and logic run on the server, while the browser stays responsive.

Traditional SSR sends HTML once; SPAs push most logic into the browser. PondLive keeps state on the server and uses PondSocket to hold a bidirectional WebSocket open. When state changes, it computes a minimal DOM diff and streams just those patches to the client.

## How It Works
1. Define views in Go: functions return HTML nodes (`pkg.Node`). For children/props, wrap with `pkg.Component`/`pkg.PropsComponent`.
2. Initial render: on first request, PondLive renders the component tree to HTML and serves it immediately.
3. Connect: a lightweight JS runtime connects back over PondSocket.
4. Interact: browser events flow to Go handlers over the socket.
5. Re-render & patch: handlers update state; PondLive re-executes the component, diffs against the previous tree, and streams a minimal patch to update the DOM.

## Core Concepts

### Rendering with Components
A component is a Go function that accepts a context and returns a tree of HTML nodes. Wrap reusable components with `pkg.Component` (or `pkg.PropsComponent`) so the runtime tracks state; the root passed to `pkg.NewApp` remains a plain function.

```go
var Card = pkg.Component(func(ctx *pkg.Ctx, children []pkg.Item) pkg.Node {
    return pkg.Div(
        pkg.Class("border p-4 rounded shadow"),
        pkg.Fragment(children...),
    )
})

func App(ctx *pkg.Ctx) pkg.Node {
    return pkg.Div(
        pkg.Class("container mx-auto"),
        Card(ctx, pkg.Text("I am inside a card!")),
    )
}
```

### Server-Side State
State lives in memory on the server for the session. Hooks like `UseState` read and update it; calling the setter triggers a re-render.

```go
count, setCount := pkg.UseState(ctx, 0)
setCount(count + 1) // triggers re-render and patch
```

### Event Handling and DOM Actions
Events can be wired directly with `pkg.On`/`pkg.OnWith` on elements; no JavaScript needed. The runtime forwards the browser event to a Go handler, runs the logic, and patches the DOM. When DOM actions or element data are needed (e.g., `getBoundingClientRect`), use a generated element ref (e.g., `UseDiv`, `UseButton`) that bundles actions.

```go
// Direct handler without a ref
pkg.Button(
    pkg.Text("Click"),
    pkg.On("click", func(evt pkg.Event) pkg.Updates {
        return nil
    }),
)

// DOM actions via generated refs
box := pkg.UseDiv(ctx)    // includes ElementActions
btn := pkg.UseButton(ctx) // includes ButtonActions

btn.OnClick(func(evt pkg.ClickEvent) pkg.Updates {
    rect, err := box.GetBoundingClientRect()
    if err == nil && rect != nil {
        _ = rect.Width // use measurements
    }
    return nil
})

return pkg.Div(
    pkg.Div(pkg.Attach(box), pkg.Text("Measure me")),
    pkg.Button(pkg.Attach(btn), pkg.Text("Get bounds")),
)
```

## Getting Started
- Install: `go get github.com/eleven-am/pondlive`
- Minimal app: create a root component `func(ctx *pkg.Ctx) pkg.Node`, then:
  ```go
  app, _ := pkg.NewApp(Root)
  http.ListenAndServe(":8080", app.Handler())
  ```
- Dev bundle: `pkg.NewApp(Root, pkg.WithDevMode())` serves `/static/pondlive-dev.js`.

## Quick Start (Counter)
```go
package main

import (
    "log"
    "net/http"

    "github.com/eleven-am/pondlive/pkg"
)

func Counter(ctx *pkg.Ctx) pkg.Node {
    count, setCount := pkg.UseState(ctx, 0)

    btn := pkg.UseButton(ctx)
    btn.OnClick(func(evt pkg.ClickEvent) pkg.Updates {
        setCount(count + 1)
        return nil
    })

    return pkg.Div(
        pkg.H1(pkg.Text("Counter")),
        pkg.Button(pkg.Attach(btn), pkg.Text("+")),
        pkg.P(pkg.Textf("Clicked %d times", count)),
    )
}

func main() {
    app, _ := pkg.NewApp(Counter, pkg.WithDevMode())
    log.Println("http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", app.Handler()))
}
```


## Examples
- Run: `cd examples/counter && go run .`
- Other samples: `examples/countdown`, `examples/auth`, `examples/scoped-styles`, `examples/stream-chat`.

## Data Flow & Lifecycle
1. Initial render: request hits the server; the root component renders HTML; response is sent.
2. Connect: the client JS connects via PondSocket at `/live`.
3. Events: browser events travel over the socket to Go handlers.
4. Re-render: state changes trigger re-render; the diff is computed server-side.
5. Patch: minimal patch is streamed back; the DOM updates in place.

## Hooks Overview
- `UseState`: in-memory state per session.
- `UseEffect`: side effects with optional deps and cleanup.
- `UseMemo`: memoized compute by deps.
- Element refs (`UseDiv`, `UseButton`, etc.): stable references plus DOM actions.
- `UseRef`: generic stable ref.
- `UseContext` / `UseProvider`: shared values through the tree.
- `UseSlots` / `UseScopedSlots`: render children/slots.
- `UseScript`: attach client JS and exchange messages.
- `UseHandler`: register HTTP handlers mounted under PondLive.
- `UseUpload`: manage uploads.
- `UseStream`: render streaming data rows.
- `UseStyles`: scoped CSS.
- `UseMetaTags`: set meta tags.
- `UseHeaders`, `UseCookie`: manage response headers/cookies.
- `UseDocument`: document-level settings.
- `UseErrorBoundary`: access error batch for error handling UI.
- `UseHydrated`: runs effect only after WebSocket connection is established.
- `UsePresence`: manage presence animations and timed visibility.

## Routing

PondLive includes a server-side router that handles URL changes without full page reloads. Route components receive both the context and a `Match` object containing route parameters.

```go
func App(ctx *pkg.Ctx) pkg.Node {
    return pkg.Routes(
        ctx,
        pkg.Route(ctx, pkg.RouteProps{Path: "/", Component: Home}),
        pkg.Route(ctx, pkg.RouteProps{Path: "/about", Component: About}),
        pkg.Route(ctx, pkg.RouteProps{Path: "/users/:id", Component: UserProfile}),
    )
}

func Home(ctx *pkg.Ctx, match pkg.Match) pkg.Node {
    return pkg.Div(pkg.Text("Welcome"))
}

func UserProfile(ctx *pkg.Ctx, match pkg.Match) pkg.Node {
    userID, _ := match.Param("id")
    return pkg.Div(pkg.Textf("User: %s", userID))
}
```

## The JavaScript Bridge (UseScript)

Some functionality requires code running in the browser — integrating a map library, managing focus, or running animations. `UseScript` bridges a Go component to a client-side closure.

```go
script := pkg.UseScript(ctx, `
    function(el, transport) {
        const onHighlight = (data) => { el.style.background = data.color }

        // Listen for messages from Go
        transport.on("highlight", onHighlight)

        // Send message to Go
        transport.send("ready", { time: Date.now() })

        // Cleanup when component unmounts
        return () => {
            el.style.background = "" // remove side-effects if desired
        }
    }
`)

// Listen for messages from JS
script.On("ready", func(val any) {
    log.Println("JS is ready:", val)
})

// Send message to JS
script.Send("highlight", map[string]any{"color": "red"})

return pkg.Div(pkg.Attach(script), pkg.Text("I have JS attached"))
```

## Serving
- `app.Handler()` is the HTTP handler.
- PondLive handles `/live` (PondSocket) and serves the client asset at `/static/pondlive.js` (dev variant in dev mode).

## State and Session
- State is per-session, in memory on the server.
- Options: `WithDevMode`, `WithDOMTimeout`, `WithIDGenerator`, `WithContext`, `WithPubSub`.
- Session IDs default to random; can be overridden.

## Styling and Meta
- `UseStyles` for component-scoped CSS.
- `UseMetaTags` for `<title>`/`<meta>` updates.
- `UseHeaders`/`UseCookie` for per-request HTTP headers/cookies.

## Quick Snippets
- Form handling: attach `pkg.On("submit", ...)` on `Form` or use generated form ref actions.
- Redirect during initial render: set redirect in the request state helpers (see session transport in codebase).
- DOM measurements: `div := pkg.UseDiv(ctx); rect, _ := div.GetBoundingClientRect()`.

## Acknowledgments

PondLive is built on [PondSocket](https://github.com/eleven-am/pondsocket), a WebSocket library that provides the bidirectional communication layer between server and client.

## Author

Roy Ossai ([@eleven-am](https://github.com/eleven-am))

## License

MIT (see [LICENSE](LICENSE)).
