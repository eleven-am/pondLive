package protocol

// DOM Protocol
type DOMServerAction string
type DOMClientAction string

const (
	DOMCallAction  DOMServerAction = "call"
	DOMSetAction   DOMServerAction = "set"
	DOMQueryAction DOMServerAction = "query"
	DOMAsyncAction DOMServerAction = "async"

	DOMResponseAction DOMClientAction = "response"
)

// DOMCallPayload represents a fire-and-forget method call on an element.
type DOMCallPayload struct {
	Ref    string `json:"ref"`
	Method string `json:"method"`
	Args   []any  `json:"args,omitempty"`
}

// DOMSetPayload represents a property assignment on an element.
type DOMSetPayload struct {
	Ref   string `json:"ref"`
	Prop  string `json:"prop"`
	Value any    `json:"value"`
}

// DOMQueryPayload represents a query request for element properties.
type DOMQueryPayload struct {
	RequestID string   `json:"requestId"`
	Ref       string   `json:"ref"`
	Selectors []string `json:"selectors"`
}

// DOMAsyncPayload represents an async method call request.
type DOMAsyncPayload struct {
	RequestID string `json:"requestId"`
	Ref       string `json:"ref"`
	Method    string `json:"method"`
	Args      []any  `json:"args,omitempty"`
}

// DOMResponsePayload represents a response from the client.
type DOMResponsePayload struct {
	RequestID string         `json:"requestId"`
	Values    map[string]any `json:"values,omitempty"`
	Result    any            `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// Bus helper methods for DOM operations

// PublishDOMCall publishes a fire-and-forget DOM method call.
func (b *Bus) PublishDOMCall(payload DOMCallPayload) {
	b.Publish(DOMHandler, string(DOMCallAction), payload)
}

// PublishDOMSet publishes a DOM property set operation.
func (b *Bus) PublishDOMSet(payload DOMSetPayload) {
	b.Publish(DOMHandler, string(DOMSetAction), payload)
}

// PublishDOMQuery publishes a DOM query request that expects a response.
func (b *Bus) PublishDOMQuery(payload DOMQueryPayload) {
	b.Publish(DOMHandler, string(DOMQueryAction), payload)
}

// PublishDOMAsync publishes an async DOM method call that expects a response.
func (b *Bus) PublishDOMAsync(payload DOMAsyncPayload) {
	b.Publish(DOMHandler, string(DOMAsyncAction), payload)
}

// SubscribeToDOMActions subscribes to all DOM actions (server → client commands).
// The callback receives the action type and the payload.
func (b *Bus) SubscribeToDOMActions(callback func(action DOMServerAction, data interface{})) *Subscription {
	return b.Subscribe(DOMHandler, func(event string, data interface{}) {
		switch DOMServerAction(event) {
		case DOMCallAction, DOMSetAction, DOMQueryAction, DOMAsyncAction:
			callback(DOMServerAction(event), data)
		}
	})
}

// SubscribeToDOMResponses subscribes to DOM responses (client → server).
// The callback receives the response payload.
func (b *Bus) SubscribeToDOMResponses(callback func(response DOMResponsePayload)) *Subscription {
	return b.Subscribe(DOMHandler, func(event string, data interface{}) {
		if event == string(DOMResponseAction) {
			if resp, ok := data.(DOMResponsePayload); ok {
				callback(resp)
			}
		}
	})
}
