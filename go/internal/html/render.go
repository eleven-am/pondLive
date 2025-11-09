package html

import "github.com/eleven-am/pondlive/go/internal/dom"

func RenderHTML(n Node) string { return dom.RenderHTML(n) }

func ComponentStartMarker(id string) string { return dom.ComponentStartComment(id) }

func ComponentEndMarker(id string) string { return dom.ComponentEndComment(id) }

func ComponentCommentPrefix() string { return dom.ComponentCommentPrefix() }
