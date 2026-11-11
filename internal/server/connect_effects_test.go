package server

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"

	rt "github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/work"
)

func effectfulComponent(effects, spawned *atomic.Int64) session.Component {
	return func(ctx *rt.Ctx) work.Node {
		rt.UseEffect(ctx, func() func() {
			effects.Add(1)
			stop := make(chan struct{})
			go func() {
				spawned.Add(1)
				<-stop
			}()
			return func() { close(stop) }
		})
		ctx.After(time.Millisecond, func() { effects.Add(100) })
		return plainComponent(ctx)
	}
}

func TestAGetOnlyRequestRunsZeroEffectsAndLeaksNoGoroutines(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	var effects, spawned atomic.Int64
	la := newLifecycleApp(t, effectfulComponent(&effects, &spawned), nil)

	for i := 0; i < 10; i++ {
		resp, _ := la.get(http.DefaultClient, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d", resp.StatusCode)
		}
	}
	time.Sleep(30 * time.Millisecond)
	if effects.Load() != 0 || spawned.Load() != 0 {
		t.Fatalf("a GET-only request ran %d effect callbacks and started %d goroutines", effects.Load(), spawned.Load())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := la.app.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	la.srv.Close()
	http.DefaultClient.CloseIdleConnections()
}

func TestAfterTheSocketConnectsEachEffectRunsExactlyOnce(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	var effects, spawned atomic.Int64
	la := newLifecycleApp(t, effectfulComponent(&effects, &spawned), nil)
	b := la.boot(nil)
	if effects.Load() != 0 {
		t.Fatalf("effects before connect = %d", effects.Load())
	}

	c := la.dial(nil)
	defer c.CloseNow()
	if joined := la.join(c, b); joined["t"] != "joined" {
		t.Fatalf("join = %v", joined)
	}
	waitUntil(t, "the effect to run", func() bool { return effects.Load() >= 1 })
	time.Sleep(20 * time.Millisecond)
	if got := effects.Load() % 100; got != 1 {
		t.Fatalf("effect callbacks after connect = %d (raw %d), want exactly 1", got, effects.Load())
	}

	c.CloseNow()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := la.app.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	la.srv.Close()
	http.DefaultClient.CloseIdleConnections()
}

func TestASessionThatNeverConnectsExpiresWithoutRunningItsEffectQueue(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	var effects, spawned atomic.Int64
	la := newLifecycleApp(t, effectfulComponent(&effects, &spawned), func(c *Config) {
		c.Transport.TokenTTL = 100 * time.Millisecond
	})
	if resp, _ := la.get(http.DefaultClient, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	la.waitLen(0)
	time.Sleep(20 * time.Millisecond)
	if effects.Load() != 0 || spawned.Load() != 0 {
		t.Fatalf("an expired pending session ran %d effects and started %d goroutines", effects.Load(), spawned.Load())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = la.app.Shutdown(ctx)
	la.srv.Close()
	http.DefaultClient.CloseIdleConnections()
}
