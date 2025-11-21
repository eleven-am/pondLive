// Package main hosts a LiveUI countdown timer application that demonstrates
// time-based updates using UseScript for client-side interval management.
// It can be started with `go run .` and will serve SSR HTML plus a websocket
// endpoint for interactive updates.
package main

import (
	"context"
	"embed"
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
		countdown,
		liveserver.WithDevMode(true),
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

	log.Println("countdown timer listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func countdown(ctx ui.Ctx) ui.Node {
	seconds, setSeconds := ui.UseState(ctx, 10)
	isRunning, setIsRunning := ui.UseState(ctx, false)

	ui.UseMetaTags(ctx, &ui.Meta{
		Title:       "LiveUI Countdown Timer",
		Description: "A countdown timer example using LiveUI and UseScript.",
		Links: []ui.LinkTag{
			{
				Rel:  "stylesheet",
				Href: "/static/tailwind.css",
			},
		},
	})

	timerScript := ui.UseScript(ctx, `
		(element, transport) => {
			let intervalId = null;

			transport.on('start', () => {
				if (intervalId) clearInterval(intervalId);
				intervalId = setInterval(() => {
					transport.send({ action: 'tick' });
				}, 1000);
			});

			transport.on('stop', () => {
				if (intervalId) {
					clearInterval(intervalId);
					intervalId = null;
				}
			});

			return () => {
				if (intervalId) clearInterval(intervalId);
			};
		}
	`)

	timerScript.OnMessage(func(data map[string]any) {
		if data["action"] == "tick" && seconds() > 0 {
			setSeconds(seconds() - 1)
			if seconds()-1 == 0 {
				setIsRunning(false)
				timerScript.Send("stop", map[string]any{})
			}
		}
	})

	start := func(h.Event) h.Updates {
		if seconds() > 0 && !isRunning() {
			setIsRunning(true)
			timerScript.Send("start", map[string]any{})
		}
		return nil
	}

	stop := func(h.Event) h.Updates {
		setIsRunning(false)
		timerScript.Send("stop", map[string]any{})
		return nil
	}

	reset := func(h.Event) h.Updates {
		setIsRunning(false)
		setSeconds(10)
		timerScript.Send("stop", map[string]any{})
		return nil
	}

	statusText := "Ready"
	statusColor := "text-slate-300"
	if isRunning() {
		statusText = "Running..."
		statusColor = "text-green-400"
	} else if seconds() == 0 {
		statusText = "Time's up!"
		statusColor = "text-red-400"
	}

	return h.Div(
		h.Class("bg-slate-900", "text-slate-100", "min-h-screen", "flex", "items-center", "justify-center", "p-4"),
		h.Div(
			h.Class("text-center", "space-y-8"),
			h.H1(
				h.Class("text-4xl", "font-bold", "text-slate-300"),
				h.Text("Countdown Timer"),
			),
			h.Div(
				h.Class("bg-slate-800", "rounded-3xl", "p-12", "shadow-2xl"),
				h.Attach(timerScript),
				h.Div(
					h.Class("text-9xl", "font-bold", "tabular-nums", "mb-4", "text-white"),
					h.Textf("%d", seconds()),
				),
				h.P(
					h.Class("text-lg", "font-medium", statusColor),
					h.Text(statusText),
				),
			),
			h.Div(
				h.Class("flex", "gap-4", "justify-center"),
				h.Button(
					h.Class("bg-green-600", "hover:bg-green-500", "disabled:opacity-30", "disabled:cursor-not-allowed", "text-white", "font-semibold", "px-8", "py-3", "rounded-xl", "transition", "min-w-24"),
					h.Attr("type", "button"),
					h.If(isRunning() || seconds() == 0, h.Attr("disabled", "")),
					h.On("click", start),
					h.Text("Start"),
				),
				h.Button(
					h.Class("bg-red-600", "hover:bg-red-500", "disabled:opacity-30", "disabled:cursor-not-allowed", "text-white", "font-semibold", "px-8", "py-3", "rounded-xl", "transition", "min-w-24"),
					h.Attr("type", "button"),
					h.If(!isRunning(), h.Attr("disabled", "")),
					h.On("click", stop),
					h.Text("Stop"),
				),
				h.Button(
					h.Class("bg-slate-600", "hover:bg-slate-500", "text-white", "font-semibold", "px-8", "py-3", "rounded-xl", "transition", "min-w-24"),
					h.Attr("type", "button"),
					h.On("click", reset),
					h.Text("Reset"),
				),
			),
		),
	)
}
