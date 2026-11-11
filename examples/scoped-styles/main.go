package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eleven-am/pondlive/pkg"
)

func newApp() (*pkg.App, error) {
	return pkg.NewApp(HomePage, pkg.WithDevMode())
}

func main() {
	app, err := newApp()
	if err != nil {
		log.Fatalf("build live app: %v", err)
	}

	srv := app.Server(":8080")

	go func() {
		log.Println("Server starting on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

var HomePage = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	pkg.UseHead(ctx, pkg.Head{
		Title:       pkg.Const("Scoped Styles Demo"),
		Description: pkg.Const("Demonstrating component-scoped CSS with UseStyles"),
	})

	return pkg.Div(
		pkg.Style("font-family", "system-ui, sans-serif"),
		pkg.Style("max-width", "800px"),
		pkg.Style("margin", "2rem auto"),
		pkg.Style("padding", "2rem"),
		pkg.H1(pkg.Text("Component Scoped Styles Demo")),
		pkg.P(pkg.Text("Both components below use the same class names (.card, .title, .content) but have different styles. The styles are automatically scoped to prevent collisions.")),
		pkg.Div(pkg.Style("margin-top", "2rem")),
		RedCard(),
		pkg.Div(pkg.Style("margin-top", "2rem")),
		BlueCard(),
	)
})

var RedCard = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	style := pkg.UseStyles(ctx, `
		.card {
			padding: 2rem;
			border: 2px solid #dc2626;
			border-radius: 8px;
			background: #fee2e2;
		}
		.title {
			color: #991b1b;
			font-size: 1.5rem;
			font-weight: bold;
			margin: 0 0 1rem 0;
		}
		.content {
			color: #7f1d1d;
			line-height: 1.6;
		}
		.card:hover {
			background: #fecaca;
			box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
		}
	`)

	return pkg.Div(
		pkg.Class(style.Class("card")),
		pkg.H2(
			pkg.Class(style.Class("title")),
			pkg.Text("Red Card Component"),
		),
		pkg.P(pkg.Class(style.Class("content")),
			pkg.Text("This card uses .card, .title, and .content classes with red styling. "),
			pkg.Text("Hover over me to see the hover effect!"),
		),
		pkg.P(
			pkg.Class(style.Class("content")),
			pkg.Text(fmt.Sprintf("Scoped classes: .card → .%s", style.Class("card"))),
		),
	)
})

var BlueCard = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	style := pkg.UseStyles(ctx, `
		.card {
			padding: 2rem;
			border: 2px solid #2563eb;
			border-radius: 8px;
			background: #dbeafe;
		}
		.title {
			color: #1e40af;
			font-size: 1.5rem;
			font-weight: bold;
			margin: 0 0 1rem 0;
		}
		.content {
			color: #1e3a8a;
			line-height: 1.6;
		}
		.card:hover {
			background: #bfdbfe;
			box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
		}
	`)

	return pkg.Div(
		pkg.Class(style.Class("card")),
		pkg.H2(pkg.Class(style.Class("title")), pkg.Text("Blue Card Component")),
		pkg.P(pkg.Class(style.Class("content")),
			pkg.Text("This card also uses .card, .title, and .content classes, but with blue styling. "),
			pkg.Text("The styles don't collide because they're scoped!"),
		),
		pkg.P(pkg.Class(style.Class("content")),
			pkg.Text(fmt.Sprintf("Scoped classes: .card → .%s", style.Class("card"))),
		),
	)
})
