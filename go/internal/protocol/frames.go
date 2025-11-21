package protocol

import dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"

// Boot initializes a new session with the initial DOM tree
type Boot struct {
	T        string           `json:"t"`
	SID      string           `json:"sid"`
	Ver      int              `json:"ver"`
	Seq      int              `json:"seq"`
	Patch    []dom2diff.Patch `json:"patch,omitempty"`
	Location Location         `json:"location"`
	Client   *ClientConfig    `json:"client,omitempty"`
	Errors   []ServerError    `json:"errors,omitempty"`
}

type ClientConfig struct {
	Endpoint       string `json:"endpoint,omitempty"`
	UploadEndpoint string `json:"upload,omitempty"`
	Debug          *bool  `json:"debug,omitempty"`
}

// Init reconnects to an existing session
type Init struct {
	T        string        `json:"t"`
	SID      string        `json:"sid"`
	Ver      int           `json:"ver"`
	Location Location      `json:"location"`
	Seq      int           `json:"seq"`
	Errors   []ServerError `json:"errors,omitempty"`
}

// Frame contains DOM patches and metadata updates
type Frame struct {
	T        string           `json:"t"`
	SID      string           `json:"sid"`
	Seq      int              `json:"seq"`
	Ver      int              `json:"ver"`
	Patch    []dom2diff.Patch `json:"patch"`
	Effects  []any            `json:"effects"`
	Nav      *NavDelta        `json:"nav,omitempty"`
	Handlers HandlerDelta     `json:"handlers,omitempty"`
	Refs     RefDelta         `json:"refs,omitempty"`
	Metrics  FrameMetrics     `json:"metrics"`
}

type NavDelta struct {
	Push    string `json:"push,omitempty"`
	Replace string `json:"replace,omitempty"`
	Back    bool   `json:"back,omitempty"`
}

type HandlerMeta struct {
	Event  string   `json:"event"`
	Listen []string `json:"listen,omitempty"`
	Props  []string `json:"props,omitempty"`
}

type HandlerDelta struct {
	Add map[string]HandlerMeta `json:"add,omitempty"`
	Del []string               `json:"del,omitempty"`
}

type RefMeta struct {
	Tag string `json:"tag"`
}

type RefDelta struct {
	Add map[string]RefMeta `json:"add,omitempty"`
	Del []string           `json:"del,omitempty"`
}

type UploadBinding struct {
	UploadID string   `json:"uploadId"`
	Accept   []string `json:"accept,omitempty"`
	Multiple bool     `json:"multiple,omitempty"`
	MaxSize  int64    `json:"maxSize,omitempty"`
}

type RefBinding struct {
	RefID string `json:"refId"`
}

type RouterBinding struct {
	PathValue string `json:"pathValue,omitempty"`
	Query     string `json:"query,omitempty"`
	Hash      string `json:"hash,omitempty"`
	Replace   string `json:"replace,omitempty"`
}

// Connection protocol
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

// Event protocol
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

// Error protocol
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

type Diagnostic struct {
	T       string        `json:"t"`
	SID     string        `json:"sid"`
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details *ErrorDetails `json:"details,omitempty"`
}

// DOM API protocol
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
	SID    string         `json:"sid,omitempty"`
	ID     string         `json:"id"`
	Values map[string]any `json:"values,omitempty"`
	Result any            `json:"result,omitempty"`
	Error  string         `json:"error,omitempty"`
}

// Metrics
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

// Script protocol
type ScriptEvent struct {
	T        string         `json:"t"`
	SID      string         `json:"sid"`
	ScriptID string         `json:"scriptId"`
	Event    string         `json:"event"`
	Data     map[string]any `json:"data"`
}

type ScriptMessage struct {
	T        string         `json:"t"`
	SID      string         `json:"sid"`
	ScriptID string         `json:"scriptId"`
	Data     map[string]any `json:"data"`
}
