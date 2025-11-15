package router2

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
