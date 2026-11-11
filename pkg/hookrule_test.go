package pkg

import (
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
)

func TestHookWithAnOuterCtxInsideAShowArmIsAnError(t *testing.T) {
	root := func(ctx *Ctx) Node {
		on := UseSignal(ctx, true)
		return Div(Show(on, func() Node {
			n := UseSignal(ctx, 0)
			return P(Text(n))
		}))
	}
	s := runtime.NewSession(runtime.Config{SessionID: "hook", Root: root, DevMode: true})
	runtimetest.Start(t, s)
	var snap runtime.Snapshot
	var err error
	_ = s.Ask(time.Second, func() error { snap, err = s.RenderSnapshot(); return nil })
	got := ""
	if err != nil {
		got = err.Error()
	} else if snap.Crashed {
		got = snap.Crash.Message
	}
	if !strings.Contains(got, "outer") {
		t.Fatalf("want a clear outer-ctx hook error, got %q", got)
	}
}

func TestHookWithAnOuterCtxInsideAForRowIsReportedInProd(t *testing.T) {
	var list *Signal[[]int]
	var rows []string
	root := func(ctx *Ctx) Node {
		list = UseSignal(ctx, []int{1})
		return Ul(For(Items(list, func(n int) int { return n }), PropsComponent(func(_ *Ctx, item ReadSignal[int]) Node {
			c := UseSignal(ctx, 0)
			return Li(Text(c))
		})))
	}
	s := runtime.NewSession(runtime.Config{SessionID: "hookprod", Root: root})
	s.Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) { rows = append(rows, d.Phase+" "+d.Message) })
	runtimetest.Start(t, s)
	var snap runtime.Snapshot
	_ = s.Ask(time.Second, func() error { snap, _ = s.RenderSnapshot(); return nil })
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "outer ctx") {
		t.Fatalf("prod must report a diagnostic naming the outer ctx, got %q (crashed=%v)", joined, snap.Crashed)
	}
}
