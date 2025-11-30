package protocol

type FrameServerAction string

const (
	FramePatchAction FrameServerAction = "patch"
)

func (b *Bus) PublishFramePatch(patches interface{}) {
	b.Publish(TopicFrame, string(FramePatchAction), patches)
}

func (b *Bus) SubscribeToFramePatches(callback func(patches interface{})) *Subscription {
	return b.Subscribe(TopicFrame, func(event string, data interface{}) {
		if event == string(FramePatchAction) {
			callback(data)
		}
	})
}
