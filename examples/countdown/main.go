package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/eleven-am/pondlive/pkg"
)

//go:embed public/*
var publicFS embed.FS

const timerJS = `
	function(element, transport) {
		let intervalId = null;
		let count = 10;

		transport.on('start', () => {
			if (intervalId) clearInterval(intervalId);
			intervalId = setInterval(() => {
				count -= 1;

				if (count > 0) {
					element.innerText = count;
					transport.send('tick', { remaining: count });
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
`

func newApp() (*pkg.App, error) {
	app, err := pkg.NewApp(countdown, pkg.WithDevMode())
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

	log.Println("countdown timer listening on http://localhost:8080")
	if err := app.Server(":8080").ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

var countdown = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	running := pkg.UseSignal(ctx, false)
	done := pkg.UseSignal(ctx, false)

	pkg.UseHead(ctx, pkg.Head{
		Title:       pkg.Const("LiveUI Countdown Timer"),
		Description: pkg.Const("A countdown timer example using LiveUI and UseScript."),
		Links:       []pkg.LinkTag{{Rel: "stylesheet", Href: "/assets/tailwind.css"}},
	})

	status := pkg.UseComputed(ctx, func() string {
		switch {
		case running.Get():
			return "Running..."
		case done.Get():
			return "Time's up!"
		default:
			return "Ready"
		}
	})
	statusClass := pkg.UseComputed(ctx, func() string {
		switch {
		case running.Get():
			return "text-green-400"
		case done.Get():
			return "text-red-400"
		default:
			return "text-slate-300"
		}
	})
	startDisabled := pkg.UseComputed(ctx, func() bool { return running.Get() || done.Get() })
	stopDisabled := pkg.UseComputed(ctx, func() bool { return !running.Get() })

	timer := pkg.UseScript(ctx, timerJS)

	timer.On("done", func(any) {
		done.Set(true)
		running.Set(false)
	})

	start := func(pkg.ClickEvent) {
		if !running.Peek() {
			running.Set(true)
			timer.Send("start", struct{}{})
		}
	}
	stop := func(pkg.ClickEvent) {
		running.Set(false)
		timer.Send("stop", struct{}{})
	}
	reset := func(pkg.ClickEvent) {
		done.Set(false)
		running.Set(false)
		timer.Send("reset", struct{}{})
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
					pkg.Attach(timer),
					pkg.Text("10"),
				),
				pkg.P(
					pkg.Class("text-lg", "font-medium", statusClass),
					pkg.Text(status),
				),
			),
			pkg.Div(
				pkg.Class("flex", "gap-4", "justify-center"),
				pkg.Button(
					pkg.Class("bg-green-600", "hover:bg-green-500", "disabled:opacity-30", "disabled:cursor-not-allowed", "text-white", "font-semibold", "px-8", "py-3", "rounded-xl", "transition", "min-w-24"),
					pkg.Type("button"),
					pkg.Disabled(startDisabled),
					pkg.OnClick(start),
					pkg.Text("Start"),
				),
				pkg.Button(
					pkg.Class("bg-red-600", "hover:bg-red-500", "disabled:opacity-30", "disabled:cursor-not-allowed", "text-white", "font-semibold", "px-8", "py-3", "rounded-xl", "transition", "min-w-24"),
					pkg.Type("button"),
					pkg.Disabled(stopDisabled),
					pkg.OnClick(stop),
					pkg.Text("Stop"),
				),
				pkg.Button(
					pkg.Class("bg-slate-600", "hover:bg-slate-500", "text-white", "font-semibold", "px-8", "py-3", "rounded-xl", "transition", "min-w-24"),
					pkg.Type("button"),
					pkg.OnClick(reset),
					pkg.Text("Reset"),
				),
			),
		),
	)
})
