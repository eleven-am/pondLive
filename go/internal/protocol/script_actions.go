package protocol

type ScriptServerAction string
type ScriptClientAction string

const (
	ScriptSendAction ScriptServerAction = "send"

	ScriptMessageAction ScriptClientAction = "message"
)

type ScriptPayload struct {
	ScriptID string      `json:"scriptId"`
	Event    string      `json:"event"`
	Data     interface{} `json:"data,omitempty"`
}

func (b *Bus) PublishScriptSend(scriptID, event string, data interface{}) {
	topic := Topic("script:" + scriptID)
	b.Publish(topic, string(ScriptSendAction), ScriptPayload{
		ScriptID: scriptID,
		Event:    event,
		Data:     data,
	})
}

func (b *Bus) PublishScriptMessage(scriptID, event string, data interface{}) {
	topic := Topic("script:" + scriptID)
	b.Publish(topic, string(ScriptMessageAction), ScriptPayload{
		ScriptID: scriptID,
		Event:    event,
		Data:     data,
	})
}

func (b *Bus) SubscribeToScript(scriptID string, callback func(action string, payload ScriptPayload)) *Subscription {
	topic := Topic("script:" + scriptID)
	return b.Subscribe(topic, func(event string, data interface{}) {
		if payload, ok := data.(ScriptPayload); ok {
			callback(event, payload)
		}
	})
}

func (b *Bus) SubscribeToScriptMessages(scriptID string, callback func(event string, data interface{})) *Subscription {
	topic := Topic("script:" + scriptID)
	return b.Subscribe(topic, func(action string, data interface{}) {
		if action == string(ScriptMessageAction) {
			if payload, ok := data.(ScriptPayload); ok {
				callback(payload.Event, payload.Data)
			}
		}
	})
}
