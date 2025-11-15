package router2

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

func TestRenderRoutesRendersNestedOutlet(t *testing.T) {
	layout := func(ctx RouteRenderContext) dom.Node {
		return &dom.Element{Tag: "div", Children: []dom.Node{
			&dom.TextNode{Value: "Layout"},
			ctx.RenderOutlet(),
		}}
	}
	profile := func(ctx RouteRenderContext) dom.Node {
		return &dom.Element{Tag: "span", Children: []dom.Node{
			&dom.TextNode{Value: "Profile"},
		}}
	}
	security := func(ctx RouteRenderContext) dom.Node {
		return &dom.Element{Tag: "span", Children: []dom.Node{
			&dom.TextNode{Value: "Security"},
		}}
	}

	store := NewStore(Location{Path: "/settings/profile"})
	node := RenderRoutes(store,
		Route("/", func(RouteRenderContext) dom.Node { return &dom.TextNode{Value: "Home"} }),
		Route("/settings/*", layout,
			Route("./profile", profile),
			Route("./security", security),
		),
	)

	root, ok := node.(*dom.Element)
	if !ok || root.Tag != "div" || len(root.Children) != 2 {
		t.Fatalf("expected layout div, got %#v", node)
	}
	if text, ok := root.Children[0].(*dom.TextNode); !ok || text.Value != "Layout" {
		t.Fatalf("expected layout text, got %#v", root.Children[0])
	}
	if child, ok := root.Children[1].(*dom.Element); !ok || child.Tag != "span" {
		t.Fatalf("expected span child, got %#v", root.Children[1])
	}
}

func TestRenderRoutesMergesParamsIntoStore(t *testing.T) {
	store := NewStore(Location{Path: "/users/123/posts/456"})
	node := RenderRoutes(store,
		Route("/users/:userID/*", func(RouteRenderContext) dom.Node { return &dom.TextNode{} },
			Route("./posts/:postID", func(RouteRenderContext) dom.Node { return &dom.TextNode{} }),
		),
	)
	if _, ok := node.(*dom.FragmentNode); ok {
		t.Fatalf("expected non-empty render")
	}
	params := store.Params()
	if params["userID"] != "123" || params["postID"] != "456" {
		t.Fatalf("expected merged params, got %#v", params)
	}
}

func TestRenderRoutesMissingMatchReturnsFragment(t *testing.T) {
	store := NewStore(Location{Path: "/none"})
	node := RenderRoutes(store, Route("/home", func(RouteRenderContext) dom.Node { return &dom.TextNode{} }))
	if _, ok := node.(*dom.FragmentNode); !ok {
		t.Fatalf("expected fragment when no match, got %#v", node)
	}
}
