package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/eleven-am/pondlive/go/pkg"
)

type CardProps struct {
	Title string
	Color string
}

var Card = pkg.PropsComponent(func(ctx *pkg.Ctx, props CardProps, children []pkg.Node) pkg.Node {
	return pkg.Div(
		pkg.Class("card"),
		pkg.Style("border", fmt.Sprintf("2px solid %s", props.Color)),
		pkg.Style("padding", "1rem"),
		pkg.Style("margin", "1rem"),
		pkg.Style("border-radius", "8px"),
		pkg.H2(pkg.Text(props.Title)),
		pkg.Div(
			pkg.Class("card-content"),
			pkg.Fragment(children...),
		),
	)
})

var Container = pkg.Component(func(ctx *pkg.Ctx, children []pkg.Node) pkg.Node {
	return pkg.Div(
		pkg.Class("container"),
		pkg.Style("max-width", "800px"),
		pkg.Style("margin", "0 auto"),
		pkg.Style("padding", "2rem"),
		pkg.Fragment(children...),
	)
})

func App(ctx *pkg.Ctx) pkg.Node {
	return pkg.Html(
		pkg.Head(
			pkg.TitleEl(pkg.Text("Children Components Example")),
		),
		pkg.Body(
			pkg.H1(pkg.Text("LiveUI Children Components")),

			Container(ctx,
				pkg.Key("main-container"),
				pkg.P(pkg.Text("This example demonstrates components that accept children:")),

				Card(ctx, CardProps{Title: "First Card", Color: "#3b82f6"},
					pkg.Key("card-1"),
					pkg.P(pkg.Text("This is the first card with some content.")),
					pkg.P(pkg.Text("Cards can have multiple children!")),
				),

				Card(ctx, CardProps{Title: "Second Card", Color: "#10b981"},
					pkg.Key("card-2"),
					pkg.P(pkg.Text("This is the second card.")),
					pkg.Ul(
						pkg.Li(pkg.Text("Item 1")),
						pkg.Li(pkg.Text("Item 2")),
						pkg.Li(pkg.Text("Item 3")),
					),
				),

				Card(ctx, CardProps{Title: "Third Card", Color: "#f59e0b"},
					pkg.P(pkg.Text("This card has no explicit key.")),
					pkg.P(pkg.Text("The framework will generate one automatically.")),
				),
			),
		),
	)
}

func main() {
	server, err := pkg.NewApp(App)
	if err != nil {
		log.Fatalf("build live app: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", server.Handler())

	log.Println("children components example listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
