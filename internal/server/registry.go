package server

import (
	"embed"
	"errors"
	"io/fs"
	"sync"
	"sync/atomic"

	"github.com/eleven-am/pondlive/internal/session"
)

const DefaultMaxSessions = 10000

var (
	ErrSessionNotFound  = errors.New("server: session not found")
	ErrDuplicateSession = errors.New("server: duplicate session id")
	ErrRegistryFull     = errors.New("server: session registry is full")
	ErrPendingFull      = errors.New("server: too many sessions waiting for their first connection")
	ErrRegistryClosed   = errors.New("server: session registry is closed")
)

//go:embed static/pondlive.js static/pondlive-dev.js static/pondlive-dev.js.map static/tus.min.js static/tus.LICENSE static/tus.NOTICE
var assetsEmbed embed.FS

var Assets, _ = fs.Sub(assetsEmbed, "static")

type StateCounts struct {
	Pending      int
	Live         int
	Disconnected int
	Closing      int
}

type Stats struct {
	Sessions StateCounts
	Capacity int
	Stuck    int
	Closed   map[string]uint64
	Evicted  uint64
	Resets   map[string]uint64
	Rejected RejectedCounts
}

type RejectedCounts struct {
	Full      uint64
	Pending   uint64
	Duplicate uint64
	Closed    uint64
}

type SessionRegistry struct {
	mu       sync.RWMutex
	sessions map[session.SessionID]*session.LiveSession
	max      int
	maxPend  int
	pending  map[*session.LiveSession]struct{}
	stuck    map[*session.LiveSession]struct{}
	closed   bool

	resetMu sync.Mutex
	resets  map[string]uint64

	closes    [session.ReasonCount]atomic.Uint64
	evicted   atomic.Uint64
	rejFull   atomic.Uint64
	rejPend   atomic.Uint64
	rejDup    atomic.Uint64
	rejClosed atomic.Uint64
}

func NewSessionRegistry(max, maxPending int) *SessionRegistry {
	if max <= 0 {
		max = DefaultMaxSessions
	}
	if maxPending <= 0 {
		maxPending = max / 4
	}
	if maxPending < 1 {
		maxPending = 1
	}
	return &SessionRegistry{
		sessions: make(map[session.SessionID]*session.LiveSession),
		max:      max,
		maxPend:  maxPending,
		pending:  make(map[*session.LiveSession]struct{}),
		stuck:    make(map[*session.LiveSession]struct{}),
		resets:   make(map[string]uint64),
	}
}

func (r *SessionRegistry) admitLocked(id session.SessionID) error {
	if r.closed {
		r.rejClosed.Add(1)
		return ErrRegistryClosed
	}
	if _, exists := r.sessions[id]; exists {
		r.rejDup.Add(1)
		return ErrDuplicateSession
	}
	if len(r.sessions) >= r.max {
		r.rejFull.Add(1)
		return ErrRegistryFull
	}
	if len(r.pending) >= r.maxPend {
		r.rejPend.Add(1)
		return ErrPendingFull
	}
	return nil
}

func (r *SessionRegistry) Admit(id session.SessionID) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.admitLocked(id)
}

func (r *SessionRegistry) leftPending(sess *session.LiveSession) {
	r.mu.Lock()
	delete(r.pending, sess)
	r.mu.Unlock()
}

func (r *SessionRegistry) Put(sess *session.LiveSession) error {
	if sess == nil {
		return ErrSessionNotFound
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	id := sess.ID()
	if err := r.admitLocked(id); err != nil {
		return err
	}
	if sess.State() == session.StatePending {
		r.pending[sess] = struct{}{}
	}
	r.sessions[id] = sess
	return nil
}

func (r *SessionRegistry) take(sess *session.LiveSession, reason session.CloseReason) bool {
	if sess == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.sessions[sess.ID()]
	if !ok || cur != sess {
		return false
	}
	if reason == session.ReasonWedged {
		r.stuck[sess] = struct{}{}
	}
	r.closes[reason].Add(1)
	if reason.Evicted() {
		r.evicted.Add(1)
	}
	delete(r.sessions, sess.ID())
	delete(r.pending, sess)
	return true
}

func (r *SessionRegistry) sessionReset(cause string) {
	r.resetMu.Lock()
	r.resets[cause]++
	r.resetMu.Unlock()
}

func (r *SessionRegistry) sessionClosed(sess *session.LiveSession, reason session.CloseReason) {
	r.take(sess, reason)
}

func (r *SessionRegistry) Remove(sess *session.LiveSession) {
	if sess == nil {
		return
	}
	r.take(sess, session.ReasonRemoved)
	sess.RequestClose(session.ReasonRemoved)
}

func (r *SessionRegistry) Lookup(id session.SessionID) (*session.LiveSession, bool) {
	r.mu.RLock()
	sess, ok := r.sessions[id]
	r.mu.RUnlock()
	return sess, ok
}

func (r *SessionRegistry) Snapshot() []*session.LiveSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*session.LiveSession, 0, len(r.sessions))
	for _, sess := range r.sessions {
		out = append(out, sess)
	}
	return out
}

func (r *SessionRegistry) Range(fn func(*session.LiveSession) bool) {
	for _, sess := range r.Snapshot() {
		if !fn(sess) {
			return
		}
	}
}

func (r *SessionRegistry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.sessions)
}

func (r *SessionRegistry) Close() {
	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()
}

func (r *SessionRegistry) stuckSessions() []*session.LiveSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*session.LiveSession, 0, len(r.stuck))
	for sess := range r.stuck {
		select {
		case <-sess.Done():
			delete(r.stuck, sess)
		default:
			out = append(out, sess)
		}
	}
	return out
}

func (r *SessionRegistry) Stats() Stats {
	stuck := len(r.stuckSessions())
	var counts StateCounts
	r.mu.RLock()
	for _, sess := range r.sessions {
		switch sess.State() {
		case session.StatePending:
			counts.Pending++
		case session.StateLive:
			counts.Live++
		case session.StateDisconnected:
			counts.Disconnected++
		default:
			counts.Closing++
		}
	}
	r.mu.RUnlock()

	closed := make(map[string]uint64)
	for reason := session.ReasonNone + 1; reason < session.ReasonCount; reason++ {
		if n := r.closes[reason].Load(); n > 0 {
			closed[reason.String()] = n
		}
	}
	r.resetMu.Lock()
	resets := make(map[string]uint64, len(r.resets))
	for cause, n := range r.resets {
		resets[cause] = n
	}
	r.resetMu.Unlock()
	return Stats{
		Resets:   resets,
		Sessions: counts,
		Capacity: r.max,
		Stuck:    stuck,
		Closed:   closed,
		Evicted:  r.evicted.Load(),
		Rejected: RejectedCounts{
			Full:      r.rejFull.Load(),
			Pending:   r.rejPend.Load(),
			Duplicate: r.rejDup.Load(),
			Closed:    r.rejClosed.Load(),
		},
	}
}
