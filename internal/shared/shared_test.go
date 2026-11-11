package shared_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"go.uber.org/goleak"

	"github.com/eleven-am/pondlive/internal/shared"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

func TestConcurrentUpdatesAreNotLost(t *testing.T) {
	sh := shared.NewShared(0)
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1000 {
				sh.Update(func(n int) int { return n + 1 })
			}
		}()
	}
	wg.Wait()
	if got := sh.Get(); got != 100_000 {
		t.Fatalf("got %d, want 100000", got)
	}
}

func TestSetAndGetRoundTrip(t *testing.T) {
	sh := shared.NewShared("a")
	sh.Set("b")
	if sh.Get() != "b" {
		t.Fatalf("got %q", sh.Get())
	}
}

func TestMemberSeesNotificationsAndStopsAfterLeave(t *testing.T) {
	sh := shared.NewShared(1)
	var n atomic.Int64
	m := sh.Join(func() { n.Add(1) })
	sh.Set(2)
	sh.Set(3)
	if n.Load() != 2 || m.Latest() != 3 || sh.Subscribers() != 1 {
		t.Fatalf("n=%d latest=%d subs=%d", n.Load(), m.Latest(), sh.Subscribers())
	}
	m.Leave()
	m.Leave()
	sh.Set(4)
	if n.Load() != 2 || sh.Subscribers() != 0 {
		t.Fatalf("notified after leave: n=%d subs=%d", n.Load(), sh.Subscribers())
	}
}

func TestFamilyKeyIsFreedAfterLastLeave(t *testing.T) {
	inits := 0
	fam := shared.NewSharedFamily(func(k string) int { inits++; return len(k) })
	a, b := fam.Key("room"), fam.Key("room")
	m1 := a.Join(func() {})
	m2 := b.Join(func() {})
	if fam.Len() != 1 {
		t.Fatalf("len = %d, want 1 entry shared by both handles", fam.Len())
	}
	a.Set(10)
	if b.Get() != 10 {
		t.Fatalf("handles for one key disagree: %d", b.Get())
	}
	m1.Leave()
	if fam.Len() != 1 {
		t.Fatal("entry freed while a subscriber remains")
	}
	m2.Leave()
	if fam.Len() != 0 {
		t.Fatalf("len = %d after the last leave, want 0", fam.Len())
	}
	if got := a.Get(); got != 4 {
		t.Fatalf("freed key reads %d, want its init value 4", got)
	}
}

func TestFamilyWriteToUnsubscribedKeyIsDiscarded(t *testing.T) {
	fam := shared.NewSharedFamily(func(string) int { return 7 })
	k := fam.Key("x")
	k.Set(99)
	ran := false
	k.Update(func(n int) int { ran = true; return n + 1 })
	if ran || k.Get() != 7 || fam.Len() != 0 {
		t.Fatalf("ran=%v get=%d len=%d", ran, k.Get(), fam.Len())
	}
	m := k.Join(func() {})
	k.Set(5)
	if m.Latest() != 5 {
		t.Fatalf("latest = %d", m.Latest())
	}
	m.Leave()
	if k.Get() != 7 {
		t.Fatalf("a fresh join must start from init, got %d", k.Get())
	}
}

func TestFamilyKeysAreIndependent(t *testing.T) {
	fam := shared.NewSharedFamily(func(k int) int { return k })
	ma := fam.Key(1).Join(func() {})
	mb := fam.Key(2).Join(func() {})
	fam.Key(1).Set(100)
	if ma.Latest() != 100 || mb.Latest() != 2 || fam.Len() != 2 {
		t.Fatalf("a=%d b=%d len=%d", ma.Latest(), mb.Latest(), fam.Len())
	}
	ma.Leave()
	mb.Leave()
	if fam.Len() != 0 {
		t.Fatalf("len = %d", fam.Len())
	}
}

func TestFamilyConcurrentJoinLeaveLeavesNothing(t *testing.T) {
	fam := shared.NewSharedFamily(func(int) int { return 0 })
	var wg sync.WaitGroup
	for g := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 500 {
				k := fam.Key((g + i) % 5)
				m := k.Join(func() {})
				k.Update(func(n int) int { return n + 1 })
				m.Leave()
			}
		}()
	}
	wg.Wait()
	if fam.Len() != 0 {
		t.Fatalf("len = %d", fam.Len())
	}
}

type countingBackend struct {
	shared.Backend
	publishes atomic.Int64
}

func (c *countingBackend) Publish(next func(shared.Envelope) any) shared.Envelope {
	c.publishes.Add(1)
	return c.Backend.Publish(next)
}

func TestBackendIsPluggable(t *testing.T) {
	var cb *countingBackend
	sh := shared.NewShared(1, shared.WithBackend(func(initial any) shared.Backend {
		cb = &countingBackend{Backend: shared.NewInProcess(initial)}
		return cb
	}))
	sh.Set(2)
	sh.Update(func(n int) int { return n + 1 })
	if cb.publishes.Load() != 2 || sh.Get() != 3 {
		t.Fatalf("publishes=%d get=%d", cb.publishes.Load(), sh.Get())
	}

	var sent atomic.Int64
	b := shared.NewBroadcast[int](shared.WithBroadcastBackend(func() shared.Backend {
		return &countingBackend{Backend: shared.NewInProcessStream()}
	}))
	leave := b.Join(func(int) { sent.Add(1) })
	b.Send(1)
	leave()
	b.Send(2)
	if sent.Load() != 1 {
		t.Fatalf("delivered %d", sent.Load())
	}
}

func TestBroadcastDeliversToAllInOrderAndStopsAfterLeave(t *testing.T) {
	b := shared.NewBroadcast[int]()
	var mu sync.Mutex
	got := [3][]int{}
	var leaves []func()
	for i := range 3 {
		leaves = append(leaves, b.Join(func(v int) {
			mu.Lock()
			got[i] = append(got[i], v)
			mu.Unlock()
		}))
	}
	for v := range 5 {
		b.Send(v)
	}
	leaves[1]()
	b.Send(5)
	if b.Subscribers() != 2 {
		t.Fatalf("subs = %d", b.Subscribers())
	}
	for _, l := range leaves {
		l()
	}
	if b.Subscribers() != 0 {
		t.Fatalf("subs = %d", b.Subscribers())
	}
	b.Send(6)
	want := []int{0, 1, 2, 3, 4, 5}
	for _, i := range []int{0, 2} {
		if len(got[i]) != 6 {
			t.Fatalf("sub %d got %v", i, got[i])
		}
		for j, v := range got[i] {
			if v != want[j] {
				t.Fatalf("sub %d got %v", i, got[i])
			}
		}
	}
	if len(got[1]) != 5 {
		t.Fatalf("left subscriber kept receiving: %v", got[1])
	}
}

func TestBroadcastConcurrentSendersShareOneOrder(t *testing.T) {
	b := shared.NewBroadcast[int]()
	var mu sync.Mutex
	var a, c []int
	l1 := b.Join(func(v int) { mu.Lock(); a = append(a, v); mu.Unlock() })
	l2 := b.Join(func(v int) { mu.Lock(); c = append(c, v); mu.Unlock() })
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 500 {
				b.Send(g*1000 + i)
			}
		}()
	}
	wg.Wait()
	l1()
	l2()
	if len(a) != 4000 || len(c) != 4000 {
		t.Fatalf("a=%d c=%d", len(a), len(c))
	}
	for i := range a {
		if a[i] != c[i] {
			t.Fatalf("subscribers saw different orders at %d", i)
		}
	}
}
