package pkg

import (
	"github.com/eleven-am/pondlive/go/internal/work"
)

type BaseEvent struct {
	work.Event
}

type ClickEvent struct {
	BaseEvent
	Detail   int
	Button   int
	Buttons  int
	ClientX  float64
	ClientY  float64
	OffsetX  float64
	OffsetY  float64
	PageX    float64
	PageY    float64
	ScreenX  float64
	ScreenY  float64
	AltKey   bool
	CtrlKey  bool
	ShiftKey bool
	MetaKey  bool
}

func (ClickEvent) props() []string {
	return []string{
		"event.detail",
		"event.button", "event.buttons",
		"event.clientX", "event.clientY",
		"event.offsetX", "event.offsetY",
		"event.pageX", "event.pageY",
		"event.screenX", "event.screenY",
		"event.altKey", "event.ctrlKey", "event.shiftKey", "event.metaKey",
	}
}

func buildClickEvent(evt work.Event) ClickEvent {
	return ClickEvent{
		BaseEvent: BaseEvent{Event: evt},
		Detail:    payloadInt(evt.Payload, "event.detail"),
		Button:    payloadInt(evt.Payload, "event.button"),
		Buttons:   payloadInt(evt.Payload, "event.buttons"),
		ClientX:   payloadFloat(evt.Payload, "event.clientX"),
		ClientY:   payloadFloat(evt.Payload, "event.clientY"),
		OffsetX:   payloadFloat(evt.Payload, "event.offsetX"),
		OffsetY:   payloadFloat(evt.Payload, "event.offsetY"),
		PageX:     payloadFloat(evt.Payload, "event.pageX"),
		PageY:     payloadFloat(evt.Payload, "event.pageY"),
		ScreenX:   payloadFloat(evt.Payload, "event.screenX"),
		ScreenY:   payloadFloat(evt.Payload, "event.screenY"),
		AltKey:    payloadBool(evt.Payload, "event.altKey"),
		CtrlKey:   payloadBool(evt.Payload, "event.ctrlKey"),
		ShiftKey:  payloadBool(evt.Payload, "event.shiftKey"),
		MetaKey:   payloadBool(evt.Payload, "event.metaKey"),
	}
}

type MouseEvent struct {
	BaseEvent
	Button    int
	Buttons   int
	ClientX   float64
	ClientY   float64
	ScreenX   float64
	ScreenY   float64
	MovementX float64
	MovementY float64
	OffsetX   float64
	OffsetY   float64
	PageX     float64
	PageY     float64
	AltKey    bool
	CtrlKey   bool
	ShiftKey  bool
	MetaKey   bool
}

func (MouseEvent) props() []string {
	return []string{
		"event.button", "event.buttons",
		"event.clientX", "event.clientY",
		"event.screenX", "event.screenY",
		"event.movementX", "event.movementY",
		"event.offsetX", "event.offsetY",
		"event.pageX", "event.pageY",
		"event.altKey", "event.ctrlKey", "event.shiftKey", "event.metaKey",
	}
}

func buildMouseEvent(evt work.Event) MouseEvent {
	return MouseEvent{
		BaseEvent: BaseEvent{Event: evt},
		Button:    payloadInt(evt.Payload, "event.button"),
		Buttons:   payloadInt(evt.Payload, "event.buttons"),
		ClientX:   payloadFloat(evt.Payload, "event.clientX"),
		ClientY:   payloadFloat(evt.Payload, "event.clientY"),
		ScreenX:   payloadFloat(evt.Payload, "event.screenX"),
		ScreenY:   payloadFloat(evt.Payload, "event.screenY"),
		MovementX: payloadFloat(evt.Payload, "event.movementX"),
		MovementY: payloadFloat(evt.Payload, "event.movementY"),
		OffsetX:   payloadFloat(evt.Payload, "event.offsetX"),
		OffsetY:   payloadFloat(evt.Payload, "event.offsetY"),
		PageX:     payloadFloat(evt.Payload, "event.pageX"),
		PageY:     payloadFloat(evt.Payload, "event.pageY"),
		AltKey:    payloadBool(evt.Payload, "event.altKey"),
		CtrlKey:   payloadBool(evt.Payload, "event.ctrlKey"),
		ShiftKey:  payloadBool(evt.Payload, "event.shiftKey"),
		MetaKey:   payloadBool(evt.Payload, "event.metaKey"),
	}
}

type KeyboardEvent struct {
	BaseEvent
	Key         string
	Code        string
	Location    int
	Repeat      bool
	AltKey      bool
	CtrlKey     bool
	ShiftKey    bool
	MetaKey     bool
	IsComposing bool
	TargetValue string
}

func (KeyboardEvent) props() []string {
	return []string{
		"event.key", "event.code", "event.location", "event.repeat",
		"event.altKey", "event.ctrlKey", "event.shiftKey", "event.metaKey",
		"event.isComposing",
		"target.value",
	}
}

func buildKeyboardEvent(evt work.Event) KeyboardEvent {
	return KeyboardEvent{
		BaseEvent:   BaseEvent{Event: evt},
		Key:         payloadString(evt.Payload, "event.key"),
		Code:        payloadString(evt.Payload, "event.code"),
		Location:    payloadInt(evt.Payload, "event.location"),
		Repeat:      payloadBool(evt.Payload, "event.repeat"),
		AltKey:      payloadBool(evt.Payload, "event.altKey"),
		CtrlKey:     payloadBool(evt.Payload, "event.ctrlKey"),
		ShiftKey:    payloadBool(evt.Payload, "event.shiftKey"),
		MetaKey:     payloadBool(evt.Payload, "event.metaKey"),
		IsComposing: payloadBool(evt.Payload, "event.isComposing"),
		TargetValue: payloadString(evt.Payload, "target.value"),
	}
}

type FocusEvent struct {
	BaseEvent
}

func (FocusEvent) props() []string {
	return []string{}
}

func buildFocusEvent(evt work.Event) FocusEvent {
	return FocusEvent{BaseEvent: BaseEvent{Event: evt}}
}

type PointerEvent struct {
	BaseEvent
	PointerType string
	PointerID   int
	Button      int
	Buttons     int
	ClientX     float64
	ClientY     float64
	MovementX   float64
	MovementY   float64
	OffsetX     float64
	OffsetY     float64
	PageX       float64
	PageY       float64
	ScreenX     float64
	ScreenY     float64
	Width       float64
	Height      float64
	Pressure    float64
	TiltX       float64
	TiltY       float64
	IsPrimary   bool
	AltKey      bool
	CtrlKey     bool
	ShiftKey    bool
	MetaKey     bool
}

func (PointerEvent) props() []string {
	return []string{
		"event.pointerType", "event.pointerId",
		"event.button", "event.buttons",
		"event.clientX", "event.clientY",
		"event.movementX", "event.movementY",
		"event.offsetX", "event.offsetY",
		"event.pageX", "event.pageY",
		"event.screenX", "event.screenY",
		"event.width", "event.height",
		"event.pressure", "event.tiltX", "event.tiltY",
		"event.isPrimary",
		"event.altKey", "event.ctrlKey", "event.shiftKey", "event.metaKey",
	}
}

func buildPointerEvent(evt work.Event) PointerEvent {
	return PointerEvent{
		BaseEvent:   BaseEvent{Event: evt},
		PointerType: payloadString(evt.Payload, "event.pointerType"),
		PointerID:   payloadInt(evt.Payload, "event.pointerId"),
		Button:      payloadInt(evt.Payload, "event.button"),
		Buttons:     payloadInt(evt.Payload, "event.buttons"),
		ClientX:     payloadFloat(evt.Payload, "event.clientX"),
		ClientY:     payloadFloat(evt.Payload, "event.clientY"),
		MovementX:   payloadFloat(evt.Payload, "event.movementX"),
		MovementY:   payloadFloat(evt.Payload, "event.movementY"),
		OffsetX:     payloadFloat(evt.Payload, "event.offsetX"),
		OffsetY:     payloadFloat(evt.Payload, "event.offsetY"),
		PageX:       payloadFloat(evt.Payload, "event.pageX"),
		PageY:       payloadFloat(evt.Payload, "event.pageY"),
		ScreenX:     payloadFloat(evt.Payload, "event.screenX"),
		ScreenY:     payloadFloat(evt.Payload, "event.screenY"),
		Width:       payloadFloat(evt.Payload, "event.width"),
		Height:      payloadFloat(evt.Payload, "event.height"),
		Pressure:    payloadFloat(evt.Payload, "event.pressure"),
		TiltX:       payloadFloat(evt.Payload, "event.tiltX"),
		TiltY:       payloadFloat(evt.Payload, "event.tiltY"),
		IsPrimary:   payloadBool(evt.Payload, "event.isPrimary"),
		AltKey:      payloadBool(evt.Payload, "event.altKey"),
		CtrlKey:     payloadBool(evt.Payload, "event.ctrlKey"),
		ShiftKey:    payloadBool(evt.Payload, "event.shiftKey"),
		MetaKey:     payloadBool(evt.Payload, "event.metaKey"),
	}
}

type TouchEvent struct {
	BaseEvent
	AltKey   bool
	CtrlKey  bool
	ShiftKey bool
	MetaKey  bool
}

func (TouchEvent) props() []string {
	return []string{
		"event.altKey", "event.ctrlKey", "event.shiftKey", "event.metaKey",
	}
}

func buildTouchEvent(evt work.Event) TouchEvent {
	return TouchEvent{
		BaseEvent: BaseEvent{Event: evt},
		AltKey:    payloadBool(evt.Payload, "event.altKey"),
		CtrlKey:   payloadBool(evt.Payload, "event.ctrlKey"),
		ShiftKey:  payloadBool(evt.Payload, "event.shiftKey"),
		MetaKey:   payloadBool(evt.Payload, "event.metaKey"),
	}
}

type DragEvent struct {
	BaseEvent
	ClientX  float64
	ClientY  float64
	ScreenX  float64
	ScreenY  float64
	OffsetX  float64
	OffsetY  float64
	PageX    float64
	PageY    float64
	AltKey   bool
	CtrlKey  bool
	ShiftKey bool
	MetaKey  bool
}

func (DragEvent) props() []string {
	return []string{
		"event.clientX", "event.clientY",
		"event.screenX", "event.screenY",
		"event.offsetX", "event.offsetY",
		"event.pageX", "event.pageY",
		"event.altKey", "event.ctrlKey", "event.shiftKey", "event.metaKey",
	}
}

func buildDragEvent(evt work.Event) DragEvent {
	return DragEvent{
		BaseEvent: BaseEvent{Event: evt},
		ClientX:   payloadFloat(evt.Payload, "event.clientX"),
		ClientY:   payloadFloat(evt.Payload, "event.clientY"),
		ScreenX:   payloadFloat(evt.Payload, "event.screenX"),
		ScreenY:   payloadFloat(evt.Payload, "event.screenY"),
		OffsetX:   payloadFloat(evt.Payload, "event.offsetX"),
		OffsetY:   payloadFloat(evt.Payload, "event.offsetY"),
		PageX:     payloadFloat(evt.Payload, "event.pageX"),
		PageY:     payloadFloat(evt.Payload, "event.pageY"),
		AltKey:    payloadBool(evt.Payload, "event.altKey"),
		CtrlKey:   payloadBool(evt.Payload, "event.ctrlKey"),
		ShiftKey:  payloadBool(evt.Payload, "event.shiftKey"),
		MetaKey:   payloadBool(evt.Payload, "event.metaKey"),
	}
}

type ScrollEvent struct {
	BaseEvent
	ScrollTop    float64
	ScrollLeft   float64
	ScrollHeight float64
	ScrollWidth  float64
	ClientHeight float64
	ClientWidth  float64
}

func (ScrollEvent) props() []string {
	return []string{
		"target.scrollTop", "target.scrollLeft",
		"target.scrollHeight", "target.scrollWidth",
		"target.clientHeight", "target.clientWidth",
	}
}

func buildScrollEvent(evt work.Event) ScrollEvent {
	return ScrollEvent{
		BaseEvent:    BaseEvent{Event: evt},
		ScrollTop:    payloadFloat(evt.Payload, "target.scrollTop"),
		ScrollLeft:   payloadFloat(evt.Payload, "target.scrollLeft"),
		ScrollHeight: payloadFloat(evt.Payload, "target.scrollHeight"),
		ScrollWidth:  payloadFloat(evt.Payload, "target.scrollWidth"),
		ClientHeight: payloadFloat(evt.Payload, "target.clientHeight"),
		ClientWidth:  payloadFloat(evt.Payload, "target.clientWidth"),
	}
}

type InputEvent struct {
	BaseEvent
	InputType   string
	Data        string
	IsComposing bool
}

func (InputEvent) props() []string {
	return []string{
		"event.inputType", "event.data", "event.isComposing",
	}
}

func buildInputEvent(evt work.Event) InputEvent {
	return InputEvent{
		BaseEvent:   BaseEvent{Event: evt},
		InputType:   payloadString(evt.Payload, "event.inputType"),
		Data:        payloadString(evt.Payload, "event.data"),
		IsComposing: payloadBool(evt.Payload, "event.isComposing"),
	}
}

type ChangeEvent struct {
	BaseEvent
	Value string
}

func (ChangeEvent) props() []string {
	return []string{"target.value"}
}

func buildChangeEvent(evt work.Event) ChangeEvent {
	return ChangeEvent{
		BaseEvent: BaseEvent{Event: evt},
		Value:     payloadString(evt.Payload, "target.value"),
	}
}

type FormEvent struct {
	BaseEvent
}

func (FormEvent) props() []string {
	return []string{}
}

func buildFormEvent(evt work.Event) FormEvent {
	return FormEvent{BaseEvent: BaseEvent{Event: evt}}
}

type MediaEvent struct {
	BaseEvent
	CurrentTime float64
	Duration    float64
	Paused      bool
	Ended       bool
	Volume      float64
	Muted       bool
	Playback    float64
}

func (MediaEvent) props() []string {
	return []string{
		"target.currentTime", "target.duration",
		"target.paused", "target.ended",
		"target.volume", "target.muted",
		"target.playbackRate",
	}
}

func buildMediaEvent(evt work.Event) MediaEvent {
	return MediaEvent{
		BaseEvent:   BaseEvent{Event: evt},
		CurrentTime: payloadFloat(evt.Payload, "target.currentTime"),
		Duration:    payloadFloat(evt.Payload, "target.duration"),
		Paused:      payloadBool(evt.Payload, "target.paused"),
		Ended:       payloadBool(evt.Payload, "target.ended"),
		Volume:      payloadFloat(evt.Payload, "target.volume"),
		Muted:       payloadBool(evt.Payload, "target.muted"),
		Playback:    payloadFloat(evt.Payload, "target.playbackRate"),
	}
}

type WheelEvent struct {
	BaseEvent
	DeltaX    float64
	DeltaY    float64
	DeltaZ    float64
	DeltaMode int
	ClientX   float64
	ClientY   float64
	AltKey    bool
	CtrlKey   bool
	ShiftKey  bool
	MetaKey   bool
}

func (WheelEvent) props() []string {
	return []string{
		"event.deltaX", "event.deltaY", "event.deltaZ", "event.deltaMode",
		"event.clientX", "event.clientY",
		"event.altKey", "event.ctrlKey", "event.shiftKey", "event.metaKey",
	}
}

func buildWheelEvent(evt work.Event) WheelEvent {
	return WheelEvent{
		BaseEvent: BaseEvent{Event: evt},
		DeltaX:    payloadFloat(evt.Payload, "event.deltaX"),
		DeltaY:    payloadFloat(evt.Payload, "event.deltaY"),
		DeltaZ:    payloadFloat(evt.Payload, "event.deltaZ"),
		DeltaMode: payloadInt(evt.Payload, "event.deltaMode"),
		ClientX:   payloadFloat(evt.Payload, "event.clientX"),
		ClientY:   payloadFloat(evt.Payload, "event.clientY"),
		AltKey:    payloadBool(evt.Payload, "event.altKey"),
		CtrlKey:   payloadBool(evt.Payload, "event.ctrlKey"),
		ShiftKey:  payloadBool(evt.Payload, "event.shiftKey"),
		MetaKey:   payloadBool(evt.Payload, "event.metaKey"),
	}
}

type AnimationEvent struct {
	BaseEvent
	AnimationName string
	ElapsedTime   float64
	PseudoElement string
}

func (AnimationEvent) props() []string {
	return []string{
		"event.animationName", "event.elapsedTime", "event.pseudoElement",
	}
}

func buildAnimationEvent(evt work.Event) AnimationEvent {
	return AnimationEvent{
		BaseEvent:     BaseEvent{Event: evt},
		AnimationName: payloadString(evt.Payload, "event.animationName"),
		ElapsedTime:   payloadFloat(evt.Payload, "event.elapsedTime"),
		PseudoElement: payloadString(evt.Payload, "event.pseudoElement"),
	}
}

type TransitionEvent struct {
	BaseEvent
	PropertyName  string
	ElapsedTime   float64
	PseudoElement string
}

func (TransitionEvent) props() []string {
	return []string{
		"event.propertyName", "event.elapsedTime", "event.pseudoElement",
	}
}

func buildTransitionEvent(evt work.Event) TransitionEvent {
	return TransitionEvent{
		BaseEvent:     BaseEvent{Event: evt},
		PropertyName:  payloadString(evt.Payload, "event.propertyName"),
		ElapsedTime:   payloadFloat(evt.Payload, "event.elapsedTime"),
		PseudoElement: payloadString(evt.Payload, "event.pseudoElement"),
	}
}

type ClipboardEvent struct {
	BaseEvent
	ClipboardData string
}

func (ClipboardEvent) props() []string {
	return []string{}
}

func buildClipboardEvent(evt work.Event) ClipboardEvent {
	return ClipboardEvent{
		BaseEvent:     BaseEvent{Event: evt},
		ClipboardData: payloadString(evt.Payload, "clipboardData"),
	}
}

type CompositionEvent struct {
	BaseEvent
	Data string
}

func (CompositionEvent) props() []string {
	return []string{"event.data"}
}

func buildCompositionEvent(evt work.Event) CompositionEvent {
	return CompositionEvent{
		BaseEvent: BaseEvent{Event: evt},
		Data:      payloadString(evt.Payload, "event.data"),
	}
}

type SelectionEvent struct {
	BaseEvent
	SelectionStart int
	SelectionEnd   int
}

func (SelectionEvent) props() []string {
	return []string{
		"target.selectionStart", "target.selectionEnd",
	}
}

func buildSelectionEvent(evt work.Event) SelectionEvent {
	return SelectionEvent{
		BaseEvent:      BaseEvent{Event: evt},
		SelectionStart: payloadInt(evt.Payload, "target.selectionStart"),
		SelectionEnd:   payloadInt(evt.Payload, "target.selectionEnd"),
	}
}

type ResizeEvent struct {
	BaseEvent
	Width  float64
	Height float64
}

func (ResizeEvent) props() []string {
	return []string{
		"target.clientWidth", "target.clientHeight",
	}
}

func buildResizeEvent(evt work.Event) ResizeEvent {
	return ResizeEvent{
		BaseEvent: BaseEvent{Event: evt},
		Width:     payloadFloat(evt.Payload, "target.clientWidth"),
		Height:    payloadFloat(evt.Payload, "target.clientHeight"),
	}
}

type ToggleEvent struct {
	BaseEvent
	Open bool
}

func (ToggleEvent) props() []string {
	return []string{"target.open"}
}

func buildToggleEvent(evt work.Event) ToggleEvent {
	return ToggleEvent{
		BaseEvent: BaseEvent{Event: evt},
		Open:      payloadBool(evt.Payload, "target.open"),
	}
}

type LoadEvent struct {
	BaseEvent
}

func (LoadEvent) props() []string {
	return []string{}
}

func buildLoadEvent(evt work.Event) LoadEvent {
	return LoadEvent{BaseEvent: BaseEvent{Event: evt}}
}

type ErrorEvent struct {
	BaseEvent
	Message  string
	Filename string
	Lineno   int
	Colno    int
}

func (ErrorEvent) props() []string {
	return []string{
		"event.message", "event.filename", "event.lineno", "event.colno",
	}
}

func buildErrorEvent(evt work.Event) ErrorEvent {
	return ErrorEvent{
		BaseEvent: BaseEvent{Event: evt},
		Message:   payloadString(evt.Payload, "event.message"),
		Filename:  payloadString(evt.Payload, "event.filename"),
		Lineno:    payloadInt(evt.Payload, "event.lineno"),
		Colno:     payloadInt(evt.Payload, "event.colno"),
	}
}

type ProgressEvent struct {
	BaseEvent
	LengthComputable bool
	Loaded           float64
	Total            float64
}

func (ProgressEvent) props() []string {
	return []string{
		"event.lengthComputable", "event.loaded", "event.total",
	}
}

func buildProgressEvent(evt work.Event) ProgressEvent {
	return ProgressEvent{
		BaseEvent:        BaseEvent{Event: evt},
		LengthComputable: payloadBool(evt.Payload, "event.lengthComputable"),
		Loaded:           payloadFloat(evt.Payload, "event.loaded"),
		Total:            payloadFloat(evt.Payload, "event.total"),
	}
}

type FullscreenEvent struct {
	BaseEvent
	IsFullscreen bool
}

func (FullscreenEvent) props() []string {
	return []string{"document.fullscreenElement"}
}

func buildFullscreenEvent(evt work.Event) FullscreenEvent {

	return FullscreenEvent{
		BaseEvent:    BaseEvent{Event: evt},
		IsFullscreen: evt.Payload["document.fullscreenElement"] != nil,
	}
}

type HashChangeEvent struct {
	BaseEvent
	OldURL string
	NewURL string
}

func (HashChangeEvent) props() []string {
	return []string{"event.oldURL", "event.newURL"}
}

func buildHashChangeEvent(evt work.Event) HashChangeEvent {
	return HashChangeEvent{
		BaseEvent: BaseEvent{Event: evt},
		OldURL:    payloadString(evt.Payload, "event.oldURL"),
		NewURL:    payloadString(evt.Payload, "event.newURL"),
	}
}

type PopStateEvent struct {
	BaseEvent
}

func (PopStateEvent) props() []string {
	return []string{}
}

func buildPopStateEvent(evt work.Event) PopStateEvent {
	return PopStateEvent{BaseEvent: BaseEvent{Event: evt}}
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func payloadInt(payload map[string]any, key string) int {
	if payload == nil {
		return 0
	}
	if v, ok := payload[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return 0
}

func payloadFloat(payload map[string]any, key string) float64 {
	if payload == nil {
		return 0
	}
	if v, ok := payload[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return 0
}

func payloadBool(payload map[string]any, key string) bool {
	if payload == nil {
		return false
	}
	if v, ok := payload[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
