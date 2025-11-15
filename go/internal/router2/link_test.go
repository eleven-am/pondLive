package router2

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

func TestRenderLinkResolvesRelativeHref(t *testing.T) {
	store := NewStore(Location{Path: "/users", Query: map[string][]string{"page": {"1"}}})
	el := RenderLink(store, LinkProps{To: "./details"}, &dom.TextNode{Value: "Details"})
	if href := el.Attrs["href"]; href != "/users/details?page=1" {
		t.Fatalf("expected resolved href, got %q", href)
	}
	if len(el.Children) != 1 {
		t.Fatalf("expected child node")
	}
}

func TestRenderLinkFallsBackWithoutStore(t *testing.T) {
	el := RenderLink(nil, LinkProps{To: "/absolute"})
	if href := el.Attrs["href"]; href != "/absolute" {
		t.Fatalf("expected passthrough href, got %q", href)
	}
}
