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
//	button := html.NewElementRef[html.ButtonDescriptor]("submitBtn", html.ButtonDescriptor{})
//	// Use button.Ref() to pass to handlers or store in component state
//	buttonRef := button.Ref()
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
//
// Example:
//
//	input := html.NewElementRef[html.InputDescriptor]("email", html.InputDescriptor{})
//	// Get internal DOM ref for passing to API constructors
//	domRef := input.DOMElementRef()
//	inputAPI := html.NewInteractionAPI(domRef, ctx)
//
// Note: This method exposes the internal DOM layer. Use it when you need to create API instances
// (InteractionAPI, MediaAPI, etc.) or when integrating with framework internals.
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
//	button.On("customEvent", func(evt html.Event) html.Updates {
//	    if detail, ok := evt.Payload["detail"].(string); ok {
//	        fmt.Println("Detail:", detail)
//	    }
//	    return nil
//	})
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
//
// Example:
//
//	button := html.NewElementRef[html.ButtonDescriptor]("submitBtn", html.ButtonDescriptor{})
//	// Create element and attach ref
//	element := html.Button(
//	    html.Attach(button), // Attaches the ref to this button
//	    html.Text("Submit"),
//	)
//
// Note: This method is typically called via the Attach() prop rather than directly.
// The attachment establishes bidirectional communication between server and client for this element.
func (r *ElementRef[T]) AttachTo(e *Element) {
	if r == nil {
		return
	}
	dom.AttachElementRef[T](r.ElementRef, e)
}

// NewElementRef creates a new typed element reference with a unique ID and descriptor.
// Element refs enable imperative control over DOM elements and event handling from the server.
//
// Example:
//
//	// Create a video element ref
//	video := html.NewElementRef[html.VideoDescriptor]("player", html.VideoDescriptor{})
//	videoAPI := html.NewMediaAPI(video.DOMElementRef(), ctx)
//
//	// Use in your component
//	videoElement := html.Video(
//	    html.Attach(video),
//	    html.Src("/video.mp4"),
//	)
//
//	// Control programmatically
//	videoAPI.Play()
//
// Note: The ID must be unique within the component scope. The descriptor type determines
// which element type this ref can be attached to, providing compile-time type safety.
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
//	input := html.NewElementRef[html.InputDescriptor]("username", html.InputDescriptor{})
//	inputAPI := html.NewInteractionAPI(input.DOMElementRef(), ctx)
//
//	// Attach ref to input element
//	element := html.Input(
//	    html.Attach(input),     // Links the ref to this input
//	    html.Type("text"),
//	    html.Placeholder("Enter username"),
//	)
//
//	// Now you can control the input from server code
//	inputAPI.Focus()
//	inputAPI.OnKeyDown(func(evt html.KeyboardEvent) html.Updates {
//	    if evt.Key == "Enter" {
//	        submitForm()
//	    }
//	    return nil
//	})
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
