package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/eleven-am/pondlive/pkg"
)

//go:embed public/*
var publicFS embed.FS

type Message struct {
	ID        int
	Author    string
	Text      string
	Timestamp time.Time
}

func newApp() (*pkg.App, error) {
	app, err := pkg.NewApp(chat, pkg.WithDevMode())
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

	log.Println("stream-chat listening on http://localhost:8080")
	if err := app.Server(":8080").ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

var chat = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	messages := pkg.UseSignal(ctx, []Message{})
	lastID := 0

	pkg.UseHead(ctx, pkg.Head{
		Title:       pkg.Const("LiveUI Stream Chat"),
		Description: pkg.Const("A chat example using For for efficient list rendering."),
		Links:       []pkg.LinkTag{{Rel: "stylesheet", Href: "/assets/tailwind.css"}},
	})

	empty := pkg.UseComputed(ctx, func() bool { return len(messages.Get()) == 0 })
	draft := pkg.UseSignal(ctx, "")
	blank := pkg.UseComputed(ctx, func() bool { return draft.Get() == "" })

	post := func(author, text string, first bool) {
		lastID++
		msg := Message{ID: lastID, Author: author, Text: text, Timestamp: time.Now()}
		messages.Update(func(list []Message) []Message {
			if first {
				return append([]Message{msg}, list...)
			}
			return append(slices.Clip(list), msg)
		})
	}

	send := func(text string) {
		if text == "" {
			return
		}
		post("You", text, false)
		draft.Set("")
	}

	onKeyDown := func(evt pkg.KeyboardEvent) {
		if evt.Key == "Enter" {
			send(draft.Peek())
		}
	}
	onSend := func(pkg.ClickEvent) { send(draft.Peek()) }
	onClear := func(pkg.ClickEvent) { messages.Set([]Message{}) }
	onPrepend := func(pkg.ClickEvent) { post("System", "This message was prepended!", true) }

	return pkg.Div(
		pkg.Class("bg-slate-900", "text-slate-100", "min-h-screen", "flex", "flex-col"),
		pkg.Div(
			pkg.Class("flex-none", "bg-slate-800", "border-b", "border-slate-700", "p-4"),
			pkg.Div(
				pkg.Class("max-w-2xl", "mx-auto", "flex", "items-center", "justify-between"),
				pkg.H1(pkg.Class("text-2xl", "font-bold"), pkg.Text("Stream Chat")),
				pkg.Div(
					pkg.Class("flex", "gap-2"),
					pkg.Button(
						pkg.Class("bg-blue-600", "hover:bg-blue-500", "text-white", "px-3", "py-1", "rounded", "text-sm"),
						pkg.OnClick(onPrepend),
						pkg.Text("Prepend"),
					),
					pkg.Button(
						pkg.Class("bg-red-600", "hover:bg-red-500", "text-white", "px-3", "py-1", "rounded", "text-sm"),
						pkg.OnClick(onClear),
						pkg.Text("Clear"),
					),
				),
			),
		),
		pkg.Div(
			pkg.Class("flex-1", "overflow-y-auto", "p-4"),
			pkg.Div(
				pkg.Class("max-w-2xl", "mx-auto", "space-y-3"),
				pkg.For(pkg.Items(messages, messageKey), messageRow),
				pkg.Show(empty, func() pkg.Node {
					return pkg.Div(
						pkg.Class("text-center", "text-slate-500", "py-12"),
						pkg.Text("No messages yet. Start typing!"),
					)
				}),
			),
		),
		pkg.Div(
			pkg.Class("flex-none", "bg-slate-800", "border-t", "border-slate-700", "p-4"),
			pkg.Div(
				pkg.Class("max-w-2xl", "mx-auto", "flex", "gap-3"),
				pkg.Input(
					pkg.Class("flex-1", "bg-slate-700", "border", "border-slate-600", "rounded-lg", "px-4", "py-2", "text-white", "placeholder-slate-400", "focus:outline-none", "focus:border-blue-500"),
					pkg.ID("draft"),
					pkg.Type("text"),
					pkg.Placeholder("Type a message..."),
					pkg.Bind(draft),
					pkg.OnKeyDown(onKeyDown),
				),
				pkg.Button(
					pkg.Class("bg-blue-600", "hover:bg-blue-500", "disabled:opacity-50", "text-white", "font-semibold", "px-6", "py-2", "rounded-lg", "transition"),
					pkg.Disabled(blank),
					pkg.OnClick(onSend),
					pkg.Text("Send"),
				),
			),
		),
	)
})

func messageKey(m Message) int { return m.ID }

var messageRow = pkg.PropsComponent(func(ctx *pkg.Ctx, item pkg.ReadSignal[Message]) pkg.Node {
	msg := item.Peek()
	bg := "bg-slate-800"
	if msg.Author == "System" {
		bg = "bg-blue-900/50"
	}
	return pkg.Div(
		pkg.Class(bg, "rounded-lg", "p-4", "border", "border-slate-700"),
		pkg.Div(
			pkg.Class("flex", "items-center", "justify-between", "mb-2"),
			pkg.Span(pkg.Class("font-semibold", "text-blue-400"), pkg.Text(msg.Author)),
			pkg.Span(pkg.Class("text-xs", "text-slate-500"), pkg.Text(msg.Timestamp.Format("15:04:05"))),
		),
		pkg.P(pkg.Class("text-slate-200"), pkg.Text(msg.Text)),
	)
})
