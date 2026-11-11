package upload

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestIssueAndReserveConsumesRemaining(t *testing.T) {
	r := NewRegistry()
	token := r.Issue("sess", GrantSpec{Files: 2})

	for i := 0; i < 2; i++ {
		id, _, err := r.Reserve(token, 0, nil)
		if err != nil || id == "" {
			t.Fatalf("reserve %d: id=%q err=%v", i, id, err)
		}
	}
	if _, _, err := r.Reserve(token, 0, nil); !errors.Is(err, ErrGrantExhausted) {
		t.Fatalf("expected exhaustion, got %v", err)
	}
}

func TestTokenIsBoundToItsSession(t *testing.T) {
	r := NewRegistry()
	token := r.Issue("sess-1", GrantSpec{})
	got, ok := SessionIDFromToken(token)
	if !ok || got != "sess-1" {
		t.Fatalf("expected the issuing session, got %q %v", got, ok)
	}
	for _, bad := range []string{"", "nodot", ".x", "x."} {
		if _, ok := SessionIDFromToken(bad); ok {
			t.Fatalf("%q must not parse", bad)
		}
	}
}

func TestSessionIDWithDotsRoundTrips(t *testing.T) {
	r := NewRegistry()
	token := r.Issue("a.b.c", GrantSpec{})
	if got, _ := SessionIDFromToken(token); got != "a.b.c" {
		t.Fatalf("got %q", got)
	}
}

func TestConcurrentReservesNeverExceedTheGrant(t *testing.T) {
	r := NewRegistry()
	token := r.Issue("sess", GrantSpec{Files: 3})

	var ok atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := r.Reserve(token, 0, nil); err == nil {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 3 {
		t.Fatalf("expected exactly 3 reservations, got %d", ok.Load())
	}
}

func TestFailedCheckDoesNotConsumeTheGrant(t *testing.T) {
	r := NewRegistry()
	token := r.Issue("sess", GrantSpec{Files: 1})
	boom := errors.New("bad type")
	if _, _, err := r.Reserve(token, 0, func(GrantSpec) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("expected the check error, got %v", err)
	}
	if _, _, err := r.Reserve(token, 0, nil); err != nil {
		t.Fatalf("a rejected request burned the grant: %v", err)
	}
}

func TestGrantExpires(t *testing.T) {
	r := NewRegistry()
	now := time.Unix(1000, 0)
	r.now = func() time.Time { return now }
	token := r.Issue("sess", GrantSpec{Files: 2})

	now = now.Add(DefaultGrantTTL - time.Second)
	if _, _, err := r.Reserve(token, 0, nil); err != nil {
		t.Fatalf("grant should still be live: %v", err)
	}
	now = now.Add(DefaultGrantTTL)
	if _, _, err := r.Reserve(token, 0, nil); !errors.Is(err, ErrGrantExpired) {
		t.Fatalf("expected expiry, got %v", err)
	}
	if r.Len() != 0 {
		t.Fatal("expired grant was not removed")
	}
}

func TestExpiredGrantsAreSweptOnIssue(t *testing.T) {
	r := NewRegistry()
	now := time.Unix(1000, 0)
	r.now = func() time.Time { return now }
	r.Issue("a", GrantSpec{})
	r.Issue("b", GrantSpec{})
	now = now.Add(DefaultGrantTTL + time.Second)
	r.Issue("c", GrantSpec{})
	if r.Len() != 1 {
		t.Fatalf("expected only the fresh grant, got %d", r.Len())
	}
}

func TestAcquireRequiresOwnership(t *testing.T) {
	r := NewRegistry()
	a := r.Issue("sess", GrantSpec{Files: 1})
	b := r.Issue("sess", GrantSpec{Files: 1})
	id, _, _ := r.Reserve(a, 0, nil)

	release, err := r.Acquire(a, id)
	if err != nil {
		t.Fatalf("owner rejected: %v", err)
	}
	release()
	if _, err := r.Acquire(b, id); !errors.Is(err, ErrUploadNotGranted) {
		t.Fatalf("another grant must not reach the upload, got %v", err)
	}
	if _, err := r.Acquire("sess.unknown", id); !errors.Is(err, ErrGrantUnknown) {
		t.Fatalf("expected unknown grant, got %v", err)
	}
}

func TestSettleRemovesFinishedGrant(t *testing.T) {
	r := NewRegistry()
	token := r.Issue("sess", GrantSpec{Files: 1})
	id, _, _ := r.Reserve(token, 0, nil)
	r.Settle(token, id)
	if r.Len() != 0 {
		t.Fatal("a fully used and settled grant must be dropped")
	}
	if _, err := r.Acquire(token, id); !errors.Is(err, ErrGrantUnknown) {
		t.Fatalf("expected the settled grant to be gone, got %v", err)
	}
}

func TestRevoke(t *testing.T) {
	r := NewRegistry()
	token := r.Issue("sess", GrantSpec{})
	r.Revoke(token)
	if _, _, err := r.Reserve(token, 0, nil); !errors.Is(err, ErrGrantUnknown) {
		t.Fatalf("expected unknown after revoke, got %v", err)
	}
	r.Revoke("")
}

func TestIssueClampsFiles(t *testing.T) {
	r := NewRegistry()
	token := r.Issue("sess", GrantSpec{Files: 100000})
	n := 0
	for {
		if _, _, err := r.Reserve(token, 0, nil); err != nil {
			break
		}
		n++
	}
	if n != MaxGrantFiles {
		t.Fatalf("expected the grant clamped to %d, got %d", MaxGrantFiles, n)
	}
	zero := r.Issue("sess", GrantSpec{})
	if _, _, err := r.Reserve(zero, 0, nil); err != nil {
		t.Fatalf("a zero-file spec must grant one file: %v", err)
	}
}

func TestNilRegistryIsSafe(t *testing.T) {
	var r *Registry
	if r.Issue("s", GrantSpec{}) != "" || r.Len() != 0 {
		t.Fatal("nil registry must be inert")
	}
	r.Revoke("x")
	r.Settle("x", "y")
	if _, _, err := r.Reserve("x", 0, nil); !errors.Is(err, ErrGrantUnknown) {
		t.Fatalf("got %v", err)
	}
	if _, err := r.Acquire("x", "y"); !errors.Is(err, ErrGrantUnknown) {
		t.Fatalf("got %v", err)
	}
}

type reapLog struct {
	mu  sync.Mutex
	ids []string
}

func (l *reapLog) reap(ids []string) {
	l.mu.Lock()
	l.ids = append(l.ids, ids...)
	l.mu.Unlock()
}

func (l *reapLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.ids...)
}

func newBoundRegistry(limits Limits) (*Registry, *reapLog) {
	r := NewRegistry()
	log := &reapLog{}
	r.Bind(BindOptions{Limits: limits, Reap: log.reap})
	return r, log
}

func TestRevokeReapsIncompleteUploads(t *testing.T) {
	r, log := newBoundRegistry(Limits{})
	token := r.Issue("sess", GrantSpec{Files: 2})
	a, _, _ := r.Reserve(token, 10, nil)
	b, _, _ := r.Reserve(token, 10, nil)
	r.Settle(token, b)
	r.Revoke(token)
	if got := log.snapshot(); len(got) != 1 || got[0] != a {
		t.Fatalf("expected only the incomplete upload %s to be reaped, got %v", a, got)
	}
	if r.Inflight() != 0 {
		t.Fatalf("inflight bytes leaked: %d", r.Inflight())
	}
}

func TestCloseReapsEverythingAndRefusesNewGrants(t *testing.T) {
	r, log := newBoundRegistry(Limits{})
	t1 := r.Issue("sess", GrantSpec{Files: 1})
	t2 := r.Issue("sess", GrantSpec{Files: 1})
	r.Reserve(t1, 1, nil)
	r.Reserve(t2, 1, nil)
	r.Close()
	if len(log.snapshot()) != 2 || r.Len() != 0 {
		t.Fatalf("close must reap all incomplete uploads: %v len=%d", log.snapshot(), r.Len())
	}
	if r.Issue("sess", GrantSpec{}) != "" {
		t.Fatal("a closed registry must not issue grants")
	}
}

func TestExpiryReapsIncompleteUploads(t *testing.T) {
	r, log := newBoundRegistry(Limits{})
	now := time.Unix(1000, 0)
	r.now = func() time.Time { return now }
	token := r.Issue("sess", GrantSpec{Files: 1})
	id, _, _ := r.Reserve(token, 1, nil)
	now = now.Add(DefaultGrantTTL + time.Second)
	r.Issue("sess", GrantSpec{})
	if got := log.snapshot(); len(got) != 1 || got[0] != id {
		t.Fatalf("expired grant's upload must be reaped, got %v", got)
	}
}

func TestGrantCapEvictsTheOldest(t *testing.T) {
	r, log := newBoundRegistry(Limits{MaxGrants: 2})
	first := r.Issue("sess", GrantSpec{Files: 1})
	id, _, _ := r.Reserve(first, 1, nil)
	second := r.Issue("sess", GrantSpec{Files: 1})
	third := r.Issue("sess", GrantSpec{Files: 1})

	if r.Len() != 2 {
		t.Fatalf("expected 2 live grants, got %d", r.Len())
	}
	if _, _, err := r.Reserve(first, 0, nil); !errors.Is(err, ErrGrantUnknown) {
		t.Fatalf("the oldest grant must be evicted, got %v", err)
	}
	for _, tok := range []string{second, third} {
		if _, _, err := r.Reserve(tok, 0, nil); err != nil {
			t.Fatalf("newer grant evicted: %v", err)
		}
	}
	if got := log.snapshot(); len(got) != 1 || got[0] != id {
		t.Fatalf("evicted grant's upload must be reaped, got %v", got)
	}
}

func TestSessionBudgetBoundsInflightBytes(t *testing.T) {
	r, _ := newBoundRegistry(Limits{SessionBudget: 100})
	token := r.Issue("sess", GrantSpec{Files: 5})
	id, _, err := r.Reserve(token, 60, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.Reserve(token, 60, nil); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("expected the budget to refuse, got %v", err)
	}
	if _, _, err := r.Reserve(token, 40, nil); err != nil {
		t.Fatalf("within budget must pass: %v", err)
	}
	r.Settle(token, id)
	if _, _, err := r.Reserve(token, 60, nil); err != nil {
		t.Fatalf("settling must free the budget: %v", err)
	}
}

func TestBindIsOnceOnly(t *testing.T) {
	r := NewRegistry()
	first := &reapLog{}
	second := &reapLog{}
	r.Bind(BindOptions{Limits: Limits{MaxGrants: 1}, Reap: first.reap})
	r.Bind(BindOptions{Limits: Limits{MaxGrants: 50}, Reap: second.reap})
	a := r.Issue("sess", GrantSpec{Files: 1})
	r.Reserve(a, 1, nil)
	r.Issue("sess", GrantSpec{Files: 1})
	if len(first.snapshot()) != 1 || len(second.snapshot()) != 0 {
		t.Fatalf("the first binding must win: %v %v", first.snapshot(), second.snapshot())
	}
}

func TestClampFiles(t *testing.T) {
	for in, want := range map[int]int{-5: 1, 0: 1, 1: 1, 7: 7, MaxGrantFiles: MaxGrantFiles, 1000000000: MaxGrantFiles} {
		if got := ClampFiles(in); got != want {
			t.Errorf("ClampFiles(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestFinishProtectsACompletedUploadFromRevokeCloseEvictionAndExpiry(t *testing.T) {
	for name, drop := range map[string]func(r *Registry, token string, now *time.Time){
		"revoke": func(r *Registry, token string, now *time.Time) { r.Revoke(token) },
		"close":  func(r *Registry, token string, now *time.Time) { r.Close() },
		"evict": func(r *Registry, token string, now *time.Time) {
			r.Issue("sess", GrantSpec{})
			r.Issue("sess", GrantSpec{})
		},
		"expiry": func(r *Registry, token string, now *time.Time) {
			*now = now.Add(DefaultGrantTTL + time.Second)
			r.Expire()
		},
	} {
		t.Run(name, func(t *testing.T) {
			r, log := newBoundRegistry(Limits{MaxGrants: 2})
			now := time.Unix(1000, 0)
			r.now = func() time.Time { return now }
			token := r.Issue("sess", GrantSpec{Files: 1})
			id, _, _ := r.Reserve(token, 10, nil)
			release, err := r.Acquire(token, id)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.Finish(token, id); err != nil {
				t.Fatal(err)
			}
			drop(r, token, &now)
			release()
			if got := log.snapshot(); len(got) != 0 {
				t.Fatalf("a finished upload belongs to the application and must never be reaped, got %v", got)
			}
			if r.Inflight() != 0 {
				t.Fatalf("finished bytes still charged: %d", r.Inflight())
			}
		})
	}
}

func TestFinishAfterRevokeFails(t *testing.T) {
	r, log := newBoundRegistry(Limits{})
	token := r.Issue("sess", GrantSpec{Files: 1})
	id, _, _ := r.Reserve(token, 10, nil)
	r.Revoke(token)
	if _, err := r.Finish(token, id); !errors.Is(err, ErrGrantUnknown) {
		t.Fatalf("expected unknown grant, got %v", err)
	}
	if got := log.snapshot(); len(got) != 1 || got[0] != id {
		t.Fatalf("an upload revoked before it finished is reaped exactly once, got %v", got)
	}
}

func TestFinishIsIdempotentAndSettleReleasesOnce(t *testing.T) {
	r, _ := newBoundRegistry(Limits{})
	token := r.Issue("sess", GrantSpec{Files: 2})
	id, _, _ := r.Reserve(token, 10, nil)
	r.Finish(token, id)
	r.Finish(token, id)
	if r.Inflight() != 0 {
		t.Fatalf("inflight %d", r.Inflight())
	}
	r.Settle(token, id)
	if r.Inflight() != 0 {
		t.Fatalf("settle double-released: %d", r.Inflight())
	}
}

func TestActiveRequestKeepsTheGrantAliveBeyondTheTTL(t *testing.T) {
	r, log := newBoundRegistry(Limits{})
	now := time.Unix(1000, 0)
	r.now = func() time.Time { return now }
	token := r.Issue("sess", GrantSpec{Files: 1})
	id, _, _ := r.Reserve(token, 10, nil)
	release, err := r.Acquire(token, id)
	if err != nil {
		t.Fatal(err)
	}

	now = now.Add(DefaultGrantTTL + 5*time.Minute)
	r.Issue("sess", GrantSpec{})
	r.Expire()
	if _, err := r.Finish(token, id); err != nil {
		t.Fatalf("a grant with an open request expired under it: %v", err)
	}
	if len(log.snapshot()) != 0 {
		t.Fatal("an upload with an open request was reaped")
	}

	release()
	now = now.Add(DefaultGrantTTL - time.Second)
	again, err := r.Acquire(token, id)
	if err != nil {
		t.Fatalf("expiry must restart from the end of the request: %v", err)
	}
	again()
	now = now.Add(2 * DefaultGrantTTL)
	r.Expire()
	if _, err := r.Acquire(token, id); err == nil {
		t.Fatal("an idle grant must still expire")
	}
}

func TestLiveUploadIDsAndExpire(t *testing.T) {
	r, log := newBoundRegistry(Limits{})
	now := time.Unix(1000, 0)
	r.now = func() time.Time { return now }
	token := r.Issue("sess", GrantSpec{Files: 2})
	a, _, _ := r.Reserve(token, 1, nil)
	b, _, _ := r.Reserve(token, 1, nil)
	if ids := r.LiveUploadIDs(); len(ids) != 2 {
		t.Fatalf("expected 2 live uploads, got %v", ids)
	}
	now = now.Add(DefaultGrantTTL + time.Second)
	r.Expire()
	if len(r.LiveUploadIDs()) != 0 || len(log.snapshot()) != 2 {
		t.Fatalf("expiry must drop and reap %s and %s: %v", a, b, log.snapshot())
	}
}

func TestPoolBoundsIncompleteBytesAcrossRegistries(t *testing.T) {
	pool := NewPool(100)
	mk := func() (*Registry, string) {
		r := NewRegistry()
		r.Bind(BindOptions{Pool: pool})
		return r, r.Issue("s", GrantSpec{Files: 5})
	}
	r1, t1 := mk()
	r2, t2 := mk()
	id1, _, err := r1.Reserve(t1, 70, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := r2.Reserve(t2, 70, nil); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("the global pool must refuse, got %v", err)
	}
	if r2.Inflight() != 0 {
		t.Fatal("a refused reservation was charged to the session")
	}
	r1.Settle(t1, id1)
	if pool.Used() != 0 {
		t.Fatalf("pool leaked %d", pool.Used())
	}
	if _, _, err := r2.Reserve(t2, 70, nil); err != nil {
		t.Fatalf("settling must free the pool: %v", err)
	}
	r2.Close()
	if pool.Used() != 0 {
		t.Fatalf("close must return the bytes to the pool, used %d", pool.Used())
	}
}

func TestBindReportsFirstBindingAndOnCloseRunsOnce(t *testing.T) {
	r := NewRegistry()
	closed := 0
	if !r.Bind(BindOptions{OnClose: func() { closed++ }}) {
		t.Fatal("first Bind must report true")
	}
	if r.Bind(BindOptions{}) {
		t.Fatal("second Bind must report false")
	}
	r.Close()
	r.Close()
	if closed != 1 {
		t.Fatalf("OnClose ran %d times", closed)
	}
}

func TestTerminateOnlyAppliesToIncompleteUploads(t *testing.T) {
	r, _ := newBoundRegistry(Limits{})
	token := r.Issue("sess", GrantSpec{Files: 2})
	a, _, _ := r.Reserve(token, 10, nil)
	b, _, _ := r.Reserve(token, 10, nil)
	r.Finish(token, b)

	if err := r.Terminate(token, b); !errors.Is(err, ErrUploadFinished) {
		t.Fatalf("a finishing upload must not be terminable, got %v", err)
	}
	if err := r.Terminate(token, a); err != nil {
		t.Fatalf("an incomplete upload must be terminable: %v", err)
	}
	if r.Inflight() != 0 {
		t.Fatalf("inflight %d", r.Inflight())
	}
	if err := r.Terminate(token, "nope"); !errors.Is(err, ErrUploadNotGranted) {
		t.Fatalf("got %v", err)
	}
	if err := r.Terminate("sess.zzz", a); !errors.Is(err, ErrGrantUnknown) {
		t.Fatalf("got %v", err)
	}
}
