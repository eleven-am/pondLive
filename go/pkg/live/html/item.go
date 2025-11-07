package html

// Item is either a Node or a Prop applied to an Element.
type Item interface {
	applyTo(*Element)
}

// Prop mutates an element's metadata.
type Prop interface {
	Item
	isProp()
}

// Apply methods to treat nodes as items.
func (n *Element) applyTo(e *Element)      { e.Children = append(e.Children, n) }
func (t *TextNode) applyTo(e *Element)     { e.Children = append(e.Children, t) }
func (f *FragmentNode) applyTo(e *Element) { e.Children = append(e.Children, f) }
func (c *CommentNode) applyTo(e *Element)  { e.Children = append(e.Children, c) }
func (c *ComponentNode) applyTo(e *Element) {
	e.Children = append(e.Children, c)
}

func el(desc ElementDescriptor, tag string, items ...Item) *Element {
	e := &Element{Tag: tag, Descriptor: desc}
	for _, it := range items {
		if it == nil {
			continue
		}
		it.applyTo(e)
	}
	return e
}
