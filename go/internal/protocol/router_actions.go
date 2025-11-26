package protocol

// Router Protocol
type RouterServerAction string
type RouterClientAction string

const (
	// Server → Client commands
	RouterPushAction    RouterServerAction = "push"
	RouterReplaceAction RouterServerAction = "replace"
	RouterBackAction    RouterServerAction = "back"
	RouterForwardAction RouterServerAction = "forward"

	// Client → Server events
	RouterPopstateAction RouterClientAction = "popstate"
)

// RouterNavPayload represents navigation data sent between server and client.
type RouterNavPayload struct {
	Path    string `json:"path"`
	Query   string `json:"query"`
	Hash    string `json:"hash"`
	Replace bool   `json:"replace"`
}

// Bus helper methods for Router operations

// PublishRouterPush publishes a push history command to the client.
func (b *Bus) PublishRouterPush(payload RouterNavPayload) {
	b.Publish(RouteHandler, string(RouterPushAction), payload)
}

// PublishRouterReplace publishes a replace history command to the client.
func (b *Bus) PublishRouterReplace(payload RouterNavPayload) {
	b.Publish(RouteHandler, string(RouterReplaceAction), payload)
}

// PublishRouterBack publishes a back history command to the client.
func (b *Bus) PublishRouterBack() {
	b.Publish(RouteHandler, string(RouterBackAction), nil)
}

// PublishRouterForward publishes a forward history command to the client.
func (b *Bus) PublishRouterForward() {
	b.Publish(RouteHandler, string(RouterForwardAction), nil)
}

// SubscribeToRouterCommands subscribes to all router commands (server → client).
// The callback receives the action type and the payload (if any).
func (b *Bus) SubscribeToRouterCommands(callback func(action RouterServerAction, data interface{})) *Subscription {
	return b.Subscribe(RouteHandler, func(event string, data interface{}) {
		switch RouterServerAction(event) {
		case RouterPushAction, RouterReplaceAction, RouterBackAction, RouterForwardAction:
			callback(RouterServerAction(event), data)
		}
	})
}

// SubscribeToRouterPopstate subscribes to popstate events (client → server).
// The callback receives the navigation payload.
func (b *Bus) SubscribeToRouterPopstate(callback func(payload RouterNavPayload)) *Subscription {
	return b.Subscribe(RouteHandler, func(event string, data interface{}) {
		if event == string(RouterPopstateAction) {
			if payload, ok := data.(RouterNavPayload); ok {
				callback(payload)
			}
		}
	})
}
