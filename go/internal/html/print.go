package html

import "github.com/eleven-am/pondlive/go/internal/dom2"

// PrintEvent represents print dialog events.
type PrintEvent struct {
	dom2.Event
}

// Props returns the list of properties this event needs from the client.
func (PrintEvent) props() []string {
	return []string{}
}

// PrintHandler provides print event handlers.
type PrintHandler struct {
	ref dom2.RefListener
}

// NewPrintHandler creates a new PrintHandler.
func NewPrintHandler(ref dom2.RefListener) *PrintHandler {
	return &PrintHandler{ref: ref}
}

// OnBeforePrint registers a handler for the "beforeprint" event.
func (h *PrintHandler) OnBeforePrint(handler func(PrintEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildPrintEvent(evt)) }
	h.ref.AddListener("beforeprint", wrapped, PrintEvent{}.props())
}

// OnAfterPrint registers a handler for the "afterprint" event.
func (h *PrintHandler) OnAfterPrint(handler func(PrintEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildPrintEvent(evt)) }
	h.ref.AddListener("afterprint", wrapped, PrintEvent{}.props())
}

func buildPrintEvent(evt dom2.Event) PrintEvent {
	return PrintEvent{
		Event: evt,
	}
}
