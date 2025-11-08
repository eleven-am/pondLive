package html

// Node is anything that renders into markup.
//
// Nodes can be used anywhere an Item is expected so they can compose
// seamlessly with the HTML builders (e.g. appending child components).
// Concrete node types already satisfy applyTo via pointer receivers; making
// Item part of the Node contract exposes that behaviour on the interface.
type Node interface {
	Item
	isNode()
	privateNodeTag()
}

// Element represents an HTML element node.
type Element struct {
	Tag        string
	Attrs      map[string]string
	Class      []string
	Style      map[string]string
	Children   []Node
	Descriptor ElementDescriptor

	Key    string
	Events map[string]EventBinding
	Unsafe *string
	RefID  string

	MutableAttrs map[string]bool
}

func (*Element) isNode()         {}
func (*Element) privateNodeTag() {}

// TextNode is a text node; Value is escaped at render time.
type TextNode struct {
	Value   string
	Mutable bool
}

func (*TextNode) isNode()         {}
func (*TextNode) privateNodeTag() {}

// FragmentNode groups siblings without a wrapper element.
type FragmentNode struct {
	Children []Node
}

func (*FragmentNode) isNode()         {}
func (*FragmentNode) privateNodeTag() {}

// CommentNode renders an HTML comment node.
type CommentNode struct {
	Value string
}

func (*CommentNode) isNode()         {}
func (*CommentNode) privateNodeTag() {}

// ComponentNode wraps a rendered component subtree so render passes can
// annotate and track its template spans.
type ComponentNode struct {
	ID    string
	Child Node
}

func (*ComponentNode) isNode()         {}
func (*ComponentNode) privateNodeTag() {}
