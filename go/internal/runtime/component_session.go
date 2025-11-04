package runtime

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/eleven-am/pondlive/go/internal/diff"
	handlers "github.com/eleven-am/pondlive/go/internal/handlers"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	render "github.com/eleven-am/pondlive/go/internal/render"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// ComponentSession drives component rendering, diffing, and event handling for a live UI connection.
type ComponentSession struct {
	owner        *LiveSession
	root         *component
	rootCallable componentCallable
	rootProps    any
	registry     handlers.Registry
	sendPatch    func([]diff.Op) error

	prev render.Structured

	dirty        map[*component]struct{}
	dirtyRoot    bool
	pendingFlush bool
	suspend      int
	flushing     bool

	uploads           map[string]*uploadSlot
	uploadByComponent map[*component]map[int]*uploadSlot
	uploadSeq         int

	pendingEffects  []effectTask
	pendingCleanups []cleanupTask
	pendingNav      *protocol.NavDelta
	pendingMetrics  *protocol.FrameMetrics

	pendingPubsub  []pubsubTask
	pubsubProvider PubsubProvider
	pubsubSubs     map[string]pubsubSubscription
	pubsubMu       sync.RWMutex

	reporter       DiagnosticReporter
	renderStack    []*component
	currentPhase   string
	errored        bool
	lastDiagnostic *Diagnostic

	meta   *Meta
	metaMu sync.RWMutex

	mu sync.Mutex
}

type pubsubTask struct {
	run func()
}

type pubsubSubscription struct {
	token    string
	topic    string
	handler  func([]byte, map[string]string)
	provider PubsubProvider
}

// DiagnosticReporter receives structured diagnostics captured during panic recovery.
type DiagnosticReporter interface {
	ReportDiagnostic(Diagnostic)
}

// NewSession constructs a session rooted at the provided component function.
func NewSession[P any](root Component[P], props P) *ComponentSession {
	sess := &ComponentSession{
		dirty: make(map[*component]struct{}),
	}
	sess.root = newComponent(sess, nil, "root", root, props)
	if sess.root != nil {
		sess.rootCallable = sess.root.callable
		sess.rootProps = props
	}
	return sess
}

// Registry exposes the handler registry, creating one if necessary.
func (s *ComponentSession) Registry() handlers.Registry {
	return s.ensureRegistry()
}

// SetRegistry injects a custom registry implementation.
func (s *ComponentSession) SetRegistry(reg handlers.Registry) { s.registry = reg }

// SetPatchSender installs the transport used to deliver diff operations.
func (s *ComponentSession) SetPatchSender(fn func([]diff.Op) error) { s.sendPatch = fn }

func (s *ComponentSession) setOwner(owner *LiveSession) { s.owner = owner }

// SetPubsubProvider wires the session to an external pub/sub provider.
func (s *ComponentSession) SetPubsubProvider(provider PubsubProvider) {
	if s == nil {
		return
	}
	s.pubsubProvider = provider
}

// SetDiagnosticReporter installs a reporter notified when diagnostics are captured.
func (s *ComponentSession) SetDiagnosticReporter(r DiagnosticReporter) { s.reporter = r }

// SetMetadata records document metadata for the most recent render cycle.
func (s *ComponentSession) SetMetadata(meta *Meta) {
	if s == nil {
		return
	}
	s.metaMu.Lock()
	if meta == nil {
		s.meta = nil
		s.metaMu.Unlock()
		return
	}
	s.meta = CloneMeta(meta)
	s.metaMu.Unlock()
}

// Metadata returns a copy of the last metadata provided during rendering.
func (s *ComponentSession) Metadata() *Meta {
	if s == nil {
		return nil
	}
	s.metaMu.RLock()
	defer s.metaMu.RUnlock()
	return CloneMeta(s.meta)
}

// InitialStructured performs an initial render and returns the structured result for SSR boot.
func (s *ComponentSession) InitialStructured() render.Structured {
	if s == nil || s.root == nil {
		return render.Structured{}
	}
	var structured render.Structured
	if err := s.withRecovery("initial", func() error {
		reg := s.ensureRegistry()
		s.SetMetadata(nil)
		node := s.root.render()
		structured = render.ToStructuredWithHandlers(node, reg)
		s.prev = structured
		s.dirtyRoot = false
		s.pendingFlush = false
		s.pendingEffects = nil
		s.pendingCleanups = nil
		s.pendingNav = nil
		s.pendingMetrics = nil
		s.pendingPubsub = nil
		if s.uploads != nil {
			for _, slot := range s.uploads {
				if slot != nil {
					slot.sess = nil
				}
			}
		}
		s.uploads = nil
		s.uploadByComponent = nil
		s.uploadSeq = 0
		return nil
	}); err != nil {
		return render.Structured{}
	}
	return structured
}

// RenderNode re-renders the root component and returns its HTML node tree.
func (s *ComponentSession) RenderNode() h.Node {
	if s == nil || s.root == nil {
		return nil
	}
	s.SetMetadata(nil)
	return s.root.render()
}

// Flush applies pending state updates by rerendering and diffing the component tree.
func (s *ComponentSession) Flush() error {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	if s.errored {
		if s.lastDiagnostic != nil {
			return DiagnosticError{diag: *s.lastDiagnostic}
		}
		return errors.New("runtime: session halted after panic")
	}
	reg := s.ensureRegistry()
	var (
		cleanups []cleanupTask
		effects  []effectTask
		pubsubs  []pubsubTask
	)
	err := s.withRecovery("flush", func() error {
		s.mu.Lock()
		defer s.mu.Unlock()

		if !s.dirtyRoot && !s.pendingFlush {
			return nil
		}
		if s.root == nil {
			return errors.New("runtime: session has no root component")
		}

		start := time.Now()
		s.SetMetadata(nil)
		node := s.root.render()
		next := render.ToStructuredWithHandlers(node, reg)
		opDiff := diff.Diff(s.prev, next)
		meta := s.Metadata()
		var metadataChanged bool
		if owner := s.owner; owner != nil {
			prevMeta := owner.currentMetadata()
			if effect, ok := buildMetadataDiff(prevMeta, meta); ok {
				metadataChanged = true
				owner.enqueueMetadataEffect(effect)
			}
		}

		navUpdate := drainNavUpdate(s)
		var navDelta *protocol.NavDelta
		if !navUpdate.Empty() {
			navDelta = &protocol.NavDelta{
				Push:    navUpdate.Push,
				Replace: navUpdate.Replace,
			}
		}

		metrics := protocol.FrameMetrics{
			RenderMs: float64(time.Since(start)) / float64(time.Millisecond),
			Ops:      len(opDiff),
		}

		shouldSend := len(opDiff) > 0 || navDelta != nil || metadataChanged
		if shouldSend {
			if s.sendPatch == nil {
				return errors.New("runtime: SendPatch is nil")
			}
			s.pendingNav = navDelta
			s.pendingMetrics = &metrics
			if err := s.sendPatch(opDiff); err != nil {
				s.pendingNav = nil
				s.pendingMetrics = nil
				return err
			}
		} else {
			s.pendingNav = nil
			s.pendingMetrics = nil
		}

		cleanups = append(cleanups, s.pendingCleanups...)
		effects = append(effects, s.pendingEffects...)
		pubsubs = append(pubsubs, s.pendingPubsub...)
		s.pendingCleanups = nil
		s.pendingEffects = nil
		s.pendingPubsub = nil
		s.prev = next
		s.dirty = make(map[*component]struct{})
		s.dirtyRoot = false
		s.pendingFlush = false
		return nil
	})
	if err != nil {
		return err
	}
	runCleanups(cleanups)
	s.runPubsubTasks(pubsubs)
	totalEffects, maxEffect, slowEffects := runEffects(effects)
	if metricsPtr := s.pendingMetrics; metricsPtr != nil {
		metricsPtr.EffectsMs = float64(totalEffects) / float64(time.Millisecond)
		metricsPtr.MaxEffectMs = float64(maxEffect) / float64(time.Millisecond)
		metricsPtr.SlowEffects = slowEffects
	}
	return nil
}

func runCleanups(tasks []cleanupTask) {
	for _, task := range tasks {
		task.run()
	}
}

func (s *ComponentSession) runPubsubTasks(tasks []pubsubTask) {
	for _, task := range tasks {
		if task.run == nil {
			continue
		}
		if err := s.withRecovery("pubsub", func() error {
			task.run()
			return nil
		}); err != nil {
			// Session has transitioned into errored mode; abort remaining tasks.
			return
		}
	}
}

func runEffects(tasks []effectTask) (total time.Duration, max time.Duration, slowCount int) {
	for _, task := range tasks {
		start := time.Now()
		task.run()
		duration := time.Since(start)
		total += duration
		if duration > max {
			max = duration
		}
		if observeEffectDuration(task.comp, duration) {
			slowCount++
		}
	}
	return
}

// Reset clears the errored flag and rebuilds the root component so rendering can resume.
func (s *ComponentSession) Reset() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	if !s.errored {
		s.mu.Unlock()
		return false
	}
	var (
		callable = s.rootCallable
		props    any
	)
	if s.root != nil {
		props = s.root.props
	}
	if props == nil {
		props = s.rootProps
	}
	s.rootProps = props
	s.errored = false
	s.lastDiagnostic = nil
	s.pendingFlush = false
	s.dirtyRoot = false
	s.dirty = make(map[*component]struct{})
	s.pendingEffects = nil
	s.pendingCleanups = nil
	s.pendingNav = nil
	s.pendingMetrics = nil
	s.pendingPubsub = nil
	s.pubsubMu.Lock()
	s.pubsubSubs = nil
	s.pubsubMu.Unlock()
	s.mu.Unlock()

	if s.root != nil {
		s.root.unmount()
	}

	var rebuilt *component
	if callable != nil {
		rebuilt = newComponentWithCallable(s, nil, "root", callable, props)
	}

	s.mu.Lock()
	if rebuilt != nil {
		if s.dirty == nil {
			s.dirty = make(map[*component]struct{})
		}
		s.dirty[rebuilt] = struct{}{}
		s.dirtyRoot = true
		s.pendingFlush = true
	}
	s.root = rebuilt
	s.rootCallable = callable
	if props != nil {
		s.rootProps = props
	}
	s.mu.Unlock()
	return rebuilt != nil
}

func (s *ComponentSession) withRecovery(phase string, fn func() error) (err error) {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	prevPhase := s.currentPhase
	s.currentPhase = phase
	defer func() { s.currentPhase = prevPhase }()
	defer func() {
		if rec := recover(); rec != nil {
			err = s.handlePanic(phase, rec)
		}
	}()
	return fn()
}

func (s *ComponentSession) handlePanic(phase string, value any) error {
	diag := Diagnostic{
		Phase:      phase,
		Message:    fmt.Sprint(value),
		Panic:      fmt.Sprintf("%v", value),
		CapturedAt: time.Now(),
		Code:       normalizeDiagnosticCode(phase),
		Stack:      string(debug.Stack()),
	}
	if comp := s.currentComponent(); comp != nil {
		diag.ComponentID = comp.id
		if comp.callable != nil {
			diag.ComponentName = comp.callable.name()
		}
	}
	if carrier, ok := value.(metadataCarrier); ok {
		diag.Metadata = cloneDiagnosticMetadata(carrier.Metadata())
	}
	if diag.Metadata == nil {
		diag.Metadata = map[string]any{}
	}
	diag.Metadata["panicType"] = fmt.Sprintf("%T", value)
	if hooker, ok := value.(hookCarrier); ok {
		diag.Hook = hooker.HookName()
		diag.HookIndex = hooker.HookIndex()
	}
	if suggester, ok := value.(suggestionCarrier); ok {
		diag.Suggestion = suggester.Suggestion()
	}

	s.mu.Lock()
	s.errored = true
	s.lastDiagnostic = &diag
	s.pendingFlush = false
	s.dirtyRoot = false
	s.dirty = make(map[*component]struct{})
	s.pendingEffects = nil
	s.pendingCleanups = nil
	s.pendingNav = nil
	s.pendingMetrics = nil
	s.pendingPubsub = nil
	s.pubsubMu.Lock()
	s.pubsubSubs = nil
	s.pubsubMu.Unlock()
	s.mu.Unlock()

	if s.reporter != nil {
		s.reporter.ReportDiagnostic(diag)
	}
	return DiagnosticError{diag: diag}
}

func (s *ComponentSession) pushComponent(c *component) {
	if s == nil || c == nil {
		return
	}
	s.renderStack = append(s.renderStack, c)
}

func (s *ComponentSession) popComponent() {
	if s == nil || len(s.renderStack) == 0 {
		return
	}
	s.renderStack = s.renderStack[:len(s.renderStack)-1]
}

func (s *ComponentSession) currentComponent() *component {
	if s == nil || len(s.renderStack) == 0 {
		return nil
	}
	return s.renderStack[len(s.renderStack)-1]
}

// Dirty reports whether the session has pending renders.
func (s *ComponentSession) Dirty() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pendingFlush || s.dirtyRoot || len(s.dirty) > 0
}

func (s *ComponentSession) markDirty(c *component) {
	if s == nil || c == nil {
		return
	}
	if s.errored {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dirty == nil {
		s.dirty = make(map[*component]struct{})
	}
	s.dirty[c] = struct{}{}
	s.dirtyRoot = true
	s.pendingFlush = true
}

func (s *ComponentSession) clearDirty(c *component) {
	if s == nil || c == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.dirty, c)
	if len(s.dirty) == 0 {
		s.dirtyRoot = false
		s.pendingFlush = false
	}
}

// HandleEvent dispatches an event to a registered handler without flushing.
func (s *ComponentSession) HandleEvent(id handlers.ID, ev handlers.Event) error {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	if s.errored {
		if s.lastDiagnostic != nil {
			return DiagnosticError{diag: *s.lastDiagnostic}
		}
		return errors.New("runtime: session halted after panic")
	}
	phase := fmt.Sprintf("event:%s", id)
	return s.withRecovery(phase, func() error {
		registry := s.ensureRegistry()
		handler, ok := registry.Get(id)
		if !ok || handler == nil {
			diag := Diagnostic{
				Code:       "handler_not_found",
				Phase:      phase,
				Message:    fmt.Sprintf("runtime: handler %s not found", id),
				Metadata:   map[string]any{"handlerId": string(id)},
				Suggestion: "Ensure the event handler is registered before dispatching events.",
			}
			return DiagnosticError{diag: diag}
		}
		if updates := handler(ev); updates != nil {
			s.markDirty(s.root)
		}
		return nil
	})
}

// DispatchEvent routes an event and flushes if state changed.
func (s *ComponentSession) DispatchEvent(id handlers.ID, ev handlers.Event) error {
	if err := s.HandleEvent(id, ev); err != nil {
		return err
	}
	s.mu.Lock()
	dirty := s.pendingFlush && s.suspend == 0
	s.mu.Unlock()
	if !dirty {
		return nil
	}
	return s.Flush()
}

// effectTask represents an effect setup scheduled after the next flush.
type effectTask struct {
	comp  *component
	index int
	setup func() Cleanup
}

func (t effectTask) run() {
	if t.comp == nil || t.comp.frame == nil {
		if t.setup != nil {
			t.setup()
		}
		return
	}
	if t.setup == nil {
		return
	}
	cleanup := t.setup()
	if t.index >= 0 && t.index < len(t.comp.frame.cells) {
		if cell, ok := t.comp.frame.cells[t.index].(*effectCell); ok {
			cell.cleanup = cleanup
		}
	}
}

// cleanupTask runs stored effect cleanups.
type cleanupTask struct {
	comp  *component
	index int
}

func (t cleanupTask) run() {
	if t.comp == nil || t.comp.frame == nil {
		return
	}
	if t.index < 0 || t.index >= len(t.comp.frame.cells) {
		return
	}
	if cell, ok := t.comp.frame.cells[t.index].(*effectCell); ok {
		if cell.cleanup != nil {
			cell.cleanup()
			cell.cleanup = nil
		}
	}
}

func (s *ComponentSession) enqueueEffect(comp *component, index int, setup func() Cleanup) {
	if s == nil {
		return
	}
	if s.errored {
		return
	}
	s.pendingEffects = append(s.pendingEffects, effectTask{comp: comp, index: index, setup: setup})
}

func (s *ComponentSession) enqueueCleanup(comp *component, index int) {
	if s == nil {
		return
	}
	if s.errored {
		return
	}
	s.pendingCleanups = append(s.pendingCleanups, cleanupTask{comp: comp, index: index})
}

func (s *ComponentSession) enqueuePubsub(fn func()) {
	if s == nil || fn == nil {
		return
	}
	if s.errored {
		return
	}
	s.mu.Lock()
	s.pendingPubsub = append(s.pendingPubsub, pubsubTask{run: fn})
	s.pendingFlush = true
	s.mu.Unlock()
}

func (s *ComponentSession) ensureRegistry() handlers.Registry {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	reg := s.registry
	if reg == nil {
		reg = handlers.NewRegistry()
		s.registry = reg
	}
	s.mu.Unlock()
	return reg
}

func (s *ComponentSession) subscribePubsub(topic string, provider PubsubProvider, handler func([]byte, map[string]string)) (string, error) {
	if s == nil {
		return "", errors.New("runtime: session is nil")
	}
	if handler == nil {
		return "", errors.New("runtime: pubsub handler is nil")
	}
	target := provider
	if target == nil {
		target = s.pubsubProvider
	}
	if target == nil || s.owner == nil {
		return "", ErrPubsubUnavailable
	}
	token, err := target.Subscribe(s.owner, topic, s.handlePubsubDelivery)
	if err != nil {
		return "", err
	}
	s.pubsubMu.Lock()
	if s.pubsubSubs == nil {
		s.pubsubSubs = make(map[string]pubsubSubscription)
	}
	s.pubsubSubs[token] = pubsubSubscription{
		token:    token,
		topic:    topic,
		handler:  handler,
		provider: target,
	}
	s.pubsubMu.Unlock()
	if s.owner != nil && target == s.pubsubProvider {
		s.owner.pubsubSubscribed(topic)
	}
	return token, nil
}

func (s *ComponentSession) unsubscribePubsub(token string) error {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	if token == "" {
		return nil
	}
	s.pubsubMu.Lock()
	sub, ok := s.pubsubSubs[token]
	if ok {
		delete(s.pubsubSubs, token)
	}
	s.pubsubMu.Unlock()
	if !ok {
		return nil
	}
	provider := sub.provider
	if provider == nil {
		provider = s.pubsubProvider
	}
	if provider == nil || s.owner == nil {
		return ErrPubsubUnavailable
	}
	if err := provider.Unsubscribe(s.owner, sub.token); err != nil {
		s.pubsubMu.Lock()
		if s.pubsubSubs == nil {
			s.pubsubSubs = make(map[string]pubsubSubscription)
		}
		s.pubsubSubs[token] = sub
		s.pubsubMu.Unlock()
		return err
	}
	if s.owner != nil && provider == s.pubsubProvider {
		s.owner.pubsubUnsubscribed(sub.topic)
	}
	return nil
}

func (s *ComponentSession) publishPubsub(topic string, provider PubsubProvider, payload []byte, meta map[string]string) error {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	target := provider
	if target == nil {
		target = s.pubsubProvider
	}
	if target == nil {
		return ErrPubsubUnavailable
	}
	data := append([]byte(nil), payload...)
	var metaCopy map[string]string
	if meta != nil {
		metaCopy = cloneStringMap(meta)
	}
	return target.Publish(topic, data, metaCopy)
}

func (s *ComponentSession) handlePubsubDelivery(topic string, payload []byte, meta map[string]string) {
	s.deliverPubsub(topic, payload, meta)
}

func (s *ComponentSession) deliverPubsub(topic string, payload []byte, meta map[string]string) {
	if s == nil {
		return
	}
	s.pubsubMu.RLock()
	if len(s.pubsubSubs) == 0 {
		s.pubsubMu.RUnlock()
		return
	}
	handlers := make([]func([]byte, map[string]string), 0, len(s.pubsubSubs))
	for _, sub := range s.pubsubSubs {
		if sub.topic == topic && sub.handler != nil {
			handlers = append(handlers, sub.handler)
		}
	}
	s.pubsubMu.RUnlock()
	if len(handlers) == 0 {
		return
	}
	for _, h := range handlers {
		handler := h
		payloadCopy := append([]byte(nil), payload...)
		var metaCopy map[string]string
		if meta != nil {
			metaCopy = cloneStringMap(meta)
		}
		s.enqueuePubsub(func() {
			handler(payloadCopy, metaCopy)
		})
	}
}
