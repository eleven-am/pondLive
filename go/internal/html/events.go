package html

import (
	"github.com/eleven-am/pondlive/go/internal/work"
)

// ============================================================================
// Base Event
// ============================================================================

// BaseEvent contains common event properties.
type BaseEvent struct {
	work.Event
}

// ============================================================================
// Click Events
// ============================================================================

// ClickEvent represents mouse click events (click, dblclick, contextmenu).
type ClickEvent struct {
	BaseEvent
	Detail   int     // Number of clicks
	Button   int     // Which mouse button (0=left, 1=middle, 2=right)
	Buttons  int     // Which buttons are pressed (bitmask)
	ClientX  float64 // X coordinate relative to viewport
	ClientY  float64 // Y coordinate relative to viewport
	OffsetX  float64 // X coordinate relative to target
	OffsetY  float64 // Y coordinate relative to target
	PageX    float64 // X coordinate relative to page
	PageY    float64 // Y coordinate relative to page
	ScreenX  float64 // X coordinate relative to screen
	ScreenY  float64 // Y coordinate relative to screen
	AltKey   bool    // Alt key pressed
	CtrlKey  bool    // Control key pressed
	ShiftKey bool    // Shift key pressed
	MetaKey  bool    // Meta key pressed
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

// ============================================================================
// Mouse Events
// ============================================================================

// MouseEvent represents mouse events (mousedown, mouseup, mousemove, etc).
type MouseEvent struct {
	BaseEvent
	Button    int     // Which mouse button
	Buttons   int     // Which buttons are pressed
	ClientX   float64 // X coordinate relative to viewport
	ClientY   float64 // Y coordinate relative to viewport
	ScreenX   float64 // X coordinate relative to screen
	ScreenY   float64 // Y coordinate relative to screen
	MovementX float64 // X movement since last event
	MovementY float64 // Y movement since last event
	OffsetX   float64 // X coordinate relative to target
	OffsetY   float64 // Y coordinate relative to target
	PageX     float64 // X coordinate relative to page
	PageY     float64 // Y coordinate relative to page
	AltKey    bool    // Alt key pressed
	CtrlKey   bool    // Control key pressed
	ShiftKey  bool    // Shift key pressed
	MetaKey   bool    // Meta key pressed
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

// ============================================================================
// Keyboard Events
// ============================================================================

// KeyboardEvent represents keyboard events (keydown, keyup, keypress).
type KeyboardEvent struct {
	BaseEvent
	Key         string // The key value (e.g., "Enter", "a", "Escape")
	Code        string // Physical key code (e.g., "KeyA", "Enter")
	Location    int    // Location of the key on keyboard
	Repeat      bool   // Is key being held down
	AltKey      bool   // Alt key pressed
	CtrlKey     bool   // Control key pressed
	ShiftKey    bool   // Shift key pressed
	MetaKey     bool   // Meta key pressed
	IsComposing bool   // Is part of composition
}

func (KeyboardEvent) props() []string {
	return []string{
		"event.key", "event.code", "event.location", "event.repeat",
		"event.altKey", "event.ctrlKey", "event.shiftKey", "event.metaKey",
		"event.isComposing",
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
	}
}

// ============================================================================
// Focus Events
// ============================================================================

// FocusEvent represents focus and blur events.
type FocusEvent struct {
	BaseEvent
}

func (FocusEvent) props() []string {
	return []string{}
}

func buildFocusEvent(evt work.Event) FocusEvent {
	return FocusEvent{BaseEvent: BaseEvent{Event: evt}}
}

// ============================================================================
// Pointer Events
// ============================================================================

// PointerEvent represents pointer events (pointerdown, pointerup, pointermove, etc).
type PointerEvent struct {
	BaseEvent
	PointerType string  // Type of pointer (mouse, pen, touch)
	PointerID   int     // Unique pointer identifier
	Button      int     // Which button
	Buttons     int     // Which buttons are pressed
	ClientX     float64 // X coordinate relative to viewport
	ClientY     float64 // Y coordinate relative to viewport
	MovementX   float64 // X movement since last event
	MovementY   float64 // Y movement since last event
	OffsetX     float64 // X coordinate relative to target
	OffsetY     float64 // Y coordinate relative to target
	PageX       float64 // X coordinate relative to page
	PageY       float64 // Y coordinate relative to page
	ScreenX     float64 // X coordinate relative to screen
	ScreenY     float64 // Y coordinate relative to screen
	Width       float64 // Width of contact geometry
	Height      float64 // Height of contact geometry
	Pressure    float64 // Pressure of the pointer (0-1)
	TiltX       float64 // Tilt angle X (-90 to 90)
	TiltY       float64 // Tilt angle Y (-90 to 90)
	IsPrimary   bool    // Is primary pointer
	AltKey      bool    // Alt key pressed
	CtrlKey     bool    // Control key pressed
	ShiftKey    bool    // Shift key pressed
	MetaKey     bool    // Meta key pressed
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

// ============================================================================
// Touch Events
// ============================================================================

// TouchEvent represents touch events (touchstart, touchend, touchmove, touchcancel).
type TouchEvent struct {
	BaseEvent
	AltKey   bool // Alt key pressed
	CtrlKey  bool // Control key pressed
	ShiftKey bool // Shift key pressed
	MetaKey  bool // Meta key pressed
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

// ============================================================================
// Drag Events
// ============================================================================

// DragEvent represents drag and drop events.
type DragEvent struct {
	BaseEvent
	ClientX  float64 // X coordinate relative to viewport
	ClientY  float64 // Y coordinate relative to viewport
	ScreenX  float64 // X coordinate relative to screen
	ScreenY  float64 // Y coordinate relative to screen
	OffsetX  float64 // X coordinate relative to target
	OffsetY  float64 // Y coordinate relative to target
	PageX    float64 // X coordinate relative to page
	PageY    float64 // Y coordinate relative to page
	AltKey   bool    // Alt key pressed
	CtrlKey  bool    // Control key pressed
	ShiftKey bool    // Shift key pressed
	MetaKey  bool    // Meta key pressed
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

// ============================================================================
// Scroll Events
// ============================================================================

// ScrollEvent represents scroll events.
type ScrollEvent struct {
	BaseEvent
	ScrollTop    float64 // Current vertical scroll position
	ScrollLeft   float64 // Current horizontal scroll position
	ScrollHeight float64 // Total scrollable height
	ScrollWidth  float64 // Total scrollable width
	ClientHeight float64 // Visible height
	ClientWidth  float64 // Visible width
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

// ============================================================================
// Input Events
// ============================================================================

// InputEvent represents input events (input, change, beforeinput).
type InputEvent struct {
	BaseEvent
	InputType   string // Type of input (insertText, deleteContentBackward, etc.)
	Data        string // The inserted characters (if any)
	IsComposing bool   // Is part of composition
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

// ChangeEvent represents change events on form elements.
type ChangeEvent struct {
	BaseEvent
}

func (ChangeEvent) props() []string {
	return []string{}
}

func buildChangeEvent(evt work.Event) ChangeEvent {
	return ChangeEvent{BaseEvent: BaseEvent{Event: evt}}
}

// ============================================================================
// Form Events
// ============================================================================

// FormEvent represents form events (submit, reset, invalid).
type FormEvent struct {
	BaseEvent
}

func (FormEvent) props() []string {
	return []string{}
}

func buildFormEvent(evt work.Event) FormEvent {
	return FormEvent{BaseEvent: BaseEvent{Event: evt}}
}

// ============================================================================
// Media Events
// ============================================================================

// MediaEvent represents media events (play, pause, ended, etc.).
type MediaEvent struct {
	BaseEvent
	CurrentTime float64 // Current playback position in seconds
	Duration    float64 // Total duration in seconds
	Paused      bool    // Is playback paused
	Ended       bool    // Has playback ended
	Volume      float64 // Current volume (0-1)
	Muted       bool    // Is audio muted
	Playback    float64 // Playback rate
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

// ============================================================================
// Wheel Events
// ============================================================================

// WheelEvent represents wheel events (wheel, mousewheel).
type WheelEvent struct {
	BaseEvent
	DeltaX    float64 // Horizontal scroll amount
	DeltaY    float64 // Vertical scroll amount
	DeltaZ    float64 // Z-axis scroll amount
	DeltaMode int     // Unit of delta values (0=pixels, 1=lines, 2=pages)
	ClientX   float64 // X coordinate relative to viewport
	ClientY   float64 // Y coordinate relative to viewport
	AltKey    bool    // Alt key pressed
	CtrlKey   bool    // Control key pressed
	ShiftKey  bool    // Shift key pressed
	MetaKey   bool    // Meta key pressed
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

// ============================================================================
// Animation & Transition Events
// ============================================================================

// AnimationEvent represents CSS animation events.
type AnimationEvent struct {
	BaseEvent
	AnimationName string  // Name of the animation
	ElapsedTime   float64 // Time the animation has been running
	PseudoElement string  // Pseudo-element the animation runs on
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

// TransitionEvent represents CSS transition events.
type TransitionEvent struct {
	BaseEvent
	PropertyName  string  // Name of the CSS property
	ElapsedTime   float64 // Time the transition has been running
	PseudoElement string  // Pseudo-element the transition runs on
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

// ============================================================================
// Clipboard Events
// ============================================================================

// ClipboardEvent represents clipboard events (copy, cut, paste).
type ClipboardEvent struct {
	BaseEvent
	ClipboardData string // Text data from clipboard (if accessible)
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

// ============================================================================
// Composition Events
// ============================================================================

// CompositionEvent represents IME composition events.
type CompositionEvent struct {
	BaseEvent
	Data string // The characters being composed
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

// ============================================================================
// Selection Events
// ============================================================================

// SelectionEvent represents text selection events.
type SelectionEvent struct {
	BaseEvent
	SelectionStart int // Start position of selection
	SelectionEnd   int // End position of selection
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

// ============================================================================
// Resize Events
// ============================================================================

// ResizeEvent represents resize events.
type ResizeEvent struct {
	BaseEvent
	Width  float64 // New width
	Height float64 // New height
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

// ============================================================================
// Toggle Events
// ============================================================================

// ToggleEvent represents toggle events (for details element).
type ToggleEvent struct {
	BaseEvent
	Open bool // Whether the element is open
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

// ============================================================================
// Load Events
// ============================================================================

// LoadEvent represents load/error events (for images, scripts, iframes).
type LoadEvent struct {
	BaseEvent
}

func (LoadEvent) props() []string {
	return []string{}
}

func buildLoadEvent(evt work.Event) LoadEvent {
	return LoadEvent{BaseEvent: BaseEvent{Event: evt}}
}

// ErrorEvent represents error events.
type ErrorEvent struct {
	BaseEvent
	Message  string // Error message
	Filename string // Source file where error occurred
	Lineno   int    // Line number
	Colno    int    // Column number
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

// ============================================================================
// Progress Events
// ============================================================================

// ProgressEvent represents progress events (for uploads, downloads).
type ProgressEvent struct {
	BaseEvent
	LengthComputable bool    // Whether total size is known
	Loaded           float64 // Bytes loaded so far
	Total            float64 // Total bytes to load
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

// ============================================================================
// Fullscreen Events
// ============================================================================

// FullscreenEvent represents fullscreen change events.
type FullscreenEvent struct {
	BaseEvent
	IsFullscreen bool // Whether currently in fullscreen
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

// ============================================================================
// Hash/PopState Events (Navigation)
// ============================================================================

// HashChangeEvent represents hashchange events.
type HashChangeEvent struct {
	BaseEvent
	OldURL string // Previous URL
	NewURL string // New URL
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

// PopStateEvent represents popstate events (browser back/forward).
type PopStateEvent struct {
	BaseEvent
	// State is not easily serializable, so we don't include it
}

func (PopStateEvent) props() []string {
	return []string{}
}

func buildPopStateEvent(evt work.Event) PopStateEvent {
	return PopStateEvent{BaseEvent: BaseEvent{Event: evt}}
}

// ============================================================================
// Payload Helpers
// ============================================================================

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
