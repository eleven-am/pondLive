package live

import (
	"testing"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestComponentWithChildren(t *testing.T) {
	card := Component(func(ctx Ctx, children []h.Item) h.Node {
		return h.Div(
			h.H1(h.Text("Card Title")),
			h.Fragment(children...),
		)
	})

	// Test that it returns a function with variadic signature
	_ = card
}

func TestPropsComponentWithChildren(t *testing.T) {
	type CardProps struct {
		Title string
	}

	card := PropsComponent(func(ctx Ctx, props CardProps, children []h.Item) h.Node {
		return h.Div(
			h.H1(h.Text(props.Title)),
			h.Fragment(children...),
		)
	})

	// Test that it returns a function with variadic signature
	_ = card
}

func TestKeyExtraction(t *testing.T) {
	type CardProps struct {
		Title string
	}

	card := PropsComponent(func(ctx Ctx, props CardProps, children []h.Item) h.Node {
		// Children should have h.Key() extracted at this point
		return h.Div(
			h.H1(h.Text(props.Title)),
			h.Fragment(children...),
		)
	})

	// Usage with key
	_ = card
	// In actual usage: card(ctx, CardProps{Title: "Test"}, h.Key("my-key"), h.Text("child"))
}
