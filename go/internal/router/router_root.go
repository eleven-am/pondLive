package router

import (
	"sync"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Type aliases for convenience
type (
	Ctx     = runtime.Ctx
	Cleanup = runtime.Cleanup
)

type routerState struct {
	getLoc func() Location
	setLoc func(Location)
}

// RouterStateCtx is the context for providing router state to child components
var RouterStateCtx = runtime.CreateContext[routerState](routerState{})

// SessionEntryCtx provides router session data (navigation, params, handlers)
var SessionEntryCtx = runtime.CreateContext[*sessionEntry](nil)

type sessionEntry struct {
	mu sync.Mutex

	handlers   sessionHandlers
	navigation sessionNavigation
	params     sessionParamStore
	render     sessionRenderState
}

type sessionHandlers struct {
	get    func() Location
	set    func(Location)
	assign func(Location)
}

type sessionNavigation struct {
	loc     Location
	history []NavMsg
	pending []NavMsg
	seed    Location
	hasSeed bool
}

type sessionParamStore struct {
	values map[string]string
}

type sessionRenderState struct {
	active       bool
	currentRoute string
	depth        int
}

type routerProps struct {
	Children []*dom2.StructuredNode
}

func Router(ctx Ctx, children ...*dom2.StructuredNode) *dom2.StructuredNode {
	return runtime.Render(ctx, routerComponent, routerProps{Children: children})
}

func routerComponent(ctx Ctx, props routerProps) *dom2.StructuredNode {
	entry := &sessionEntry{}

	sess := ctx.Session()
	initial := getInitialLocationFromSession(sess)

	get, set := runtime.UseState(ctx, initial, runtime.WithEqual(LocEqual))

	state := routerState{}
	state.getLoc = func() Location { return cloneLocation(get()) }
	state.setLoc = func(next Location) {
		canon := canonicalizeLocation(next)
		set(canon)
	}

	entry.mu.Lock()
	entry.handlers.get = state.getLoc
	entry.handlers.set = state.setLoc
	entry.render.active = true
	entry.navigation.loc = get()
	entry.mu.Unlock()

	runtime.UseEffect(ctx, func() runtime.Cleanup {
		return func() {
			entry.mu.Lock()
			entry.render.active = false
			entry.mu.Unlock()
		}
	})

	current := state.getLoc()

	return SessionEntryCtx.Provide(ctx, entry, func(ectx runtime.Ctx) *dom2.StructuredNode {
		return RouterStateCtx.Provide(ectx, state, func(rctx runtime.Ctx) *dom2.StructuredNode {
			return LocationCtx.Provide(rctx, current, func(lctx runtime.Ctx) *dom2.StructuredNode {
				return renderRouterChildren(lctx, props.Children...)
			})
		})
	})
}

func requireRouterState(ctx Ctx) routerState {
	state := RouterStateCtx.Use(ctx)
	if state.getLoc == nil || state.setLoc == nil {

		if entry := loadSessionRouterEntry(ctx); entry != nil {
			entry.mu.Lock()
			loc := entry.navigation.loc
			setter := entry.handlers.set
			entry.mu.Unlock()

			if loc.Path != "" {
				setFunc := setter
				if setFunc == nil {
					setFunc = func(Location) {}
				}
				return routerState{
					getLoc: func() Location { return cloneLocation(loc) },
					setLoc: setFunc,
				}
			}
		}

		panic(ErrMissingRouter)
	}
	return state
}

func storeSessionParams(ctx runtime.Ctx, params map[string]string) {
	if entry := ensureSessionRouterEntry(ctx); entry != nil {
		entry.mu.Lock()
		if len(params) == 0 {
			entry.params.values = nil
		} else {
			entry.params.values = copyParams(params)
		}
		entry.mu.Unlock()
	}
}
