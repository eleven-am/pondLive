package main

import "github.com/eleven-am/pondlive/pkg"

var page = pkg.Component(func(ctx *pkg.Ctx) pkg.Node { return pkg.Div() })

func build() pkg.Node {
	return pkg.Div(pkg.Route("/", page))
}

func main() {}
