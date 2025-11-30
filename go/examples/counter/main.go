// Package main hosts a full LiveUI application that renders a TailwindCSS-styled
// counter and wires both the HTTP boot route and PondSocket transport. It can be
// started with `go run .` and will serve SSR HTML plus a websocket endpoint for
// interactive updates.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/eleven-am/pondlive/go/pkg"
)

//go:embed public/*
var publicFS embed.FS

func main() {
	app, err := pkg.NewApp(
		counter,
		pkg.WithDevMode(),
	)
	if err != nil {
		log.Fatalf("build live app: %v", err)
	}

	assets, err := fs.Sub(publicFS, "public/assets")
	if err != nil {
		log.Fatalf("load assets: %v", err)
	}

	app.Mux().Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))

	log.Println("tailwind counter listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", app.Handler()); err != nil {
		log.Fatal(err)
	}
}

func counter(ctx *pkg.Ctx) pkg.Node {
	count, setCount := pkg.UseState(ctx, 0)
	pkg.UseMetaTags(ctx, &pkg.Meta{
		Title:       fmt.Sprintf("LiveUI Tailwind Counter: %d", count),
		Description: "A simple counter example using LiveUI and TailwindCSS.",
		Meta:        nil,
		Links: []pkg.LinkTag{
			{
				Rel:  "stylesheet",
				Href: "/assets/tailwind.css",
			},
		},
		Scripts: nil,
	})

	decRef := pkg.UseButton(ctx)
	decRef.OnClick(func(evt pkg.ClickEvent) pkg.Updates {
		x, err := decRef.GetBoundingClientRect()
		fmt.Printf("Button bounding rect: %+v, err: %v\n", x, err)
		setCount(count - 1)
		return nil
	})

	increment := func(pkg.Event) pkg.Updates {
		setCount(count + 1)
		return nil
	}

	return pkg.Div(
		pkg.Class("bg-slate-900", "text-slate-100", "min-h-screen", "flex", "items-center", "justify-center"),
		pkg.Div(
			pkg.Class("bg-slate-800", "rounded-2xl", "shadow-xl", "p-8", "w-full", "max-w-sm", "space-y-6"),
			pkg.Header(
				pkg.Class("text-center"),
				pkg.H1(
					pkg.Class("text-3xl", "font-semibold"),
					pkg.Text("LiveUI Tailwind Counter"),
				),
				pkg.P(
					pkg.Class("mt-2", "text-slate-300"),
					pkg.Text("This example combines TailwindCSS styling with LiveUI state."),
				),
			),
			pkg.Div(
				pkg.Class("flex", "items-center", "justify-center", "space-x-4"),
				pkg.Button(
					pkg.Class("bg-slate-700", "hover:bg-slate-600", "text-lg", "font-medium", "px-4", "py-2", "rounded-xl", "transition"),
					pkg.Attr("type", "button"),
					pkg.Attach(decRef),
					pkg.Text("-"),
				),
				pkg.Div(
					pkg.Class("text-4xl", "font-bold", "tabular-nums", "w-20", "text-center"),
					pkg.Textf("%d", count),
				),
				pkg.Button(
					pkg.Class("bg-indigo-500", "hover:bg-indigo-400", "text-lg", "font-medium", "px-4", "py-2", "rounded-xl", "transition"),
					pkg.Attr("type", "button"),
					pkg.On("click", increment),
					pkg.Text("+"),
				),
			),
			pkg.Footer(
				pkg.Class("text-xs", "text-slate-400", "text-center"),
				pkg.Text("Try clicking the buttons while watching websocket traffic."),
			),
		),
	)
}
