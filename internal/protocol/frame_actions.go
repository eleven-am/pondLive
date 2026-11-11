package protocol

type FrameServerAction string

const (
	FramePatchAction FrameServerAction = "patch"
	FrameResetAction FrameServerAction = "reset"
)

func (b *Bus) PublishFrameEdits(edits []Edit) {
	b.Publish(TopicFrame, string(FramePatchAction), edits)
}

func (b *Bus) SubscribeToFrameEdits(callback func(edits []Edit)) *Subscription {
	return b.Upsert(TopicFrame, func(event string, data interface{}) {
		if event != string(FramePatchAction) {
			return
		}
		if edits, ok := data.([]Edit); ok {
			callback(edits)
		}
	})
}
