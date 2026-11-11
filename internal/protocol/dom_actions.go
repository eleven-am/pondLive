package protocol

type DOMServerAction string
type DOMClientAction string

const (
	DOMCallAction  DOMServerAction = "call"
	DOMSetAction   DOMServerAction = "set"
	DOMQueryAction DOMServerAction = "query"
	DOMAsyncAction DOMServerAction = "async"

	DOMObserveAction   DOMServerAction = "observe"
	DOMUnobserveAction DOMServerAction = "unobserve"

	DOMResponseAction DOMClientAction = "response"
	DOMValueAction    DOMClientAction = "value"
)

type DOMCallPayload struct {
	K      string `json:"k"`
	ID     uint32 `json:"id"`
	Method string `json:"method"`
	Args   []any  `json:"args,omitempty"`
}

type DOMSetPayload struct {
	K     string `json:"k"`
	ID    uint32 `json:"id"`
	Prop  string `json:"prop"`
	Value any    `json:"value"`
}

type DOMQueryPayload struct {
	K         string   `json:"k"`
	RequestID string   `json:"requestId"`
	ID        uint32   `json:"id"`
	Selectors []string `json:"selectors"`
}

type DOMAsyncPayload struct {
	K         string `json:"k"`
	RequestID string `json:"requestId"`
	ID        uint32 `json:"id"`
	Method    string `json:"method"`
	Args      []any  `json:"args,omitempty"`
}

type DOMObservePayload struct {
	K    string `json:"k"`
	ID   uint32 `json:"id"`
	Prop string `json:"prop"`
	Ms   int    `json:"ms"`
}

type DOMUnobservePayload struct {
	ID   uint32 `json:"id"`
	Prop string `json:"prop"`
}

type DOMValuePayload struct {
	ID    uint32 `json:"id"`
	Prop  string `json:"prop"`
	Value any    `json:"value"`
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

func (b *Bus) PublishDOMObserve(payload DOMObservePayload) {
	b.Publish(DOMHandler, string(DOMObserveAction), payload)
}

func (b *Bus) PublishDOMUnobserve(payload DOMUnobservePayload) {
	b.Publish(DOMHandler, string(DOMUnobserveAction), payload)
}

func (b *Bus) SubscribeToDOMValues(callback func(value DOMValuePayload)) *Subscription {
	return b.Subscribe(DOMHandler, func(event string, data interface{}) {
		if event == string(DOMValueAction) {
			if v, ok := DecodePayload[DOMValuePayload](data); ok {
				callback(v)
			}
		}
	})
}

func (b *Bus) SubscribeToDOMActions(callback func(action DOMServerAction, data interface{})) *Subscription {
	return b.Upsert(DOMHandler, func(event string, data interface{}) {
		switch DOMServerAction(event) {
		case DOMCallAction, DOMSetAction, DOMQueryAction, DOMAsyncAction, DOMObserveAction, DOMUnobserveAction:
			callback(DOMServerAction(event), data)
		}
	})
}

func (b *Bus) SubscribeToDOMResponses(callback func(response DOMResponsePayload)) *Subscription {
	return b.Subscribe(DOMHandler, func(event string, data interface{}) {
		if event == string(DOMResponseAction) {
			if resp, ok := DecodePayload[DOMResponsePayload](data); ok {
				callback(resp)
			}
		}
	})
}
