package html

import "github.com/eleven-am/pondlive/go/internal/dom"

type (
	RefListener = dom.RefListener
)

// ElementRef is a typed reference to a DOM element, providing type-safe access to element-specific APIs.
// It wraps the internal dom.ElementRef and exposes a safe, typed interface for interacting with elements.
//
// Example:
//
//	buttonRef := ui.UseElement[*h.ButtonRef](ctx)
//	buttonRef.OnClick(func(evt h.ClickEvent) h.Updates {
//	    handleClick()
//	    return nil
//	})
//
//	return h.Button(h.Attach(buttonRef), h.Text("Submit"))
type ElementRef[T ElementDescriptor] struct {
	*dom.ElementRef[T]
}

// Ref returns the ElementRef itself, primarily used for nil-safe chaining and type assertions.
// This method is useful when working with optional refs or when you need to pass the ref explicitly.
//
// Example:
//
//	var optionalRef *html.ElementRef[html.DivDescriptor]
//	// Safely get ref without panicking on nil
//	if ref := optionalRef.Ref(); ref != nil {
//	    // Use ref safely
//	}
//
// Note: This method returns nil if called on a nil receiver, making it safe for nil-checking patterns.
func (r *ElementRef[T]) Ref() *ElementRef[T] {
	if r == nil {
		return nil
	}
	return r
}

// DOMElementRef returns the underlying dom.ElementRef for advanced use cases or internal framework operations.
// Most users should use the typed ElementRef methods directly rather than accessing the DOM layer.
func (r *ElementRef[T]) DOMElementRef() *dom.ElementRef[T] {
	if r == nil {
		return nil
	}
	return r.ElementRef
}

// On registers a generic event handler that automatically captures all serializable event properties.
// The handler receives an Event with all properties in evt.Payload (map[string]any).
// Users must type-assert values from the payload map themselves.
//
// Example:
//
//	buttonRef := ui.UseElement[*h.ButtonRef](ctx)
//	buttonRef.On("customEvent", func(evt h.Event) h.Updates {
//	    if detail, ok := evt.Payload["detail"].(string); ok {
//	        fmt.Println("Detail:", detail)
//	    }
//	    return nil
//	})
//
//	return h.Button(h.Attach(buttonRef), h.Text("Click me"))
//
// This method coexists with typed methods (OnClick, OnChange, etc.) which provide type safety.
// Use typed methods when available; use On() for custom events or when you need maximum flexibility.
func (r *ElementRef[T]) On(eventName string, handler func(Event) Updates) {
	if r == nil || r.ElementRef == nil || handler == nil {
		return
	}
	r.ElementRef.AddListener(eventName, handler, []string{dom.CaptureAllProperties})
}

// AttachTo attaches this ElementRef to an HTML element, establishing the connection between
// the ref and its corresponding DOM element in the virtual DOM tree.
// This method is typically called via the Attach() prop rather than directly.
func (r *ElementRef[T]) AttachTo(e *Element) {
	if r == nil {
		return
	}
	dom.AttachElementRef[T](r.ElementRef, e)
}

// NewElementRef creates a new typed element reference with a unique ID and descriptor.
// Element refs enable imperative control over DOM elements and event handling from the server.
func NewElementRef[T ElementDescriptor](id string, descriptor T) *ElementRef[T] {
	raw := dom.NewElementRef(id, descriptor)
	if raw == nil {
		return nil
	}
	return &ElementRef[T]{ElementRef: raw}
}

// Attachment interface represents any object that can be attached to an Element.
// ElementRef implements this interface, allowing refs to be attached via the Attach() prop.
type Attachment interface {
	AttachTo(*Element)
}

// Attach creates a Prop that attaches an ElementRef (or other Attachment) to an element.
// This is the standard way to connect refs to their corresponding DOM elements.
//
// Example:
//
//	inputRef := ui.UseElement[*h.InputRef](ctx)
//
//	// Attach ref to input element
//	inputRef.OnKeyDown(func(evt h.KeyboardEvent) h.Updates {
//	    if evt.Key == "Enter" {
//	        submitForm()
//	    }
//	    return nil
//	})
//
//	return h.Input(
//	    h.Attach(inputRef),     // Links the ref to this input
//	    h.Type("text"),
//	    h.Placeholder("Enter username"),
//	)
//
// Note: An element can only have one ref attached. Attaching multiple refs to the same element
// will result in undefined behavior. Use Attach() as a prop when creating the element.
func Attach(target Attachment) Prop {
	if target == nil {
		return nil
	}
	return attachmentProp{target: target}
}

type attachmentProp struct {
	target Attachment
}

func (attachmentProp) isProp() {}

func (p attachmentProp) ApplyTo(e *Element) {
	if e == nil || p.target == nil {
		return
	}
	p.target.AttachTo(e)
}

// no builder registry needed; ref construction handled in generated façade
