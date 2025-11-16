package router

import (
	"sync"

	h "github.com/eleven-am/pondlive/go/internal/html"
	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
)

// Global router state storage keyed by session
var (
	sessionStates sync.Map // *runtime.ComponentSession -> *routerSessionState
)

// Helper functions to access router state

func ensureRouterState(sess *runtime.ComponentSession) *routerSessionState {
	if sess == nil {
		return nil
	}
	if state, ok := sessionStates.Load(sess); ok {
		return state.(*routerSessionState)
	}
	state := &routerSessionState{
		entry: sessionEntry{},
	}
	sessionStates.Store(sess, state)
	return state
}

func loadRouterState(sess *runtime.ComponentSession) *routerSessionState {
	if sess == nil {
		return nil
	}
	if state, ok := sessionStates.Load(sess); ok {
		return state.(*routerSessionState)
	}
	return nil
}

// Session entry accessors (standalone functions, not methods)

func ensureSessionRouterEntry(sess *runtime.ComponentSession) *sessionEntry {
	state := ensureRouterState(sess)
	if state == nil {
		return nil
	}
	return &state.entry
}

func loadSessionRouterEntry(sess *runtime.ComponentSession) *sessionEntry {
	state := loadRouterState(sess)
	if state == nil {
		return nil
	}
	return &state.entry
}

// Link placeholder accessors

func storeLinkPlaceholder(sess *runtime.ComponentSession, frag *h.FragmentNode, node *linkNode) {
	if sess == nil || frag == nil || node == nil {
		return
	}
	if state := ensureRouterState(sess); state != nil {
		state.linkPlaceholders.Store(frag, node)
	}
}

func takeLinkPlaceholder(sess *runtime.ComponentSession, frag *h.FragmentNode) (*linkNode, bool) {
	if sess == nil || frag == nil {
		return nil, false
	}
	if state := loadRouterState(sess); state != nil {
		if value, ok := state.linkPlaceholders.LoadAndDelete(frag); ok {
			if node, okCast := value.(*linkNode); okCast {
				return node, true
			}
		}
	}
	return nil, false
}

func clearLinkPlaceholder(sess *runtime.ComponentSession, frag *h.FragmentNode) {
	if sess == nil || frag == nil {
		return
	}
	if state := loadRouterState(sess); state != nil {
		state.linkPlaceholders.Delete(frag)
	}
}

// Routes placeholder accessors

func storeRoutesPlaceholder(sess *runtime.ComponentSession, frag *h.FragmentNode, node *routesNode) {
	if sess == nil || frag == nil || node == nil {
		return
	}
	if state := ensureRouterState(sess); state != nil {
		state.routesPlaceholders.Store(frag, node)
	}
}

func takeRoutesPlaceholder(sess *runtime.ComponentSession, frag *h.FragmentNode) (*routesNode, bool) {
	if sess == nil || frag == nil {
		return nil, false
	}
	if state := loadRouterState(sess); state != nil {
		if value, ok := state.routesPlaceholders.LoadAndDelete(frag); ok {
			if node, okCast := value.(*routesNode); okCast {
				return node, true
			}
		}
	}
	return nil, false
}

func clearRoutesPlaceholder(sess *runtime.ComponentSession, frag *h.FragmentNode) {
	if sess == nil || frag == nil {
		return
	}
	if state := loadRouterState(sess); state != nil {
		state.routesPlaceholders.Delete(frag)
	}
}
