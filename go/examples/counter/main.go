// Package main hosts a full LiveUI application that renders a TailwindCSS-styled
// counter and wires both the HTTP boot route and PondSocket transport. It can be
// started with `go run .` and will serve SSR HTML plus a websocket endpoint for
// interactive updates.
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	ui "github.com/eleven-am/pondlive/go/pkg/live"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
	liveserver "github.com/eleven-am/pondlive/go/pkg/live/server"
)

//go:embed public/*
var publicFS embed.FS

func main() {
	ctx := context.Background()
	app, err := liveserver.NewApp(
		ctx,
		counter,
		liveserver.WithDevMode(false),
	)
	if err != nil {
		log.Fatalf("build live app: %v", err)
	}

	assets, err := fs.Sub(publicFS, "public/assets")
	if err != nil {
		log.Fatalf("load assets: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(assets))))
	mux.Handle("/", app.Handler())

	log.Println("tailwind counter listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func counter(ctx ui.Ctx) h.Node {
	count, setCount := ui.UseState(ctx, 0)
	ui.UseMetadata(ctx, &ui.Meta{
		Title:       fmt.Sprintf("LiveUI Tailwind Counter: %d", count()),
		Description: "A simple counter example using LiveUI and TailwindCSS.",
		Meta:        nil,
		Links: []h.LinkTag{
			{
				Rel:  "stylesheet",
				Href: "/static/tailwind.css",
			},
		},
		Scripts: nil,
	})

	buttonRef := ui.UseElement[*h.ButtonRef](ctx)
	buttonRef.OnClick(func(evt h.ClickEvent) h.Updates {
		setCount(count() - 1)
		buttonRef.Focus()
		return nil
	})

	increment := func(h.Event) h.Updates {
		setCount(count() + 1)
		return nil
	}

	return h.Div(
		h.Class("bg-slate-900", "text-slate-100", "min-h-screen", "flex", "items-center", "justify-center"),
		h.Div(
			h.Class("bg-slate-800", "rounded-2xl", "shadow-xl", "p-8", "w-full", "max-w-sm", "space-y-6"),
			h.Header(
				h.Class("text-center"),
				h.H1(
					h.Class("text-3xl", "font-semibold"),
					h.Text("LiveUI Tailwind Counter"),
				),
				h.P(
					h.Class("mt-2", "text-slate-300"),
					h.Text("This example combines TailwindCSS styling with LiveUI state."),
				),
			),
			h.Div(
				h.Class("flex", "items-center", "justify-center", "space-x-4"),
				h.Button(
					h.Class("bg-slate-700", "hover:bg-slate-600", "text-lg", "font-medium", "px-4", "py-2", "rounded-xl", "transition"),
					h.Attr("type", "button"),
					h.Attach(buttonRef),
					h.Text("-"),
				),
				h.Div(
					h.Class("text-4xl", "font-bold", "tabular-nums", "w-20", "text-center"),
					h.Textf("%d", count()),
				),
				h.Button(
					h.Class("bg-indigo-500", "hover:bg-indigo-400", "text-lg", "font-medium", "px-4", "py-2", "rounded-xl", "transition"),
					h.Attr("type", "button"),
					h.On("click", increment),
					h.Text("+"),
				),
			),
			h.Footer(
				h.Class("text-xs", "text-slate-400", "text-center"),
				h.Text("Try clicking the buttons while watching websocket traffic."),
			),
		),
	)
}
