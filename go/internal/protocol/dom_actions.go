package protocol

type DOMServerAction string
type DOMClientAction string

const (
	DOMCallAction  DOMServerAction = "call"
	DOMSetAction   DOMServerAction = "set"
	DOMQueryAction DOMServerAction = "query"
	DOMAsyncAction DOMServerAction = "async"

	DOMResponseAction DOMClientAction = "response"
)

type DOMCallPayload struct {
	Ref    string `json:"ref"`
	Method string `json:"method"`
	Args   []any  `json:"args,omitempty"`
}

type DOMSetPayload struct {
	Ref   string `json:"ref"`
	Prop  string `json:"prop"`
	Value any    `json:"value"`
}

type DOMQueryPayload struct {
	RequestID string   `json:"requestId"`
	Ref       string   `json:"ref"`
	Selectors []string `json:"selectors"`
}

type DOMAsyncPayload struct {
	RequestID string `json:"requestId"`
	Ref       string `json:"ref"`
	Method    string `json:"method"`
	Args      []any  `json:"args,omitempty"`
}

type DOMResponsePayload struct {
	RequestID string         `json:"requestId"`
	Values    map[string]any `json:"values,omitempty"`
	Result    any            `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
}

func (b *Bus) PublishDOMCall(payload DOMCallPayload) {
	b.Publish(DOMHandler, string(DOMCallAction), payload)
}

func (b *Bus) PublishDOMSet(payload DOMSetPayload) {
	b.Publish(DOMHandler, string(DOMSetAction), payload)
}

func (b *Bus) PublishDOMQuery(payload DOMQueryPayload) {
	b.Publish(DOMHandler, string(DOMQueryAction), payload)
}

func (b *Bus) PublishDOMAsync(payload DOMAsyncPayload) {
	b.Publish(DOMHandler, string(DOMAsyncAction), payload)
}

func (b *Bus) SubscribeToDOMActions(callback func(action DOMServerAction, data interface{})) *Subscription {
	return b.Subscribe(DOMHandler, func(event string, data interface{}) {
		switch DOMServerAction(event) {
		case DOMCallAction, DOMSetAction, DOMQueryAction, DOMAsyncAction:
			callback(DOMServerAction(event), data)
		}
	})
}

func (b *Bus) SubscribeToDOMResponses(callback func(response DOMResponsePayload)) *Subscription {
	return b.Subscribe(DOMHandler, func(event string, data interface{}) {
		if event == string(DOMResponseAction) {
			if resp, ok := data.(DOMResponsePayload); ok {
				callback(resp)
			}
		}
	})
}
