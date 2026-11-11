package reactive

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestGoidMatchesStack(t *testing.T) {
	if err := VerifyGoid(); err != nil {
		t.Fatal(err)
	}
}

type actorRig struct {
	g     *Graph
	box   chan func()
	done  chan struct{}
	ready chan struct{}
}

func startActor(setup func(g *Graph)) *actorRig {
	r := &actorRig{g: NewGraph(), box: make(chan func(), 1024), done: make(chan struct{}), ready: make(chan struct{})}
	go func() {
		defer close(r.done)
		r.g.BindActor()
		r.g.SetPoster(func(fn func()) { r.box <- fn })
		setup(r.g)
		close(r.ready)
		for fn := range r.box {
			fn()
		}
	}()
	<-r.ready
	return r
}

func (r *actorRig) stop() {
	close(r.box)
	<-r.done
}

func TestOffActorWritesArePostedAndNeverLost(t *testing.T) {
	var sig *Signal[int]
	var runs atomic.Int64
	var last atomic.Int64
	r := startActor(func(g *Graph) {
		sig = NewSignal(g, 0)
		root := g.NewRoot()
		NewEffect(root, KindRender, func() { last.Store(int64(sig.Get())); runs.Add(1) })
	})
	const writers, per = 8, 200
	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < per; i++ {
				sig.Update(func(v int) int { return v + 1 })
			}
		}()
	}
	wg.Wait()
	finished := make(chan struct{})
	r.box <- func() {
		if err := r.g.RunQueue(QueueRender); err != nil {
			t.Error(err)
		}
		close(finished)
	}
	<-finished
	r.stop()
	if sig.Peek() != writers*per || last.Load() != writers*per {
		t.Fatalf("lost relative updates: value=%d observed=%d", sig.Peek(), last.Load())
	}
}

func TestOffActorReadIsViolation(t *testing.T) {
	var sig *Signal[int]
	r := startActor(func(g *Graph) { sig = NewSignal(g, 1) })
	defer r.stop()
	mustPanic(t, OffActorRead, func() { sig.Get() })
	if sig.Peek() != 1 {
		t.Fatal()
	}
}

func TestOffActorWriteWithoutPosterIsViolation(t *testing.T) {
	g := NewGraph()
	s := NewSignal(g, 1)
	ready, stop := make(chan struct{}), make(chan struct{})
	go func() {
		g.BindActor()
		close(ready)
		<-stop
	}()
	<-ready
	defer close(stop)
	mustPanic(t, OffActorWrite, func() { s.Set(2) })
}
