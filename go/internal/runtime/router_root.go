package runtime

import (
	"sync"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type routerState struct {
	getLoc func() Location
	setLoc func(Location)
}

var routerStateCtx = NewContext(routerState{})

type sessionEntry struct {
	mu          sync.Mutex
	get         func() Location
	set         func(Location)
	assign      func(Location)
	loc         Location
	navs        []NavMsg
	pendingNavs []NavMsg
	params      map[string]string
	active      bool
	pattern     string
	routeDepth  int
}

var sessionEntries sync.Map // key: *ComponentSession -> *sessionEntry

type routerProps struct {
	Children []h.Node
}

func Router(ctx Ctx, children ...h.Node) h.Node {
	return Render(ctx, routerComponent, routerProps{Children: children})
}

func routerComponent(ctx Ctx, props routerProps) h.Node {
	sess := ctx.Session()
	initial := initialLocation(sess)
	get, set := UseState(ctx, initial, WithEqual(LocEqual))
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

func registerSessionEntry(sess *ComponentSession, get func() Location, set func(Location), assign func(Location)) *sessionEntry {
	if sess == nil {
		return nil
	}
	entry := &sessionEntry{}
	actual, _ := sessionEntries.LoadOrStore(sess, entry)
	stored := actual.(*sessionEntry)
	stored.mu.Lock()
	stored.get = get
	stored.set = set
	stored.assign = assign
	stored.mu.Unlock()
	return stored
}

func requestTemplateReset(sess *ComponentSession) {
	if sess == nil {
		return
	}
	sess.requestTemplateReset()
}

func setSessionRendering(sess *ComponentSession, active bool) {
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

func sessionRendering(sess *ComponentSession) bool {
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

func storeSessionLocation(sess *ComponentSession, loc Location) {
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
	if owner := sess.owner; owner != nil {
		owner.SetRoute(canon.Path, encodeQuery(canon.Query), nil)
	}
}

func currentSessionLocation(sess *ComponentSession) Location {
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

func initialLocation(sess *ComponentSession) Location {
	if loc, ok := consumeSeed(sess); ok {
		return canonicalizeLocation(loc)
	}
	return currentSessionLocation(sess)
}

func storeSessionParams(sess *ComponentSession, params map[string]string) {
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

func sessionParams(sess *ComponentSession) map[string]string {
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
func InternalSeedSessionParams(sess *ComponentSession, params map[string]string) {
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
	Children []h.Node
}

func renderRouterChildren(ctx Ctx, children ...h.Node) h.Node {
	return Render(ctx, routerChildrenComponent, routerChildrenProps{Children: children})
}

func routerChildrenComponent(ctx Ctx, props routerChildrenProps) h.Node {
	if len(props.Children) == 0 {
		return h.Fragment()
	}
	normalized := make([]h.Node, 0, len(props.Children))
	for _, child := range props.Children {
		normalized = append(normalized, normalizeRouterNode(ctx, child))
	}
	return h.Fragment(normalized...)
}

func normalizeRouterNode(ctx Ctx, node h.Node) h.Node {
	if node == nil {
		return nil
	}
	switch v := node.(type) {
	case *routesNode:
		routesPlaceholders.Delete(v.FragmentNode)
		return normalizeRouterNode(ctx, renderRoutes(ctx, v.entries))
	case *linkNode:
		linkPlaceholders.Delete(v.FragmentNode)
		return renderLink(ctx, v.props, v.children...)
	case *h.Element:
		if v == nil || len(v.Children) == 0 || v.Unsafe != nil {
			return node
		}
		children := v.Children
		updated := make([]h.Node, len(children))
		changed := false
		for i, child := range children {
			normalized := normalizeRouterNode(ctx, child)
			if normalized != child {
				changed = true
			}
			updated[i] = normalized
		}
		if !changed {
			return node
		}
		clone := *v
		clone.Children = updated
		return &clone
	case *h.FragmentNode:
		if placeholder, ok := consumeLinkPlaceholder(v); ok {
			return renderLink(ctx, placeholder.props, placeholder.children...)
		}
		if placeholder, ok := consumeRoutesPlaceholder(v); ok {
			return normalizeRouterNode(ctx, renderRoutes(ctx, placeholder.entries))
		}
		if v == nil || len(v.Children) == 0 {
			return node
		}
		children := v.Children
		updated := make([]h.Node, len(children))
		changed := false
		for i, child := range children {
			normalized := normalizeRouterNode(ctx, child)
			if normalized != child {
				changed = true
			}
			updated[i] = normalized
		}
		if !changed {
			return node
		}
		return h.Fragment(updated...)
	default:
		return node
	}
}
