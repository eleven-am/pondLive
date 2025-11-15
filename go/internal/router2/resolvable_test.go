package router2

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

func TestResolveTreeReplacesRoutesAndLinks(t *testing.T) {
	store := NewStore(Location{Path: "/settings/profile"})
	root := &dom.Element{
		Tag: "main",
		Children: []dom.Node{
			RoutesNode("/",
				Route("/settings/*", func(ctx RouteRenderContext) dom.Node {
					return &dom.Element{Tag: "section", Children: []dom.Node{ctx.RenderOutlet()}}
				},
					Route("./profile", func(RouteRenderContext) dom.Node {
						return LinkNode(LinkProps{To: "./edit"}, &dom.TextNode{Value: "Edit"})
					}),
				),
			),
		},
	}

	resolved := ResolveTree(root, store)
	el, ok := resolved.(*dom.Element)
	if !ok || el.Tag != "main" {
		t.Fatalf("expected main element, got %#v", resolved)
	}
	section, ok := el.Children[0].(*dom.Element)
	if !ok || section.Tag != "section" {
		t.Fatalf("expected section child, got %#v", el.Children[0])
	}
	anchor, ok := section.Children[0].(*dom.Element)
	if !ok || anchor.Tag != "a" {
		t.Fatalf("expected link child, got %#v", section.Children[0])
	}
	if href := anchor.Attrs["href"]; href != "/settings/profile/edit" {
		t.Fatalf("expected resolved href, got %q", href)
	}
}
