package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/eleven-am/pondlive/pkg"
)

//go:embed public/*
var publicFS embed.FS

type Message struct {
	Author    string
	Text      string
	Timestamp time.Time
}

func main() {
	app, err := pkg.NewApp(chat, pkg.WithDevMode())
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

	log.Println("stream-chat listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func chat(ctx *pkg.Ctx) pkg.Node {
	inputValue, setInputValue := pkg.UseState(ctx, "")
	messageCount := pkg.UseRef(ctx, 0)

	pkg.UseMetaTags(ctx, &pkg.Meta{
		Title:       "LiveUI Stream Chat",
		Description: "A chat example using UseStream for efficient list rendering.",
		Links: []pkg.LinkTag{
			{Rel: "stylesheet", Href: "/assets/tailwind.css"},
		},
	})

	messagesNode, messages := pkg.UseStream(ctx, func(item pkg.StreamItem[Message]) pkg.Node {
		return renderMessage(item)
	})

	inputRef := pkg.UseInput(ctx)
	sendRef := pkg.UseButton(ctx)
	clearRef := pkg.UseButton(ctx)
	prependRef := pkg.UseButton(ctx)

	inputRef.OnChange(func(evt pkg.ChangeEvent) pkg.Updates {
		fmt.Printf("[OnChange] evt.Value=%q, current inputValue=%q\n", evt.Value, inputValue)
		setInputValue(evt.Value)
		return nil
	})

	inputRef.OnKeyDown(func(evt pkg.KeyboardEvent) pkg.Updates {
		fmt.Printf("[OnKeyDown] key=%q, targetValue=%q\n", evt.Key, evt.TargetValue)
		if evt.Key == "Enter" && evt.TargetValue != "" {
			messageCount.Current++
			messages.Append(pkg.StreamItem[Message]{
				Key: fmt.Sprintf("msg-%d", messageCount.Current),
				Value: Message{
					Author:    "You",
					Text:      evt.TargetValue,
					Timestamp: time.Now(),
				},
			})
			setInputValue("")
		}
		return nil
	})

	sendRef.OnClick(func(evt pkg.ClickEvent) pkg.Updates {
		if inputValue != "" {
			messageCount.Current++
			messages.Append(pkg.StreamItem[Message]{
				Key: fmt.Sprintf("msg-%d", messageCount.Current),
				Value: Message{
					Author:    "You",
					Text:      inputValue,
					Timestamp: time.Now(),
				},
			})
			setInputValue("")
		}
		return nil
	})

	clearRef.OnClick(func(evt pkg.ClickEvent) pkg.Updates {
		messages.Clear()
		return nil
	})

	prependRef.OnClick(func(evt pkg.ClickEvent) pkg.Updates {
		messageCount.Current++
		messages.Prepend(pkg.StreamItem[Message]{
			Key: fmt.Sprintf("msg-%d", messageCount.Current),
			Value: Message{
				Author:    "System",
				Text:      "This message was prepended!",
				Timestamp: time.Now(),
			},
		})
		return nil
	})

	return pkg.Div(
		pkg.Class("bg-slate-900", "text-slate-100", "min-h-screen", "flex", "flex-col"),
		pkg.Div(
			pkg.Class("flex-none", "bg-slate-800", "border-b", "border-slate-700", "p-4"),
			pkg.Div(
				pkg.Class("max-w-2xl", "mx-auto", "flex", "items-center", "justify-between"),
				pkg.H1(
					pkg.Class("text-2xl", "font-bold"),
					pkg.Text("Stream Chat"),
				),
				pkg.Div(
					pkg.Class("flex", "gap-2"),
					pkg.Button(
						pkg.Class("bg-blue-600", "hover:bg-blue-500", "text-white", "px-3", "py-1", "rounded", "text-sm"),
						pkg.Attach(prependRef),
						pkg.Text("Prepend"),
					),
					pkg.Button(
						pkg.Class("bg-red-600", "hover:bg-red-500", "text-white", "px-3", "py-1", "rounded", "text-sm"),
						pkg.Attach(clearRef),
						pkg.Text("Clear"),
					),
				),
			),
		),
		pkg.Div(
			pkg.Class("flex-1", "overflow-y-auto", "p-4"),
			pkg.Div(
				pkg.Class("max-w-2xl", "mx-auto", "space-y-3"),
				messagesNode,
				pkg.If(len(messages.Items()) == 0,
					pkg.Div(
						pkg.Class("text-center", "text-slate-500", "py-12"),
						pkg.Text("No messages yet. Start typing!"),
					),
				),
			),
		),
		pkg.Div(
			pkg.Class("flex-none", "bg-slate-800", "border-t", "border-slate-700", "p-4"),
			pkg.Div(
				pkg.Class("max-w-2xl", "mx-auto", "flex", "gap-3"),
				pkg.Input(
					pkg.Class("flex-1", "bg-slate-700", "border", "border-slate-600", "rounded-lg", "px-4", "py-2", "text-white", "placeholder-slate-400", "focus:outline-none", "focus:border-blue-500"),
					pkg.Attr("type", "text"),
					pkg.Attr("placeholder", "Type a message..."),
					pkg.Attr("value", inputValue),
					pkg.Attach(inputRef),
				),
				pkg.Button(
					pkg.Class("bg-blue-600", "hover:bg-blue-500", "disabled:opacity-50", "text-white", "font-semibold", "px-6", "py-2", "rounded-lg", "transition"),
					pkg.If(inputValue == "", pkg.Attr("disabled", "")),
					pkg.Attach(sendRef),
					pkg.Text("Send"),
				),
			),
		),
	)
}

func renderMessage(item pkg.StreamItem[Message]) pkg.Node {
	msg := item.Value
	isSystem := msg.Author == "System"

	bgColor := "bg-slate-800"
	if isSystem {
		bgColor = "bg-blue-900/50"
	}

	return pkg.Div(
		pkg.Class(bgColor, "rounded-lg", "p-4", "border", "border-slate-700"),
		pkg.Div(
			pkg.Class("flex", "items-center", "justify-between", "mb-2"),
			pkg.Span(
				pkg.Class("font-semibold", "text-blue-400"),
				pkg.Text(msg.Author),
			),
			pkg.Span(
				pkg.Class("text-xs", "text-slate-500"),
				pkg.Text(msg.Timestamp.Format("15:04:05")),
			),
		),
		pkg.P(
			pkg.Class("text-slate-200"),
			pkg.Text(msg.Text),
		),
	)
}
