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
	app, err := pkg.NewApp(
		root,
		pkg.WithDevMode(),
	)
	if err != nil {
		log.Fatalf("build live app: %v", err)
	}

	assets, err := fs.Sub(publicFS, "public/assets")
	if err != nil {
		log.Fatalf("load assets: %v", err)
	}

	app.Mux().Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))

	log.Println("auth example listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", app.Handler()); err != nil {
		log.Fatal(err)
	}
}

func root(ctx *pkg.Ctx) pkg.Node {
	pkg.UseMetaTags(ctx, &pkg.Meta{
		Title: "Auth Example",
		Links: []pkg.LinkTag{
			{Rel: "stylesheet", Href: "/assets/tailwind.css"},
		},
	})

	return pkg.Div(
		pkg.Class("bg-slate-900", "text-slate-100", "min-h-screen"),
		pkg.Routes(ctx,
			pkg.Route(ctx, pkg.RouteProps{Path: "/login", Component: loginPage}),
			pkg.Route(ctx, pkg.RouteProps{Path: "/", Component: mainLayout},
				pkg.Route(ctx, pkg.RouteProps{Path: "/", Component: homePage}),
				pkg.Route(ctx, pkg.RouteProps{Path: "/dashboard", Component: dashboardPage}),
			),
		),
	)
}

func mainLayout(ctx *pkg.Ctx, match pkg.Match) pkg.Node {
	session, setSession := pkg.UseCookie(ctx, "session")
	isLoggedIn := session != ""

	logoutRef := pkg.UseButton(ctx)
	logoutRef.OnClick(func(evt pkg.ClickEvent) pkg.Updates {
		setSession("", &pkg.CookieOptions{
			Path:   "/",
			MaxAge: -1,
		})
		pkg.Navigate(ctx, "/")
		return nil
	})

	return pkg.Div(
		pkg.Class("min-h-screen", "flex", "flex-col"),
		pkg.Header(
			pkg.Class("bg-slate-800", "px-6", "py-4", "flex", "justify-between", "items-center"),
			pkg.Link(ctx, pkg.LinkProps{To: "/"},
				pkg.Span(
					pkg.Class("text-xl", "font-bold", "text-indigo-400"),
					pkg.Text("Auth Demo"),
				),
			),
			pkg.Nav(
				pkg.Class("flex", "items-center", "space-x-4"),
				pkg.Ternary(isLoggedIn,
					pkg.Fragment(
						pkg.Span(
							pkg.Class("text-slate-300"),
							pkg.Textf("Hello, %s", session),
						),
						pkg.Link(ctx, pkg.LinkProps{To: "/dashboard"},
							pkg.Span(
								pkg.Class("text-slate-400", "hover:text-slate-200"),
								pkg.Text("Dashboard"),
							),
						),
						pkg.Button(
							pkg.Attr("type", "button"),
							pkg.Attach(logoutRef),
							pkg.Class("bg-red-500", "hover:bg-red-400", "px-3", "py-1", "rounded", "text-sm"),
							pkg.Text("Logout"),
						),
					),
					pkg.Link(ctx, pkg.LinkProps{To: "/login"},
						pkg.Span(
							pkg.Class("bg-indigo-500", "hover:bg-indigo-400", "px-4", "py-2", "rounded"),
							pkg.Text("Login"),
						),
					),
				),
			),
		),
		pkg.Main(
			pkg.Class("flex-1"),
			pkg.Outlet(ctx),
		),
	)
}

func homePage(ctx *pkg.Ctx, match pkg.Match) pkg.Node {
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
}

func loginPage(ctx *pkg.Ctx, match pkg.Match) pkg.Node {
	session, setSession := pkg.UseCookie(ctx, "session")
	if session != "" {
		return pkg.Redirect(ctx, pkg.RedirectProps{To: "/dashboard"})
	}

	username, setUsername := pkg.UseState(ctx, "")
	errorMsg, setError := pkg.UseState(ctx, "")

	usernameRef := pkg.UseInput(ctx)
	usernameRef.OnChange(func(evt pkg.ChangeEvent) pkg.Updates {
		setUsername(evt.Value)
		setError("")
		return nil
	})

	formRef := pkg.UseForm(ctx)
	formRef.OnSubmit(func(evt pkg.FormEvent) pkg.Updates {
		if username == "" {
			setError("Username is required")
			return nil
		}
		setSession(username, &pkg.CookieOptions{
			Path:     "/",
			HttpOnly: true,
		})
		pkg.Navigate(ctx, "/dashboard")
		return nil
	})

	return pkg.Div(
		pkg.Class("flex", "items-center", "justify-center", "min-h-screen"),
		pkg.Div(
			pkg.Class("bg-slate-800", "rounded-2xl", "shadow-xl", "p-8", "w-full", "max-w-sm", "space-y-6"),
			pkg.H2(
				pkg.Class("text-2xl", "font-bold", "text-center"),
				pkg.Text("Login"),
			),
			pkg.Form(
				pkg.Attach(formRef),
				pkg.Class("space-y-4"),
				pkg.Div(
					pkg.Label(
						pkg.Class("block", "text-sm", "font-medium", "mb-2"),
						pkg.Attr("for", "username"),
						pkg.Text("Username"),
					),
					pkg.Input(
						pkg.Attr("id", "username"),
						pkg.Attr("type", "text"),
						pkg.Attr("placeholder", "Enter any username"),
						pkg.Attr("value", username),
						pkg.Attach(usernameRef),
						pkg.Class("w-full", "px-4", "py-2", "bg-slate-700", "rounded-lg", "focus:outline-none", "focus:ring-2", "focus:ring-indigo-500"),
					),
				),
				pkg.If(errorMsg != "",
					pkg.P(
						pkg.Class("text-red-400", "text-sm"),
						pkg.Text(errorMsg),
					),
				),
				pkg.Button(
					pkg.Attr("type", "submit"),
					pkg.Class("w-full", "bg-indigo-500", "hover:bg-indigo-400", "px-4", "py-2", "rounded-lg", "font-medium"),
					pkg.Text("Sign In"),
				),
			),
			pkg.Div(
				pkg.Class("text-center"),
				pkg.Link(ctx, pkg.LinkProps{To: "/"},
					pkg.Span(
						pkg.Class("text-slate-400", "hover:text-slate-300", "text-sm"),
						pkg.Text("Back to Home"),
					),
				),
			),
		),
	)
}

func dashboardPage(ctx *pkg.Ctx, match pkg.Match) pkg.Node {
	session, _ := pkg.UseCookie(ctx, "session")
	if session == "" {
		return pkg.Redirect(ctx, pkg.RedirectProps{To: "/login"})
	}

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
