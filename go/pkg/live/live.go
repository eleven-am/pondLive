package live

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
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
	HeaderState                       = session.HeaderState
)

// Component wraps a stateless component function so it can be invoked directly
// from HTML builders without manually calling Render.
//
// Example:
//
//	counter := live.Component(func(ctx live.Ctx) h.Node {
//	       return h.Div()
//	})
//
// Within another component you can render it with:
//
//	counter(ctx, live.WithKey("counter"))
//
// Prefer invoking the returned function instead of calling Render for
// stateless children.
func Component(fn func(Ctx) h.Node) func(Ctx, ...RenderOption) h.Node {
	if fn == nil {
		return nil
	}
	wrapped := func(ctx Ctx) *dom.StructuredNode {
		return fn(ctx)
	}
	return runtime.NoPropsComponent(wrapped, fn)
}

// PropsComponent wraps a component function that expects props so it can be
// called directly with a context, props, and optional render options.
//
// Example:
//
//	card := live.PropsComponent(func(ctx live.Ctx, props CardProps) h.Node {
//	       return h.Div(h.Text(props.Title))
//	})
//
// Render it via:
//
//	card(ctx, CardProps{Title: "Inbox"}, live.WithKey("card"))
func PropsComponent[P any](fn func(Ctx, P) h.Node) func(Ctx, P, ...RenderOption) h.Node {
	if fn == nil {
		return nil
	}
	wrapped := func(ctx Ctx, props P) *dom.StructuredNode {
		return fn(ctx, props)
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

func UseHeader(ctx Ctx) HeaderState {
	return session.UseHeader(ctx)
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
