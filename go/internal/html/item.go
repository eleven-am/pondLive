package html

import "github.com/eleven-am/pondlive/go/internal/dom"

// Prop mutates an element's metadata.
type Prop interface {
	dom.Item
	isProp()
}

func el(desc ElementDescriptor, tag string, items ...Item) *Element {
	e := &Element{Tag: tag, Descriptor: desc}
	for _, it := range items {
		if it == nil {
			continue
		}
		it.ApplyTo(e)
	}
	return e
}

// El constructs an Element with the provided descriptor, tag name, and content.
func El(desc ElementDescriptor, tag string, items ...Item) *Element {
	return el(desc, tag, items...)
}
