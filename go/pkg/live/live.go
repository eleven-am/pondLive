package live

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/router"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/session"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type (
	Ctx                               = runtime.Ctx
	RenderOption                      = runtime.RenderOption
	StateOpt[T any]                   = runtime.StateOpt[T]
	Cleanup                           = runtime.Cleanup
	Ref[T any]                        = runtime.Ref[T]
	ElementRef[T h.ElementDescriptor] = h.ElementRef[T]
	ElementDescriptor                 = h.ElementDescriptor
	Node                              = h.Node
	Context[T any]                    = runtime.Context[T]
	SessionID                         = session.SessionID
	Session                           = runtime.ComponentSession
	ScrollOptions                     = dom.ScrollOptions
	PubsubMessage[T any]              = runtime.PubsubMessage[T]
	PubsubHandle[T any]               = runtime.PubsubHandle[T]
	PubsubOption[T any]               = runtime.PubsubOption[T]
	StreamItem[T any]                 = runtime.StreamItem[T]
	StreamHandle[T any]               = runtime.StreamHandle[T]
	RuntimeComponent[P any]           = runtime.Component[P]
	NavMsg                            = router.NavMsg
	PopMsg                            = router.PopMsg
	Styles                            = runtime.Styles
	ScriptHandle                      = runtime.ScriptHandle
	UploadHandle                      = runtime.UploadHandle
	UploadConfig                      = runtime.UploadConfig
	UploadEvent                       = runtime.UploadEvent
	HeadersHandle                     = headers.Handle
	CookieOptions                     = headers.CookieOptions
)

// Component wraps a component function that accepts children as a slice.
// Children can include h.Key() at the top level to set the component's render key.
//
// Example:
//
//	card := live.Component(func(ctx live.Ctx, children []h.Item) h.Node {
//	       return h.Div(
//	           h.H1(h.Text("Card")),
//	           h.Fragment(children...),
//	       )
//	})
//
// Render it with:
//
//	card(ctx, h.Key("my-card"), h.Text("Child 1"), h.Text("Child 2"))
//
// The h.Key() is extracted and used as the component's identity, not rendered as a DOM element.
func Component(fn func(Ctx, []h.Item) h.Node) func(Ctx, ...h.Item) h.Node {
	if fn == nil {
		return nil
	}
	wrapped := func(ctx Ctx, children []dom.Item) *dom.StructuredNode {
		return fn(ctx, children)
	}
	return runtime.NoPropsComponent(wrapped, fn)
}

// PropsComponent wraps a component function that accepts props and children as a slice.
// Children can include h.Key() at the top level to set the component's render key.
//
// Example:
//
//	card := live.PropsComponent(func(ctx live.Ctx, props CardProps, children []h.Item) h.Node {
//	       return h.Div(
//	           h.H1(h.Text(props.Title)),
//	           h.Fragment(children...),
//	       )
//	})
//
// Render it with:
//
//	card(ctx, CardProps{Title: "Inbox"}, h.Key("my-card"), h.Text("Message 1"), h.Text("Message 2"))
//
// The h.Key() is extracted and used as the component's identity, not rendered as a DOM element.
func PropsComponent[P any](fn func(Ctx, P, []h.Item) h.Node) func(Ctx, P, ...h.Item) h.Node {
	if fn == nil {
		return nil
	}
	wrapped := func(ctx Ctx, props P, children []dom.Item) *dom.StructuredNode {
		return fn(ctx, props, children)
	}
	return runtime.PropsComponent(wrapped, fn)
}

// Render invokes the supplied child component with props, returning its node.
// Use it within your component to manually compose children. Combine with
// WithKey to give siblings stable identities in lists.
//
// Deprecated: Wrap the child with Component or PropsComponent and call the
// returned function directly.
func Render[P any](ctx Ctx, fn RuntimeComponent[P], props P, opts ...RenderOption) h.Node {
	return runtime.Render(ctx, fn, props, opts...)
}

// WithKey assigns a deterministic key to a child rendered via Render. Helpful
// when rendering slices so LiveUI can diff elements predictably.
func WithKey(key string) RenderOption { return runtime.WithKey(key) }

// UseState creates reactive state scoped to the component. It returns getter
// and setter closures; calling the setter schedules a render. Supply
// WithEqual to suppress renders when the value hasn't meaningfully changed.
//
// Example:
//
//	func Counter(ctx live.Ctx) h.Node {
//	    count, setCount := live.UseState(ctx, 0)
//
//	    return h.Div(
//	        h.Text(fmt.Sprintf("Count: %d", count())),
//	        h.Button(
//	            h.OnClick(func() h.Updates {
//	                setCount(count() + 1)
//	                return nil
//	            }),
//	            h.Text("Increment"),
//	        ),
//	    )
//	}
//
// With custom equality to avoid unnecessary rerenders:
//
//	user, setUser := live.UseState(ctx, User{}, live.WithEqual(func(a, b User) bool {
//	    return a.ID == b.ID && a.Name == b.Name
//	}))
func UseState[T any](ctx Ctx, initial T, opts ...StateOpt[T]) (func() T, func(T)) {
	return runtime.UseState(ctx, initial, opts...)
}

// UseMemo memoizes compute until any dependency changes. It's useful for
// expensive calculations or deriving values from props/state without
// recomputing every render.
//
// Example:
//
//	func ProductList(ctx live.Ctx) h.Node {
//	    searchQuery, _ := live.UseState(ctx, "")
//	    products, _ := live.UseState(ctx, []Product{})
//
//	    // Only recompute filtered list when products or search query changes
//	    filteredProducts := live.UseMemo(ctx, func() []Product {
//	        if searchQuery() == "" {
//	            return products()
//	        }
//	        var filtered []Product
//	        for _, p := range products() {
//	            if strings.Contains(strings.ToLower(p.Name), strings.ToLower(searchQuery())) {
//	                filtered = append(filtered, p)
//	            }
//	        }
//	        return filtered
//	    }, products(), searchQuery())
//
//	    return h.Div(/* render filteredProducts */)
//	}
func UseMemo[T any](ctx Ctx, compute func() T, deps ...any) T {
	return runtime.UseMemo(ctx, compute, deps...)
}

// UseEffect runs setup after render and optionally returns a cleanup that runs
// on dependency change or unmount. Provide deps to limit when the effect
// re-executes.
//
// Example - Run once on mount:
//
//	func Dashboard(ctx live.Ctx) h.Node {
//	    data, setData := live.UseState(ctx, []Item{})
//
//	    live.UseEffect(ctx, func() live.Cleanup {
//	        // Fetch data when component mounts
//	        items, err := fetchDashboardData()
//	        if err == nil {
//	            setData(items)
//	        }
//	        return nil  // No cleanup needed
//	    })  // Empty deps = run once on mount
//
//	    return h.Div(/* render data */)
//	}
//
// Example - Run when dependencies change:
//
//	func UserProfile(ctx live.Ctx, userID string) h.Node {
//	    profile, setProfile := live.UseState(ctx, Profile{})
//
//	    live.UseEffect(ctx, func() live.Cleanup {
//	        // Fetch profile when userID changes
//	        p, err := fetchProfile(userID)
//	        if err == nil {
//	            setProfile(p)
//	        }
//	        return nil
//	    }, userID)  // Re-run when userID changes
//
//	    return h.Div(/* render profile */)
//	}
//
// Example - With cleanup:
//
//	func Timer(ctx live.Ctx) h.Node {
//	    count, setCount := live.UseState(ctx, 0)
//
//	    live.UseEffect(ctx, func() live.Cleanup {
//	        ticker := time.NewTicker(1 * time.Second)
//	        go func() {
//	            for range ticker.C {
//	                setCount(count() + 1)
//	            }
//	        }()
//
//	        // Cleanup: stop ticker when component unmounts or deps change
//	        return func() {
//	            ticker.Stop()
//	        }
//	    })
//
//	    return h.Div(h.Text(fmt.Sprintf("Elapsed: %d seconds", count())))
//	}
func UseEffect(ctx Ctx, setup func() Cleanup, deps ...any) {
	runtime.UseEffect(ctx, setup, deps...)
}

// UseRef returns a pointer holding mutable state that persists across renders.
// It's ideal for tracking DOM handles or other imperative data without triggering rerenders.
//
// Example - Track previous value:
//
//	func ValueTracker(ctx live.Ctx) h.Node {
//	    value, setValue := live.UseState(ctx, 0)
//	    prevValue := live.UseRef(ctx, 0)
//
//	    live.UseEffect(ctx, func() live.Cleanup {
//	        prevValue.Cur = value()
//	        return nil
//	    }, value())
//
//	    return h.Div(
//	        h.Text(fmt.Sprintf("Current: %d, Previous: %d", value(), prevValue.Cur)),
//	    )
//	}
//
// Example - Store mutable data without triggering rerenders:
//
//	func AnimationComponent(ctx live.Ctx) h.Node {
//	    frameCount := live.UseRef(ctx, 0)
//
//	    live.UseEffect(ctx, func() live.Cleanup {
//	        ticker := time.NewTicker(16 * time.Millisecond)  // ~60fps
//	        go func() {
//	            for range ticker.C {
//	                frameCount.Cur++
//	                // Process animation frame without rerendering
//	                processFrame(frameCount.Cur)
//	            }
//	        }()
//	        return func() { ticker.Stop() }
//	    })
//
//	    return h.Canvas(/* render animation */)
//	}
func UseRef[T any](ctx Ctx, zero T) *Ref[T] {
	return runtime.UseRef(ctx, zero)
}

type hookable[R any] interface {
	HookBuild(any) R
}

// UseElement returns a fully-wrapped HTML ref (e.g., *h.DivRef, *h.ButtonRef) so callers
// can attach event handlers and call DOM methods without extra boilerplate.
//
// Example - Basic element ref with event handler:
//
//	func InteractiveButton(ctx live.Ctx) h.Node {
//	    buttonRef := live.UseElement[*h.ButtonRef](ctx)
//
//	    buttonRef.OnClick(func(evt h.ClickEvent) h.Updates {
//	        fmt.Println("Button clicked!")
//	        return nil
//	    })
//
//	    return h.Button(h.Attach(buttonRef), h.Text("Click me"))
//	}
//
// Example - Call DOM methods:
//
//	func FocusableInput(ctx live.Ctx) h.Node {
//	    inputRef := live.UseElement[*h.InputRef](ctx)
//	    shouldFocus, _ := live.UseState(ctx, false)
//
//	    live.UseEffect(ctx, func() live.Cleanup {
//	        if shouldFocus() {
//	            inputRef.Focus()  // Call DOM method
//	        }
//	        return nil
//	    }, shouldFocus())
//
//	    return h.Input(h.Attach(inputRef), h.Type("text"))
//	}
//
// Example - Access element properties:
//
//	func ScrollTracker(ctx live.Ctx) h.Node {
//	    divRef := live.UseElement[*h.DivRef](ctx)
//
//	    divRef.OnScroll(func(evt h.ScrollEvent) h.Updates {
//	        fmt.Printf("Scroll position: %f\n", evt.ScrollTop)
//	        return nil
//	    })
//
//	    return h.Div(h.Attach(divRef), h.Text("Scrollable content"))
//	}
func UseElement[R hookable[R]](ctx Ctx) R {
	var zero R
	return zero.HookBuild(ctx)
}

// UseStream renders and manages a keyed list. It returns a fragment node and a
// handle exposing mutation helpers for the backing collection. Each item must have
// a unique key for efficient diffing and updates.
//
// Example - Basic todo list:
//
//	func TodoList(ctx live.Ctx) h.Node {
//	    node, handle := live.UseStream(ctx, func(item live.StreamItem[Todo]) h.Node {
//	        return h.Li(
//	            h.Text(item.Value.Text),
//	            h.Button(
//	                h.OnClick(func() h.Updates {
//	                    handle.Remove(item.Key)
//	                    return nil
//	                }),
//	                h.Text("Delete"),
//	            ),
//	        )
//	    })
//
//	    return h.Div(
//	        h.Button(
//	            h.OnClick(func() h.Updates {
//	                newTodo := Todo{Text: "New task"}
//	                handle.Append(live.StreamItem[Todo]{
//	                    Key:   fmt.Sprintf("todo-%d", time.Now().Unix()),
//	                    Value: newTodo,
//	                })
//	                return nil
//	            }),
//	            h.Text("Add Todo"),
//	        ),
//	        h.Ul(node),  // Render the stream
//	    )
//	}
//
// Example - Real-time message list with updates:
//
//	func ChatMessages(ctx live.Ctx) h.Node {
//	    node, handle := live.UseStream(ctx, func(item live.StreamItem[Message]) h.Node {
//	        return h.Div(
//	            h.Text(fmt.Sprintf("%s: %s", item.Value.Author, item.Value.Text)),
//	        )
//	    })
//
//	    live.UseEffect(ctx, func() live.Cleanup {
//	        // Subscribe to new messages
//	        unsubscribe := subscribeToMessages(func(msg Message) {
//	            handle.Prepend(live.StreamItem[Message]{
//	                Key:   msg.ID,
//	                Value: msg,
//	            })
//	        })
//	        return unsubscribe
//	    })
//
//	    return h.Div(node)
//	}
func UseStream[T any](ctx Ctx, renderRow func(StreamItem[T]) h.Node, initial ...StreamItem[T]) (h.Node, StreamHandle[T]) {
	return runtime.UseStream(ctx, renderRow, initial...)
}

// WithEqual customizes UseState comparisons. If eq(old, new) is true, the
// setter skips scheduling a render.
func WithEqual[T any](eq func(a, b T) bool) StateOpt[T] {
	return runtime.WithEqual(eq)
}

// UseSelect subscribes to a context value, projecting it with pick. The eq
// function controls whether the projected value changed, avoiding unnecessary
// rerenders when unrelated context fields update.
//
// Example - Subscribe to specific field:
//
//	type AppState struct {
//	    User      User
//	    Theme     string
//	    Locale    string
//	}
//
//	var AppContext = live.NewContext(AppState{})
//
//	func UserProfile(ctx live.Ctx) h.Node {
//	    // Only render when User changes, not when Theme or Locale changes
//	    user := live.UseSelect(ctx, AppContext,
//	        func(state AppState) User { return state.User },
//	        func(a, b User) bool { return a.ID == b.ID && a.Name == b.Name },
//	    )
//
//	    return h.Div(
//	        h.H1(h.Text(user.Name)),
//	        h.P(h.Text(user.Email)),
//	    )
//	}
//
// Example - Compute derived value:
//
//	func CartItemCount(ctx live.Ctx) h.Node {
//	    // Only render when cart item count changes
//	    itemCount := live.UseSelect(ctx, CartContext,
//	        func(cart Cart) int { return len(cart.Items) },
//	        func(a, b int) bool { return a == b },
//	    )
//
//	    return h.Span(h.Text(fmt.Sprintf("%d items", itemCount)))
//	}
//
// NewContext creates a context handle with a default value. Use Provide on the
// returned context to supply overrides, and Use to read it down the tree.
func NewContext[T any](def T) *Context[T] {
	return runtime.CreateContext(def)
}

// UseStyles parses CSS, scopes selectors to the component, and returns a Styles
// object for accessing scoped class names and the style tag node.
//
// Example:
//
//	func Card(ctx live.Ctx) h.Node {
//	    styles := live.UseStyles(ctx, `
//	        .card {
//	            background: #fff;
//	            border-radius: 8px;
//	            padding: 16px;
//	        }
//	        .card:hover {
//	            box-shadow: 0 4px 12px rgba(0,0,0,0.1);
//	        }
//	        @media (max-width: 768px) {
//	            .card { padding: 8px; }
//	        }
//	    `)
//
//	    return h.Div(
//	        styles.StyleTag(),
//	        h.Div(
//	            h.Class(styles.Class("card")),
//	            h.Text("Hello"),
//	        ),
//	    )
//	}
func UseStyles(ctx Ctx, css string) *Styles {
	return runtime.UseStyles(ctx, css)
}

// UseScript creates a client-side script that runs in the browser. The script
// receives the element it's attached to and a transport object for bidirectional
// communication with the server.
//
// Example - Auto-incrementing counter:
//
//	func AutoCounter(ctx live.Ctx) h.Node {
//	    count, setCount := live.UseState(ctx, 0)
//
//	    script := live.UseScript(ctx, `
//	        (element, transport) => {
//	            const interval = setInterval(() => {
//	                transport.send({ tick: true });
//	            }, 1000);
//	            return () => clearInterval(interval);
//	        }
//	    `)
//
//	    script.OnMessage(func(data map[string]any) {
//	        setCount(count() + 1)
//	    })
//
//	    div := h.Div(h.Textf("Count: %d", count()))
//	    script.AttachTo(div)
//	    return div
//	}
func UseScript(ctx Ctx, script string) ScriptHandle {
	return runtime.UseScript(ctx, script)
}

// UseUpload creates a file upload handler that manages client-side file selection
// and server-side file processing. It combines UseScript for the client upload UI
// and UseHandler for receiving the uploaded file via HTTP POST.
//
// The returned UploadHandle provides methods to configure upload constraints and
// handle upload lifecycle events:
//
//   - Accept(cfg): Configure max file size, accepted file types, and multiple file support
//   - OnReady(fn): Called when user selects a file, before upload starts
//   - OnChange(fn): Called when upload begins with file metadata
//   - OnProgress(fn): Called during upload with loaded/total bytes
//   - Progress(): Returns current upload progress
//   - OnComplete(fn): Called on server when file is received - process the file here
//   - OnError(fn): Called if upload fails
//   - OnCancelled(fn): Called if upload is cancelled
//   - Cancel(): Programmatically cancel an ongoing upload
//
// Example - Basic file upload with processing:
//
//	func FileUploader(ctx live.Ctx) h.Node {
//	    upload := live.UseUpload(ctx)
//
//	    upload.Accept(live.UploadConfig{
//	        MaxSize: 10 * 1024 * 1024, // 10MB
//	        Accept:  []string{"image/*", ".pdf"},
//	    })
//
//	    upload.OnComplete(func(file multipart.File, header *multipart.FileHeader) error {
//	        // Process the uploaded file
//	        data, err := io.ReadAll(file)
//	        if err != nil {
//	            return err
//	        }
//	        // Save file, process image, etc.
//	        return saveFile(header.Filename, data)
//	    })
//
//	    input := h.Input(h.Type("file"))
//	    upload.AttachTo(input)
//
//	    return h.Div(
//	        input,
//	        h.Div(h.Textf("Upload progress: %d%%", progress.Progress().Loaded*100/progress.Progress().Total)),
//	    )
//	}
func UseUpload(ctx Ctx) UploadHandle {
	return runtime.UseUpload(ctx)
}

// UseHeaders provides access to HTTP request headers and cookie management.
// It works in both server-side rendering (SSR) and WebSocket modes, allowing you
// to read incoming request headers and set/delete cookies that work across both contexts.
//
// The returned HeadersHandle provides these methods:
//
//   - Get(name): Read an HTTP request header value
//   - GetCookie(name): Read a cookie from the request
//   - SetCookie(name, value): Set a cookie with default options
//   - SetCookieWithOptions(name, value, options): Set a cookie with custom options
//   - DeleteCookie(name): Delete a cookie
//
// Example - Read request headers:
//
//	func Dashboard(ctx live.Ctx) h.Node {
//	    headers := live.UseHeaders(ctx)
//
//	    userAgent, _ := headers.Get("User-Agent")
//	    acceptLang, _ := headers.Get("Accept-Language")
//
//	    return h.Div(
//	        h.P(h.Textf("Browser: %s", userAgent)),
//	        h.P(h.Textf("Language: %s", acceptLang)),
//	    )
//	}
//
// Example - Cookie-based authentication:
//
//	func AuthenticatedPage(ctx live.Ctx) h.Node {
//	    headers := live.UseHeaders(ctx)
//
//	    // Read authentication token from cookie
//	    token, authenticated := headers.GetCookie("auth_token")
//	    if !authenticated {
//	        return h.Div(h.Text("Please log in"))
//	    }
//
//	    return h.Div(h.Textf("Welcome! Token: %s", token))
//	}
//
// Example - Set and delete cookies:
//
//	func LoginForm(ctx live.Ctx) h.Node {
//	    headers := live.UseHeaders(ctx)
//	    loggedIn, setLoggedIn := live.UseState(ctx, false)
//
//	    handleLogin := func() h.Updates {
//	        // Set cookie with custom options
//	        headers.SetCookieWithOptions("auth_token", "abc123", live.HeadersHandle{
//	            Path:     "/",
//	            MaxAge:   86400,  // 24 hours
//	            Secure:   true,
//	            HttpOnly: true,
//	            SameSite: http.SameSiteStrictMode,
//	        })
//	        setLoggedIn(true)
//	        return nil
//	    }
//
//	    handleLogout := func() h.Updates {
//	        headers.DeleteCookie("auth_token")
//	        setLoggedIn(false)
//	        return nil
//	    }
//
//	    if loggedIn() {
//	        return h.Button(h.OnClick(handleLogout), h.Text("Logout"))
//	    }
//	    return h.Button(h.OnClick(handleLogin), h.Text("Login"))
//	}
//
// Example - User preferences with cookies:
//
//	func ThemeSwitcher(ctx live.Ctx) h.Node {
//	    headers := live.UseHeaders(ctx)
//
//	    // Read theme preference from cookie
//	    savedTheme, _ := headers.GetCookie("theme")
//	    theme, setTheme := live.UseState(ctx, savedTheme)
//	    if theme() == "" {
//	        setTheme("light")  // Default
//	    }
//
//	    toggleTheme := func() h.Updates {
//	        newTheme := "dark"
//	        if theme() == "dark" {
//	            newTheme = "light"
//	        }
//
//	        // Save preference to cookie
//	        headers.SetCookieWithOptions("theme", newTheme, live.HeadersHandle{
//	            Path:   "/",
//	            MaxAge: 31536000,  // 1 year
//	        })
//	        setTheme(newTheme)
//	        return nil
//	    }
//
//	    return h.Div(
//	        h.Class(theme()),
//	        h.Button(h.OnClick(toggleTheme), h.Textf("Switch to %s mode",
//	            if theme() == "dark" { "light" } else { "dark" })),
//	    )
//	}
//
// Note: GetCookie reads from the initial HTTP request headers, not cookies set
// during the current render. To read newly set cookies, you need to wait for the
// next request or page reload.
func UseHeaders(ctx Ctx) HeadersHandle {
	return headers.UseHeaders(ctx)
}
