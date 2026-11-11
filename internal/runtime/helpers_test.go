package runtime_test

import (
	"testing"

	"go.uber.org/goleak"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type (
	Ctx  = runtime.Ctx
	Node = work.Node
	Item = work.Item
)

func el(tag string, items ...Item) Node { return work.BuildElement(tag, items...) }

func txt(s string) Node { return work.NewText(s) }

func bound[T any](v T) Node { return work.TextOf(v) }

func cls(parts ...any) Item { return work.Class(parts...) }

func attr[T any](name string, v T) Item { return work.Attribute(name, v) }

func on(event string, fn func(work.Event)) Item { return work.On(event, fn) }

func frag(items ...Item) Node { return work.NewFragment(items...) }

func noopHandler(work.Event) {}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, goleak.IgnoreAnyFunction("github.com/tus/tusd/v2/pkg/handler.newDelayedContext.func1"))
}

func requireEqual(t testing.TB, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
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
