package router

import (
	"sync"

	h "github.com/eleven-am/pondlive/go/internal/html"
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

var routerStateCtx = runtime.NewContext(routerState{})

type sessionEntry struct {
	mu sync.Mutex

	handlers   sessionHandlers
	navigation sessionNavigation
	params     sessionParamStore
	render     sessionRenderState
}

type routerSessionState struct {
	entry              sessionEntry
	linkPlaceholders   sync.Map // *h.FragmentNode -> *linkNode
	routesPlaceholders sync.Map // *h.FragmentNode -> *routesNode
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
	Children []h.Node
}

func Router(ctx Ctx, children ...h.Node) h.Node {
	return runtime.Render(ctx, routerComponent, routerProps{Children: children})
}

func routerComponent(ctx Ctx, props routerProps) h.Node {
	sess := ctx.Session()
	initial := initialLocation(sess)
	get, set := runtime.UseState(ctx, initial, runtime.WithEqual(LocEqual))
	assign := func(next Location) {
		canon := canonicalizeLocation(next)
		set(canon)
	}

	state := routerState{}
	state.getLoc = func() Location { return cloneLocation(get()) }
	state.setLoc = func(next Location) {
		canon := canonicalizeLocation(next)
		set(canon)
		storeSessionLocation(sess, canon)
	}

	entry := registerSessionEntry(sess, state.getLoc, state.setLoc, assign)
	current := state.getLoc()
	storeSessionLocation(sess, current)
	if entry != nil {
		setSessionRendering(sess, true)
		defer setSessionRendering(sess, false)
	}
	return routerStateCtx.Provide(ctx, state, func() h.Node {
		return LocationCtx.Provide(ctx, current, func() h.Node {
			return renderRouterChildren(ctx, props.Children...)
		})
	})
}

func requireRouterState(ctx Ctx) routerState {
	state := routerStateCtx.Use(ctx)
	if state.getLoc == nil || state.setLoc == nil {
		if sess := ctx.Session(); sess != nil {
			if entry := loadSessionRouterEntry(sess); entry != nil {
				entry.mu.Lock()
				loc := entry.navigation.loc
				setter := entry.handlers.set
				entry.mu.Unlock()
				if setter != nil {
					return routerState{
						getLoc: func() Location { return cloneLocation(loc) },
						setLoc: setter,
					}
				}
			}
		}
		panic(ErrMissingRouter)
	}
	return state
}

func registerSessionEntry(sess *runtime.ComponentSession, get func() Location, set func(Location), assign func(Location)) *sessionEntry {
	if sess == nil {
		return nil
	}
	entry := ensureSessionRouterEntry(sess)
	entry.mu.Lock()
	entry.handlers.get = get
	entry.handlers.set = set
	entry.handlers.assign = assign
	entry.mu.Unlock()
	return entry
}

func requestTemplateReset(sess *runtime.ComponentSession) {
	if sess == nil {
		return
	}
	sess.RequestTemplateReset()
}

func setSessionRendering(sess *runtime.ComponentSession, active bool) {
	if sess == nil {
		return
	}
	if entry := loadSessionRouterEntry(sess); entry != nil {
		entry.mu.Lock()
		entry.render.active = active
		entry.mu.Unlock()
	}
}

func sessionRendering(sess *runtime.ComponentSession) bool {
	if sess == nil {
		return false
	}
	if entry := loadSessionRouterEntry(sess); entry != nil {
		entry.mu.Lock()
		defer entry.mu.Unlock()
		return entry.render.active
	}
	return false
}

func storeSessionLocation(sess *runtime.ComponentSession, loc Location) {
	if sess == nil {
		return
	}
	canon := canonicalizeLocation(loc)
	if entry := ensureSessionRouterEntry(sess); entry != nil {
		entry.mu.Lock()
		entry.navigation.loc = canon
		entry.params.values = nil
		entry.mu.Unlock()
	}
	if owner := sess.Owner(); owner != nil {
		owner.SetRoute(canon.Path, encodeQuery(canon.Query), nil)
	}
}

func currentSessionLocation(sess *runtime.ComponentSession) Location {
	if sess == nil {
		return canonicalizeLocation(Location{Path: "/"})
	}
	if entry := loadSessionRouterEntry(sess); entry != nil {
		entry.mu.Lock()
		loc := entry.navigation.loc
		entry.mu.Unlock()
		if loc.Path != "" {
			return canonicalizeLocation(loc)
		}
	}
	return canonicalizeLocation(Location{Path: "/"})
}

func initialLocation(sess *runtime.ComponentSession) Location {
	if loc, ok := consumeSeed(sess); ok {
		return canonicalizeLocation(loc)
	}
	return currentSessionLocation(sess)
}

func storeSessionParams(sess *runtime.ComponentSession, params map[string]string) {
	if sess == nil {
		return
	}
	if entry := ensureSessionRouterEntry(sess); entry != nil {
		entry.mu.Lock()
		if len(params) == 0 {
			entry.params.values = nil
		} else {
			entry.params.values = copyParams(params)
		}
		entry.mu.Unlock()
	}
}

func sessionParams(sess *runtime.ComponentSession) map[string]string {
	if sess == nil {
		return nil
	}
	if entry := loadSessionRouterEntry(sess); entry != nil {
		entry.mu.Lock()
		defer entry.mu.Unlock()
		if len(entry.params.values) == 0 {
			return map[string]string{}
		}
		return copyParams(entry.params.values)
	}
	return map[string]string{}
}

// SeedSessionParams pre-populates the parameter map used by UseParams during hydration.
// InternalSeedSessionParams records route params during SSR boot. Internal use only.
func InternalSeedSessionParams(sess *runtime.ComponentSession, params map[string]string) {
	if sess == nil {
		return
	}
	entry := ensureSessionRouterEntry(sess)
	entry.mu.Lock()
	if len(params) == 0 {
		entry.params.values = nil
	} else {
		entry.params.values = copyParams(params)
	}
	entry.mu.Unlock()
}
