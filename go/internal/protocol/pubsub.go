package protocol

// PubsubControl instructs the client transport to join or leave a pubsub topic.
type PubsubControl struct {
	T     string `json:"t"`
	Op    string `json:"op"`
	Topic string `json:"topic"`
}
