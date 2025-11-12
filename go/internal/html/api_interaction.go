package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// ============================================================================
// Event Types
// ============================================================================

// MouseEvent represents mouse events (mousedown, mouseup, mousemove, etc).
type MouseEvent struct {
	Event
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

func buildMouseEvent(evt Event) MouseEvent {
	return MouseEvent{
		Event:     evt,
		Button:    payloadInt(evt.Payload, "event.button", 0),
		Buttons:   payloadInt(evt.Payload, "event.buttons", 0),
		ClientX:   payloadFloat(evt.Payload, "event.clientX", 0),
		ClientY:   payloadFloat(evt.Payload, "event.clientY", 0),
		ScreenX:   payloadFloat(evt.Payload, "event.screenX", 0),
		ScreenY:   payloadFloat(evt.Payload, "event.screenY", 0),
		MovementX: payloadFloat(evt.Payload, "event.movementX", 0),
		MovementY: payloadFloat(evt.Payload, "event.movementY", 0),
		OffsetX:   payloadFloat(evt.Payload, "event.offsetX", 0),
		OffsetY:   payloadFloat(evt.Payload, "event.offsetY", 0),
		PageX:     payloadFloat(evt.Payload, "event.pageX", 0),
		PageY:     payloadFloat(evt.Payload, "event.pageY", 0),
		AltKey:    payloadBool(evt.Payload, "event.altKey", false),
		CtrlKey:   payloadBool(evt.Payload, "event.ctrlKey", false),
		ShiftKey:  payloadBool(evt.Payload, "event.shiftKey", false),
		MetaKey:   payloadBool(evt.Payload, "event.metaKey", false),
	}
}

// ClickEvent represents mouse click events (click, dblclick, contextmenu).
type ClickEvent struct {
	Event
	Detail   int     // Number of clicks
	AltKey   bool    // Alt key pressed
	CtrlKey  bool    // Control key pressed
	ShiftKey bool    // Shift key pressed
	MetaKey  bool    // Meta key pressed
	Button   int     // Which mouse button
	Buttons  int     // Which buttons are pressed
	ClientX  float64 // X coordinate relative to viewport
	ClientY  float64 // Y coordinate relative to viewport
	OffsetX  float64 // X coordinate relative to target
	OffsetY  float64 // Y coordinate relative to target
	PageX    float64 // X coordinate relative to page
	PageY    float64 // Y coordinate relative to page
	ScreenX  float64 // X coordinate relative to screen
	ScreenY  float64 // Y coordinate relative to screen
}

func (ClickEvent) props() []string {
	return []string{
		"event.detail",
		"event.altKey", "event.ctrlKey", "event.shiftKey", "event.metaKey",
		"event.button", "event.buttons",
		"event.clientX", "event.clientY",
		"event.offsetX", "event.offsetY",
		"event.pageX", "event.pageY",
		"event.screenX", "event.screenY",
	}
}

func buildClickEvent(evt Event) ClickEvent {
	return ClickEvent{
		Event:    evt,
		Detail:   payloadInt(evt.Payload, "event.detail", 0),
		AltKey:   payloadBool(evt.Payload, "event.altKey", false),
		CtrlKey:  payloadBool(evt.Payload, "event.ctrlKey", false),
		ShiftKey: payloadBool(evt.Payload, "event.shiftKey", false),
		MetaKey:  payloadBool(evt.Payload, "event.metaKey", false),
		Button:   payloadInt(evt.Payload, "event.button", 0),
		Buttons:  payloadInt(evt.Payload, "event.buttons", 0),
		ClientX:  payloadFloat(evt.Payload, "event.clientX", 0),
		ClientY:  payloadFloat(evt.Payload, "event.clientY", 0),
		OffsetX:  payloadFloat(evt.Payload, "event.offsetX", 0),
		OffsetY:  payloadFloat(evt.Payload, "event.offsetY", 0),
		PageX:    payloadFloat(evt.Payload, "event.pageX", 0),
		PageY:    payloadFloat(evt.Payload, "event.pageY", 0),
		ScreenX:  payloadFloat(evt.Payload, "event.screenX", 0),
		ScreenY:  payloadFloat(evt.Payload, "event.screenY", 0),
	}
}

// FocusEvent represents focus and blur events.
type FocusEvent struct {
	Event
}

func (FocusEvent) props() []string {
	return []string{}
}

func buildFocusEvent(evt Event) FocusEvent {
	return FocusEvent{Event: evt}
}

// KeyboardEvent represents keyboard events (keydown, keyup, keypress).
type KeyboardEvent struct {
	Event
	Key         string // The key value
	Code        string // Physical key code
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

func buildKeyboardEvent(evt Event) KeyboardEvent {
	return KeyboardEvent{
		Event:       evt,
		Key:         PayloadString(evt.Payload, "event.key", ""),
		Code:        PayloadString(evt.Payload, "event.code", ""),
		Location:    payloadInt(evt.Payload, "event.location", 0),
		Repeat:      payloadBool(evt.Payload, "event.repeat", false),
		AltKey:      payloadBool(evt.Payload, "event.altKey", false),
		CtrlKey:     payloadBool(evt.Payload, "event.ctrlKey", false),
		ShiftKey:    payloadBool(evt.Payload, "event.shiftKey", false),
		MetaKey:     payloadBool(evt.Payload, "event.metaKey", false),
		IsComposing: payloadBool(evt.Payload, "event.isComposing", false),
	}
}

// PointerEvent represents pointer events (pointerdown, pointerup, pointermove, etc).
type PointerEvent struct {
	Event
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
		"event.isPrimary",
		"event.altKey", "event.ctrlKey", "event.shiftKey", "event.metaKey",
	}
}

func buildPointerEvent(evt Event) PointerEvent {
	return PointerEvent{
		Event:       evt,
		PointerType: PayloadString(evt.Payload, "event.pointerType", ""),
		PointerID:   payloadInt(evt.Payload, "event.pointerId", 0),
		Button:      payloadInt(evt.Payload, "event.button", 0),
		Buttons:     payloadInt(evt.Payload, "event.buttons", 0),
		ClientX:     payloadFloat(evt.Payload, "event.clientX", 0),
		ClientY:     payloadFloat(evt.Payload, "event.clientY", 0),
		MovementX:   payloadFloat(evt.Payload, "event.movementX", 0),
		MovementY:   payloadFloat(evt.Payload, "event.movementY", 0),
		OffsetX:     payloadFloat(evt.Payload, "event.offsetX", 0),
		OffsetY:     payloadFloat(evt.Payload, "event.offsetY", 0),
		PageX:       payloadFloat(evt.Payload, "event.pageX", 0),
		PageY:       payloadFloat(evt.Payload, "event.pageY", 0),
		ScreenX:     payloadFloat(evt.Payload, "event.screenX", 0),
		ScreenY:     payloadFloat(evt.Payload, "event.screenY", 0),
		IsPrimary:   payloadBool(evt.Payload, "event.isPrimary", false),
		AltKey:      payloadBool(evt.Payload, "event.altKey", false),
		CtrlKey:     payloadBool(evt.Payload, "event.ctrlKey", false),
		ShiftKey:    payloadBool(evt.Payload, "event.shiftKey", false),
		MetaKey:     payloadBool(evt.Payload, "event.metaKey", false),
	}
}

// TouchEvent represents touch events (touchstart, touchend, touchmove, touchcancel).
type TouchEvent struct {
	Event
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

func buildTouchEvent(evt Event) TouchEvent {
	return TouchEvent{
		Event:    evt,
		AltKey:   payloadBool(evt.Payload, "event.altKey", false),
		CtrlKey:  payloadBool(evt.Payload, "event.ctrlKey", false),
		ShiftKey: payloadBool(evt.Payload, "event.shiftKey", false),
		MetaKey:  payloadBool(evt.Payload, "event.metaKey", false),
	}
}

// DragEvent represents drag and drop events.
type DragEvent struct {
	Event
	ClientX  float64 // X coordinate relative to viewport
	ClientY  float64 // Y coordinate relative to viewport
	ScreenX  float64 // X coordinate relative to screen
	ScreenY  float64 // Y coordinate relative to screen
	AltKey   bool    // Alt key pressed
	CtrlKey  bool    // Control key pressed
	ShiftKey bool    // Shift key pressed
	MetaKey  bool    // Meta key pressed
}

func (DragEvent) props() []string {
	return []string{
		"event.clientX", "event.clientY",
		"event.screenX", "event.screenY",
		"event.altKey", "event.ctrlKey", "event.shiftKey", "event.metaKey",
	}
}

func buildDragEvent(evt Event) DragEvent {
	return DragEvent{
		Event:    evt,
		ClientX:  payloadFloat(evt.Payload, "event.clientX", 0),
		ClientY:  payloadFloat(evt.Payload, "event.clientY", 0),
		ScreenX:  payloadFloat(evt.Payload, "event.screenX", 0),
		ScreenY:  payloadFloat(evt.Payload, "event.screenY", 0),
		AltKey:   payloadBool(evt.Payload, "event.altKey", false),
		CtrlKey:  payloadBool(evt.Payload, "event.ctrlKey", false),
		ShiftKey: payloadBool(evt.Payload, "event.shiftKey", false),
		MetaKey:  payloadBool(evt.Payload, "event.metaKey", false),
	}
}

// ============================================================================
// InteractionAPI
// ============================================================================

// InteractionAPI provides actions and events for all user interaction (mouse, keyboard, touch, pointer, drag, focus).
type InteractionAPI[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.Dispatcher
}

func NewInteractionAPI[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.Dispatcher) *InteractionAPI[T] {
	return &InteractionAPI[T]{ref: ref, ctx: ctx}
}

// ============================================================================
// Actions
// ============================================================================

// Click simulates a click on the element.
func (a *InteractionAPI[T]) Click() {
	dom.DOMCall[T](a.ctx, a.ref, "click")
}

// Focus sets focus on the element.
func (a *InteractionAPI[T]) Focus() {
	dom.DOMCall[T](a.ctx, a.ref, "focus")
}

// Blur removes focus from the element.
func (a *InteractionAPI[T]) Blur() {
	dom.DOMCall[T](a.ctx, a.ref, "blur")
}

// ============================================================================
// Mouse Events
// ============================================================================

// OnMouseDown registers a handler for the "mousedown" event.
func (a *InteractionAPI[T]) OnMouseDown(handler func(MouseEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	a.ref.AddListener("mousedown", wrapped, MouseEvent{}.props())
}

// OnMouseUp registers a handler for the "mouseup" event.
func (a *InteractionAPI[T]) OnMouseUp(handler func(MouseEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	a.ref.AddListener("mouseup", wrapped, MouseEvent{}.props())
}

// OnMouseMove registers a handler for the "mousemove" event.
func (a *InteractionAPI[T]) OnMouseMove(handler func(MouseEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	a.ref.AddListener("mousemove", wrapped, MouseEvent{}.props())
}

// OnMouseEnter registers a handler for the "mouseenter" event.
func (a *InteractionAPI[T]) OnMouseEnter(handler func(MouseEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	a.ref.AddListener("mouseenter", wrapped, MouseEvent{}.props())
}

// OnMouseLeave registers a handler for the "mouseleave" event.
func (a *InteractionAPI[T]) OnMouseLeave(handler func(MouseEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	a.ref.AddListener("mouseleave", wrapped, MouseEvent{}.props())
}

// OnMouseOver registers a handler for the "mouseover" event.
func (a *InteractionAPI[T]) OnMouseOver(handler func(MouseEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	a.ref.AddListener("mouseover", wrapped, MouseEvent{}.props())
}

// OnMouseOut registers a handler for the "mouseout" event.
func (a *InteractionAPI[T]) OnMouseOut(handler func(MouseEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	a.ref.AddListener("mouseout", wrapped, MouseEvent{}.props())
}

// ============================================================================
// Click Events
// ============================================================================

// OnClick registers a handler for the "click" event.
func (a *InteractionAPI[T]) OnClick(handler func(ClickEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildClickEvent(evt)) }
	a.ref.AddListener("click", wrapped, ClickEvent{}.props())
}

// OnDoubleClick registers a handler for the "dblclick" event.
func (a *InteractionAPI[T]) OnDoubleClick(handler func(ClickEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildClickEvent(evt)) }
	a.ref.AddListener("dblclick", wrapped, ClickEvent{}.props())
}

// OnContextMenu registers a handler for the "contextmenu" event.
func (a *InteractionAPI[T]) OnContextMenu(handler func(ClickEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildClickEvent(evt)) }
	a.ref.AddListener("contextmenu", wrapped, ClickEvent{}.props())
}

// ============================================================================
// Focus Events
// ============================================================================

// OnFocus registers a handler for the "focus" event.
func (a *InteractionAPI[T]) OnFocus(handler func(FocusEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildFocusEvent(evt)) }
	a.ref.AddListener("focus", wrapped, FocusEvent{}.props())
}

// OnBlur registers a handler for the "blur" event.
func (a *InteractionAPI[T]) OnBlur(handler func(FocusEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildFocusEvent(evt)) }
	a.ref.AddListener("blur", wrapped, FocusEvent{}.props())
}

// ============================================================================
// Keyboard Events
// ============================================================================

// OnKeyDown registers a handler for the "keydown" event.
func (a *InteractionAPI[T]) OnKeyDown(handler func(KeyboardEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildKeyboardEvent(evt)) }
	a.ref.AddListener("keydown", wrapped, KeyboardEvent{}.props())
}

// OnKeyUp registers a handler for the "keyup" event.
func (a *InteractionAPI[T]) OnKeyUp(handler func(KeyboardEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildKeyboardEvent(evt)) }
	a.ref.AddListener("keyup", wrapped, KeyboardEvent{}.props())
}

// OnKeyPress registers a handler for the "keypress" event.
func (a *InteractionAPI[T]) OnKeyPress(handler func(KeyboardEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildKeyboardEvent(evt)) }
	a.ref.AddListener("keypress", wrapped, KeyboardEvent{}.props())
}

// ============================================================================
// Pointer Events
// ============================================================================

// OnPointerDown registers a handler for the "pointerdown" event.
func (a *InteractionAPI[T]) OnPointerDown(handler func(PointerEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	a.ref.AddListener("pointerdown", wrapped, PointerEvent{}.props())
}

// OnPointerUp registers a handler for the "pointerup" event.
func (a *InteractionAPI[T]) OnPointerUp(handler func(PointerEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	a.ref.AddListener("pointerup", wrapped, PointerEvent{}.props())
}

// OnPointerMove registers a handler for the "pointerMove" event.
func (a *InteractionAPI[T]) OnPointerMove(handler func(PointerEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	a.ref.AddListener("pointermove", wrapped, PointerEvent{}.props())
}

// OnPointerEnter registers a handler for the "pointerenter" event.
func (a *InteractionAPI[T]) OnPointerEnter(handler func(PointerEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	a.ref.AddListener("pointerenter", wrapped, PointerEvent{}.props())
}

// OnPointerLeave registers a handler for the "pointerleave" event.
func (a *InteractionAPI[T]) OnPointerLeave(handler func(PointerEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	a.ref.AddListener("pointerleave", wrapped, PointerEvent{}.props())
}

// OnPointerOver registers a handler for the "pointerover" event.
func (a *InteractionAPI[T]) OnPointerOver(handler func(PointerEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	a.ref.AddListener("pointerover", wrapped, PointerEvent{}.props())
}

// OnPointerOut registers a handler for the "pointerout" event.
func (a *InteractionAPI[T]) OnPointerOut(handler func(PointerEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	a.ref.AddListener("pointerout", wrapped, PointerEvent{}.props())
}

// OnPointerCancel registers a handler for the "pointercancel" event.
func (a *InteractionAPI[T]) OnPointerCancel(handler func(PointerEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	a.ref.AddListener("pointercancel", wrapped, PointerEvent{}.props())
}

// ============================================================================
// Touch Events
// ============================================================================

// OnTouchStart registers a handler for the "touchstart" event.
func (a *InteractionAPI[T]) OnTouchStart(handler func(TouchEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildTouchEvent(evt)) }
	a.ref.AddListener("touchstart", wrapped, TouchEvent{}.props())
}

// OnTouchEnd registers a handler for the "touchend" event.
func (a *InteractionAPI[T]) OnTouchEnd(handler func(TouchEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildTouchEvent(evt)) }
	a.ref.AddListener("touchend", wrapped, TouchEvent{}.props())
}

// OnTouchMove registers a handler for the "touchmove" event.
func (a *InteractionAPI[T]) OnTouchMove(handler func(TouchEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildTouchEvent(evt)) }
	a.ref.AddListener("touchmove", wrapped, TouchEvent{}.props())
}

// OnTouchCancel registers a handler for the "touchcancel" event.
func (a *InteractionAPI[T]) OnTouchCancel(handler func(TouchEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildTouchEvent(evt)) }
	a.ref.AddListener("touchcancel", wrapped, TouchEvent{}.props())
}

// ============================================================================
// Drag Events
// ============================================================================

// OnDrag registers a handler for the "drag" event.
func (a *InteractionAPI[T]) OnDrag(handler func(DragEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	a.ref.AddListener("drag", wrapped, DragEvent{}.props())
}

// OnDragStart registers a handler for the "dragstart" event.
func (a *InteractionAPI[T]) OnDragStart(handler func(DragEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	a.ref.AddListener("dragstart", wrapped, DragEvent{}.props())
}

// OnDragEnd registers a handler for the "dragend" event.
func (a *InteractionAPI[T]) OnDragEnd(handler func(DragEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	a.ref.AddListener("dragend", wrapped, DragEvent{}.props())
}

// OnDragEnter registers a handler for the "dragenter" event.
func (a *InteractionAPI[T]) OnDragEnter(handler func(DragEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	a.ref.AddListener("dragenter", wrapped, DragEvent{}.props())
}

// OnDragLeave registers a handler for the "dragleave" event.
func (a *InteractionAPI[T]) OnDragLeave(handler func(DragEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	a.ref.AddListener("dragleave", wrapped, DragEvent{}.props())
}

// OnDragOver registers a handler for the "dragover" event.
func (a *InteractionAPI[T]) OnDragOver(handler func(DragEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	a.ref.AddListener("dragover", wrapped, DragEvent{}.props())
}

// OnDrop registers a handler for the "drop" event.
func (a *InteractionAPI[T]) OnDrop(handler func(DragEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	a.ref.AddListener("drop", wrapped, DragEvent{}.props())
}
