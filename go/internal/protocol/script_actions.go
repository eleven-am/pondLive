package protocol

// Script Protocol
type ScriptServerAction string
type ScriptClientAction string

const (
	// Server → Client
	ScriptSendAction ScriptServerAction = "send"

	// Client → Server
	ScriptMessageAction ScriptClientAction = "message"
)

// ScriptPayload represents a message between server and client for a specific script.
type ScriptPayload struct {
	ScriptID string      `json:"scriptId"`
	Event    string      `json:"event"`
	Data     interface{} `json:"data,omitempty"`
}

// Bus helper methods for Script operations

// PublishScriptSend publishes a server→client script message.
func (b *Bus) PublishScriptSend(scriptID, event string, data interface{}) {
	topic := Topic("script:" + scriptID)
	b.Publish(topic, string(ScriptSendAction), ScriptPayload{
		ScriptID: scriptID,
		Event:    event,
		Data:     data,
	})
}

// PublishScriptMessage publishes a client→server script message.
func (b *Bus) PublishScriptMessage(scriptID, event string, data interface{}) {
	topic := Topic("script:" + scriptID)
	b.Publish(topic, string(ScriptMessageAction), ScriptPayload{
		ScriptID: scriptID,
		Event:    event,
		Data:     data,
	})
}

// SubscribeToScript subscribes to all events for a specific script.
// The callback receives the action type and payload.
func (b *Bus) SubscribeToScript(scriptID string, callback func(action string, payload ScriptPayload)) *Subscription {
	topic := Topic("script:" + scriptID)
	return b.Subscribe(topic, func(event string, data interface{}) {
		if payload, ok := data.(ScriptPayload); ok {
			callback(event, payload)
		}
	})
}

// SubscribeToScriptMessages subscribes to client→server messages for a script.
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
