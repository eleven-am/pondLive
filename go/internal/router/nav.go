package router

import runtime "github.com/eleven-am/pondlive/go/internal/runtime"

// NavUpdateHandler drains navigation events for NavProvider bridges.
type NavUpdateHandler interface {
	DrainNav([]NavEvent)
}

// NavDispatcher adapts RouterStore pending events to NavUpdateHandler consumers.
type NavDispatcher struct {
	store   *RouterStore
	handler NavUpdateHandler
}

func NewNavDispatcher(store *RouterStore, handler NavUpdateHandler) *NavDispatcher {
	return &NavDispatcher{store: store, handler: handler}
}

// Start registers the handler with the store and returns a cancel func.
func (d *NavDispatcher) Start() func() {
	if d == nil || d.store == nil || d.handler == nil {
		return func() {}
	}
	return d.store.RegisterNavHandler(d.handler)
}

// Drain forces a dispatch of pending events.
func (d *NavDispatcher) Drain() {
	if d == nil || d.store == nil {
		return
	}
	d.store.DrainAndDispatch()
}

type sessionNavHandler struct {
	sess *runtime.ComponentSession
}

func (h *sessionNavHandler) DrainNav(events []NavEvent) {
	if h == nil || h.sess == nil {
		return
	}
	for _, event := range events {
		msg := navEventToNavMsg(event)
		runtime.InternalEnqueueNavMessage(h.sess, msg)
	}
}

func navEventToNavMsg(event NavEvent) runtime.NavMsg {
	msg := runtime.NavMsg{
		Path: event.Target.Path,
		Q:    encodeQuery(event.Target.Query),
		Hash: event.Target.Hash,
	}
	switch event.Kind {
	case NavKindReplace:
		msg.T = "replace"
	case NavKindBack:
		msg.T = "back"
	default:
		msg.T = "nav"
	}
	return msg
}
