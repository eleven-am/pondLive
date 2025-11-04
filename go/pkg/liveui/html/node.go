package html

// Node is anything that renders into markup.
type Node interface {
	isNode()
	privateNodeTag()
}

// Element represents an HTML element node.
type Element struct {
	Tag      string
	Attrs    map[string]string
	Class    []string
	Style    map[string]string
	Children []Node

	Key    string
	Events map[string]EventHandler
	Unsafe *string
}

func (*Element) isNode()         {}
func (*Element) privateNodeTag() {}

// TextNode is a text node; Value is escaped at render time.
type TextNode struct {
	Value string
}

func (*TextNode) isNode()         {}
func (*TextNode) privateNodeTag() {}

// FragmentNode groups siblings without a wrapper element.
type FragmentNode struct {
	Children []Node
}

func (*FragmentNode) isNode()         {}
func (*FragmentNode) privateNodeTag() {}
