package protocol

import "github.com/eleven-am/pondlive/go/internal/diff"

type Boot struct {
	TemplatePayload
	T        string        `json:"t"`
	SID      string        `json:"sid"`
	Ver      int           `json:"ver"`
	Seq      int           `json:"seq"`
	Location Location      `json:"location"`
	Client   *ClientConfig `json:"client,omitempty"`
	Errors   []ServerError `json:"errors,omitempty"`
}

type ClientConfig struct {
	Endpoint       string `json:"endpoint,omitempty"`
	UploadEndpoint string `json:"upload,omitempty"`
	Debug          *bool  `json:"debug,omitempty"`
}

type Init struct {
	TemplatePayload
	T        string        `json:"t"`
	SID      string        `json:"sid"`
	Ver      int           `json:"ver"`
	Location Location      `json:"location"`
	Seq      int           `json:"seq"`
	Errors   []ServerError `json:"errors,omitempty"`
}

type TemplateFrame struct {
	TemplatePayload
	T     string         `json:"t"`
	SID   string         `json:"sid"`
	Ver   int            `json:"ver"`
	Scope *TemplateScope `json:"scope,omitempty"`
}

type TemplateScope struct {
	ComponentID string        `json:"componentId"`
	ParentID    string        `json:"parentId,omitempty"`
	ParentPath  []PathSegment `json:"parentPath,omitempty"`
}

type TemplatePayload struct {
	HTML           string                 `json:"html,omitempty"`
	TemplateHash   string                 `json:"templateHash,omitempty"`
	S              []string               `json:"s"`
	D              []DynamicSlot          `json:"d"`
	Slots          []SlotMeta             `json:"slots"`
	SlotPaths      []SlotPath             `json:"slotPaths,omitempty"`
	ListPaths      []ListPath             `json:"listPaths,omitempty"`
	ComponentPaths []ComponentPath        `json:"componentPaths,omitempty"`
	Handlers       map[string]HandlerMeta `json:"handlers,omitempty"`
	Bindings       TemplateBindings       `json:"bindings,omitempty"`
	Refs           RefDelta               `json:"refs,omitempty"`
}

type TemplateBindings struct {
	Slots   BindingTable    `json:"slots,omitempty"`
	Uploads []UploadBinding `json:"uploads,omitempty"`
	Refs    []RefBinding    `json:"refs,omitempty"`
	Router  []RouterBinding `json:"router,omitempty"`
}

type DynamicSlot struct {
	Kind  string            `json:"kind"`
	Text  string            `json:"text,omitempty"`
	Attrs map[string]string `json:"attrs,omitempty"`
	List  []ListRow         `json:"list,omitempty"`
}

type ListRow struct {
	Key            string           `json:"key"`
	Slots          []int            `json:"slots,omitempty"`
	Bindings       TemplateBindings `json:"bindings,omitempty"`
	SlotPaths      []SlotPath       `json:"slotPaths,omitempty"`
	ListPaths      []ListPath       `json:"listPaths,omitempty"`
	ComponentPaths []ComponentPath  `json:"componentPaths,omitempty"`
	RootCount      int              `json:"rootCount,omitempty"`
}

type PathSegment struct {
	Kind  string `json:"kind"`
	Index int    `json:"index"`
}

type SlotMeta struct {
	AnchorID int `json:"anchorId"`
}

type SlotPath struct {
	Slot           int           `json:"slot"`
	ComponentID    string        `json:"componentId"`
	Path           []PathSegment `json:"path"`
	TextChildIndex int           `json:"textChildIndex"`
}

type ListPath struct {
	Slot        int           `json:"slot"`
	ComponentID string        `json:"componentId"`
	Path        []PathSegment `json:"path,omitempty"`
	AtRoot      bool          `json:"atRoot,omitempty"`
}

type ComponentPath struct {
	ComponentID string        `json:"componentId"`
	ParentID    string        `json:"parentId,omitempty"`
	ParentPath  []PathSegment `json:"parentPath,omitempty"`
	FirstChild  []PathSegment `json:"firstChild,omitempty"`
	LastChild   []PathSegment `json:"lastChild,omitempty"`
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

type UploadBinding struct {
	ComponentID string        `json:"componentId"`
	Path        []PathSegment `json:"path,omitempty"`
	UploadID    string        `json:"uploadId"`
	Accept      []string      `json:"accept,omitempty"`
	Multiple    bool          `json:"multiple,omitempty"`
	MaxSize     int64         `json:"maxSize,omitempty"`
}

type RefBinding struct {
	ComponentID string        `json:"componentId"`
	Path        []PathSegment `json:"path,omitempty"`
	RefID       string        `json:"refId"`
}

type RouterBinding struct {
	ComponentID string        `json:"componentId"`
	Path        []PathSegment `json:"path,omitempty"`
	PathValue   string        `json:"pathValue,omitempty"`
	Query       string        `json:"query,omitempty"`
	Hash        string        `json:"hash,omitempty"`
	Replace     string        `json:"replace,omitempty"`
}

type BindingTable map[int][]SlotBinding
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

type RouterReset struct {
	T           string `json:"t"`
	SID         string `json:"sid"`
	ComponentID string `json:"componentId"`
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
	T        string            `json:"t"`
	SID      string            `json:"sid"`
	Seq      int               `json:"seq"`
	Ver      int               `json:"ver"`
	Delta    FrameDelta        `json:"delta"`
	Patch    []diff.Op         `json:"patch"`
	Effects  []any             `json:"effects"`
	Nav      *NavDelta         `json:"nav,omitempty"`
	Handlers HandlerDelta      `json:"handlers"`
	Refs     RefDelta          `json:"refs"`
	Bindings *TemplateBindings `json:"bindings,omitempty"`
	Metrics  FrameMetrics      `json:"metrics"`
}

type FrameDelta struct {
	Statics bool        `json:"statics"`
	Slots   interface{} `json:"slots"`
}

type NavDelta struct {
	Push    string `json:"push,omitempty"`
	Replace string `json:"replace,omitempty"`
	Back    bool   `json:"back,omitempty"`
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
	Handler string   `json:"handler,omitempty"`
	Listen  []string `json:"listen,omitempty"`
	Props   []string `json:"props,omitempty"`
}

type Diagnostic struct {
	T       string        `json:"t"`
	SID     string        `json:"sid"`
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details *ErrorDetails `json:"details,omitempty"`
}

type DOMRequest struct {
	T      string   `json:"t"`
	ID     string   `json:"id"`
	Ref    string   `json:"ref"`
	Props  []string `json:"props,omitempty"`
	Method string   `json:"method,omitempty"`
	Args   []any    `json:"args,omitempty"`
}

type DOMResponse struct {
	T      string         `json:"t"`
	ID     string         `json:"id"`
	Values map[string]any `json:"values,omitempty"`
	Result any            `json:"result,omitempty"`
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
