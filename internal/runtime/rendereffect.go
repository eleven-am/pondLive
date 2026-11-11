package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/eleven-am/pondlive/internal/effectguard"
	"github.com/eleven-am/pondlive/internal/goroutines"
	"github.com/eleven-am/pondlive/internal/reactive"
)

const (
	SlowRenderEffect = 5 * time.Millisecond

	renderEffectLabel   = "pondlive"
	renderEffectSession = "pondlive.session"
	renderEffectPrefix  = "render-effect:"
)

type renderEffectCell struct {
	s       *Session
	o       *reactive.Owner
	fn      func() func()
	cleanup func()
	eff     *reactive.Effect
}

func UseRenderEffect(ctx *Ctx, fn func() func()) {
	reactive.Cell(ctx.o, func() *renderEffectCell {
		w := &renderEffectCell{s: ctx.s, o: ctx.o, fn: fn}
		ctx.o.OnCleanup(w.runCleanup)
		w.eff = reactive.NewEffect(ctx.o, reactive.KindRender, w.run)
		return w
	})
}

func (w *renderEffectCell) run() {
	w.runCleanup()
	w.s.runRenderEffect(w)
}

func (w *renderEffectCell) runCleanup() {
	c := w.cleanup
	if c == nil {
		return
	}
	w.cleanup = nil
	if err := w.s.recoverUser(ErrCodeEffectCleanup, "render-effect:cleanup", w.o, c); err != nil {
		w.s.deliver(w.o, err)
	}
}

func (s *Session) runRenderEffect(w *renderEffectCell) {
	name := w.o.Info.Name
	exit := effectguard.Enter(&effectguard.Scope{Component: name, Dev: s.devMode, Dropped: s.droppedInRenderEffect})
	defer exit()
	var before int
	var start time.Time
	if s.devMode {
		before = runtime.NumGoroutine()
		start = time.Now()
	}
	err := s.recoverUser(ErrCodeEffect, "render-effect:run", w.o, func() {
		if !s.devMode {
			w.cleanup = w.fn()
			return
		}
		labels := pprof.Labels(renderEffectLabel, renderEffectPrefix+name, renderEffectSession, s.SessionID)
		pprof.Do(context.Background(), labels, func(context.Context) { w.cleanup = w.fn() })
	})
	if err != nil {
		s.deliver(w.o, err)
	}
	if !s.devMode {
		return
	}
	if elapsed := time.Since(start); elapsed > SlowRenderEffect {
		s.warn(&Warning{
			Code:      WarnRenderEffectSlow,
			Component: name,
			Message:   fmt.Sprintf("UseRenderEffect in %s ran for %s (limit %s); render effects block the first paint, keep them to bookkeeping and move slow work to UseEffect", name, elapsed.Round(time.Microsecond), SlowRenderEffect),
		})
	}
	if runtime.NumGoroutine() > before {
		s.warnLeaked(true)
	}
}

func (s *Session) droppedInRenderEffect(op, component string) {
	s.Report(nil, Report{
		Severity:  SeverityWarning,
		Code:      string(WarnRenderEffectForbidden),
		Component: component,
		Message:   fmt.Sprintf("%s was called inside UseRenderEffect of %q and was dropped", op, component),
		Meta:      map[string]any{"call": op},
	})
}

type leakedGoroutine struct {
	component string
	state     string
	current   string
	created   string
}

func (s *Session) warnLeaked(transientOK bool) {
	for _, g := range scanRenderEffectGoroutines(s.SessionID) {
		if transientOK && (strings.HasPrefix(g.state, "running") || strings.HasPrefix(g.state, "runnable")) {
			continue
		}
		key := g.component + "|" + g.created
		if _, seen := s.leakSeen[key]; seen {
			continue
		}
		if s.leakSeen == nil {
			s.leakSeen = make(map[string]struct{})
		}
		s.leakSeen[key] = struct{}{}
		stack := g.created
		if stack == "" {
			stack = g.current
		}
		s.warn(&Warning{
			Code:      WarnRenderEffectLeak,
			Component: g.component,
			Message:   fmt.Sprintf("a goroutine started inside UseRenderEffect of %s is still alive [%s]; render effects must not start goroutines, use UseEffect (runs after the socket connects, with a cleanup)", g.component, g.state),
			Stack:     stack,
		})
	}
}

const scanDeadline = 250 * time.Millisecond

func scanRenderEffectGoroutines(session string) []leakedGoroutine {
	deadline := time.Now().Add(scanDeadline)
	for {
		found, unmatched, ok := scanRenderEffectGoroutinesOnce(session)
		if ok {
			return found
		}
		if !time.Now().Before(deadline) {
			return append(found, unmatched...)
		}
		time.Sleep(time.Millisecond)
	}
}

func scanRenderEffectGoroutinesOnce(session string) (found, unmatched []leakedGoroutine, ok bool) {
	var brief bytes.Buffer
	if err := pprof.Lookup("goroutine").WriteTo(&brief, 1); err != nil {
		return nil, nil, true
	}
	type group struct {
		component string
		frames    []string
		current   string
	}
	var groups []group
	for _, block := range strings.Split(brief.String(), "\n\n") {
		var labels map[string]string
		var frames []string
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "# labels: "):
				_ = json.Unmarshal([]byte(strings.TrimPrefix(line, "# labels: ")), &labels)
			case strings.HasPrefix(line, "#\t"):
				fields := strings.Split(line, "\t")
				frames = append(frames, strings.TrimSpace(fields[len(fields)-1]))
			}
		}
		comp, isEffect := strings.CutPrefix(labels[renderEffectLabel], renderEffectPrefix)
		if !isEffect || labels[renderEffectSession] != session {
			continue
		}
		groups = append(groups, group{component: comp, frames: frames, current: block})
	}
	if len(groups) == 0 {
		return nil, nil, true
	}
	var full bytes.Buffer
	if err := pprof.Lookup("goroutine").WriteTo(&full, 2); err != nil {
		return nil, nil, true
	}
	parsed := goroutines.Parse(full.String())
	ok = true
	for _, grp := range groups {
		matched := false
		for _, g := range parsed {
			if goroutines.SameFrames(grp.frames, g.Frames) {
				found = append(found, leakedGoroutine{component: grp.component, state: g.State, current: g.Body, created: g.Created})
				matched = true
			}
		}
		if !matched {
			ok = false
			unmatched = append(unmatched, leakedGoroutine{component: grp.component, state: "unsettled", current: grp.current})
		}
	}
	return found, unmatched, ok
}
