// Package html provides a public façade for the LiveUI HTML package.
// It re-exports the safe public API from internal/html while keeping
// implementation details hidden from external users.
package html

import internalhtml "github.com/eleven-am/pondlive/go/internal/html"

// Core interfaces and types
type (
	// Node is anything that renders into markup.
	Node = internalhtml.Node

	// Element represents an HTML element node.
	Element = internalhtml.Element

	// ComponentNode wraps a rendered component subtree.
	ComponentNode = internalhtml.ComponentNode

	// TextNode is a plain text node.
	TextNode = internalhtml.TextNode

	// FragmentNode renders a group of children without a wrapper.
	FragmentNode = internalhtml.FragmentNode

	// CommentNode renders an HTML comment.
	CommentNode = internalhtml.CommentNode

	// Item is either a Node or a Prop applied to an Element.
	Item = internalhtml.Item

	// Prop mutates an element's metadata.
	Prop = internalhtml.Prop

	// ElementDescriptor describes the compile-time identity of a generated HTML element.
	ElementDescriptor = internalhtml.ElementDescriptor

	// ElementRef provides a typed handle to a rendered element instance.
	ElementRef[T ElementDescriptor] = internalhtml.ElementRef[T]

	// LinkTag describes a <link> element for metadata.
	LinkTag = internalhtml.LinkTag
)

// Event types
type (
	// Event represents a DOM event payload delivered to the server.
	Event = internalhtml.Event

	// Modifiers captures keyboard and mouse modifier state for an event.
	Modifiers = internalhtml.Modifiers

	// Updates marks a handler return value that can trigger rerenders.
	Updates = internalhtml.Updates

	// EventHandler represents a server-side event handler for a DOM event.
	EventHandler = internalhtml.EventHandler

	// EventOptions configures additional metadata for a DOM event handler.
	EventOptions = internalhtml.EventOptions
)

// Builder helpers
var (
	// Text creates an escaped text node.
	Text = internalhtml.Text

	// Textf formats according to fmt.Sprintf and wraps result in a text node.
	Textf = internalhtml.Textf

	// Fragment constructs a fragment node from children.
	Fragment = internalhtml.Fragment

	// Comment creates an HTML comment node.
	Comment = internalhtml.Comment

	// WrapComponent wraps a component subtree so render passes can attach metadata.
	WrapComponent = internalhtml.WrapComponent

	// If includes the node when cond is true; otherwise it contributes nothing.
	If = internalhtml.If

	// IfFn evaluates fn when cond is true.
	IfFn = internalhtml.IfFn

	// Ternary returns whenTrue when cond is true, otherwise whenFalse.
	Ternary = internalhtml.Ternary
)

// Prop functions
var (
	// Attr sets an arbitrary attribute on the element.
	Attr = internalhtml.Attr

	// MutableAttr sets an attribute and marks it as dynamic.
	MutableAttr = internalhtml.MutableAttr

	// ID sets the id attribute.
	ID = internalhtml.ID

	// Href sets the href attribute.
	Href = internalhtml.Href

	// Src sets the src attribute.
	Src = internalhtml.Src

	// Target sets the target attribute.
	Target = internalhtml.Target

	// Rel sets the rel attribute.
	Rel = internalhtml.Rel

	// Title sets the title attribute.
	Title = internalhtml.Title

	// Alt sets the alt attribute.
	Alt = internalhtml.Alt

	// Type sets the type attribute.
	Type = internalhtml.Type

	// Value sets the value attribute.
	Value = internalhtml.Value

	// Name sets the name attribute.
	Name = internalhtml.Name

	// Data sets a data-* attribute.
	Data = internalhtml.Data

	// Aria sets an aria-* attribute.
	Aria = internalhtml.Aria

	// Class appends CSS class tokens to the element.
	Class = internalhtml.Class

	// Style sets an inline style declaration.
	Style = internalhtml.Style

	// Key assigns a stable identity for keyed lists.
	Key = internalhtml.Key

	// UnsafeHTML sets pre-escaped inner HTML for the element.
	UnsafeHTML = internalhtml.UnsafeHTML

	// On attaches an event handler for the named DOM event.
	On = internalhtml.On

	// OnWith attaches an event handler together with additional options.
	OnWith = internalhtml.OnWith

	// LinkTags renders link descriptors into HTML nodes.
	LinkTags = internalhtml.LinkTags
)

// Attach binds an element ref to an element instance.
func Attach[T ElementDescriptor](ref *ElementRef[T]) Prop {
	return internalhtml.Attach(ref)
}
