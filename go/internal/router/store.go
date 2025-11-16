package router

import (
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eleven-am/pondlive/go/internal/route"
)

// Location is an alias to route.Location for canonical URL representation.
type Location = route.Location

// NavKind describes the intent of a navigation event.
type NavKind uint8

const (
	NavKindPush NavKind = iota
	NavKindReplace
	NavKindBack
)

// NavEvent captures a navigation update that still needs to be delivered to the client.
type NavEvent struct {
	Seq    uint64
	Kind   NavKind
	Target Location
	Source string
	Time   time.Time
}

// RouterStore centralizes all mutable router state for a session.
type RouterStore struct {
	mu          sync.RWMutex
	loc         Location
	params      map[string]string
	history     []NavEvent
	nextSeq     atomic.Uint64
	listeners   map[uint64]func(Location)
	nextWatcher uint64
	navMu       sync.Mutex
	navHandlers []NavUpdateHandler
	queue       NavQueue
}

// Snapshot captures RouterStore state for hydration and SSR.
type Snapshot struct {
	Location Location
	Params   map[string]string
	History  []NavEvent
}

// NewStore constructs a RouterStore seeded with an initial location.
func NewStore(initial Location) *RouterStore {
	canon := canonicalizeLocation(initial)
	return &RouterStore{loc: canon.Clone(), params: map[string]string{}, listeners: map[uint64]func(Location){}}
}

// Location returns a copy of the current canonical location.
func (s *RouterStore) Location() Location {
	if s == nil {
		return Location{Path: "/", Query: url.Values{}}
	}
	s.mu.RLock()
	loc := s.loc.Clone()
	s.mu.RUnlock()
	return loc
}

// SetLocation overwrites the current location without recording a navigation event.
func (s *RouterStore) SetLocation(loc Location) {
	if s == nil {
		return
	}
	canon := canonicalizeLocation(loc)
	s.mu.Lock()
	s.loc = canon.Clone()
	listeners := s.snapshotListenersLocked()
	s.mu.Unlock()
	notifyListeners(listeners, canon)
}

// RecordNavigation updates the store location and appends a pending navigation event.
func (s *RouterStore) RecordNavigation(kind NavKind, target Location) NavEvent {
	return s.RecordNavigationWithSource(kind, target, "")
}

// RecordNavigationWithSource records a navigation event annotated with a source label.
func (s *RouterStore) RecordNavigationWithSource(kind NavKind, target Location, source string) NavEvent {
	if s == nil {
		return NavEvent{}
	}
	canon := canonicalizeLocation(target)
	event := NavEvent{Seq: s.nextSeq.Add(1), Kind: kind, Target: canon.Clone(), Source: source, Time: time.Now()}
	s.mu.Lock()
	s.loc = canon.Clone()
	s.history = append(s.history, event)
	listeners := s.snapshotListenersLocked()
	s.mu.Unlock()
	notifyListeners(listeners, canon)
	s.queue.Enqueue([]NavEvent{event})
	return event
}

// RecordBack enqueues a back navigation event without mutating location.
func (s *RouterStore) RecordBack() NavEvent {
	return s.RecordBackWithSource("")
}

// RecordBackWithSource enqueues a back navigation event annotated with a source label.
func (s *RouterStore) RecordBackWithSource(source string) NavEvent {
	if s == nil {
		return NavEvent{}
	}
	event := NavEvent{Seq: s.nextSeq.Add(1), Kind: NavKindBack, Source: source, Time: time.Now()}
	s.mu.Lock()
	s.history = append(s.history, event)
	s.mu.Unlock()
	s.queue.Enqueue([]NavEvent{event})
	return event
}

// History exposes the recorded navigation list.
func (s *RouterStore) History() []NavEvent {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	history := make([]NavEvent, len(s.history))
	copy(history, s.history)
	s.mu.RUnlock()
	return history
}

// Params returns a shallow copy of the current route params.
func (s *RouterStore) Params() map[string]string {
	if s == nil {
		return map[string]string{}
	}
	s.mu.RLock()
	params := cloneParams(s.params)
	s.mu.RUnlock()
	return params
}

// SetParams replaces the current params map.
func (s *RouterStore) SetParams(params map[string]string) {
	if s == nil {
		return
	}
	clone := cloneParams(params)
	s.mu.Lock()
	if len(clone) == 0 {
		s.params = map[string]string{}
	} else {
		s.params = clone
	}
	s.mu.Unlock()
}

// Snapshot returns a serializable view of the store state.
func (s *RouterStore) Snapshot() Snapshot {
	if s == nil {
		return Snapshot{}
	}
	return Snapshot{
		Location: s.Location(),
		Params:   s.Params(),
		History:  s.History(),
	}
}

// ApplySnapshot primes the store using serialized state.
func (s *RouterStore) ApplySnapshot(snap Snapshot) {
	if s == nil {
		return
	}
	s.SetLocation(snap.Location)
	s.SetParams(snap.Params)
	s.mu.Lock()
	s.history = cloneNavEvents(snap.History)
	s.mu.Unlock()
}

// Seed initializes the store with the provided location, params, and history.
func (s *RouterStore) Seed(loc Location, params map[string]string, history []NavEvent) {
	if s == nil {
		return
	}
	snap := Snapshot{
		Location: canonicalizeLocation(loc),
		Params:   cloneParams(params),
		History:  cloneNavEvents(history),
	}
	s.ApplySnapshot(snap)
}

// Subscribe registers a listener invoked whenever the location changes.
func (s *RouterStore) Subscribe(fn func(Location)) func() {
	if s == nil || fn == nil {
		return func() {}
	}
	s.mu.Lock()
	id := s.nextWatcher
	s.nextWatcher++
	if s.listeners == nil {
		s.listeners = make(map[uint64]func(Location))
	}
	s.listeners[id] = fn
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		delete(s.listeners, id)
		s.mu.Unlock()
	}
}

func (s *RouterStore) snapshotListenersLocked() []func(Location) {
	if len(s.listeners) == 0 {
		return nil
	}
	fns := make([]func(Location), 0, len(s.listeners))
	for _, fn := range s.listeners {
		fns = append(fns, fn)
	}
	return fns
}

func (s *RouterStore) RegisterNavHandler(handler NavUpdateHandler) func() {
	if s == nil || handler == nil {
		return func() {}
	}
	s.navMu.Lock()
	s.navHandlers = append(s.navHandlers, handler)
	s.navMu.Unlock()
	return func() {
		s.navMu.Lock()
		filtered := s.navHandlers[:0]
		for _, h := range s.navHandlers {
			if h != handler {
				filtered = append(filtered, h)
			}
		}
		s.navHandlers = filtered
		s.navMu.Unlock()
	}
}

func (s *RouterStore) DrainAndDispatch() {
	if s == nil {
		return
	}
	events := s.DrainNavEvents()
	if len(events) == 0 {
		return
	}
	s.navMu.Lock()
	handlers := append([]NavUpdateHandler(nil), s.navHandlers...)
	s.navMu.Unlock()
	for _, handler := range handlers {
		handler.DrainNav(events)
	}
}

// DrainNavEvents returns all pending navigation events in FIFO order.
func (s *RouterStore) DrainNavEvents() []NavEvent {
	if s == nil {
		return nil
	}
	return s.queue.Drain()
}

func notifyListeners(listeners []func(Location), loc Location) {
	if len(listeners) == 0 {
		return
	}
	for _, fn := range listeners {
		fn(loc.Clone())
	}
}

// canonicalizeLocation aligns paths, queries, and hash fragments with the same semantics as the legacy router.
func canonicalizeLocation(loc Location) Location {
	parts := route.NormalizeParts(loc.Path)
	return Location{
		Path:  parts.Path,
		Query: canonicalizeValues(loc.Query),
		Hash:  normalizeHash(loc.Hash, parts.Hash),
	}
}

func normalizeHash(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return route.NormalizeHash(trimmed)
	}
	if fallback != "" {
		return route.NormalizeHash(fallback)
	}
	return ""
}

func canonicalizeValues(values url.Values) url.Values {
	if len(values) == 0 {
		return url.Values{}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(url.Values, len(values))
	for _, key := range keys {
		out[key] = canonicalizeList(values[key])
	}
	return out
}

func canonicalizeList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	trimmed := make([]string, 0, len(values))
	for _, v := range values {
		trimmed = append(trimmed, strings.TrimSpace(v))
	}
	sort.Strings(trimmed)
	return trimmed
}

func cloneParams(params map[string]string) map[string]string {
	if len(params) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(params))
	for k, v := range params {
		out[k] = v
	}
	return out
}

func cloneNavEvents(events []NavEvent) []NavEvent {
	if len(events) == 0 {
		return nil
	}
	out := make([]NavEvent, len(events))
	for i, event := range events {
		out[i] = NavEvent{
			Seq:    event.Seq,
			Kind:   event.Kind,
			Target: event.Target.Clone(),
			Source: event.Source,
			Time:   event.Time,
		}
	}
	return out
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return url.Values{}
	}
	clone := make(url.Values, len(values))
	for k, v := range values {
		clone[k] = append([]string(nil), v...)
	}
	return clone
}
