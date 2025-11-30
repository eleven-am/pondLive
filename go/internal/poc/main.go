package main

import (
	"fmt"
	"log"

	"github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server"
	"github.com/eleven-am/pondlive/go/internal/session"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type CounterProps struct {
	InitialValue int
	Label        string
}

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
	app, err := server.New(server.Config{
		Component: App,
		SessionConfig: &session.Config{
			DevMode: true,
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(app.Server(":8080").ListenAndServe())
}
