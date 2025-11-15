package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	ui "github.com/eleven-am/pondlive/go/pkg/live"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
	"github.com/eleven-am/pondlive/go/pkg/live/server"
)

func main() {
	ctx := context.Background()
	app, err := server.NewApp(ctx, HomePage, server.WithDevMode(true))
	if err != nil {
		log.Fatalf("build live app: %v", err)
	}

	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", app.Handler()); err != nil {
		log.Fatal(err)
	}
}

func HomePage(ctx ui.Ctx) h.Node {
	ui.UseMetadata(ctx, &ui.Meta{
		Title:       "Scoped Styles Demo",
		Description: "Demonstrating component-scoped CSS with UseStyles",
	})

	return h.Html(
		h.Head(
			h.Meta(h.Attr("charset", "utf-8")),
			h.TitleEl(h.Text("Scoped Styles Demo")),
		),
		h.Body(
			h.Div(
				h.Style("font-family", "system-ui, sans-serif"),
				h.Style("max-width", "800px"),
				h.Style("margin", "2rem auto"),
				h.Style("padding", "2rem"),

				h.H1(h.Text("Component Scoped Styles Demo")),
				h.P(h.Text("Both components below use the same class names (.card, .title, .content) but have different styles. The styles are automatically scoped to prevent collisions.")),

				h.Div(h.Style("margin-top", "2rem")),

				ui.Render(ctx, RedCard, struct{}{}),

				h.Div(h.Style("margin-top", "2rem")),

				ui.Render(ctx, BlueCard, struct{}{}),
			),
		),
	)
}

func RedCard(ctx ui.Ctx, _ struct{}) h.Node {
	style := ui.UseStyles(ctx, `
		.card {
			padding: 2rem;
			border: 2px solid #dc2626;
			border-radius: 8px;
			background: #fee2e2;
		}
		.title {
			color: #991b1b;
			font-size: 1.5rem;
			font-weight: bold;
			margin: 0 0 1rem 0;
		}
		.content {
			color: #7f1d1d;
			line-height: 1.6;
		}
		.card:hover {
			background: #fecaca;
			box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
		}
	`)

	return h.Div(
		style.StyleTag(),
		h.Div(h.Class(style.Class("card")),
			h.H2(h.Class(style.Class("title")), h.Text("Red Card Component")),
			h.P(h.Class(style.Class("content")),
				h.Text("This card uses .card, .title, and .content classes with red styling. "),
				h.Text("Hover over me to see the hover effect!"),
			),
			h.P(h.Class(style.Class("content")),
				h.Text(fmt.Sprintf("Scoped classes: .card → .%s", style.Class("card"))),
			),
		),
	)
}

func BlueCard(ctx ui.Ctx, _ struct{}) h.Node {
	style := ui.UseStyles(ctx, `
		.card {
			padding: 2rem;
			border: 2px solid #2563eb;
			border-radius: 8px;
			background: #dbeafe;
		}
		.title {
			color: #1e40af;
			font-size: 1.5rem;
			font-weight: bold;
			margin: 0 0 1rem 0;
		}
		.content {
			color: #1e3a8a;
			line-height: 1.6;
		}
		.card:hover {
			background: #bfdbfe;
			box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
		}
	`)

	return h.Div(
		style.StyleTag(),
		h.Div(h.Class(style.Class("card")),
			h.H2(h.Class(style.Class("title")), h.Text("Blue Card Component")),
			h.P(h.Class(style.Class("content")),
				h.Text("This card also uses .card, .title, and .content classes, but with blue styling. "),
				h.Text("The styles don't collide because they're scoped!"),
			),
			h.P(h.Class(style.Class("content")),
				h.Text(fmt.Sprintf("Scoped classes: .card → .%s", style.Class("card"))),
			),
		),
	)
}
