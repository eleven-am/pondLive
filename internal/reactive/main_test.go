package reactive

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func setup() (*Graph, *Owner) {
	g := NewGraph()
	return g, g.NewRoot()
}

func flush(t testing.TB, g *Graph) {
	t.Helper()
	if err := g.RunQueue(QueueRender); err != nil {
		t.Fatal(err)
	}
	if err := g.RunQueue(QueueWatch); err != nil {
		t.Fatal(err)
	}
}

func mustPanic(t *testing.T, want Violation, fn func()) {
	t.Helper()
	defer func() {
		t.Helper()
		v := recover()
		de, ok := v.(*DevError)
		if !ok || de.Violation != want {
			t.Fatalf("expected violation %d, got %v", want, v)
		}
	}()
	fn()
}
