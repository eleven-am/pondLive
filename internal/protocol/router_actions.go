package protocol

type RouterServerAction string
type RouterClientAction string

const (
	RouterPushAction     RouterServerAction = "push"
	RouterReplaceAction  RouterServerAction = "replace"
	RouterBackAction     RouterServerAction = "back"
	RouterForwardAction  RouterServerAction = "forward"
	RouterRedirectAction RouterServerAction = "redirect"

	RouterPopstateAction RouterClientAction = "popstate"
)

type RouterNavPayload struct {
	Path    string `json:"path"`
	Query   string `json:"query"`
	Hash    string `json:"hash"`
	Replace bool   `json:"replace"`
}

func (b *Bus) PublishRouterPush(payload RouterNavPayload) {
	b.Publish(RouteHandler, string(RouterPushAction), payload)
}

func (b *Bus) PublishRouterReplace(payload RouterNavPayload) {
	b.Publish(RouteHandler, string(RouterReplaceAction), payload)
}

type RouterPopstatePayload struct {
	Path  string `json:"path"`
	Query string `json:"query"`
	Hash  string `json:"hash"`
}

type RouterRedirectPayload struct {
	URL string `json:"url"`
}

func (b *Bus) PublishRouterRedirect(payload RouterRedirectPayload) {
	b.Publish(RouteHandler, string(RouterRedirectAction), payload)
}

func (b *Bus) PublishRouterBack() {
	b.Publish(RouteHandler, string(RouterBackAction), nil)
}

func (b *Bus) PublishRouterForward() {
	b.Publish(RouteHandler, string(RouterForwardAction), nil)
}

func (b *Bus) SubscribeToRouterCommands(callback func(action RouterServerAction, data interface{})) *Subscription {
	return b.Upsert(RouteHandler, func(event string, data interface{}) {
		switch RouterServerAction(event) {
		case RouterPushAction, RouterReplaceAction, RouterBackAction, RouterForwardAction, RouterRedirectAction:
			callback(RouterServerAction(event), data)
		}
	})
}

func (b *Bus) SubscribeToRouterPopstate(callback func(payload RouterPopstatePayload)) *Subscription {
	return b.Upsert(RouteHandler, func(event string, data interface{}) {
		if event == string(RouterPopstateAction) {
			if payload, ok := DecodePayload[RouterPopstatePayload](data); ok {
				callback(payload)
			}
		}
	})
}
