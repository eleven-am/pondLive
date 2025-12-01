// Package main hosts a LiveUI countdown timer application that demonstrates
// time-based updates using UseScript for client-side interval management.
// It can be started with `go run .` and will serve SSR HTML plus a websocket
// endpoint for interactive updates.
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/eleven-am/pondlive/go/pkg"
)

//go:embed public/*
var publicFS embed.FS

func main() {
	app, err := pkg.NewApp(countdown)
	if err != nil {
		log.Fatalf("build live app: %v", err)
	}

	assets, err := fs.Sub(publicFS, "public/assets")
	if err != nil {
		log.Fatalf("load assets: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))
	mux.Handle("/", app.Handler())

	log.Println("countdown timer listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func countdown(ctx *pkg.Ctx) pkg.Node {
	isRunning, setIsRunning := pkg.UseState(ctx, false)
	isDone, setIsDone := pkg.UseState(ctx, false)

	pkg.UseMetaTags(ctx, &pkg.Meta{
		Title:       "LiveUI Countdown Timer",
		Description: "A countdown timer example using LiveUI and UseScript.",
		Links: []pkg.LinkTag{
			{
				Rel:  "stylesheet",
				Href: "/assets/tailwind.css",
			},
		},
	})

	startRef := pkg.UseButton(ctx)
	stopRef := pkg.UseButton(ctx)
	resetRef := pkg.UseButton(ctx)

	timerScript := pkg.UseScript(ctx, `
		function(element, transport) {
			let intervalId = null;
			let count = 10;

			transport.on('start', () => {
				if (intervalId) clearInterval(intervalId);
				intervalId = setInterval(() => {
					count -= 1;

					if (count > 0) {
						element.innerText = count;
					} else {
						element.innerText = 0;
						transport.send('done', 'Timer finished');
						clearInterval(intervalId);
						intervalId = null;
					}

				}, 1000);
			});

			transport.on('stop', () => {
				if (intervalId) {
					clearInterval(intervalId);
					intervalId = null;
				}
			});

			transport.on('reset', () => {
				if (intervalId) {
					clearInterval(intervalId);
					intervalId = null;
				}
				count = 10;
				element.innerText = count;
			});

			return () => {
				if (intervalId) clearInterval(intervalId);
			};
		}
	`)

	timerScript.On("done", func(data interface{}) {
		setIsDone(true)
		setIsRunning(false)
	})

	startRef.OnClick(func(evt pkg.ClickEvent) pkg.Updates {
		if !isRunning {
			setIsRunning(true)
			timerScript.Send("start", struct{}{})
		}
		return nil
	})

	stopRef.OnClick(func(evt pkg.ClickEvent) pkg.Updates {
		setIsRunning(false)
		timerScript.Send("stop", struct{}{})
		return nil
	})

	resetRef.OnClick(func(evt pkg.ClickEvent) pkg.Updates {
		setIsDone(false)
		setIsRunning(false)
		timerScript.Send("reset", struct{}{})
		return nil
	})

	statusText := "Ready"
	statusColor := "text-slate-300"
	if isRunning {
		statusText = "Running..."
		statusColor = "text-green-400"
	} else if !isRunning && isDone {
		statusText = "Time's up!"
		statusColor = "text-red-400"
	}

	return pkg.Div(
		pkg.Class("bg-slate-900", "text-slate-100", "min-h-screen", "flex", "items-center", "justify-center", "p-4"),
		pkg.Div(
			pkg.Class("text-center", "space-y-8"),
			pkg.H1(
				pkg.Class("text-4xl", "font-bold", "text-slate-300"),
				pkg.Text("Countdown Timer"),
			),
			pkg.Div(
				pkg.Class("bg-slate-800", "rounded-3xl", "p-12", "shadow-2xl"),
				pkg.Div(
					pkg.Class("text-9xl", "font-bold", "tabular-nums", "mb-4", "text-white"),
					pkg.Textf("%d", 10),
					pkg.Attach(timerScript),
				),
				pkg.P(
					pkg.Class("text-lg", "font-medium", statusColor),
					pkg.Text(statusText),
				),
			),
			pkg.Div(
				pkg.Class("flex", "gap-4", "justify-center"),
				pkg.Button(
					pkg.Class("bg-green-600", "hover:bg-green-500", "disabled:opacity-30", "disabled:cursor-not-allowed", "text-white", "font-semibold", "px-8", "py-3", "rounded-xl", "transition", "min-w-24"),
					pkg.Attr("type", "button"),
					pkg.If(isRunning || isDone, pkg.Attr("disabled", "")),
					pkg.Attach(startRef),
					pkg.Text("Start"),
				),
				pkg.Button(
					pkg.Class("bg-red-600", "hover:bg-red-500", "disabled:opacity-30", "disabled:cursor-not-allowed", "text-white", "font-semibold", "px-8", "py-3", "rounded-xl", "transition", "min-w-24"),
					pkg.Attr("type", "button"),
					pkg.If(!isRunning, pkg.Attr("disabled", "")),
					pkg.Attach(stopRef),
					pkg.Text("Stop"),
				),
				pkg.Button(
					pkg.Class("bg-slate-600", "hover:bg-slate-500", "text-white", "font-semibold", "px-8", "py-3", "rounded-xl", "transition", "min-w-24"),
					pkg.Attr("type", "button"),
					pkg.Attach(resetRef),
					pkg.Text("Reset"),
				),
			),
		),
	)
}
