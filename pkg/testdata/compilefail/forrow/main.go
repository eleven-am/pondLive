package main

import "github.com/eleven-am/pondlive/pkg"

func row(ctx *pkg.Ctx, item pkg.ReadSignal[int]) pkg.Node { return pkg.Li() }

func build(items pkg.ReadSignal[[]int]) {
	_ = pkg.For(pkg.Items(items, func(n int) int { return n }), row)
}

func main() {}
