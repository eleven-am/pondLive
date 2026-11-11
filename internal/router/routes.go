package router

import (
	"maps"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type RouteDef struct {
	pattern   string
	component func() work.Node
}

func Route(path string, component func() work.Node) RouteDef {
	return RouteDef{pattern: path, component: component}
}

type routesProps struct{ defs []RouteDef }

func Routes(defs ...RouteDef) work.Node { return routesNode(routesProps{defs: defs}) }

type selection struct {
	idx   int
	state *MatchState
}

func selectionEqual(a, b *selection) bool {
	return a.idx == b.idx && matchStateEqual(a.state, b.state)
}

func mergeParams(parent, own map[string]string) map[string]string {
	if len(parent) == 0 {
		return own
	}
	out := make(map[string]string, len(parent)+len(own))
	maps.Copy(out, parent)
	maps.Copy(out, own)
	return out
}

func rootOf(rest string) string {
	if rest == "" {
		return "/"
	}
	return rest
}

var routesNode = runtime.PropsComponent(func(ctx *runtime.Ctx, props routesProps) work.Node {
	location := headers.UseRequest(ctx).URL()
	parent := runtime.UseContext(ctx, matchCtx).Peek()

	patterns := make([]string, len(props.defs))
	for i, d := range props.defs {
		patterns[i] = d.pattern
	}

	sel := runtime.UseComputed(ctx, func() *selection {
		full := location.Get().Path
		m, idx, ok := route.BestMatch(parent.rest.Get(), "", patterns)
		if !ok {
			return &selection{idx: -1, state: &MatchState{Path: full}}
		}
		own := make(map[string]string, len(m.Params))
		for k, v := range m.Params {
			if k != "*" {
				own[k] = v
			}
		}
		return &selection{idx: idx, state: &MatchState{
			Matched: true,
			Pattern: m.Pattern,
			Path:    full,
			Params:  mergeParams(parent.params.Get(), own),
			Rest:    m.Rest,
		}}
	}, runtime.WithEqual(selectionEqual))
	idx := runtime.UseComputed(ctx, func() int { return sel.Get().idx })

	cases := make([]runtime.SwitchArg, len(props.defs))
	for i, d := range props.defs {
		def := d
		cases[i] = runtime.Case(i, func() work.Node {
			return routeMount(mountProps{component: def.component, sel: sel})
		})
	}
	return runtime.Switch(idx, cases...)
})

type mountProps struct {
	component func() work.Node
	sel       runtime.ReadSignal[*selection]
}

var routeMount = runtime.PropsComponent(func(ctx *runtime.Ctx, props mountProps) work.Node {
	sel := props.sel
	state := runtime.UseComputed(ctx, func() *MatchState { return sel.Get().state }, runtime.WithEqual(matchStateEqual))
	params := runtime.UseComputed(ctx, func() map[string]string { return state.Get().Params }, runtime.WithEqual(paramsEqual))
	matched := runtime.UseComputed(ctx, func() bool { return state.Get().Matched })
	rest := runtime.UseComputed(ctx, func() string { return rootOf(state.Get().Rest) })

	runtime.Provide(ctx, matchCtx, runtime.UseSignal(ctx, &matchInfo{state: state, params: params, matched: matched, rest: rest}))

	if props.component == nil {
		return work.NewFragment()
	}
	return props.component()
})
