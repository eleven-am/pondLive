package html

import "github.com/eleven-am/pondlive/go/internal/dom"

type (
	Event        = dom.Event
	Modifiers    = dom.Modifiers
	Updates      = dom.Updates
	EventHandler = dom.EventHandler
	EventOptions = dom.EventOptions
	EventBinding = dom.EventBinding
)

func Rerender() Updates { return dom.Rerender() }

func mergeEventOptions(base, extra EventOptions) EventOptions {
	return dom.MergeEventOptions(base, extra)
}

func defaultEventOptions(event string) EventOptions {
	return dom.DefaultEventOptions(event)
}

func mergeEventBinding(existing, addition EventBinding) EventBinding {
	return dom.MergeEventBinding(existing, addition)
}
