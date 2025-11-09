package protocol

import "github.com/eleven-am/pondlive/go/internal/diff"

type Boot struct {
	T        string                     `json:"t"`
	SID      string                     `json:"sid"`
	Ver      int                        `json:"ver"`
	Seq      int                        `json:"seq"`
	HTML     string                     `json:"html"`
	S        []string                   `json:"s"`
	D        []DynamicSlot              `json:"d"`
	Slots    []SlotMeta                 `json:"slots"`
	Handlers map[string]HandlerMeta     `json:"handlers"`
	Bindings BindingTable               `json:"bindings,omitempty"`
	Markers  map[string]ComponentMarker `json:"markers,omitempty"`
	Refs     map[string]RefMeta         `json:"refs,omitempty"`
	Location Location                   `json:"location"`
	Client   *ClientConfig              `json:"client,omitempty"`
	Errors   []ServerError              `json:"errors,omitempty"`
}

type ClientConfig struct {
	Endpoint       string `json:"endpoint,omitempty"`
	UploadEndpoint string `json:"upload,omitempty"`
	Debug          *bool  `json:"debug,omitempty"`
}

type Init struct {
	T        string                     `json:"t"`
	SID      string                     `json:"sid"`
	Ver      int                        `json:"ver"`
	S        []string                   `json:"s"`
	D        []DynamicSlot              `json:"d"`
	Slots    []SlotMeta                 `json:"slots"`
	Handlers map[string]HandlerMeta     `json:"handlers"`
	Bindings BindingTable               `json:"bindings,omitempty"`
	Markers  map[string]ComponentMarker `json:"markers,omitempty"`
	Refs     map[string]RefMeta         `json:"refs,omitempty"`
	Location Location                   `json:"location"`
	Seq      int                        `json:"seq"`
	Errors   []ServerError              `json:"errors,omitempty"`
}

type DynamicSlot struct {
	Kind  string            `json:"kind"`
	Text  string            `json:"text,omitempty"`
	Attrs map[string]string `json:"attrs,omitempty"`
	List  []ListRow         `json:"list,omitempty"`
}

type ListRow struct {
	Key      string                     `json:"key"`
	Slots    []int                      `json:"slots,omitempty"`
	SlotMeta []SlotMeta                 `json:"slotMeta,omitempty"`
	Bindings BindingTable               `json:"bindings,omitempty"`
	Markers  map[string]ComponentMarker `json:"markers,omitempty"`
}

type SlotMeta struct {
	AnchorID   int   `json:"anchorId"`
	ParentPath []int `json:"parentPath,omitempty"`
	ChildIndex *int  `json:"childIndex,omitempty"`
}

type HandlerMeta struct {
	Event  string   `json:"event"`
	Listen []string `json:"listen,omitempty"`
	Props  []string `json:"props,omitempty"`
}

type SlotBinding struct {
	Event   string   `json:"event"`
	Handler string   `json:"handler"`
	Listen  []string `json:"listen,omitempty"`
	Props   []string `json:"props,omitempty"`
}

type BindingTable map[int][]SlotBinding

type ComponentMarker struct {
	ParentPath []int `json:"parentPath,omitempty"`
	Start      int   `json:"start"`
	End        int   `json:"end"`
}

type Join struct {
	T   string   `json:"t"`
	SID string   `json:"sid"`
	Ver int      `json:"ver"`
	Ack int      `json:"ack"`
	Loc Location `json:"loc"`
}

type ResumeOK struct {
	T      string        `json:"t"`
	SID    string        `json:"sid"`
	From   int           `json:"from"`
	To     int           `json:"to"`
	Errors []ServerError `json:"errors,omitempty"`
}

type ClientAck struct {
	T   string `json:"t"`
	SID string `json:"sid"`
	Seq int    `json:"seq"`
}

type ClientEvent struct {
	T       string                 `json:"t"`
	SID     string                 `json:"sid"`
	HID     string                 `json:"hid"`
	CSeq    int                    `json:"cseq"`
	Payload map[string]interface{} `json:"payload"`
}

type EventAck struct {
	T    string `json:"t"`
	SID  string `json:"sid"`
	CSeq int    `json:"cseq"`
}

type ServerError struct {
	T       string        `json:"t"`
	SID     string        `json:"sid"`
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details *ErrorDetails `json:"details,omitempty"`
}

type ErrorDetails struct {
	Phase         string         `json:"phase,omitempty"`
	ComponentID   string         `json:"componentId,omitempty"`
	ComponentName string         `json:"componentName,omitempty"`
	Hook          string         `json:"hook,omitempty"`
	HookIndex     int            `json:"hookIndex,omitempty"`
	Suggestion    string         `json:"suggestion,omitempty"`
	Stack         string         `json:"stack,omitempty"`
	Panic         string         `json:"panic,omitempty"`
	CapturedAt    string         `json:"capturedAt,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type Frame struct {
	T        string       `json:"t"`
	SID      string       `json:"sid"`
	Seq      int          `json:"seq"`
	Ver      int          `json:"ver"`
	Delta    FrameDelta   `json:"delta"`
	Patch    []diff.Op    `json:"patch"`
	Effects  []any        `json:"effects"`
	Nav      *NavDelta    `json:"nav,omitempty"`
	Handlers HandlerDelta `json:"handlers"`
	Refs     RefDelta     `json:"refs"`
	Metrics  FrameMetrics `json:"metrics"`
}

type FrameDelta struct {
	Statics bool        `json:"statics"`
	Slots   interface{} `json:"slots"`
}

type NavDelta struct {
	Push    string `json:"push,omitempty"`
	Replace string `json:"replace,omitempty"`
}

type HandlerDelta struct {
	Add map[string]HandlerMeta `json:"add,omitempty"`
	Del []string               `json:"del,omitempty"`
}

type RefDelta struct {
	Add map[string]RefMeta `json:"add,omitempty"`
	Del []string           `json:"del,omitempty"`
}

type RefMeta struct {
	Tag    string                  `json:"tag"`
	Events map[string]RefEventMeta `json:"events,omitempty"`
}

type RefEventMeta struct {
	Listen []string `json:"listen,omitempty"`
	Props  []string `json:"props,omitempty"`
}

type DOMRequest struct {
	T     string   `json:"t"`
	ID    string   `json:"id"`
	Ref   string   `json:"ref"`
	Props []string `json:"props,omitempty"`
}

type DOMResponse struct {
	T      string         `json:"t"`
	ID     string         `json:"id"`
	Values map[string]any `json:"values,omitempty"`
	Error  string         `json:"error,omitempty"`
}

type FrameMetrics struct {
	RenderMs    float64 `json:"renderMs"`
	Ops         int     `json:"ops"`
	EffectsMs   float64 `json:"effectsMs,omitempty"`
	MaxEffectMs float64 `json:"maxEffectMs,omitempty"`
	SlowEffects int     `json:"slowEffects,omitempty"`
}

type Location struct {
	Path  string `json:"path"`
	Query string `json:"q"`
	Hash  string `json:"hash"`
}
