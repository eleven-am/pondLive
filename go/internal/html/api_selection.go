package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// SelectionEvent represents text selection events.
type SelectionEvent struct {
	Event
}

// props returns the list of properties this event needs from the client.
func (SelectionEvent) props() []string {
	return []string{}
}

func buildSelectionEvent(evt Event) SelectionEvent {
	return SelectionEvent{
		Event: evt,
	}
}

// SelectionAPI provides actions and events for text selection in input and textarea elements.
type SelectionAPI[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.Dispatcher
}

func NewSelectionAPI[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.Dispatcher) *SelectionAPI[T] {
	return &SelectionAPI[T]{ref: ref, ctx: ctx}
}

// ============================================================================
// Actions
// ============================================================================

// Select selects all text in the input or textarea element.
//
// Example:
//
//	inputRef := ui.UseElement[*h.InputRef](ctx)
//	inputRef.Select()  // Select all text
//
//	return h.Input(h.Attach(inputRef), h.Value("Select this text"))
func (a *SelectionAPI[T]) Select() {
	dom.DOMCall[T](a.ctx, a.ref, "select")
}

// SetSelectionRange sets the selection range for text in the input or textarea element.
// Start and end are character positions (0-indexed).
//
// Example:
//
//	inputRef := ui.UseElement[*h.InputRef](ctx)
//	inputRef.SetSelectionRange(0, 5)  // Select first 5 characters
//
//	return h.Input(h.Attach(inputRef), h.Value("Hello world"))
func (a *SelectionAPI[T]) SetSelectionRange(start, end int) {
	dom.DOMCall[T](a.ctx, a.ref, "setSelectionRange", start, end)
}

// ============================================================================
// Events
// ============================================================================

// OnSelectStart registers a handler for the "selectstart" event, fired when text selection begins.
//
// Example:
//
//	inputRef := ui.UseElement[*h.InputRef](ctx)
//	inputRef.OnSelectStart(func(evt h.SelectionEvent) h.Updates {
//	    logSelectionStart()
//	    return nil
//	})
//
//	return h.Input(h.Attach(inputRef), h.Value("Selectable text"))
func (a *SelectionAPI[T]) OnSelectStart(handler func(SelectionEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildSelectionEvent(evt)) }
	a.ref.AddListener("selectstart", wrapped, SelectionEvent{}.props())
}

// OnSelectionChange registers a handler for the "selectionchange" event, fired when the text selection changes.
//
// Example:
//
//	inputRef := ui.UseElement[*h.InputRef](ctx)
//	inputRef.OnSelectionChange(func(evt h.SelectionEvent) h.Updates {
//	    updateSelectionIndicator()
//	    return nil
//	})
//
//	return h.Input(h.Attach(inputRef), h.Value("Selectable text"))
func (a *SelectionAPI[T]) OnSelectionChange(handler func(SelectionEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildSelectionEvent(evt)) }
	a.ref.AddListener("selectionchange", wrapped, SelectionEvent{}.props())
}
