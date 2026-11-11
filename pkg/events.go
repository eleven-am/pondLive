package pkg

import "github.com/eleven-am/pondlive/internal/work"

type BaseEvent struct {
	work.Event
}

func (b *BaseEvent) setEvent(e work.Event) { b.Event = e }

type TouchPoint struct {
	ID               int
	ClientX, ClientY float64
}

type FileMeta struct {
	Name         string
	Size         int64
	Type         string
	LastModified int64
}

type ClickEvent struct {
	BaseEvent
	Detail   int     `ev:"detail"`
	Button   int     `ev:"button"`
	Buttons  int     `ev:"buttons,opt"`
	ClientX  float64 `ev:"clientX"`
	ClientY  float64 `ev:"clientY"`
	OffsetX  float64 `ev:"offsetX"`
	OffsetY  float64 `ev:"offsetY"`
	PageX    float64 `ev:"pageX,opt"`
	PageY    float64 `ev:"pageY,opt"`
	ScreenX  float64 `ev:"screenX,opt"`
	ScreenY  float64 `ev:"screenY,opt"`
	AltKey   bool    `ev:"altKey"`
	CtrlKey  bool    `ev:"ctrlKey"`
	ShiftKey bool    `ev:"shiftKey"`
	MetaKey  bool    `ev:"metaKey"`
}

type MouseEvent struct {
	BaseEvent
	Button    int     `ev:"button"`
	Buttons   int     `ev:"buttons"`
	ClientX   float64 `ev:"clientX"`
	ClientY   float64 `ev:"clientY"`
	MovementX float64 `ev:"movementX"`
	MovementY float64 `ev:"movementY"`
	OffsetX   float64 `ev:"offsetX,opt"`
	OffsetY   float64 `ev:"offsetY,opt"`
	PageX     float64 `ev:"pageX,opt"`
	PageY     float64 `ev:"pageY,opt"`
	ScreenX   float64 `ev:"screenX,opt"`
	ScreenY   float64 `ev:"screenY,opt"`
	AltKey    bool    `ev:"altKey"`
	CtrlKey   bool    `ev:"ctrlKey"`
	ShiftKey  bool    `ev:"shiftKey"`
	MetaKey   bool    `ev:"metaKey"`
}

type KeyboardEvent struct {
	BaseEvent
	Key         string `ev:"key"`
	Code        string `ev:"code"`
	Repeat      bool   `ev:"repeat"`
	IsComposing bool   `ev:"isComposing"`
	AltKey      bool   `ev:"altKey"`
	CtrlKey     bool   `ev:"ctrlKey"`
	ShiftKey    bool   `ev:"shiftKey"`
	MetaKey     bool   `ev:"metaKey"`
	Location    int    `ev:"location,opt"`
	TargetValue string `ev:"value,opt"`
}

type FocusEvent struct {
	BaseEvent
}

type PointerEvent struct {
	BaseEvent
	PointerType string  `ev:"pointerType"`
	PointerID   int     `ev:"pointerId"`
	Button      int     `ev:"button"`
	Buttons     int     `ev:"buttons"`
	ClientX     float64 `ev:"clientX"`
	ClientY     float64 `ev:"clientY"`
	Pressure    float64 `ev:"pressure"`
	IsPrimary   bool    `ev:"isPrimary"`
	AltKey      bool    `ev:"altKey"`
	CtrlKey     bool    `ev:"ctrlKey"`
	ShiftKey    bool    `ev:"shiftKey"`
	MetaKey     bool    `ev:"metaKey"`
	MovementX   float64 `ev:"movementX,opt"`
	MovementY   float64 `ev:"movementY,opt"`
	OffsetX     float64 `ev:"offsetX,opt"`
	OffsetY     float64 `ev:"offsetY,opt"`
	PageX       float64 `ev:"pageX,opt"`
	PageY       float64 `ev:"pageY,opt"`
	ScreenX     float64 `ev:"screenX,opt"`
	ScreenY     float64 `ev:"screenY,opt"`
	Width       float64 `ev:"width,opt"`
	Height      float64 `ev:"height,opt"`
	TiltX       float64 `ev:"tiltX,opt"`
	TiltY       float64 `ev:"tiltY,opt"`
}

type TouchEvent struct {
	BaseEvent
	Touches  []TouchPoint `ev:"touches"`
	AltKey   bool         `ev:"altKey"`
	CtrlKey  bool         `ev:"ctrlKey"`
	ShiftKey bool         `ev:"shiftKey"`
	MetaKey  bool         `ev:"metaKey"`
}

type DragEvent struct {
	BaseEvent
	ClientX  float64 `ev:"clientX"`
	ClientY  float64 `ev:"clientY"`
	AltKey   bool    `ev:"altKey"`
	CtrlKey  bool    `ev:"ctrlKey"`
	ShiftKey bool    `ev:"shiftKey"`
	MetaKey  bool    `ev:"metaKey"`
	OffsetX  float64 `ev:"offsetX,opt"`
	OffsetY  float64 `ev:"offsetY,opt"`
	PageX    float64 `ev:"pageX,opt"`
	PageY    float64 `ev:"pageY,opt"`
	ScreenX  float64 `ev:"screenX,opt"`
	ScreenY  float64 `ev:"screenY,opt"`
	Text     string  `ev:"text,opt"`
}

type WheelEvent struct {
	BaseEvent
	DeltaX    float64 `ev:"deltaX"`
	DeltaY    float64 `ev:"deltaY"`
	DeltaMode int     `ev:"deltaMode"`
	ClientX   float64 `ev:"clientX"`
	ClientY   float64 `ev:"clientY"`
	CtrlKey   bool    `ev:"ctrlKey"`
	MetaKey   bool    `ev:"metaKey"`
	DeltaZ    float64 `ev:"deltaZ,opt"`
	AltKey    bool    `ev:"altKey,opt"`
	ShiftKey  bool    `ev:"shiftKey,opt"`
}

type ScrollEvent struct {
	BaseEvent
	ScrollTop    float64 `ev:"scrollTop"`
	ScrollLeft   float64 `ev:"scrollLeft"`
	ScrollHeight float64 `ev:"scrollHeight,opt"`
	ScrollWidth  float64 `ev:"scrollWidth,opt"`
	ClientHeight float64 `ev:"clientHeight,opt"`
	ClientWidth  float64 `ev:"clientWidth,opt"`
}

type ChangeEvent struct {
	BaseEvent
	Value       string     `ev:"value"`
	Values      []string   `ev:"selected"`
	Checked     bool       `ev:"checked"`
	Files       []FileMeta `ev:"files"`
	IsComposing bool       `ev:"isComposing,opt"`
}

type BeforeInputEvent struct {
	BaseEvent
	InputType string `ev:"inputType"`
	Data      string `ev:"data"`
}

type FormEvent struct {
	BaseEvent
}

type MediaEvent struct {
	BaseEvent
	CurrentTime float64 `ev:"currentTime"`
	Duration    float64 `ev:"duration"`
	Paused      bool    `ev:"paused"`
	Ended       bool    `ev:"ended,opt"`
	Volume      float64 `ev:"volume,opt"`
	Muted       bool    `ev:"muted,opt"`
	Playback    float64 `ev:"playbackRate,opt"`
}

type AnimationEvent struct {
	BaseEvent
	AnimationName string  `ev:"animationName"`
	ElapsedTime   float64 `ev:"elapsedTime"`
	PseudoElement string  `ev:"pseudoElement"`
}

type TransitionEvent struct {
	BaseEvent
	PropertyName  string  `ev:"propertyName"`
	ElapsedTime   float64 `ev:"elapsedTime"`
	PseudoElement string  `ev:"pseudoElement"`
}

type ClipboardEvent struct {
	BaseEvent
	Text string `ev:"text"`
}

type CompositionEvent struct {
	BaseEvent
	Data string `ev:"data"`
}

type SelectionEvent struct {
	BaseEvent
	SelectionStart int `ev:"selectionStart"`
	SelectionEnd   int `ev:"selectionEnd"`
}

type ToggleEvent struct {
	BaseEvent
	Open bool `ev:"open"`
}

type DialogEvent struct {
	BaseEvent
	ReturnValue string `ev:"returnValue"`
}

type LoadEvent struct {
	BaseEvent
	NaturalWidth  int `ev:"naturalWidth,opt"`
	NaturalHeight int `ev:"naturalHeight,opt"`
}

type ErrorEvent struct {
	BaseEvent
}

type ProgressEvent struct {
	BaseEvent
	LengthComputable bool    `ev:"lengthComputable"`
	Loaded           float64 `ev:"loaded"`
	Total            float64 `ev:"total"`
}

type FullscreenChangeEvent struct {
	BaseEvent
	Active bool `ev:"fullscreen"`
}
