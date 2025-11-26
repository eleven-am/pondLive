package protocol

// Handler Protocol
type HandlerClientAction string

const (
	// Client → Server
	HandlerInvokeAction HandlerClientAction = "invoke"
)

// Bus helper methods for Handler operations

// PublishHandlerInvoke publishes an event handler invocation from the client.
func (b *Bus) PublishHandlerInvoke(handlerID string, event interface{}) {
	topic := Topic(handlerID)
	b.Publish(topic, string(HandlerInvokeAction), event)
}

// SubscribeToHandlerInvoke subscribes to handler invocations for a specific handler ID.
// The callback receives the event data.
func (b *Bus) SubscribeToHandlerInvoke(handlerID string, callback func(event interface{})) *Subscription {
	topic := Topic(handlerID)
	return b.Subscribe(topic, func(action string, data interface{}) {
		if action == string(HandlerInvokeAction) {
			callback(data)
		}
	})
}
