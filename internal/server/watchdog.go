package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/session"
)

type watchdog struct {
	registry *SessionRegistry
	cfg      WatchdogConfig

	mu       sync.Mutex
	timer    *time.Timer
	stopped  bool
	reported map[session.SessionID]uint64
}

func newWatchdog(registry *SessionRegistry, cfg WatchdogConfig) *watchdog {
	return &watchdog{registry: registry, cfg: cfg, reported: make(map[session.SessionID]uint64)}
}

func (w *watchdog) start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped || w.timer != nil {
		return
	}
	w.timer = time.AfterFunc(w.cfg.Interval, w.tick)
}

func (w *watchdog) stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stopped = true
	if w.timer != nil {
		w.timer.Stop()
	}
}

func (w *watchdog) tick() {
	w.sweep()
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return
	}
	w.timer.Reset(w.cfg.Interval)
}

func (w *watchdog) sweep() {
	seen := make(map[session.SessionID]struct{})
	for _, sess := range w.registry.Snapshot() {
		id := sess.ID()
		seen[id] = struct{}{}
		rt := sess.Session()
		elapsed, seq, running := rt.StepRunning()
		if !running {
			continue
		}
		if elapsed >= w.cfg.WedgeStep {
			if w.registry.take(sess, session.ReasonWedged) {
				sess.Wedge(fmt.Sprintf("actor step ran for %s; session removed", elapsed.Round(time.Millisecond)))
			}
			continue
		}
		if elapsed >= w.cfg.SlowStep && w.reported[id] != seq {
			w.reported[id] = seq
			sess.Report(protocol.Diagnostic{
				Phase:   "actor:slow-step",
				Message: fmt.Sprintf("actor step has been running for %s", elapsed.Round(time.Millisecond)),
				Metadata: map[string]any{
					"kind":    rt.StepKind(),
					"elapsed": elapsed.String(),
				},
			})
		}
	}
	for id := range w.reported {
		if _, ok := seen[id]; !ok {
			delete(w.reported, id)
		}
	}
}
