package router

import (
	"sync"

	ui "github.com/eleven-am/go/pondlive/pkg/live"
	h "github.com/eleven-am/go/pondlive/pkg/live/html"
)

type routerState struct {
	getLoc func() Location
	setLoc func(Location)
}

var routerStateCtx = ui.NewContext(routerState{})

type sessionEntry struct {
	mu     sync.Mutex
	get    func() Location
	set    func(Location)
	loc    Location
	navs   []NavMsg
	params map[string]string
	active bool
}

var sessionEntries sync.Map // key: *ui.Session -> *sessionEntry

type routerProps struct {
	Children []ui.Node
}

func Router(ctx ui.Ctx, children ...ui.Node) ui.Node {
	return ui.Render(ctx, routerComponent, routerProps{Children: children})
}

func routerComponent(ctx ui.Ctx, props routerProps) ui.Node {
	sess := ctx.Session()
	initial := initialLocation(sess)
	get, set := ui.UseState(ctx, initial, ui.WithEqual(LocEqual))

	state := routerState{}
	state.getLoc = func() Location { return cloneLocation(get()) }
	state.setLoc = func(next Location) {
		canon := canonicalizeLocation(next)
		set(canon)
		storeSessionLocation(sess, canon)
	}

	entry := registerSessionEntry(sess, state.getLoc, state.setLoc)
	current := state.getLoc()
	storeSessionLocation(sess, current)
	_ = routerStateCtx.Provide(ctx, state)
	_ = LocationCtx.Provide(ctx, current)
	if entry != nil {
		setSessionRendering(sess, true)
		defer setSessionRendering(sess, false)
	}
	return renderRouterChildren(ctx, props.Children...)
}

func requireRouterState(ctx ui.Ctx) routerState {
	state := routerStateCtx.Use(ctx)
	if state.getLoc == nil || state.setLoc == nil {
		if sess := ctx.Session(); sess != nil {
			if v, ok := sessionEntries.Load(sess); ok {
				entry := v.(*sessionEntry)
				entry.mu.Lock()
				loc := entry.loc
				setter := entry.set
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

func registerSessionEntry(sess *ui.Session, get func() Location, set func(Location)) *sessionEntry {
	if sess == nil {
		return nil
	}
	entry := &sessionEntry{}
	actual, _ := sessionEntries.LoadOrStore(sess, entry)
	stored := actual.(*sessionEntry)
	stored.mu.Lock()
	stored.get = get
	stored.set = set
	stored.mu.Unlock()
	return stored
}

func setSessionRendering(sess *ui.Session, active bool) {
	if sess == nil {
		return
	}
	if v, ok := sessionEntries.Load(sess); ok {
		entry := v.(*sessionEntry)
		entry.mu.Lock()
		entry.active = active
		entry.mu.Unlock()
	}
}

func sessionRendering(sess *ui.Session) bool {
	if sess == nil {
		return false
	}
	if v, ok := sessionEntries.Load(sess); ok {
		entry := v.(*sessionEntry)
		entry.mu.Lock()
		defer entry.mu.Unlock()
		return entry.active
	}
	return false
}

func storeSessionLocation(sess *ui.Session, loc Location) {
	if sess == nil {
		return
	}
	canon := canonicalizeLocation(loc)
	if v, ok := sessionEntries.Load(sess); ok {
		entry := v.(*sessionEntry)
		entry.mu.Lock()
		entry.loc = canon
		entry.params = nil
		entry.mu.Unlock()
	}
}

func currentSessionLocation(sess *ui.Session) Location {
	if sess == nil {
		return canonicalizeLocation(Location{Path: "/"})
	}
	if v, ok := sessionEntries.Load(sess); ok {
		entry := v.(*sessionEntry)
		entry.mu.Lock()
		loc := entry.loc
		entry.mu.Unlock()
		if loc.Path != "" {
			return canonicalizeLocation(loc)
		}
	}
	return canonicalizeLocation(Location{Path: "/"})
}

func initialLocation(sess *ui.Session) Location {
	if loc, ok := consumeSeed(sess); ok {
		return canonicalizeLocation(loc)
	}
	return currentSessionLocation(sess)
}

func storeSessionParams(sess *ui.Session, params map[string]string) {
	if sess == nil {
		return
	}
	if v, ok := sessionEntries.Load(sess); ok {
		entry := v.(*sessionEntry)
		entry.mu.Lock()
		if len(params) == 0 {
			entry.params = nil
		} else {
			entry.params = copyParams(params)
		}
		entry.mu.Unlock()
	}
}

func sessionParams(sess *ui.Session) map[string]string {
	if sess == nil {
		return nil
	}
	if v, ok := sessionEntries.Load(sess); ok {
		entry := v.(*sessionEntry)
		entry.mu.Lock()
		defer entry.mu.Unlock()
		if len(entry.params) == 0 {
			return map[string]string{}
		}
		return copyParams(entry.params)
	}
	return map[string]string{}
}

// SeedSessionParams pre-populates the parameter map used by UseParams during hydration.
// InternalSeedSessionParams records route params during SSR boot. Internal use only.
func InternalSeedSessionParams(sess *ui.Session, params map[string]string) {
	if sess == nil {
		return
	}
	actual, _ := sessionEntries.LoadOrStore(sess, &sessionEntry{})
	entry := actual.(*sessionEntry)
	entry.mu.Lock()
	if len(params) == 0 {
		entry.params = nil
	} else {
		entry.params = copyParams(params)
	}
	entry.mu.Unlock()
}

type routerChildrenProps struct {
	Children []ui.Node
}

func renderRouterChildren(ctx ui.Ctx, children ...ui.Node) ui.Node {
	return ui.Render(ctx, routerChildrenComponent, routerChildrenProps{Children: children})
}

func routerChildrenComponent(ctx ui.Ctx, props routerChildrenProps) ui.Node {
	if len(props.Children) == 0 {
		return h.Fragment()
	}
	normalized := make([]ui.Node, 0, len(props.Children))
	for _, child := range props.Children {
		switch v := child.(type) {
		case *routesNode:
			normalized = append(normalized, renderRoutes(ctx, v.entries))
		case *linkNode:
			normalized = append(normalized, renderLink(ctx, v.props))
		default:
			normalized = append(normalized, child)
		}
	}
	return h.Fragment(normalized...)
}
