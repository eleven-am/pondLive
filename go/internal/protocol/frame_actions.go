package protocol

// Frame Protocol
type FrameServerAction string

const (
	// Server → Client
	FramePatchAction FrameServerAction = "patch"
)

// Bus helper methods for Frame operations

// PublishFramePatch publishes view diff patches to the client.
func (b *Bus) PublishFramePatch(patches interface{}) {
	b.Publish(TopicFrame, string(FramePatchAction), patches)
}

// SubscribeToFramePatches subscribes to frame patch events (server → client).
// The callback receives the patches' payload.
func (b *Bus) SubscribeToFramePatches(callback func(patches interface{})) *Subscription {
	return b.Subscribe(TopicFrame, func(event string, data interface{}) {
		if event == string(FramePatchAction) {
			callback(data)
		}
	})
}
