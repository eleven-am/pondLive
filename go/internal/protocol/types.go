package protocol

type Frame struct {
	Handler Topic       `json:"handler"`
	Action  string      `json:"action"`
	Payload interface{} `json:"payload,omitempty"`
}

type Topic string

const (
	RouteHandler Topic = "router"
	DOMHandler   Topic = "dom"
	TopicFrame   Topic = "frame"
)
