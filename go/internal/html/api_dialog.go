package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom2"
)

// DialogEvent represents dialog events (close, cancel).
type DialogEvent struct {
	dom2.Event
}

// props returns the list of properties this event needs from the client.
func (DialogEvent) props() []string {
	return []string{}
}

func buildDialogEvent(evt dom2.Event) DialogEvent {
	return DialogEvent{
		Event: evt,
	}
}

// DialogAPI provides actions and events for dialog elements.
type DialogAPI[T dom2.ElementDescriptor] struct {
	ref *dom2.ElementRef[T]
	ctx dom2.Dispatcher
}

func NewDialogAPI[T dom2.ElementDescriptor](ref *dom2.ElementRef[T], ctx dom2.Dispatcher) *DialogAPI[T] {
	return &DialogAPI[T]{ref: ref, ctx: ctx}
}

// ============================================================================
// Actions
// ============================================================================

// Show displays the dialog non-modally (without blocking interaction with other page content).
//
// Example:
//
//	dialogRef := ui.UseElement[*h.DialogRef](ctx)
//	dialogRef.Show()
//
//	return h.Dialog(h.Attach(dialogRef), h.Text("Non-modal dialog"))
func (a *DialogAPI[T]) Show() {
	dom2.DOMCall[T](a.ctx, a.ref, "show")
}

// ShowModal displays the dialog modally with backdrop and focus trap.
// This blocks interaction with other page content until the dialog is closed.
//
// Example:
//
//	dialogRef := ui.UseElement[*h.DialogRef](ctx)
//	dialogRef.ShowModal()
//
//	return h.Dialog(h.Attach(dialogRef), h.Text("Modal dialog"))
func (a *DialogAPI[T]) ShowModal() {
	dom2.DOMCall[T](a.ctx, a.ref, "showModal")
}

// Close closes the dialog with an optional return value.
// The return value can be accessed in the OnClose event handler.
//
// Example:
//
//	dialogRef := ui.UseElement[*h.DialogRef](ctx)
//	dialogRef.Close("confirmed")  // Close with return value
//
//	return h.Dialog(h.Attach(dialogRef), h.Text("Dialog content"))
func (a *DialogAPI[T]) Close(returnValue string) {
	if returnValue == "" {
		dom2.DOMCall[T](a.ctx, a.ref, "close")
	} else {
		dom2.DOMCall[T](a.ctx, a.ref, "close", returnValue)
	}
}

// ============================================================================
// Events
// ============================================================================

// OnClose registers a handler for the "close" event, fired when the dialog is closed.
//
// Example:
//
//	dialogRef := ui.UseElement[*h.DialogRef](ctx)
//	dialogRef.OnClose(func(evt h.DialogEvent) h.Updates {
//	    handleDialogClose()
//	    return nil
//	})
//
//	return h.Dialog(h.Attach(dialogRef), h.Text("Dialog content"))
func (a *DialogAPI[T]) OnClose(handler func(DialogEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildDialogEvent(evt)) }
	a.ref.AddListener("close", wrapped, DialogEvent{}.props())
}

// OnCancel registers a handler for the "cancel" event, fired when user presses Escape key.
// Call evt.PreventDefault() to prevent the dialog from closing.
//
// Example:
//
//	dialogRef := ui.UseElement[*h.DialogRef](ctx)
//	dialogRef.OnCancel(func(evt h.DialogEvent) h.Updates {
//	    if hasUnsavedChanges() {
//	        evt.PreventDefault()  // Prevent closing
//	        showConfirmation()
//	    }
//	    return nil
//	})
//
//	return h.Dialog(h.Attach(dialogRef), h.Text("Dialog content"))
func (a *DialogAPI[T]) OnCancel(handler func(DialogEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildDialogEvent(evt)) }
	a.ref.AddListener("cancel", wrapped, DialogEvent{}.props())
}
