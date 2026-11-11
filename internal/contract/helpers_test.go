package contract

import (
	"strconv"
	"testing"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
	"go.uber.org/goleak"
	"golang.org/x/net/html"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, goleak.IgnoreAnyFunction("github.com/tus/tusd/v2/pkg/handler.newDelayedContext.func1"))
}

type ctx = runtime.Ctx
type node = work.Node
type item = work.Item

func el(tag string, items ...item) node { return work.BuildElement(tag, items...) }

func txt(s string) node { return work.NewText(s) }

func bound[T any](v T) node { return work.TextOf(v) }

func attr[T any](name string, v T) item { return work.Attribute(name, v) }

func cls(parts ...any) item { return work.Class(parts...) }

func on(event string, fn func(work.Event)) item { return work.On(event, fn) }

func noop(work.Event) {}

func doc(body ...item) node {
	return el("html", el("head"), el("body", body...))
}

func itoa(n int) string { return strconv.Itoa(n) }

type bag struct {
	vals map[string]any
	sigs map[string]any
}

func newBag(init map[string]any) *bag {
	b := &bag{vals: map[string]any{}, sigs: map[string]any{}}
	for k, v := range init {
		b.vals[k] = v
	}
	return b
}

func (b *bag) clone() *bag { return newBag(b.vals) }

func use[T any](c *ctx, b *bag, key string) *runtime.Signal[T] {
	sig := runtime.UseSignal(c, b.vals[key].(T))
	b.sigs[key] = sig
	return sig
}

func put[T any](b *bag, key string, v T) {
	b.vals[key] = v
	b.sigs[key].(*runtime.Signal[T]).Set(v)
}

type stepFn func(a *runtimetest.App, b *bag)

func bagCase(name string, init map[string]any, view func(c *ctx, b *bag) node, steps ...stepFn) scenarioCase {
	return scenarioCase{name: name, build: func() instance { return bagInstance(init, view, steps) }}
}

func prodBagCase(name string, init map[string]any, view func(c *ctx, b *bag) node, steps ...stepFn) scenarioCase {
	sc := bagCase(name, init, view, steps...)
	sc.prod = true
	return sc
}

func bagInstance(init map[string]any, view func(c *ctx, b *bag) node, steps []stepFn) instance {
	b := newBag(init)
	inst := instance{
		root: func(c *ctx) node { return view(c, b) },
		fresh: func() func(c *ctx) node {
			fb := b.clone()
			return func(c *ctx) node { return view(c, fb) }
		},
	}
	for _, s := range steps {
		s := s
		inst.steps = append(inst.steps, func(a *runtimetest.App) { s(a, b) })
	}
	return inst
}

func set[T any](key string, v T) stepFn {
	return func(a *runtimetest.App, b *bag) { a.Do(func() { put(b, key, v) }) }
}

func change(fn func(b *bag)) stepFn {
	return func(a *runtimetest.App, b *bag) { a.Do(func() { fn(b) }) }
}

func idOfTag(t testing.TB, a *runtimetest.App, tag string, nth int) uint32 {
	t.Helper()
	ids := idsOf(a)
	for _, id := range ids {
		n := a.Model.Node(id)
		if n != nil && n.Type == html.ElementNode && n.Data == tag {
			if nth == 0 {
				return id
			}
			nth--
		}
	}
	t.Fatalf("no element %s #%d in %s", tag, nth, a.Model.HTML())
	return 0
}

type classWhen struct {
	name string
	on   runtime.ReadSignal[bool]
}

func (c classWhen) Get() string {
	if c.on.Get() {
		return c.name
	}
	return ""
}

func (c classWhen) Peek() string {
	if c.on.Peek() {
		return c.name
	}
	return ""
}
