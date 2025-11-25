package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// CounterProps defines props for the Counter component
type CounterProps struct {
	InitialValue int
	Label        string
}

// Counter is a component with props
var Counter = html.PropsComponent(func(ctx *runtime.Ctx, props CounterProps, children []work.Node) work.Node {
	count, setCount := runtime.UseState(ctx, props.InitialValue)
	decrement := func(evt work.Event) work.Updates {
		setCount(count - 1)
		return nil
	}

	increment := func(evt work.Event) work.Updates {
		setCount(count + 1)
		return nil
	}

	return html.Div(
		html.H1(html.Text(fmt.Sprintf("%s: %d", props.Label, count))),
		html.Button(
			html.On("click", decrement),
			html.Text("-"),
		),
		html.Button(
			html.On("click", increment),
			html.Text("+"),
		),
	)
})

// App renders the application body content
// The HTML document structure (<html>, <head>, <body>) is added by the boot infrastructure
func App(ctx *runtime.Ctx) work.Node {
	return html.Div(
		html.H1(html.Text("Counter Demo with Props")),
		Counter(ctx, CounterProps{
			InitialValue: 0,
			Label:        "Main Counter",
		}),
		html.Hr(),
		Counter(ctx, CounterProps{
			InitialValue: 100,
			Label:        "Second Counter",
		}),
	)
}

func main() {
	ctx := context.Background()
	app := server.NewApp(ctx)

	registry := server.NewSessionRegistry()

	stopSweeper := registry.StartSweeper(0)
	defer stopSweeper()

	_, err := server.Register(app.PondManager(), "/live", registry)
	if err != nil {
		log.Fatal(err)
	}

	ssrHandler := server.NewSSRHandler(server.SSRConfig{
		Registry:    registry,
		Component:   App,
		ClientAsset: "/static/pondlive.js",
	})

	app.Handle("/{path...}", ssrHandler)

	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", app.Handler()))
}
