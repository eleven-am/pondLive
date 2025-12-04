package view

import "github.com/eleven-am/pondlive/internal/metadata"

type Node interface {
	viewNode()
}

type Element struct {
	Tag      string              `json:"tag,omitempty"`
	Attrs    map[string][]string `json:"attrs,omitempty"`
	Style    map[string]string   `json:"style,omitempty"`
	Children []Node              `json:"children,omitempty"`

	UnsafeHTML string `json:"unsafeHtml,omitempty"`

	Key   string `json:"key,omitempty"`
	RefID string `json:"refId,omitempty"`

	Handlers []metadata.HandlerMeta `json:"handlers,omitempty"`

	Script     *metadata.ScriptMeta `json:"script,omitempty"`
	Stylesheet *metadata.Stylesheet `json:"stylesheet,omitempty"`
}

type Text struct {
	Text string `json:"text,omitempty"`
}

type Comment struct {
	Comment string `json:"comment,omitempty"`
}

type Fragment struct {
	Fragment bool   `json:"fragment"`
	Children []Node `json:"children,omitempty"`
}

func (e *Element) viewNode()  {}
func (t *Text) viewNode()     {}
func (c *Comment) viewNode()  {}
func (f *Fragment) viewNode() {}
