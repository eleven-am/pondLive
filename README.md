# PondLive

PondLive is a Go library for building interactive web interfaces entirely in Go. Components and state live on the server. The browser keeps a WebSocket open and applies small, exact edits.

## How It Works

1. **Components run once.** A component body runs when it mounts. It builds a tree and registers signals, handlers and bindings. It does not run again when state changes.
2. **Signals hold state.** A write to a signal runs only the bindings that read it. Each binding sends one edit addressed by node ID, for example "set the text of node 7 to `1`". A click on a counter sends one small `t` edit.
3. **Structure changes in three places.** `Show`, `Switch` and `For` add and remove whole subtrees. Everything else updates in place.
4. **SSR and live use the same emitter.** The first request renders HTML. A script connects to `/live`, and events flow to Go handlers by node ID.
5. **A rebuild re-runs the bodies.** After a connection problem the server re-invokes your components in a serialize pass. Hooks resolve to their existing state, nothing new is created, and the output carries the same IDs as the live page. The rules below exist for that pass.

## Components

A component is declared in one of two forms. `pkg.NewApp`, `pkg.Route` and the row of `pkg.For` are meant to take component values, and a raw `func(*pkg.Ctx) pkg.Node` does not compile there. That is the whole of the compile-time check: a plain `func() pkg.Node` has the same type as the result of `pkg.Component`, so it compiles in `NewApp` and `Route` too. It simply has no ctx, so it cannot use hooks, which is why a `Show` arm or any helper closure cannot call them. Declare anything that needs hooks with `Component` or `PropsComponent`.

- `pkg.Component(func(ctx *pkg.Ctx) pkg.Node)` returns `func() pkg.Node`.
- `pkg.PropsComponent(func(ctx *pkg.Ctx, props P) pkg.Node)` returns `func(P) pkg.Node`.

```go
var Card = pkg.PropsComponent(func(ctx *pkg.Ctx, props CardProps) pkg.Node {
    return pkg.Div(
        pkg.Class("border p-4 rounded shadow"),
        pkg.H2(pkg.Text(props.Title)),
        props.Children,
    )
})

type CardProps struct {
    Title    string
    Children pkg.Node
}

var App = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
    return Card(CardProps{Title: "Hello", Children: pkg.P(pkg.Text("Inside the card"))})
})
```

Slots are gone. Pass nodes as typed props (`Children pkg.Node`). `pkg.Component` is the same without props: `var Footer = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {...})`, used as `Footer()`.

## Signals

```go
count := pkg.UseSignal(ctx, 0)               // *Signal[int]: Get, Peek, Set, Update
double := pkg.UseComputed(ctx, func() int {  // ReadSignal[int]
    return count.Get() * 2
})
pkg.UseEffect(ctx, func() func() {           // runs after the socket connects; reruns when a signal it reads changes
    log.Println("count is", count.Get())
    return nil                                // optional cleanup
})
```

- `Get` reads and tracks. `Peek` reads without tracking.
- `Update(func(T) T)` composes: two updates in one handler both apply.
- `pkg.Const(v)` wraps a constant as a `ReadSignal`.
- `pkg.Derive(src, fn)` is a read-only view of another signal: `Get` and `Peek` return `fn` of the source's value. It is not a hook and has no state of its own, so it can be made anywhere (`req.Header(name)` is one). Use `UseComputed` when you want the result remembered and compared.
- `pkg.RenderHTML(node)` renders any node to a string, with no session (for example an SVG you want as text).
- `pkg.WithEqual(eq)` overrides the equality check that decides whether a write is a change.
- Props are read-only signals when you need them to change: pass a `pkg.ReadSignal[T]` field.

## Hooks

There are 16 `Use*` hooks, plus `Provide`, which publishes a context value that a descendant reads with `UseContext`. The 16 are `UseSignal`, `UseComputed`, `UseEffect`, `UseRenderEffect`, `UseContext`, `UseShared`, `UseBroadcast`, `UseRef`, `UseScript`, `UseStyles`, `UseRouter`, `UseRequest`, `UseHandler`, `UseUpload`, `UseHead` and `UseErrorBoundary`. `UseRef[T]` is the one hook for element refs; `T` is one of seven capability types (see Events and Refs). Whether the browser is connected is `pkg.UseRequest(ctx).Connected()`.

## Effects

Two hooks run your code outside the tree. They differ in when.

**`UseEffect(ctx, fn)` runs after the browser's socket connects.** Use it for timers, subscriptions and I/O: anything that starts work.

- It is auto-tracked: signals read inside `fn` are its dependencies, and `fn` runs again when one changes.
- The function `fn` returns is its cleanup. It runs before the next run and when the component unmounts.
- **Nothing starts before the socket connects.** A plain page request renders the HTML and queues the effects of every mounted component. When the socket first claims the session, the queue runs once, parents before children and siblings in mount order. Components mounted after that run their effects right after they mount. If the socket never connects, the session expires and the queue is discarded, so a crawler or a half-open tab starts no goroutines, timers or queries. Timers made with `ctx.After` and `Broadcast` events follow the same rule.
- `livetest.AssertNothingStartsBeforeConnect(t, App)` (import `github.com/eleven-am/pondlive/pkg/livetest`) renders `App` with one page request and fails the test if an effect ran, a goroutine was started while rendering, or a goroutine survived shutdown.

**`UseRenderEffect(ctx, fn)` runs during render, before that component's HTML is written.** Use it for bookkeeping that the first HTML depends on, such as registering the component with a parent registry (`UseHead` is built on it).

- It runs once at mount and is skipped in every serialize pass. Its returned function runs on unmount. If `fn` read signals, it runs again, after cleanup, when one of them changes.
- It must stay cheap and local. Once the socket has connected, these calls are refused: `Shared.Set` and `Update`, `Broadcast.Send`, `req.Push`, `Replace`, `Back` and `Forward`, router navigation, script `Send`, every call on an element ref (`Focus`, `Play`, `Selection`, ...), and any other post to the session. Before the socket connects, navigation is not refused, because it only shapes the HTTP response: `req.Push` and `req.Replace` answer with a 302 or 303, exactly like `SetCookie`. `pkg.Redirect` is plain public code built on that rule. The refusal covers the render effect's own goroutine; Go has no goroutine ancestry, so a goroutine the effect starts is not inside it. The dev leak detector reports one that is still running when the effect returns (`RENDER_EFFECT_LEAKED_GOROUTINE`). In dev mode a refused call panics with a message naming the call and the component, and the error appears in the overlay. In production the call is dropped and a `RENDER_EFFECT_FORBIDDEN_CALL` warning diagnostic is reported. Reading a ref's live signals is allowed.
- Writing a signal that is already rendered is fine: the page is re-serialized before it is sent. Changing state that is not a signal after it was rendered is caught by the dev mount self-check.
- In dev mode the effect also runs under a profiler label. A goroutine it starts that is still alive afterwards or at connect is reported with its component and creation stack, and an effect that runs longer than 5 ms is reported as slow. These are warnings, not errors: they appear in a dismissible amber panel in the corner, held until the socket connects. Production has no panel.

## The Rules

A component body runs once when it mounts. **It runs again on its own whenever the page has to be rebuilt after a connection problem.** For that reason:

1. **Same inputs, same output.** Given the same props, hook values and context, a body must return the same tree.
   - Do not render `time.Now()`, random numbers, map iteration order, environment or package variables, or data you query inside the body.
   - Load data into a signal (in a handler, a `UseEffect`, or a `Feed` loader) and render the signal.
2. **No side effects in the body.**
   - Do not start goroutines, write to files, databases or the network, mutate package state, or call `Set`/`Update` (a dev error).
   - Do those things in handlers or `UseEffect`.
   - Initial-value expressions such as `UseSignal(ctx, expensive())` are evaluated again on a rebuild, and the result is discarded, so keep them cheap.
3. **Call hooks only at the top of a component body**, unconditionally, with that component's own ctx. Declare every hook as a variable first, then build the tree from those variables. Do not call a hook inside a tree expression (`pkg.Text(pkg.UseComputed(...))`), inside a `Show`/`Switch` arm, or inside a `For` row closure using an outer component's ctx. A row's own `PropsComponent` ctx is its own body, so hooks at the top of the row are fine. Using an outer ctx there is an error in dev mode and a reported diagnostic in production: "a hook was called with the ctx of X after its body finished".
4. **Plain Go variables are fine for handler bookkeeping.**
   - Example: `lastID := 0` with `lastID++` in a handler. The live handlers keep the variable from the first run; the rebuild's copy is never used.
   - **Anything that renders a changing value must be a signal.** That includes text that a region body (`Show`/`For` row) computes from a variable a handler mutates.
5. **`Get()` in the body is a dev error**, because the value would be frozen. Read state in the tree, in `UseComputed`/`UseEffect`, or in handlers. `Peek()` is the explicit untracked read, for values that never change for this owner, such as a feed row's item.
6. **Values are immutable**: replace them, never mutate a slice you passed to `Set`. **Writes from your own goroutine are posted** to the session automatically. **Reads from your own goroutine are a data race** and a dev error.
7. **Shared is context for all sessions.**

Dev mode (`pkg.WithDevMode()`) runs extra determinism checks, including a second serialize pass that must match the first, and reports violations as errors.

## Tree Primitives

```go
pkg.Text(sig)                 // string, number, bool, or a ReadSignal of any of them
pkg.Textf("n = %d", 3)        // static formatted text
pkg.Attr("title", sig)        // attribute; a ReadSignal[bool] controls presence
pkg.Attr("lang", sig, pkg.OmitEmpty()) // absent while the value is empty
pkg.Class("px-2 py-1", sig)   // static strings and ReadSignal[string] parts, merged as Tailwind classes
pkg.Style("color", sig)
pkg.Styles(map[string]any{"color": sig, "order": 3})
pkg.Disabled(isSaving)        // bool or ReadSignal[bool]
```

Attributes, classes and styles accept plain values, signals and computeds. Anything else panics at render with the offending type.

**`Attr` is the one attribute implementation.** `ID`, `Href`, `Src`, `Target`, `Rel`, `Title`, `Alt`, `Type`, `Name`, `Placeholder`, `Data(k, v)` and `Aria(k, v)` are `Attr` with the name filled in. The boolean helpers `Disabled`, `Required`, `Readonly`, `Autofocus`, `Autoplay`, `Controls`, `Loop`, `Muted` and `Multiple` take a `bool` or a `ReadSignal[bool]` (`pkg.Disabled(false)` renders nothing; any other type panics). `Muted` only sets the starting state; the live state is the `Muted` signal of a `MediaRef`.

**`Class` is the one class API.** It takes strings (split on spaces) and `ReadSignal[string]` parts. Whenever the class is evaluated (on the first render and on every update) the combined result goes through [tailwind-merge-go](https://github.com/Oudwins/tailwind-merge-go), so `Class("px-2 py-1", "px-4")` renders `py-1 px-4`, and exact duplicates of your own classes are dropped. A conditional class is a computed that returns the class or `""`; there is no `ClassIf` and no `CN`:

```go
active := pkg.UseSignal(ctx, false)
tone := pkg.UseComputed(ctx, func() string {
    if active.Get() {
        return "bg-red-500"
    }
    return ""
})
return pkg.Div(pkg.Class("p-2 bg-blue-500", tone))
```

The merge follows Tailwind 3.0 to 3.4 (most v4 names merge too; v4 arbitrary-value syntax such as `w-(--x)` is left alone). Classes it does not know are always kept. Merging costs about 33 ns and one allocation when the same string was merged before, and about 15 microseconds when it is new, so a class that produces thousands of different strings is the expensive case. `Attr("class", ...)` and `Attr("style", ...)` are errors in dev mode (use `Class` and `Style`).

**`Styles`** takes a map and calls `Style` for each entry in sorted key order, so the output is deterministic and any value may be a signal. (The scoped stylesheet handle returned by `UseStyles` is `*pkg.ScopedStyles`.)

**One way and two way.** `Attr` and every helper above are one way: the server drives the attribute, and the user cannot change it. State the user can change in the browser is set **only** with `Bind` (below): `value` on `input`/`textarea`/`select`, `checked` on a checkbox or radio, the selection of a `select multiple`, and `open` on `details`/`dialog`. There are no `Value`, `Checked`, `Selected` or `Open` helpers, and `Attr("value" | "checked" | "selected" | "open", ...)` on such an element is an error in dev mode (an `attribute:bind-owned` diagnostic in production) that points to `Bind`. A starting state is a constant: `Bind(true)`, `Bind("hello")`. A checkbox's or radio's `value`, an `option`'s `value` and a hidden input's `value` are plain attributes.

```go
pkg.Show(cond, func() pkg.Node { return pkg.P(pkg.Text("yes")) },
    pkg.Else(func() pkg.Node { return pkg.P(pkg.Text("no")) }))

pkg.Switch(mode,
    pkg.Case("edit", func() pkg.Node { return editor() }),
    pkg.Case("view", func() pkg.Node { return viewer() }),
    pkg.Default(func() pkg.Node { return pkg.Text("?") }))
```

`Show` is a `Switch` on a bool: it shares the one region implementation, so both behave identically (keys, exit, moves). Region bodies are closures, not nodes: a region keeps the closure for later mounts and never retains a tree. A region arm or a `For` row may be any content. One element, component or region is used as it is. Anything else (a `Fragment` of several nodes, text, nothing) is wrapped automatically in its own region, so inserts, moves and removals still address it as one unit. A wrapped unit has no single element to carry `data-exiting`, so on exit every top-level element of the unit gets `data-exiting` and `inert` instead (a component or region at the top level of the unit is not marked, wrap it in an element if you need the hook). A unit with no top-level element stays in the DOM for the duration unstyled and reports a `REGION_EXIT_NO_ELEMENT` warning.

**Exit animations.** `pkg.Exit(d)` is an option for `Show`, `Switch` and `For`. It keeps a removed arm or row in the DOM for `d` with a `data-exiting` attribute, so CSS can animate it:

```css
li[data-exiting] { opacity: 0; transition: opacity 200ms; }
```

A click on an exiting row is dropped. `pkg.WaitExit()` makes a `Switch` wait for the old arm to finish before the new arm mounts.

Static content needs no helper: use a plain Go `if` or `for` around the node you build. `Show` and `For` are for values that change.

## Lists

`For` is the one list primitive. It takes a source and a row function. A row is its own owner, so it can use hooks.

```go
pkg.For(pkg.Items(messages, func(m Message) int { return m.ID }),
    pkg.PropsComponent(func(ctx *pkg.Ctx, item pkg.ReadSignal[Message]) pkg.Node {
        return pkg.Li(pkg.Text(item.Peek().Text))
    }))
```

The row is a `PropsComponent` whose prop is the item signal, so it has its own ctx and its hooks go at the top of the row body. `For` calls it as an ordinary component; there is no separate per-row plumbing.

- `pkg.Items(sig, key)` is the **retained** source. Each row keeps an item signal. A row whose key is unchanged but whose item changed updates in place. Inserts are one edit, moves are the minimum.
- `*pkg.Feed` from `pkg.NewFeed(key, load)` is the **non-retained** source. The framework keeps keys and node IDs, never the items, so a very long list costs almost nothing per row.

```go
var Row = pkg.PropsComponent(func(_ *pkg.Ctx, item pkg.ReadSignal[Message]) pkg.Node {
    return pkg.Li(pkg.Text(item.Peek().Text))
})

var Chat = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
    feed := pkg.NewFeed(func(m Message) int { return m.ID }, loadMessages)
    send := func(pkg.ClickEvent) {
        msg := next()
        saveMessage(msg)
        feed.Append(msg)
    }
    return pkg.Div(
        pkg.Ul(pkg.For(feed, Row)),
        pkg.Button(pkg.OnClick(send), pkg.Text("Send")),
    )
})
```

A feed offers `Append`, `Prepend`, `InsertBefore(key, ...)`, `Replace`, `Delete` and `Reload`. `load` is your source of truth: it runs on the first render, on `Reload` and on a rebuild, so it must return what is stored, including every row you pushed live. The framework does not keep pushed rows, because that would cost the item's size per row (about 130 bytes more per row for a small struct). Instead, dev mode remembers which keys were pushed since the last load and, when a rebuild's `load` leaves one out, fails the rebuild with an error that names the keys. In production a row that `load` omits is dropped by the rebuild. Mounting one feed in two places is an error.

## Events and Refs

Event handlers are attributes of an element. `pkg.On` and `pkg.OnWith` are the generic form; every typed helper delegates to one generic function that builds the typed event for you:

```go
pkg.Button(pkg.Text("Click"), pkg.On("click", func(evt pkg.Event) { ... }))
pkg.Button(pkg.Text("Click"), pkg.OnClick(func(evt pkg.ClickEvent) { ... }))
pkg.Input(pkg.OnChange(func(evt pkg.ChangeEvent) { ... }), pkg.OnKeyDown(func(evt pkg.KeyboardEvent) { ... }, pkg.Key("Enter")))
pkg.Form(pkg.OnSubmit(func(pkg.FormEvent) { ... }))
```

There is a helper for each typed event: `OnClick`, `OnDoubleClick`, `OnContextMenu`, `OnAuxClick`, the `OnMouse*`, `OnPointer*` (including `OnLostPointerCapture`), `OnTouch*`, `OnDrag*`/`OnDrop`, `OnKeyDown`/`OnKeyUp`, `OnFocus`/`OnBlur`/`OnFocusIn`/`OnFocusOut` (the `In`/`Out` forms bubble), `OnChange`, `OnBeforeInput`, `OnSelect`/`OnSelectionChange`, `OnInvalid`, `OnSubmit` (prevents the default), `OnReset`, `OnToggle`/`OnBeforeToggle`, `OnClose`/`OnCancel` (dialog), `OnLoad`, `OnError`, `OnFullscreenChange`, `OnScroll`/`OnScrollEnd`, `OnWheel`, `OnAnimation*`, `OnTransition*`, `OnCopy`/`OnCut`/`OnPaste`, `OnComposition*`, the media events (`OnPlay`, `OnPause`, `OnEnded`, `OnTimeUpdate`, `OnLoadStart`, ...) and `OnProgress`. Handlers are plain `func(event)` with no return value and run on the session's actor, in the order the browser sent them, so they may read and write signals freely. There is no `OnInput`, `OnFileChange` or `OnKeyPress`.

**`OnChange` follows React.** It fires on every edit of a text input, textarea or contenteditable element (the DOM `input` event) and on commit for a checkbox, radio, select or file input (the DOM `change` event). `ChangeEvent` carries `Value`, `Checked`, `Values` (for `select multiple`) and `Files` (name, size, type and modified time only; bytes go through `UseUpload`). For "finished editing" use `Bind(sig, pkg.Lazy())` for a value, `OnBlur`, or `OnKeyDown(h, pkg.Key("Enter"))` for an action.

**Options.** Every typed helper takes options. `pkg.EventOptions{Prevent: true, Stop: true, Passive: true, Once: true, Capture: true}` sets flags (the struct is itself an option). `Prevent` and `Stop` act in the browser on every matching DOM event, before and independent of any rate limit, so `preventDefault` is never throttled. The rest:

| option | effect |
| --- | --- |
| `pkg.Key("Enter", "Escape")` | `keydown`/`keyup` only. The browser drops other keys before preventing, stopping or sending anything, so "Enter submits and is swallowed, other keys type normally" is one option. |
| `pkg.CapturePointer()` | calls `setPointerCapture` inside the `pointerdown` listener (it cannot be done from a server round trip). Pair it with `OnPointerMove` and `OnLostPointerCapture` for sliders and drag handles. |
| `pkg.Throttle(d)`, `pkg.Frame()`, `pkg.Debounce(d)`, `pkg.Immediate()` | rate control, below. |
| `pkg.Fields("pageX")`, `pkg.AllFields()` | payload fields, below. |

**Rate control.** `mousemove`, `pointermove`, `touchmove`, `scroll`, `wheel`, `drag`, `dragover` and `timeupdate` are sent at most once per animation frame by default. `Throttle(d)` sends the first event at once and then at most one per `d`; `Debounce(d)` sends after `d` of quiet; `Frame()` is one per frame; `Immediate()` (or `Throttle(0)`) turns the default off. A throttled or debounced event **always delivers its last value** (the trailing edge), and a pending trailing event is always sent before any later event, so a debounced input never arrives after the click that follows it. The same functions rate-limit `Bind` (`pkg.Bind(x, pkg.Debounce(...))`) and live ref signals (`video.CurrentTime(pkg.Throttle(...))`), through one client-side limiter. If several handlers on one element ask for different rates for the same event they share one listener: the finest rate wins and an `EVENT_RATE_CONFLICT` warning is raised; setting both `Debounce` and `Throttle` on one handler raises `EVENT_RATE_BOTH` and `Debounce` is used. `scroll`, `wheel`, `touchstart` and `touchmove` are passive by default unless you ask to `Prevent`.

**Payloads are small.** Each typed event sends only the fields its helper needs and your handler can use, not everything the browser has. `ClickEvent` sends the button, client and offset position and the four modifier keys; the page, screen and `Buttons` fields exist on the struct but stay zero unless you ask: `pkg.OnClick(h, pkg.Fields("pageX", "pageY"))` or `pkg.AllFields()`. `KeyboardEvent` does not send the input's text (`pkg.Fields("value")`); `ScrollEvent` sends `ScrollTop` and `ScrollLeft`; `MediaEvent` sends the time, duration and paused state. An unknown field name panics when the helper is built, so the page can only ask the browser for names from the framework's list (the list is `protocol.EventFields`; the browser has no list of its own, to keep the client small).

**Signal or event?** If you want to know what a value *is* (the text in an input, whether a box is checked, where the video is, whether an element has focus, its size or scroll position), use `Bind` or a ref signal: it stays current, you read it when you need it, and it costs nothing until read. If you want to react to something that *happened* (a click, a key press, a drop, a submit, a media `ended`), handle the event. State is a signal, an action is an event. Do not rebuild a signal out of `OnChange` plus a `UseSignal`, and do not poll a ref signal in an effect to detect an action.

**Refs.** `pkg.UseRef[T](ctx)` returns a ref of capability type `T`; attach it with `pkg.Attach(ref)`. Handlers are never registered on a ref, and a ref is never the way to read or set a field's value (that is `Bind`, below). Every ref has `Attached() pkg.ReadSignal[bool]`: true while its element is in the DOM, false after it detaches.

| type | for | methods |
| --- | --- | --- |
| `ElementRef` | any element | `Focus`, `Blur`, `Click`, `ScrollIntoView`, `SetScrollTop/Left`, `ComputedStyle`, `CheckVisibility`, `Matches`; live `Focused`, `Rect`, `Scroll` |
| `InputRef` | `input`, `textarea`, `select`, `button`, `fieldset`, `meter`, `progress`, `option`, `optgroup` | `Select`, `SetSelectionRange`, `SetRangeText`, `Selection` (callback) |
| `FormRef` | `form` | `Submit`, `Reset`, `RequestSubmit`, `CheckValidity`, `ReportValidity` |
| `MediaRef` | `audio`, `video` | `Play`, `Pause`, `Load`, `SetCurrentTime`, `SetVolume`, `SetMuted`, `SetPlaybackRate`; live `CurrentTime`, `Duration`, `Paused`, `Ended`, `Volume`, `Muted` |
| `DialogRef` | `dialog` | `Show`, `ShowModal`, `Close`, `Open`, `ReturnValue` |
| `CanvasRef` | `canvas` | `ToDataURL`, `SetWidth`, `SetHeight`, `Width`, `Height` |
| `DetailsRef` | `details` | `SetOpen`, `Toggle`, `Open` |

Every other type embeds `ElementRef`. Refs expose only these typed methods: there is no raw `Call`, `Set`, `Query` or `AsyncCall`. The browser enforces the same list: every message names the ref type, and the client ignores a method, property or observed value that is not on that type's list (the table is generated from the Go source with `make client-wire`). **`UseScript` is the single escape hatch** for anything else you need to do to an element. Attaching a ref to a tag its type does not support is an error in dev mode (and a `REF_INCOMPATIBLE_TAG` diagnostic in production, the ref stays unattached). Refs and `UseScript` use the same attach mechanism, so attaching either to two live elements is the same error.

**Live values are signals.** The values that change while the page is open (the media state, `Focused`, `Rect`, `Scroll`) are `ReadSignal`s kept current by browser events, with no round trip. Nothing is subscribed until the signal is first read (`Get` or `Peek`); the browser then starts listening, and stops when the element unmounts. Before the page connects, a signal holds the starting value from the HTML (`muted`, `autoplay`, `autofocus`, otherwise the zero value). `CurrentTime`, `Rect` and `Scroll` are sent at most once per animation frame by default and the others on every change; `pkg.Throttle(d)`, `pkg.Frame()` and `pkg.Immediate()` change that (`Throttle(0)` sends every change, and `Frame()` limits any of them to one per frame). The server always sends the rate explicitly. Options are applied by the first call, for example `video.CurrentTime(pkg.Throttle(250 * time.Millisecond))`.

**One-off reads.** A value with no browser event, such as `CanvasRef.ToDataURL`, `ElementRef.ComputedStyle` or `InputRef.Selection`, is read with a callback that runs later on the session's actor. Nothing blocks the actor waiting for the browser, so a slow or missing reply never delays other events. The callback gets a non-nil error if the browser does not answer within `WithDOMTimeout` (5 seconds by default), and the same kind of error if the ref does not allow the call.

**Misuse is reported, not dropped.** A call on a ref before the browser connects (for example in the body), or on a ref that is not attached to a mounted element, raises a `REF_NOT_CONNECTED` or `REF_DETACHED` warning: a diagnostic in the log and, in dev mode, a panel entry. Calls inside `UseRenderEffect` are refused as described under Effects.

**Bind: form fields.** `pkg.Bind(x)` is how a value reaches a field, and the only way (there is no `pkg.Value`, and refs have no `SetValue`). The element and the value type decide what is bound:

| value | element | binds |
| --- | --- | --- |
| `string`, signal | `input` (text-like), `textarea`, `select` | `value` |
| `bool`, signal | `input` checkbox or radio | `checked` |
| `[]string`, signal | `select multiple` | the selected options |
| `bool`, signal | `details`, `dialog` | `open` (`toggle` for `details`, `toggle` and `close` for `dialog`) |
| `string`, signal | `option`, `progress`, `meter`, `li`, `button`, `data`, hidden/button inputs | the `value` attribute, one way |

```go
name := pkg.UseSignal(ctx, "")
agree := pkg.UseSignal(ctx, false)
shout := pkg.UseComputed(ctx, func() string { return strings.ToUpper(name.Get()) })
return pkg.Form(
    pkg.Input(pkg.Type("text"), pkg.Bind(name, pkg.Debounce(200*time.Millisecond))),
    pkg.Input(pkg.Type("checkbox"), pkg.Bind(agree)),
    pkg.P(pkg.Text(shout)),
)
```

- A constant sets the attribute. A read-only signal (`UseComputed`, `ReadSignal`) is one way, server to browser. A writable `*Signal` on an editable control is **two way**: typing updates the signal, and `name.Set("")` clears the field. Controls that are `readonly` or `disabled`, and the one-way elements above, are never two way.
- **No echo.** A change that came from a field's own input is not sent back to that field, so the cursor never jumps. If your code rewrites the value in response (upper-casing it, say), the rewritten value is sent.
- `pkg.Debounce(d)` waits until the user pauses before updating the signal, and `pkg.Throttle(d)` limits how often. `pkg.Lazy()` updates on `change` (blur or Enter) instead of every `input`. They compose. Without `Lazy`, `Bind` listens to the same event as `OnChange`, and the bound signal is updated before any `OnChange` handler on the same element runs, so `sig.Peek()` equals `evt.Value` inside it.
- `Bind(open)` on a `details` or `dialog` opens and closes it without a round trip and follows the user closing it (a dialog closed with Escape or a `method=dialog` form sets the signal to `false`). It also follows a dialog opened or closed through the ref (`DialogRef.Show()`, `ShowModal()`, `Close()`), so a rebuild does not undo it. The signal opens a dialog non-modally; to open one modally call `DialogRef.ShowModal()`.
- A read-only signal bound to an editable, non-readonly control raises a `BIND_READ_ONLY_SIGNAL` warning ("typing here won't update this value; pass a writable Signal for two-way"): the overlay panel in dev, a diagnostic in production.
- A value type the element cannot take (a `Signal[bool]` on a text input) is an error in dev mode and a `BIND_TYPE_MISMATCH` diagnostic in production.

Keep value logic in signals: derive with `UseComputed`, react with `UseEffect`, and use `OnChange` only for user-action logic (validate on edit, autosave).

**Testing the browser side.** `pkg/livetest` simulates the browser for tests: `page := livetest.Mount(t, App)`, then `page.ByID("name").Type("Ada")` (returns whether the server echoed the value back), `.Check(true)`, `.Choose("a", "b")`, `.Toggle(true)` (details/dialog), `.Click()`, `.Press("Enter")` (a key down), and `.PushCurrentTime(12.5)`, `.PushRect(...)`, `.PushScroll(...)` to feed the live signals. `page.HTML()` is the current DOM and `page.Do(func(){...})` runs code on the session, for example `draft.Set("")`.

## Context

```go
var ThemeCtx = pkg.NewContext[string]()

var App = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
    theme := pkg.UseSignal(ctx, "dark")
    pkg.Provide(ctx, ThemeCtx, theme)
    return Child()
})

var Child = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
    theme := pkg.UseContext(ctx, ThemeCtx)
    return pkg.Text(theme)
})
```

`UseContext` returns the provider's `*Signal`. A missing provider is an error: it panics with `context <type> has no provider`, which reaches the nearest error boundary. `NewContext` takes no default.

## Shared and Broadcast

**Shared is context for all sessions.** One value, visible in every session that reads it.

```go
var online = pkg.NewShared(0)

var Badge = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
    n := pkg.UseShared(ctx, online) // ReadSignal[int], read-only
    return pkg.Text(n)
})

// anywhere, from any goroutine or handler
online.Update(func(n int) int { return n + 1 })
```

- `Set` and `Update` are safe from any goroutine, and concurrent `Update`s are never lost.
- **A write reaches every session, including the writer, on the next tick. It is not read back synchronously in the same handler.** `online.Set(5); online.Get()` returns 5, but the mirror signal from `UseShared` updates on the session's next step. A slow session sees only the latest value.
- `pkg.NewSharedFamily(init).Key(k)` gives one `*Shared` per key. A key exists while at least one session reads it; a write to a key nobody reads is discarded, and `Get` returns `init(k)`. Keep `init` cheap and pure.
- `Update`'s function runs under the hub lock: do not call `Set` or `Update` on the same `Shared` inside it.

`Broadcast` is an event stream with no stored value:

```go
var chat = pkg.NewBroadcast[Message]()

pkg.UseBroadcast(ctx, chat, func(m Message) { messages.Update(...) }) // runs on the session's actor
chat.Send(msg)
```

Each session has an inbox of 256 events. When it overflows, the oldest events are dropped and a `broadcast:drop` diagnostic is reported once per burst. A session that has rendered but not connected yet keeps the events sent to it (within the same 256) and delivers them when the socket connects; the handler never runs for a page that does not connect.

## Routing

A route is a pattern and a component. `pkg.Routes(defs...)` renders the one that matches, and `pkg.Route(path, component)` makes a definition. A `RouteDef` is a value, not a node: it only goes to `Routes`.

```go
var App = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
    return pkg.Routes(
        pkg.Route("/", Home),
        pkg.Route("/users/*", UsersLayout),
        pkg.Route("*", NotFound),
    )
})

var UsersLayout = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
    return pkg.Div(
        pkg.NavLink(pkg.NavLinkProps{To: "/users", End: true, ActiveClass: "active", Children: pkg.Text("All")}),
        pkg.Routes(
            pkg.Route("/", UserList),
            pkg.Route("/:id", User),
            pkg.Route("*", UserNotFound),
        ),
    )
})

var User = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
    router := pkg.UseRouter(ctx)
    id := pkg.UseComputed(ctx, func() string { return router.Params.Get()["id"] })
    return pkg.H1(pkg.Text(id))
})
```

- **Patterns.** A path with no `/*` matches exactly. `"/x/*"` matches `/x` and everything under it, and marks a route that has sub-routes: the component renders its own `pkg.Routes(...)`, whose paths are relative to what was left after `/x`. `"*"` is the fallback and matches anything. `:name` captures one segment.
- **The most specific match wins, whatever the order.** A static segment beats a `:param`, a `:param` beats `*`, and an exact path beats the `/*` form of the same path. A level with no match and no `"*"` renders nothing.
- **Matching is a computed over `UseRequest(ctx).URL()`.** A matched component stays mounted while its route is unchanged: moving between `/users/1` and `/users/2`, or between two children of a layout, updates `Params` and swaps only the inner route, and the layout keeps its state.
- **Params accumulate.** The remainder of the path and the params so far flow down through public context. `UseRouter(ctx).Params` is the params of every level from the root to the component that asks, merged.

`pkg.UseRouter(ctx)` returns a `*Router`:

- Matching signals: `Params`, `Matched`, `Match`.
- URL signals, derived from `UseRequest(ctx).URL()`: `Location`, `Path`, `Query`, `Hash`.
- Convenience methods, each a thin wrapper over the request: `Navigate` and `NavigateWith` (`req.Push`), `Replace(href, opts...)`, `ReplaceWith(fn, opts...)` and `SetQuery(key, value)` (`req.Replace`), `Back` and `Forward`.

`pkg.Link`, `pkg.NavLink` and `pkg.Redirect` take props structs. During SSR, `Navigate` and `Redirect` answer with an HTTP redirect (302 for push, 303 for replace); the response carries any cookies set at the same time, and the session is closed.

## Head

```go
pkg.UseHead(ctx, pkg.Head{
    Title:       title,                    // ReadSignal[string]; nil means "not set by this entry"
    Description: pkg.Const("About this page"),
    Links:       []pkg.LinkTag{{Rel: "stylesheet", Href: "/assets/app.css"}},
})
```

Entries merge by depth, then mount order. For a scalar field the deepest entry that sets it wins, and unmounting restores the previous one. `HTMLLang`, `HTMLDir`, `HTMLClass` and `BodyClass` are signals; the class fields are unioned. `Meta`, `Links` and `HeadScripts` are static lists merged by key. `Icon`, `IconBackground` and `IconColor` generate favicon links.

## Request and Cookies

```go
req := pkg.UseRequest(ctx)
req.Header("User-Agent")                     // ReadSignal[string]
req.Headers()                                 // ReadSignal[http.Header]
session := req.Cookie("session")              // ReadSignal[string]
req.SetCookie("session", "abc", &pkg.CookieOptions{Path: "/", HttpOnly: true})
req.URL()                                     // ReadSignal[pkg.Location]
req.Push("/users/2")                          // new history entry
req.Replace("/users/2?tab=a")                 // replace the history entry
req.Replace("/new-home", pkg.Status(301))     // first render: HTTP 301 instead of 303
req.Push("https://example.com/docs")          // another origin: full browser navigation
req.Back(); req.Forward()
```

The request owns the URL and all navigation. `req.URL()` is the single current URL: it starts from the HTTP request and follows `Push`, `Replace`, back/forward (popstate) and reconnects. It is read-only; only the navigation methods change it, and each tells the browser. There is no plain `Path`, `Query` or `Hash` on the request. `UseRouter` does the route matching and derives its URL signals from `req.URL()`; `router.Navigate(href)` is the same as `req.Push(href)`.

Before the page is live, `Push` answers with an HTTP 302 and `Replace` with a 303; `pkg.Status(code)` overrides the status of `Replace` (3xx only). On a live page, a same-origin target is an in-app push or replace (the URL signal updates, the route swaps, no reload), and an `http` or `https` target on another origin makes the browser do a full navigation. Any other target, such as `javascript:`, a URL containing a backslash or whitespace, or a non-3xx status, is dropped and reported as a `NAVIGATION_UNSAFE_URL` warning. The client checks the scheme again before it navigates. `pkg.Redirect(RedirectProps{To, Push, Status})` is the component form. It replaces by default, so Back does not return to the redirecting URL; set `Push` to add a history entry instead. `Status` applies to the replace form.

Headers and cookies are signals: they update when the request headers change, for example on the WebSocket upgrade, so read them with `Get()` inside the tree, a `UseComputed` or a handler. `Header` and `Headers` expose only the allowlisted request headers: `Cookie`, `User-Agent`, `Accept`, `Accept-Language`, `Referer`, `X-Forwarded-For`, `X-Forwarded-Proto`, `X-Forwarded-Host` and `X-Real-Ip`. Credentials such as `Authorization` are never retained unless you list them with `WithRetainedHeaders`. `SetCookie` works before and after the page is live; call it from handlers, not from a body.

## Styles and Scripts

`pkg.UseStyles(ctx, css)` returns scoped CSS: `style.Class("card")` gives the scoped class name, so two components can both define `.card`. It is two public steps: the pure function `pkg.ScopeCSS(key, css)` does the scoping, and `UseHead` registers the resulting sheet (`pkg.Head{Styles: []pkg.StyleTag{...}}`), so the sheet is in the first HTML and one sheet is shared by every component that asks for the same one.

`pkg.UseScript(ctx, js)` binds a client-side closure to an element:

```go
script := pkg.UseScript(ctx, `
    function(el, transport) {
        transport.on("highlight", (data) => { el.style.background = data.color })
        transport.send("ready", {})
        return () => { el.style.background = "" }
    }
`)
script.On("ready", func(v any) { ready.Set(true) })
script.Send("highlight", map[string]any{"color": "red"})
return pkg.Div(pkg.Attach(script), pkg.Text("I have JS attached"))
```

The client builds the closure with `new Function`, so a Content-Security-Policy must allow `unsafe-eval` for pages that use `UseScript`.

`pkg.UseRequest(ctx).Connected()` is a `ReadSignal[bool]` that is true while the browser is connected.

## Uploads and HTTP Handlers

- `pkg.UseUpload(ctx)` returns an `*UploadHandle`: attach it to a file input, configure it with `Accept`, and use `OnComplete`, `OnError`, `OnCancelled` and `Cancel`. `Progress()` is a `ReadSignal[UploadProgress]`.
- `pkg.UseHandler(ctx, method, chain...)` registers an HTTP handler under PondLive and returns a handle with `URL()`, `GenerateToken()` and `Destroy()`.

## Error Boundaries

```go
boundary := pkg.UseErrorBoundary(ctx)
errs := boundary.Errors() // ReadSignal[*ErrorBatch]
boundary.Reset()
```

A panic in a body, a handler, a computed or an effect reaches the nearest boundary. At the root, production serves a crash page and dev mode shows an overlay.

**Reporting your own problems.** `pkg.Report(ctx, level, code, message)` uses the same path as the framework's own diagnostics (ref misuse, unsafe navigation, forbidden render-effect calls). It always writes a diagnostic to the log (`warning:CODE` or `error:CODE`). In dev mode a `pkg.ReportWarning` also adds an entry to the dismissible warning panel, and a `pkg.ReportError` raises the error overlay through the nearest boundary. Production only logs. Calls made in a serialize pass are ignored, so a report in a body is made once, at mount.

```go
pkg.Report(ctx, pkg.ReportWarning, "SLOW_QUERY", "the report query took 4s")
```

There is exactly one reporting path: `pkg.Report` and the framework's own diagnostics share it, and the protocol and infrastructure diagnostics (overflow, uploads, feeds, portals, cookie-token eviction, session lifecycle) go through it too, as log-only diagnostics.

## Getting Started

```go
package main

import (
    "log"

    "github.com/eleven-am/pondlive/pkg"
)

var Counter = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
    count := pkg.UseSignal(ctx, 0)
    inc := func(pkg.ClickEvent) { count.Update(func(n int) int { return n + 1 }) }

    return pkg.Div(
        pkg.H1(pkg.Text("Counter")),
        pkg.Button(pkg.OnClick(inc), pkg.Text("+")),
        pkg.P(pkg.Text(count)),
    )
})

func main() {
    app, err := pkg.NewApp(Counter, pkg.WithDevMode())
    if err != nil {
        log.Fatal(err)
    }
    log.Fatal(app.Server(":8080").ListenAndServe())
}
```

- Install: `go get github.com/eleven-am/pondlive`.
- Examples: `cd examples/counter && go run .`. Others: `examples/countdown`, `examples/auth`, `examples/scoped-styles`, `examples/stream-chat`.
- Dev bundle: `pkg.WithDevMode()` serves `/static/pondlive-dev.js` and enables the determinism checks.

## Serving
- `app.Handler()` is the HTTP handler; `app.Server(addr)` returns an `http.Server` with header, read, write and idle timeouts and a header size limit; use it (not `http.ListenAndServe(addr, app.Handler())`, which applies no limits). `WithHTTPConfig` overrides the defaults; a negative timeout or size disables that limit.
- `app.Shutdown(ctx)` closes every session, every live connection and the upload handler. Call it before `Server.Shutdown`; it is safe to call more than once.
- `app.Stats()` reports the number of sessions per state (pending, live, disconnected), why sessions were closed, and how many page requests were refused (`Rejected.Full`, `Rejected.Pending`, `Rejected.Duplicate`, `Rejected.Closed`), and how many removed sessions are still stuck in user code (`Stuck`); use it for metrics. `Shutdown` returns an error naming the stuck count if they do not finish before its context ends.
- PondLive handles the WebSocket at `/live` (same-origin only unless `WithAllowedOrigins` is set) and serves the client asset at `/static/pondlive.js` (dev variant in dev mode).

## State and Session
- State is per-session, in memory on the server.
- Options: `WithDevMode`, `WithDOMTimeout`, `WithIDGenerator`, `WithContext`, `WithAllowedOrigins`, `WithAttachSecret`, `WithAttachTokenTTL`, `WithReadLimit`, `WithOutboundBudget`, `WithPingInterval`, `WithRetainedHeaders`, `WithMaxSessions`, `WithMaxPendingSessions`, `WithSessionLifetime`, `WithRenderTimeout`, `WithHandlerQueueCap`, `WithHTTPConfig`, `WithWatchdog`, `WithUploadConfig`.
- Session IDs default to random; can be overridden. A custom generator that returns an id already in use makes the page request fail with `500` and a diagnostic; it never joins the old session.
- Requests to `UseHandler` URLs queue behind the session's current work; at most 32 may wait per session (`WithHandlerQueueCap`), and further requests get `503` with `Retry-After` immediately.
- A session whose actor spends more than 30 seconds in one step (`WithWatchdog`) is treated as wedged and removed; a step slower than one second is reported as a diagnostic.

## Author

Roy Ossai ([@eleven-am](https://github.com/eleven-am))

## License

MIT (see [LICENSE](LICENSE)).
