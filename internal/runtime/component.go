package runtime

import (
	"reflect"
	"runtime"

	"github.com/eleven-am/pondlive/internal/work"
)

type Node = work.Node

type componentFn = func(ctx *Ctx) work.Node

func Component(fn func(ctx *Ctx) work.Node) func() work.Node {
	ident, name := funcIdentity(fn)
	return func() work.Node {
		return &work.ComponentNode{Fn: componentFn(fn), Ident: ident, Name: name}
	}
}

func PropsComponent[P any](fn func(ctx *Ctx, props P) work.Node) func(P) work.Node {
	ident, name := funcIdentity(fn)
	return func(props P) work.Node {
		return &work.ComponentNode{
			Fn:    componentFn(func(ctx *Ctx) work.Node { return fn(ctx, props) }),
			Ident: ident,
			Name:  name,
			Props: props,
		}
	}
}

func funcIdentity(fn any) (uintptr, string) {
	pc := reflect.ValueOf(fn).Pointer()
	if f := runtime.FuncForPC(pc); f != nil {
		return pc, f.Name()
	}
	return pc, ""
}

func inlineComponent(ctx *Ctx, n work.Node) work.Node {
	if cn, ok := n.(*work.ComponentNode); ok {
		if fn, ok := cn.Fn.(componentFn); ok {
			return fn(ctx)
		}
	}
	return n
}
