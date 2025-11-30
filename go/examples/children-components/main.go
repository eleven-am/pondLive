package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	ui "github.com/eleven-am/pondlive/go/pkg/live"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
	liveserver "github.com/eleven-am/pondlive/go/pkg/live/server"
)

type CardProps struct {
	Title string
	Color string
}

var Card = ui.PropsComponent(func(ctx ui.Ctx, props CardProps, children []h.Item) h.Node {
	return h.Div(
		h.Class("card"),
		h.Style("border", fmt.Sprintf("2px solid %s", props.Color)),
		h.Style("padding", "1rem"),
		h.Style("margin", "1rem"),
		h.Style("border-radius", "8px"),
		h.H2(h.Text(props.Title)),
		h.Div(
			h.Class("card-content"),
			h.Fragment(children...),
		),
	)
})

var Container = ui.Component(func(ctx ui.Ctx, children []h.Item) h.Node {
	return h.Div(
		h.Class("container"),
		h.Style("max-width", "800px"),
		h.Style("margin", "0 auto"),
		h.Style("padding", "2rem"),
		h.Fragment(children...),
	)
})

func App(ctx ui.Ctx) h.Node {
	return h.Html(
		h.Head(
			h.Title("Children Components Example"),
		),
		h.Body(
			h.H1(h.Text("LiveUI Children Components")),

			Container(ctx,
				h.Key("main-container"),
				h.P(h.Text("This example demonstrates components that accept children:")),

				Card(ctx, CardProps{Title: "First Card", Color: "#3b82f6"},
					h.Key("card-1"),
					h.P(h.Text("This is the first card with some content.")),
					h.P(h.Text("Cards can have multiple children!")),
				),

				Card(ctx, CardProps{Title: "Second Card", Color: "#10b981"},
					h.Key("card-2"),
					h.P(h.Text("This is the second card.")),
					h.Ul(
						h.Li(h.Text("Item 1")),
						h.Li(h.Text("Item 2")),
						h.Li(h.Text("Item 3")),
					),
				),

				Card(ctx, CardProps{Title: "Third Card", Color: "#f59e0b"},
					h.P(h.Text("This card has no explicit key.")),
					h.P(h.Text("The framework will generate one automatically.")),
				),
			),
		),
	)
}

func main() {
	ctx := context.Background()
	server, err := liveserver.NewApp(
		ctx,
		App,
	)

	if err != nil {
		log.Fatalf("build live app: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", server.Handler())

	log.Println("countdown timer listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
