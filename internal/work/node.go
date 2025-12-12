package work

import "github.com/eleven-am/pondlive/internal/metadata"

type Node interface {
	Item
	workNode()
}

type Element struct {
	Tag      string
	Attrs    map[string][]string
	Style    map[string]string
	Children []Node

	UnsafeHTML string

	Key   string
	RefID string

	Handlers map[string]Handler

	Script     *metadata.ScriptMeta
	Stylesheet *metadata.Stylesheet

	Descriptor ElementDescriptor
	Metadata   map[string]any
}

type Text struct {
	Value string
}

type Comment struct {
	Value string
}

type Fragment struct {
	Children []Node
	Attrs    []Item
	Metadata map[string]any
}

type ComponentNode struct {
	Fn            any
	Props         any
	InputChildren []Node
	InputAttrs    []Item
	Key           string
	Name          string
}

type PortalNode struct {
	Children []Node
}

type PortalTarget struct{}

func (e *Element) workNode()       {}
func (t *Text) workNode()          {}
func (c *Comment) workNode()       {}
func (f *Fragment) workNode()      {}
func (c *ComponentNode) workNode() {}
func (p *PortalNode) workNode()    {}
func (p *PortalTarget) workNode()  {}

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
	for _, attr := range f.Attrs {
		attr.ApplyTo(parent)
	}
	parent.Children = append(parent.Children, f)
}

func (c *ComponentNode) ApplyTo(parent *Element) {
	parent.Children = append(parent.Children, c)
}

func (p *PortalNode) ApplyTo(parent *Element) {
	parent.Children = append(parent.Children, p)
}

func (p *PortalTarget) ApplyTo(parent *Element) {
	parent.Children = append(parent.Children, p)
}
