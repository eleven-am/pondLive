package main

import "github.com/eleven-am/pondlive/pkg"

var page = pkg.Component(func(ctx *pkg.Ctx) pkg.Node { return pkg.Div() })
var row = pkg.PropsComponent(func(ctx *pkg.Ctx, item pkg.ReadSignal[int]) pkg.Node { return pkg.Li() })

func build(items pkg.ReadSignal[[]int]) {
	_, _ = pkg.NewApp(page)
	_ = pkg.Routes(pkg.Route("/", page), pkg.Route("/x/*", page), pkg.Route("*", page))
	_ = pkg.For(pkg.Items(items, func(n int) int { return n }), row)
}

func main() {}
