package server

import (
	"errors"
	"sync"
	"time"

	"github.com/eleven-am/pondlive/go/internal/server/store"
	"github.com/eleven-am/pondlive/go/internal/session"
)

var (
	// ErrSessionNotFound reports attempts to fetch sessions that don't exist in the registry.
	ErrSessionNotFound = errors.New("server: session not found")
)

// Transport is the session transport interface.
type Transport = session.Transport

type sessionEntry struct {
	session   *session.LiveSession
	transport session.Transport
	connID    string
}

type transportRelease struct {
	session   *session.LiveSession
	transport session.Transport
}

func (rel transportRelease) release() {
	if rel.transport != nil {
		_ = rel.transport.Close()
	}
	if rel.session != nil {
		rel.session.SetTransport(nil)
	}
}

// SessionRegistry manages runtime2 LiveSession instances.
type SessionRegistry struct {
	mu             sync.RWMutex
	sessions       map[session.SessionID]*sessionEntry
	connections    map[string]*sessionEntry
	ttl            store.TTLStore
	touchObservers map[*session.LiveSession]func()
}

// NewSessionRegistry constructs an in-memory registry.
func NewSessionRegistry() *SessionRegistry {
	return NewSessionRegistryWithTTL(store.NewInMemoryTTLStore())
}

// NewSessionRegistryWithTTL constructs a registry backed by the provided TTL store.
func NewSessionRegistryWithTTL(ttl store.TTLStore) *SessionRegistry {
	return &SessionRegistry{
		sessions:       make(map[session.SessionID]*sessionEntry),
		connections:    make(map[string]*sessionEntry),
		ttl:            ttl,
		touchObservers: make(map[*session.LiveSession]func()),
	}
}

// Put registers the provided session.
func (r *SessionRegistry) Put(sess *session.LiveSession) {
	if sess == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	id := sess.ID()
	if _, exists := r.sessions[id]; exists {
		return
	}
	entry := &sessionEntry{session: sess}
	r.sessions[id] = entry
	r.attachSessionLocked(entry.session)
}

// Remove deletes a session from the registry.
func (r *SessionRegistry) Remove(id session.SessionID) {
	r.mu.Lock()
	release := r.removeSessionLocked(id, true)
	r.mu.Unlock()
	release.release()
}

// Attach binds a transport to the given session and connection id.
func (r *SessionRegistry) Attach(id session.SessionID, connID string, transport session.Transport) (*session.LiveSession, error) {
	if connID == "" || transport == nil {
		return nil, errors.New("server: missing connection or transport")
	}

	var releases []transportRelease
	r.mu.Lock()
	entry, ok := r.sessions[id]
	if !ok {
		r.mu.Unlock()
		return nil, ErrSessionNotFound
	}

	if entry.transport != nil && entry.transport != transport {
		releases = append(releases, transportRelease{session: entry.session, transport: entry.transport})
	}
	if entry.connID != "" && entry.connID != connID {
		delete(r.connections, entry.connID)
	}

	if bound, ok := r.connections[connID]; ok && bound != entry {
		releases = append(releases, transportRelease{session: bound.session, transport: bound.transport})
		bound.transport = nil
		bound.connID = ""
	}

	entry.transport = transport
	entry.connID = connID
	r.connections[connID] = entry
	r.attachSessionLocked(entry.session)
	r.mu.Unlock()

	for _, rel := range releases {
		rel.release()
	}

	entry.session.SetTransport(transport)
	return entry.session, nil
}

// Detach clears the connection binding.
func (r *SessionRegistry) Detach(connID string) {
	if connID == "" {
		return
	}
	var release transportRelease
	r.mu.Lock()
	if entry, ok := r.connections[connID]; ok {
		release = transportRelease{session: entry.session, transport: entry.transport}
		entry.transport = nil
		entry.connID = ""
		delete(r.connections, connID)
	}
	r.mu.Unlock()
	release.release()
}

// Lookup finds a session by id.
func (r *SessionRegistry) Lookup(id session.SessionID) (*session.LiveSession, bool) {
	r.mu.RLock()
	entry, ok := r.sessions[id]
	r.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return entry.session, true
}

// LookupWithConnection verifies the binding for a connection id.
func (r *SessionRegistry) LookupWithConnection(id session.SessionID, connID string) (*session.LiveSession, session.Transport, bool) {
	r.mu.RLock()
	entry, ok := r.sessions[id]
	r.mu.RUnlock()
	if !ok || entry == nil {
		return nil, nil, false
	}
	if connID == "" || entry.connID != connID {
		return entry.session, nil, false
	}
	return entry.session, entry.transport, entry.transport != nil
}

// LookupByConnection returns the session currently bound to a connection id.
func (r *SessionRegistry) LookupByConnection(connID string) (*session.LiveSession, session.Transport, bool) {
	if connID == "" {
		return nil, nil, false
	}
	r.mu.RLock()
	entry, ok := r.connections[connID]
	r.mu.RUnlock()
	if !ok || entry == nil {
		return nil, nil, false
	}
	return entry.session, entry.transport, entry.transport != nil
}

// ConnectionForSession returns the connection ID and transport for a session.
func (r *SessionRegistry) ConnectionForSession(id session.SessionID) (string, session.Transport, bool) {
	r.mu.RLock()
	entry, ok := r.sessions[id]
	r.mu.RUnlock()
	if !ok || entry == nil {
		return "", nil, false
	}
	return entry.connID, entry.transport, entry.connID != "" && entry.transport != nil
}

// SweepExpired prunes expired sessions.
func (r *SessionRegistry) SweepExpired() []session.SessionID {
	var expired []session.SessionID
	var releases []transportRelease

	if r.ttl != nil {
		if ids, err := r.ttl.Expired(time.Now()); err == nil && len(ids) > 0 {
			r.mu.Lock()
			for _, id := range ids {
				releases = append(releases, r.removeSessionLocked(id, false))
				expired = append(expired, id)
			}
			r.mu.Unlock()
		}
	}

	r.mu.Lock()
	for id, entry := range r.sessions {
		if entry.session == nil || !entry.session.IsExpired() {
			continue
		}
		releases = append(releases, r.removeSessionLocked(id, true))
		expired = append(expired, id)
	}
	r.mu.Unlock()

	for _, rel := range releases {
		rel.release()
	}
	return expired
}

// StartSweeper periodically sweeps expired sessions.
func (r *SessionRegistry) StartSweeper(interval time.Duration) func() {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				r.SweepExpired()
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return func() {
		close(done)
	}
}

func (r *SessionRegistry) attachSessionLocked(sess *session.LiveSession) {
	if sess == nil || r.ttl == nil {
		return
	}
	if remove := r.touchObservers[sess]; remove != nil {
		remove()
	}
	if ttl := sess.TTL(); ttl > 0 {
		_ = r.ttl.Touch(sess.ID(), ttl)
	}
	remove := sess.AddTouchObserver(func(time.Time) {
		if ttl := sess.TTL(); ttl > 0 {
			_ = r.ttl.Touch(sess.ID(), ttl)
		}
	})
	r.touchObservers[sess] = remove
}

func (r *SessionRegistry) removeSessionLocked(id session.SessionID, dropTTL bool) transportRelease {
	entry, ok := r.sessions[id]
	if !ok {
		return transportRelease{}
	}
	if entry.connID != "" {
		delete(r.connections, entry.connID)
	}
	if remove := r.touchObservers[entry.session]; remove != nil {
		remove()
		delete(r.touchObservers, entry.session)
	}
	if dropTTL && r.ttl != nil {
		_ = r.ttl.Remove(id)
	}
	delete(r.sessions, id)
	return transportRelease{session: entry.session, transport: entry.transport}
}
