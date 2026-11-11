package runtime

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/work"
)

func TestQueuedEffectsDoNotWakeTheFlushLoopBeforeConnect(t *testing.T) {
	var ran atomic.Int32
	s := NewSession(Config{SessionID: "d1", Root: func(ctx *Ctx) work.Node {
		UseEffect(ctx, func() func() { ran.Add(1); return nil })
		return work.BuildElement("div")
	}})
	s.DeferEffects()
	s.Start()
	t.Cleanup(func() { s.Close() })
	if err := s.Ask(5*time.Second, func() error { _, err := s.RenderSnapshot(); return err }); err != nil {
		t.Fatal(err)
	}

	var pending bool
	onActor(t, s, func() { pending = s.hasWork() })
	if pending {
		t.Fatal("the flush loop sees work while only deferred effects are queued")
	}
	before := s.FlushCount()
	for i := 0; i < 5; i++ {
		onActor(t, s, func() {})
	}
	if got := s.FlushCount(); got != before || ran.Load() != 0 {
		t.Fatalf("flushes %d -> %d, effects run = %d before connect", before, got, ran.Load())
	}

	s.GoLive()
	onActor(t, s, func() {})
	if ran.Load() != 1 {
		t.Fatalf("effects run after connect = %d", ran.Load())
	}
}
