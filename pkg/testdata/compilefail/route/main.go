package main

import "github.com/eleven-am/pondlive/pkg"

func page(ctx *pkg.Ctx) pkg.Node { return pkg.Div() }

func build() { _ = pkg.Route("/", page) }

func main() {}
