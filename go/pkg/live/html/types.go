package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	internalhtml "github.com/eleven-am/pondlive/go/internal/html"
)

type (
	// Core types from dom
	Item              = dom.Item
	Node              = *dom.StructuredNode
	ElementDescriptor = dom.ElementDescriptor
	Event             = dom.Event
	Modifiers         = dom.Modifiers
	Updates           = dom.Updates
	EventHandler      = dom.EventHandler
	EventOptions      = dom.EventOptions
	EventBinding      = dom.EventBinding

	// Event types from internal/html
	AnimationEvent   = internalhtml.AnimationEvent
	ClickEvent       = internalhtml.ClickEvent
	ClipboardEvent   = internalhtml.ClipboardEvent
	CompositionEvent = internalhtml.CompositionEvent
	DialogEvent      = internalhtml.DialogEvent
	DragEvent        = internalhtml.DragEvent
	FocusEvent       = internalhtml.FocusEvent
	FormEvent        = internalhtml.FormEvent
	FullscreenEvent  = internalhtml.FullscreenEvent
	HashChangeEvent  = internalhtml.HashChangeEvent
	InputEvent       = internalhtml.InputEvent
	KeyboardEvent    = internalhtml.KeyboardEvent
	LifecycleEvent   = internalhtml.LifecycleEvent
	LoadEvent        = internalhtml.LoadEvent
	MediaEvent       = internalhtml.MediaEvent
	MouseEvent       = internalhtml.MouseEvent
	PointerEvent     = internalhtml.PointerEvent
	PrintEvent       = internalhtml.PrintEvent
	ResizeEvent      = internalhtml.ResizeEvent
	ScrollEvent      = internalhtml.ScrollEvent
	SelectionEvent   = internalhtml.SelectionEvent
	StorageEvent     = internalhtml.StorageEvent
	ToggleEvent      = internalhtml.ToggleEvent
	TouchEvent       = internalhtml.TouchEvent
	TransitionEvent  = internalhtml.TransitionEvent
	VisibilityEvent  = internalhtml.VisibilityEvent
	WheelEvent       = internalhtml.WheelEvent

	// Ref types from dom
	ElementRef[T dom.ElementDescriptor] = dom.ElementRef[T]
	RefListener                         = dom.RefListener
)
