package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom2"
	internalhtml "github.com/eleven-am/pondlive/go/internal/html"
)

type (
	// Core types from dom2
	Item              = dom2.Item
	Node              = *dom2.StructuredNode
	ElementDescriptor = dom2.ElementDescriptor
	Event             = dom2.Event
	Modifiers         = dom2.Modifiers
	Updates           = dom2.Updates
	EventHandler      = dom2.EventHandler
	EventOptions      = dom2.EventOptions
	EventBinding      = dom2.EventBinding

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

	// Ref and upload types from dom2
	ElementRef[T dom2.ElementDescriptor] = dom2.ElementRef[T]
	RefListener                          = dom2.RefListener
	UploadBinding                        = dom2.UploadBinding
)
