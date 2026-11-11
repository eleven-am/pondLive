package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/eleven-am/pondlive/pkg"
)

//go:embed public/*
var publicFS embed.FS

func newApp() (*pkg.App, error) {
	app, err := pkg.NewApp(counter, pkg.WithDevMode())
	if err != nil {
		return nil, err
	}

	assets, err := fs.Sub(publicFS, "public/assets")
	if err != nil {
		return nil, err
	}

	app.Mux().Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))
	return app, nil
}

func main() {
	app, err := newApp()
	if err != nil {
		log.Fatalf("build live app: %v", err)
	}

	log.Println("tailwind counter listening on http://localhost:8080")
	if err := app.Server(":8080").ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

var counter = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	count := pkg.UseSignal(ctx, 0)
	title := pkg.UseComputed(ctx, func() string {
		return fmt.Sprintf("LiveUI Tailwind Counter: %d", count.Get())
	})

	pkg.UseHead(ctx, pkg.Head{
		Title:       title,
		Description: pkg.Const("A simple counter example using LiveUI and TailwindCSS."),
		Links:       []pkg.LinkTag{{Rel: "stylesheet", Href: "/assets/tailwind.css"}},
	})

	dec := func(pkg.ClickEvent) { count.Update(func(n int) int { return n - 1 }) }
	inc := func(pkg.Event) { count.Update(func(n int) int { return n + 1 }) }

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
					pkg.Text("This example combines TailwindCSS styling with LiveUI signals."),
				),
			),
			pkg.Div(
				pkg.Class("flex", "items-center", "justify-center", "space-x-4"),
				pkg.Button(
					pkg.Class("bg-slate-700", "hover:bg-slate-600", "text-lg", "font-medium", "px-4", "py-2", "rounded-xl", "transition"),
					pkg.Type("button"),
					pkg.OnClick(dec),
					pkg.Text("-"),
				),
				pkg.Div(
					pkg.Class("text-4xl", "font-bold", "tabular-nums", "w-20", "text-center"),
					pkg.Text(count),
				),
				pkg.Button(
					pkg.Class("bg-indigo-500", "hover:bg-indigo-400", "text-lg", "font-medium", "px-4", "py-2", "rounded-xl", "transition"),
					pkg.Type("button"),
					pkg.On("click", inc),
					pkg.Text("+"),
				),
			),
			pkg.Footer(
				pkg.Class("text-xs", "text-slate-400", "text-center"),
				pkg.Text("Try clicking the buttons while watching websocket traffic."),
			),
		),
	)
})
