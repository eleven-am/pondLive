package view

import "github.com/eleven-am/pondlive/go/internal/metadata"

// Node represents a node in the persistent view tree.
type Node interface {
	viewNode()
}

// Element represents an HTML element in the view tree.
type Element struct {
	Tag      string              `json:"tag,omitempty"`
	Attrs    map[string][]string `json:"attrs,omitempty"`
	Style    map[string]string   `json:"style,omitempty"`
	Children []Node              `json:"children,omitempty"` // Only Element/Text/Comment/Fragment (no Components)

	UnsafeHTML string `json:"unsafeHtml,omitempty"` // Mutually exclusive with Children

	Key   string `json:"key,omitempty"`
	RefID string `json:"refId,omitempty"`

	Handlers []metadata.HandlerMeta `json:"handlers,omitempty"` // Handler IDs + EventOptions (not functions)

	Script     *metadata.ScriptMeta `json:"script,omitempty"`
	Stylesheet *metadata.Stylesheet `json:"stylesheet,omitempty"`
}

// Text represents a text node in the view tree.
type Text struct {
	Text string `json:"text,omitempty"`
}

// Comment represents a comment node in the view tree.
type Comment struct {
	Comment string `json:"comment,omitempty"`
}

// Fragment represents a collection of children with no wrapper.
type Fragment struct {
	Fragment bool   `json:"fragment"`
	Children []Node `json:"children,omitempty"`
}

func (e *Element) viewNode()  {}
func (t *Text) viewNode()     {}
func (c *Comment) viewNode()  {}
func (f *Fragment) viewNode() {}
