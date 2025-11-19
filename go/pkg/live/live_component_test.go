package live

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/runtime"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestComponentInvokesRuntimeRender(t *testing.T) {
	var renders int
	counter := Component(func(ctx Ctx) h.Node {
		renders++
		return h.Div()
	})

	root := runtime.Component[struct{}](func(ctx runtime.Ctx, _ struct{}) *h.StructuredNode {
		return counter(ctx)
	})

	sess := runtime.NewSession(root, struct{}{})
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	node := sess.Tree()
	if renders != 1 {
		t.Fatalf("expected component to render once, got %d", renders)
	}
	if node == nil {
		t.Fatalf("expected node, got nil")
	}
}

func TestPropsComponentForwardsProps(t *testing.T) {
	type props struct {
		Label string
	}
	var seen props
	card := PropsComponent(func(ctx Ctx, p props) h.Node {
		seen = p
		return h.Div(h.Text(p.Label))
	})

	root := runtime.Component[struct{}](func(ctx runtime.Ctx, _ struct{}) *h.StructuredNode {
		return card(ctx, props{Label: "hello"})
	})

	sess := runtime.NewSession(root, struct{}{})
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if seen.Label != "hello" {
		t.Fatalf("expected props to forward, got %q", seen.Label)
	}
}

func TestComponentForwardsRenderOptions(t *testing.T) {
	child := Component(func(ctx Ctx) h.Node {
		return h.Div()
	})

	root := runtime.Component[struct{}](func(ctx runtime.Ctx, _ struct{}) *h.StructuredNode {
		return h.Fragment(
			child(ctx, WithKey("dup")),
			child(ctx, WithKey("dup")),
		)
	})

	sess := runtime.NewSession(root, struct{}{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected duplicate key panic")
		}
	}()
	_ = sess.Flush()
}
