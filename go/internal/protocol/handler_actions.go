package protocol

type HandlerClientAction string

const (
	HandlerInvokeAction HandlerClientAction = "invoke"
)

func (b *Bus) PublishHandlerInvoke(handlerID string, event interface{}) {
	topic := Topic(handlerID)
	b.Publish(topic, string(HandlerInvokeAction), event)
}

func (b *Bus) SubscribeToHandlerInvoke(handlerID string, callback func(event interface{})) *Subscription {
	topic := Topic(handlerID)
	return b.Upsert(topic, func(action string, data interface{}) {
		if action == string(HandlerInvokeAction) {
			callback(data)
		}
	})
}
