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

func newApp() (*pkg.App, error) {
	app, err := pkg.NewApp(root, pkg.WithDevMode())
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

	log.Println("auth example listening on http://localhost:8080")
	if err := app.Server(":8080").ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

var root = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	pkg.UseHead(ctx, pkg.Head{
		Title: pkg.Const("Auth Example"),
		Links: []pkg.LinkTag{{Rel: "stylesheet", Href: "/assets/tailwind.css"}},
	})

	return pkg.Div(
		pkg.Class("bg-slate-900", "text-slate-100", "min-h-screen"),
		pkg.Routes(
			pkg.Route("/login", loginPage),
			pkg.Route("/*", mainLayout),
		),
	)
})

var mainLayout = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	request := pkg.UseRequest(ctx)
	router := pkg.UseRouter(ctx)
	session := request.Cookie("session")
	loggedIn := pkg.UseComputed(ctx, func() bool { return session.Get() != "" })
	greeting := pkg.UseComputed(ctx, func() string { return "Hello, " + session.Get() })

	logout := func(pkg.ClickEvent) {
		request.SetCookie("session", "", &pkg.CookieOptions{Path: "/", MaxAge: -1})
		router.Navigate("/")
	}

	return pkg.Div(
		pkg.Class("min-h-screen", "flex", "flex-col"),
		pkg.Header(
			pkg.Class("bg-slate-800", "px-6", "py-4", "flex", "justify-between", "items-center"),
			pkg.Link(pkg.LinkProps{To: "/", Children: pkg.Span(
				pkg.Class("text-xl", "font-bold", "text-indigo-400"),
				pkg.Text("Auth Demo"),
			)}),
			pkg.Nav(
				pkg.Class("flex", "items-center", "space-x-4"),
				pkg.Show(loggedIn,
					func() pkg.Node {
						return pkg.Fragment(
							pkg.Span(pkg.Class("text-slate-300"), pkg.Text(greeting)),
							pkg.Link(pkg.LinkProps{To: "/dashboard", Children: pkg.Span(
								pkg.Class("text-slate-400", "hover:text-slate-200"),
								pkg.Text("Dashboard"),
							)}),
							pkg.Button(
								pkg.Type("button"),
								pkg.OnClick(logout),
								pkg.Class("bg-red-500", "hover:bg-red-400", "px-3", "py-1", "rounded", "text-sm"),
								pkg.Text("Logout"),
							),
						)
					},
					pkg.Else(func() pkg.Node {
						return pkg.Link(pkg.LinkProps{To: "/login", Children: pkg.Span(
							pkg.Class("bg-indigo-500", "hover:bg-indigo-400", "px-4", "py-2", "rounded"),
							pkg.Text("Login"),
						)})
					}),
				),
			),
		),
		pkg.Main(
			pkg.Class("flex-1"),
			pkg.Routes(
				pkg.Route("/", homePage),
				pkg.Route("/dashboard", dashboardPage),
			),
		),
	)
})

var homePage = pkg.Component(func(*pkg.Ctx) pkg.Node {
	return pkg.Div(
		pkg.Class("flex", "flex-col", "items-center", "justify-center", "min-h-[calc(100vh-80px)]", "space-y-6"),
		pkg.H1(
			pkg.Class("text-4xl", "font-bold"),
			pkg.Text("Welcome to Auth Example"),
		),
		pkg.P(
			pkg.Class("text-slate-400"),
			pkg.Text("A simple login/logout demo using cookies and routing"),
		),
	)
})

var loginPage = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	request := pkg.UseRequest(ctx)
	router := pkg.UseRouter(ctx)
	session := request.Cookie("session")
	loggedIn := pkg.UseComputed(ctx, func() bool { return session.Get() != "" })
	username := pkg.UseSignal(ctx, "")
	errorMsg := pkg.UseSignal(ctx, "")
	hasError := pkg.UseComputed(ctx, func() bool { return errorMsg.Get() != "" })

	onUsername := func(pkg.ChangeEvent) {
		errorMsg.Set("")
	}

	onSubmit := func(pkg.FormEvent) {
		name := username.Peek()
		if name == "" {
			errorMsg.Set("Username is required")
			return
		}
		request.SetCookie("session", name, &pkg.CookieOptions{Path: "/", HttpOnly: true})
		router.Navigate("/dashboard")
	}

	return pkg.Div(
		pkg.Class("flex", "items-center", "justify-center", "min-h-screen"),
		pkg.Show(loggedIn, func() pkg.Node {
			return pkg.Redirect(pkg.RedirectProps{To: "/dashboard"})
		}),
		pkg.Div(
			pkg.Class("bg-slate-800", "rounded-2xl", "shadow-xl", "p-8", "w-full", "max-w-sm", "space-y-6"),
			pkg.H2(
				pkg.Class("text-2xl", "font-bold", "text-center"),
				pkg.Text("Login"),
			),
			pkg.Form(
				pkg.OnSubmit(onSubmit),
				pkg.Class("space-y-4"),
				pkg.Div(
					pkg.Label(
						pkg.Class("block", "text-sm", "font-medium", "mb-2"),
						pkg.Attr("for", "username"),
						pkg.Text("Username"),
					),
					pkg.Input(
						pkg.ID("username"),
						pkg.Type("text"),
						pkg.Placeholder("Enter any username"),
						pkg.Bind(username),
						pkg.OnChange(onUsername),
						pkg.Class("w-full", "px-4", "py-2", "bg-slate-700", "rounded-lg", "focus:outline-none", "focus:ring-2", "focus:ring-indigo-500"),
					),
				),
				pkg.Show(hasError, func() pkg.Node {
					return pkg.P(
						pkg.Class("text-red-400", "text-sm"),
						pkg.Text(errorMsg),
					)
				}),
				pkg.Button(
					pkg.Type("submit"),
					pkg.Class("w-full", "bg-indigo-500", "hover:bg-indigo-400", "px-4", "py-2", "rounded-lg", "font-medium"),
					pkg.Text("Sign In"),
				),
			),
			pkg.Div(
				pkg.Class("text-center"),
				pkg.Link(pkg.LinkProps{To: "/", Children: pkg.Span(
					pkg.Class("text-slate-400", "hover:text-slate-300", "text-sm"),
					pkg.Text("Back to Home"),
				)}),
			),
		),
	)
})

var dashboardPage = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	session := pkg.UseRequest(ctx).Cookie("session")
	loggedOut := pkg.UseComputed(ctx, func() bool { return session.Get() == "" })

	return pkg.Show(loggedOut,
		func() pkg.Node { return pkg.Redirect(pkg.RedirectProps{To: "/login"}) },
		pkg.Else(func() pkg.Node { return dashboardBody(session) }),
	)
})

func dashboardBody(session pkg.ReadSignal[string]) pkg.Node {
	return pkg.Div(
		pkg.Class("flex", "items-center", "justify-center", "min-h-[calc(100vh-80px)]"),
		pkg.Div(
			pkg.Class("bg-slate-800", "rounded-2xl", "shadow-xl", "p-8", "w-full", "max-w-md", "space-y-6"),
			pkg.H2(
				pkg.Class("text-2xl", "font-bold", "text-center"),
				pkg.Text("Dashboard"),
			),
			pkg.Div(
				pkg.Class("bg-slate-700", "rounded-lg", "p-4"),
				pkg.P(
					pkg.Class("text-slate-300"),
					pkg.Text("Welcome, "),
					pkg.Span(
						pkg.Class("text-indigo-400", "font-semibold"),
						pkg.Text(session),
					),
					pkg.Text("!"),
				),
				pkg.P(
					pkg.Class("text-slate-400", "text-sm", "mt-2"),
					pkg.Text("This is a protected page. Only logged-in users can see this."),
				),
			),
		),
	)
}
