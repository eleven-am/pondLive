package work

import (
	"fmt"

	"github.com/eleven-am/pondlive/internal/reactive"
)

type Node interface {
	Item
	workNode()
}

type AttrKind uint8

const (
	AttrStatic AttrKind = iota
	AttrPresent
	AttrAbsent
	AttrDynamic
	AttrDynamicPresence
	AttrDynamicOptional
)

type Attr struct {
	Name     string
	Kind     AttrKind
	Value    string
	Source   reactive.Readable[string]
	Presence reactive.Readable[bool]
}

func (a Attr) Dynamic() bool {
	return a.Kind == AttrDynamic || a.Kind == AttrDynamicPresence || a.Kind == AttrDynamicOptional
}

type ClassPart struct {
	Static string
	Source reactive.Readable[string]
}

func (c ClassPart) Dynamic() bool { return c.Source != nil }

type StyleDecl struct {
	Prop   string
	Value  string
	Source reactive.Readable[string]
}

type Raw struct {
	Static string
	Source reactive.Readable[string]
}

type Element struct {
	Tag         string
	Attrs       []Attr
	Classes     []ClassPart
	Styles      []StyleDecl
	Events      []EventBinding
	Children    []Node
	Raw         *Raw
	Attachments []any
	Bind        *BindSpec
	Descriptor  ElementDescriptor
}

type Text struct {
	Value string
}

type BoundText struct {
	Source reactive.Readable[string]
}

type Comment struct {
	Value string
}

type Fragment struct {
	Children []Node
	Attrs    []Item
}

type ComponentNode struct {
	Fn    any
	Ident uintptr
	Name  string
	Props any
}

type RegionNode struct {
	Impl any
}

type PortalNode struct {
	Children []Node
}

type PortalTarget struct{}

type UnitRegion struct {
	Children []Node
}

func (e *Element) workNode()       {}
func (t *Text) workNode()          {}
func (t *BoundText) workNode()     {}
func (c *Comment) workNode()       {}
func (f *Fragment) workNode()      {}
func (c *ComponentNode) workNode() {}
func (r *RegionNode) workNode()    {}
func (p *PortalNode) workNode()    {}
func (p *PortalTarget) workNode()  {}
func (u *UnitRegion) workNode()    {}

func (e *Element) applyTo(parent *Element)       { parent.Children = append(parent.Children, e) }
func (t *Text) applyTo(parent *Element)          { parent.Children = append(parent.Children, t) }
func (t *BoundText) applyTo(parent *Element)     { parent.Children = append(parent.Children, t) }
func (c *Comment) applyTo(parent *Element)       { parent.Children = append(parent.Children, c) }
func (c *ComponentNode) applyTo(parent *Element) { parent.Children = append(parent.Children, c) }
func (r *RegionNode) applyTo(parent *Element)    { parent.Children = append(parent.Children, r) }
func (p *PortalNode) applyTo(parent *Element)    { parent.Children = append(parent.Children, p) }
func (p *PortalTarget) applyTo(parent *Element)  { parent.Children = append(parent.Children, p) }
func (u *UnitRegion) applyTo(parent *Element)    { parent.Children = append(parent.Children, u) }

func (f *Fragment) applyTo(parent *Element) {
	for _, attr := range f.Attrs {
		attr.applyTo(parent)
	}
	parent.Children = append(parent.Children, f)
}

func (e *Element) Dynamic() bool {
	if len(e.Events) > 0 || len(e.Attachments) > 0 || (e.Bind != nil && !e.Bind.Constant()) || (e.Raw != nil && e.Raw.Source != nil) {
		return true
	}
	for _, a := range e.Attrs {
		if a.Dynamic() {
			return true
		}
	}
	for _, c := range e.Classes {
		if c.Dynamic() {
			return true
		}
	}
	for _, s := range e.Styles {
		if s.Source != nil {
			return true
		}
	}
	return false
}

func (e *Element) String() string { return fmt.Sprintf("<%s>", e.Tag) }
