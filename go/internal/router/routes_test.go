package router

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestRenderRoutesRendersNestedOutlet(t *testing.T) {
	layout := func(ctx runtime.Ctx, match Match) dom.Node {
		return h.Div(h.Text("Layout"), Outlet(ctx))
	}
	profile := func(ctx runtime.Ctx, match Match) dom.Node {
		return h.Span(h.Text("Profile"))
	}
	security := func(ctx runtime.Ctx, match Match) dom.Node {
		return h.Span(h.Text("Security"))
	}

	store := NewStore(Location{Path: "/settings/profile"})
	var ctx runtime.Ctx
	node := RenderRoutes(ctx, store,
		Route(ctx, RouteProps{Path: "/", Component: func(ctx runtime.Ctx, _ Match) dom.Node { return h.Text("Home") }}),
		Route(ctx, RouteProps{Path: "/settings/*", Component: layout},
			Route(ctx, RouteProps{Path: "./profile", Component: profile}),
			Route(ctx, RouteProps{Path: "./security", Component: security}),
		),
	)

	root, ok := node.(*dom.Element)
	if !ok || root.Tag != "div" || len(root.Children) != 2 {
		t.Fatalf("expected layout div, got %#v", node)
	}
}

func TestRenderRoutesMergesParamsIntoStore(t *testing.T) {
	store := NewStore(Location{Path: "/users/123/posts/456"})
	var ctx runtime.Ctx
	node := RenderRoutes(ctx, store,
		Route(ctx, RouteProps{Path: "/users/:userID/*", Component: func(ctx runtime.Ctx, _ Match) dom.Node { return h.Text("User") }},
			Route(ctx, RouteProps{Path: "./posts/:postID", Component: func(ctx runtime.Ctx, _ Match) dom.Node { return h.Text("Post") }}),
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
	var ctx runtime.Ctx
	node := RenderRoutes(ctx, store, Route(ctx, RouteProps{Path: "/home", Component: func(ctx runtime.Ctx, _ Match) dom.Node { return h.Text("Home") }}))
	if _, ok := node.(*dom.FragmentNode); !ok {
		t.Fatalf("expected fragment when no match, got %#v", node)
	}
	if params := store.Params(); len(params) != 0 {
		t.Fatalf("expected params cleared on miss, got %#v", params)
	}
}

func TestRenderRoutesWithSeededStore(t *testing.T) {
	t.Skip("covered by router integration tests")
	store := NewStore(Location{})
	store.Seed(Location{Path: "/users/7"}, nil, nil)
	var ctx runtime.Ctx
	node := RenderRoutes(ctx, store,
		Route(ctx, RouteProps{Path: "/users/:id", Component: func(ctx runtime.Ctx, match Match) dom.Node {
			if match.Params["id"] == "" {
				t.Fatalf("expected params from seed")
			}
			return h.Div(h.Text(match.Params["id"]))
		}}),
	)
	el, ok := node.(*dom.Element)
	if !ok || el.Tag != "div" {
		t.Fatalf("expected seeded div render, got %#v", node)
	}
	if text, _ := el.Children[0].(*dom.TextNode); text == nil || text.Value != "7" {
		t.Fatalf("expected text node with id, got %#v", el.Children)
	}
	params := store.Params()
	if params["id"] != "7" {
		t.Fatalf("expected params stored after render, got %#v", params)
	}
}

func TestRouteMatchParamsMerge(t *testing.T) {
	root := &RouteMatch{
		Match: Match{Params: map[string]string{"user": "1"}},
		Child: &RouteMatch{
			Match: Match{Params: map[string]string{"post": "2"}},
			Child: &RouteMatch{Match: Match{Params: map[string]string{"comment": "3"}}},
		},
	}
	params := root.Params()
	if params["user"] != "1" || params["post"] != "2" || params["comment"] != "3" {
		t.Fatalf("expected merged params, got %#v", params)
	}
}

func TestRouteTreeResolveClearsParamsOnMiss(t *testing.T) {
	store := NewStore(Location{Path: "/unknown"})
	store.SetParams(map[string]string{"old": "value"})
	tree := CompileTree("/", DefineRoute("/known", func(ctx runtime.Ctx, rctx RouteRenderContext) h.Node {
		return h.Text("known")
	}))
	if match := tree.Resolve(store); match != nil {
		t.Fatalf("expected no match, got %#v", match)
	}
	if params := store.Params(); len(params) != 0 {
		t.Fatalf("expected params cleared when no match, got %#v", params)
	}
}
