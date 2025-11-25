package work

import "github.com/eleven-am/pondlive/go/internal/metadata"

// Node represents a node in the ephemeral work tree.
// All Nodes are also Items (can be applied as children to Elements).
type Node interface {
	Item
	workNode()
}

// Element represents an HTML element.
type Element struct {
	Tag      string
	Attrs    map[string][]string
	Style    map[string]string
	Children []Node

	UnsafeHTML string // Mutually exclusive with Children

	Key   string
	RefID string

	Handlers map[string]Handler // Handler includes Fn + EventOptions

	Script     *metadata.ScriptMeta
	Stylesheet *metadata.Stylesheet

	Descriptor ElementDescriptor // Non-serialized type safety
	Metadata   map[string]any    // Non-serialized arbitrary data
}

// Text represents a text node.
type Text struct {
	Value string
}

// Comment represents a comment node.
type Comment struct {
	Value string
}

// Fragment represents a collection of children with no wrapper.
type Fragment struct {
	Children []Node
	Metadata map[string]any // Non-serialized arbitrary data
}

// Component represents a call to a component function.
type Component struct {
	Fn            any    // func(runtime.Ctx, any, []Node) Node
	Props         any    // Props to pass to the component
	InputChildren []Node // Children passed from parent
	Key           string // Optional reconciliation key
}

func (e *Element) workNode()   {}
func (t *Text) workNode()      {}
func (c *Comment) workNode()   {}
func (f *Fragment) workNode()  {}
func (c *Component) workNode() {}

// ApplyTo implements Item interface for all nodes
func (e *Element) ApplyTo(parent *Element) {
	parent.Children = append(parent.Children, e)
}

func (t *Text) ApplyTo(parent *Element) {
	parent.Children = append(parent.Children, t)
}

func (c *Comment) ApplyTo(parent *Element) {
	parent.Children = append(parent.Children, c)
}

func (f *Fragment) ApplyTo(parent *Element) {
	parent.Children = append(parent.Children, f)
}

func (c *Component) ApplyTo(parent *Element) {
	parent.Children = append(parent.Children, c)
}
