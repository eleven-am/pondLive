package runtime

import (
	"testing"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type noopProps struct{}

func noopRouterComponent(ctx Ctx, _ noopProps) h.Node {
	return h.Fragment()
}

func TestLinkPlaceholdersAreSessionScoped(t *testing.T) {
	sessA := NewSession(noopRouterComponent, noopProps{})
	sessB := NewSession(noopRouterComponent, noopProps{})

	frag := h.Fragment()
	node := &linkNode{FragmentNode: frag, props: LinkProps{To: "/a"}}

	sessA.storeLinkPlaceholder(frag, node)

	if got, ok := consumeLinkPlaceholder(sessB, frag); ok || got != nil {
		t.Fatalf("expected no placeholder for session B, got ok=%v node=%v", ok, got)
	}

	got, ok := consumeLinkPlaceholder(sessA, frag)
	if !ok || got != node {
		t.Fatalf("expected placeholder for session A, ok=%v node=%v", ok, got)
	}

	if _, ok := consumeLinkPlaceholder(sessA, frag); ok {
		t.Fatal("expected placeholder to be cleared after consumption")
	}
}

func TestRoutesPlaceholdersDoNotLeakBetweenSessions(t *testing.T) {
	sessA := NewSession(noopRouterComponent, noopProps{})
	sessB := NewSession(noopRouterComponent, noopProps{})

	frag := h.Fragment()
	noopRoute := func(ctx Ctx, _ Match) h.Node { return h.Fragment() }
	node := &routesNode{FragmentNode: frag, entries: []routeEntry{{pattern: "/", component: noopRoute}}}

	sessA.storeRoutesPlaceholder(frag, node)

	if got, ok := consumeRoutesPlaceholder(sessB, frag); ok || got != nil {
		t.Fatalf("expected no routes placeholder for session B, ok=%v node=%v", ok, got)
	}

	got, ok := consumeRoutesPlaceholder(sessA, frag)
	if !ok || got != node {
		t.Fatalf("expected routes placeholder for session A, ok=%v node=%v", ok, got)
	}

	if _, ok := consumeRoutesPlaceholder(sessA, frag); ok {
		t.Fatal("expected routes placeholder to be cleared after consumption")
	}
}
