package runtime

import (
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

type Snapshot struct {
	HTML    string
	Crashed bool
	Crash   *Error
}

func (s *Session) DeferEffects() {
	if s == nil || s.starting.Load() {
		return
	}
	s.deferred.Store(true)
}

func (s *Session) OnClosed(fn func()) {
	if s == nil || s.starting.Load() {
		return
	}
	s.onClosed = fn
}

func (s *Session) GoLive() {
	if s == nil || !s.deferred.Swap(false) {
		return
	}
	s.ResyncRefs()
	timers := s.deferredTimers
	s.deferredTimers = nil
	for _, tm := range timers {
		if !tm.dead {
			s.arm(tm, tm.d, tm.fn)
		}
	}
	drains := s.heldDrains
	s.heldDrains = nil
	for _, drain := range drains {
		drain()
	}
	if s.devMode {
		s.warnLeaked(false)
		for _, w := range s.heldWarnings {
			s.publishWarning(w)
		}
		s.heldWarnings = nil
	}
	s.flush()
}

func (s *Session) SettleIfNeeded() error {
	if !s.mounted {
		return nil
	}
	s.drainSetters()
	if !s.hasWork() {
		return nil
	}
	s.flush()
	if s.closed {
		return ErrSessionClosed
	}
	return nil
}

func (s *Session) Mounted() bool { return s != nil && s.mounted }

func (s *Session) mountRoot() (string, error) {
	if s.rootFn == nil {
		return "", errors.New("runtime: session has no root component")
	}
	s.root = s.graph.NewRoot()
	s.root.Info = reactive.ComponentInfo{Name: "root"}
	w := newWriter(s.devMode)
	e := s.newEmitter(w, false)
	var panicked any
	s.holdFeeds = true
	defer func() {
		s.holdFeeds = false
		s.dropFeedCaches()
	}()
	func() {
		defer func() { panicked = recover() }()
		s.rootMounting = true
		e.ownerBody(s.root, s.rootFn, false)
		s.rootMounting = false
		s.runArmed()
	}()
	s.rootMounting = false
	if panicked != nil {
		return "", fmt.Errorf("runtime: mount failed: %v", panicked)
	}
	s.mounted = true
	html := w.String()
	s.docLen = len(html)
	if err := s.selfCheckDocument(html, w); err != nil {
		return "", err
	}
	return html, nil
}

func (s *Session) RenderSnapshot() (Snapshot, error) {
	if s.closed {
		return Snapshot{}, ErrSessionClosed
	}
	var html string
	if !s.mounted {
		var err error
		if html, err = s.mountRoot(); err != nil {
			return Snapshot{}, err
		}
	}
	if s.hasWork() {
		s.flush()
		html = ""
	}
	if s.crash != nil && s.root != nil && s.rootFailed() {
		return Snapshot{Crashed: true, Crash: s.crash}, nil
	}
	if html == "" {
		var err error
		if html, err = s.DocumentHTML(); err != nil {
			return Snapshot{}, err
		}
	}
	return Snapshot{HTML: html, Crashed: s.crash != nil, Crash: s.crash}, nil
}

func (s *Session) rootFailed() bool {
	_, bad := s.failed[s.root]
	return bad
}

func (s *Session) DocumentHTML() (html string, err error) {
	if !s.mounted {
		return "", errors.New("runtime: session has not been rendered")
	}
	defer func() {
		if r := recover(); r != nil {
			err = s.nondeterministic(r)
		}
	}()
	defer s.ensureFeedCaches()()
	s.graph.Serialize(func() {
		w := newWriter(false)
		w.buf = make([]byte, 0, s.docLen+s.docLen/32)
		e := s.newEmitter(w, true)
		s.root.ResetCursor()
		e.ownerSerialize(s.root, s.rootFn, false, false)
		html = w.String()
		s.docLen = len(html)
	})
	return html, nil
}

func (s *Session) nondeterministic(r any) error {
	var det *reactive.DeterminismError
	switch v := r.(type) {
	case *reactive.DeterminismError:
		det = v
	case error:
		det = &reactive.DeterminismError{Detail: v.Error()}
	default:
		det = &reactive.DeterminismError{Detail: fmt.Sprint(v)}
	}
	err := &NondeterministicError{Component: det.Component, Detail: det.Detail}
	s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "reset:nondeterministic",
		Message: err.Error(),
		Meta:    map[string]any{"component": det.Component},
	})
	return err
}

func (s *Session) ResetStep() (string, error) {
	if err := s.resyncFeeds(); err != nil {
		return "", err
	}
	if s.hasWork() {
		s.flush()
	}
	defer s.dropFeedCaches()
	html, err := s.DocumentHTML()
	if err != nil {
		return "", err
	}
	if s.devMode {
		again, err := s.DocumentHTML()
		if err != nil {
			return "", err
		}
		if again != html {
			return "", s.nondeterministic(&reactive.DeterminismError{Component: "root", Detail: "two serialize passes differ at byte " + fmt.Sprint(firstDiff(html, again))})
		}
	}
	return html, nil
}

func (s *Session) StepRunning() (elapsed time.Duration, seq uint64, running bool) {
	if s == nil {
		return 0, 0, false
	}
	seq = s.stepSeq.Load()
	start := s.stepStart.Load()
	if start == 0 {
		return 0, seq, false
	}
	return time.Since(time.Unix(0, start)), seq, true
}

func (s *Session) StepKind() string {
	switch msgKind(s.stepKind.Load()) {
	case msgSetter:
		return "setter"
	case msgClientEvent:
		return "client-event"
	default:
		return "step"
	}
}

func (s *Session) Wedge(reason string) {
	if s == nil || !s.running.Load() {
		return
	}
	closeMsg := message{kind: msgStep, run: func(s *Session) { s.closeStep() }}
	if err := s.mb.requestFinal(closeMsg); err != nil {
		return
	}
	s.mb.seal()
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
	s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "actor:wedged",
		Message: reason,
		Meta: map[string]any{
			"code": string(ErrCodeSession),
		},
	})
}

func (m *mailbox) seal() {
	m.mu.Lock()
	m.closed = true
	m.q = nil
	m.head = 0
	m.mu.Unlock()
}

func (s *Session) closeStep() {
	if s.closed {
		return
	}
	s.closed = true
	s.mb.close()

	if s.root != nil {
		func() {
			defer func() { _ = recover() }()
			s.root.Dispose()
		}()
	}
	s.handlers = nil
	s.refs = nil
	s.scripts = nil
	s.httpHandlers = nil
	s.closeDOM()
	if s.UploadRegistry != nil {
		s.UploadRegistry.Close()
	}
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
	s.reportOrphanedCompletions()
	s.root = nil
	s.edits = nil
	s.deferredTimers = nil
	s.heldDrains = nil
	s.heldWarnings = nil
	if s.onClosed != nil {
		s.onClosed()
	}
}

func withComponent(component string, extra map[string]any) map[string]any {
	meta := maps.Clone(extra)
	if meta == nil {
		meta = make(map[string]any, 1)
	}
	if _, ok := meta["component"]; !ok {
		meta["component"] = component
	}
	return meta
}

func (s *Session) warn(w *Warning) {
	s.warnSeq++
	w.ID = s.warnSeq
	w.Timestamp = time.Now()
	meta := withComponent(w.Component, w.Meta)
	phase := w.Phase
	if phase == "" {
		phase = "warning:" + string(w.Code)
	}
	s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: phase,
		Message: w.Message,
		Stack:   w.Stack,
		Meta:    meta,
	})
	if !s.devMode {
		return
	}
	if s.deferred.Load() {
		s.heldWarnings = append(s.heldWarnings, w)
		return
	}
	s.publishWarning(w)
}

func (s *Session) publishWarning(w *Warning) {
	s.late = append(s.late, func() {
		if b := s.rootBoundary; b != nil {
			b.pushWarning(w)
		}
	})
}

func RenderHTML(n work.Node) string {
	s := NewSession(Config{SessionID: "render-html", Root: func(*Ctx) work.Node { return n }})
	defer s.Close()
	html, err := s.mountRoot()
	if err != nil {
		return ""
	}
	return html
}
