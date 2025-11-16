package router

import (
	h "github.com/eleven-am/pondlive/go/pkg/live/html"

	"github.com/eleven-am/pondlive/go/internal/dom"
	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
)

var (
	routerStoreCtx = runtime.NewContext((*RouterStore)(nil))
	routeBaseCtx   = runtime.NewContext("/")
	snapshotCtx    = runtime.NewContext((*Snapshot)(nil))
)

// storeCaptureHook is used in tests to observe the active RouterStore.
var storeCaptureHook func(*RouterStore)

type routerProps struct {
	Children []h.Node
}

type routerChildrenProps struct {
	Children []h.Node
	Store    *RouterStore
}

// Router provides router state to its descendants and resolves all router
// placeholders before returning the rendered node tree.
func Router(ctx runtime.Ctx, children ...h.Node) h.Node {
	return runtime.Render(ctx, routerComponent, routerProps{Children: children})
}

func routerComponent(ctx runtime.Ctx, props routerProps) h.Node {
	sess := ctx.Session()
	initial := Location{Path: "/"}
	if sess != nil {
		initial = fromRuntimeLocation(runtime.InternalInitialLocation(sess))
	}
	store := useRouterStore(ctx, initial)
	if storeCaptureHook != nil {
		storeCaptureHook(store)
	}
	getVersion, setVersion := runtime.UseState(ctx, 0)
	runtime.UseEffect(ctx, func() runtime.Cleanup {
		return store.Subscribe(func(Location) {
			setVersion(getVersion() + 1)
		})
	}, store)
	if sess != nil {
		runtime.InternalRegisterRouterHandlers(sess,
			func() runtime.Location { return toRuntimeLocation(store.Location()) },
			func(loc runtime.Location) { store.SetLocation(fromRuntimeLocation(loc)) },
			func(loc runtime.Location) { store.SetLocation(fromRuntimeLocation(loc)) },
		)
		runtime.InternalStoreLocation(sess, toRuntimeLocation(store.Location()))
		runtime.UseEffect(ctx, func() runtime.Cleanup {
			handler := &sessionNavHandler{sess: sess}
			dispatcher := NewNavDispatcher(store, handler)
			cancelNav := dispatcher.Start()
			cancelSubscribe := store.Subscribe(func(loc Location) {
				runtime.InternalStoreLocation(sess, toRuntimeLocation(loc))
			})
			return func() {
				cancelNav()
				cancelSubscribe()
			}
		}, store)
	}
	applied := runtime.UseRef(ctx, false)
	if snap := snapshotCtx.Use(ctx); snap != nil && !applied.Cur {
		store.ApplySnapshot(*snap)
		applied.Cur = true
	}
	node := routerStoreCtx.Provide(ctx, store, func() h.Node {
		loc := toRuntimeLocation(store.Location())
		params := store.Params()
		return runtime.LocationCtx.Provide(ctx, loc, func() h.Node {
			return runtime.ParamsCtx.Provide(ctx, params, func() h.Node {
				return routeBaseCtx.Provide(ctx, "/", func() h.Node {
					return renderRouterChildren(ctx, props.Children, store)
				})
			})
		})
	})
	store.DrainAndDispatch()
	return node
}

func renderRouterChildren(ctx runtime.Ctx, children []h.Node, store *RouterStore) h.Node {
	return runtime.Render(ctx, routerChildrenComponent, routerChildrenProps{
		Children: children,
		Store:    store,
	})
}

func routerChildrenComponent(ctx runtime.Ctx, props routerChildrenProps) h.Node {
	resolved := make([]dom.Node, 0, len(props.Children))
	for _, child := range props.Children {
		if child == nil {
			continue
		}
		resolved = append(resolved, ResolveTree(ctx, child, props.Store))
	}
	if len(resolved) == 0 {
		return h.Fragment()
	}
	return h.Fragment(resolved...)
}

func useRouterStore(ctx runtime.Ctx, initial Location) *RouterStore {
	init := initial
	if init.Path == "" {
		init.Path = "/"
	}
	store := runtime.UseMemo(ctx, func() *RouterStore {
		return NewStore(init)
	}, nil)
	if store == nil {
		return NewStore(Location{Path: "/"})
	}
	return store
}

// WithBase adjusts the base path for nested route definitions.
func WithBase(ctx runtime.Ctx, base string, render func() h.Node) h.Node {
	if render == nil {
		return h.Fragment()
	}
	if ctx.ComponentID() == "" {
		cleanup := pushContextlessBase(base)
		defer cleanup()
		return render()
	}
	return routeBaseCtx.Provide(ctx, base, render)
}

// WithSnapshot seeds descendants with a pre-rendered router snapshot.
func WithSnapshot(ctx runtime.Ctx, snap Snapshot, render func() h.Node) h.Node {
	if render == nil {
		return h.Fragment()
	}
	clone := snap
	return snapshotCtx.Provide(ctx, &clone, render)
}

func toRuntimeLocation(loc Location) runtime.Location {
	return runtime.Location{Path: loc.Path, Query: cloneValues(loc.Query), Hash: loc.Hash}
}

func fromRuntimeLocation(loc runtime.Location) Location {
	return Location{Path: loc.Path, Query: cloneValues(loc.Query), Hash: loc.Hash}
}
