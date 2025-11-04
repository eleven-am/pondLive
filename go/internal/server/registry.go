package server

import (
	"errors"
	"sync"
	"time"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server/store"
)

var (
	// ErrSessionNotFound reports attempts to attach or fetch sessions that are unknown to the registry.
	ErrSessionNotFound = errors.New("live: session not found")
)

// Transport extends the runtime transport with lifecycle hooks used by the registry.
type Transport interface {
	runtime.Transport
	Close() error
	SendEventAck(protocol.EventAck) error
	SendServerError(protocol.ServerError) error
}

// SessionRegistry stores live sessions and tracks which PondSocket connection currently owns them.
type SessionRegistry struct {
	mu             sync.RWMutex
	sessions       map[runtime.SessionID]*sessionEntry
	connections    map[string]*sessionEntry
	ttl            store.TTLStore
	touchObservers map[*runtime.LiveSession]func()
}

type sessionEntry struct {
	session   *runtime.LiveSession
	transport Transport
	connID    string
}

type transportRelease struct {
	session   *runtime.LiveSession
	transport Transport
}

func (rel transportRelease) release() {
	if rel.session == nil || rel.transport == nil {
		return
	}
	rel.session.DetachTransport(rel.transport)
	_ = rel.transport.Close()
}

// NewSessionRegistry constructs an empty registry.
func NewSessionRegistry() *SessionRegistry {
	return NewSessionRegistryWithTTL(store.NewInMemoryTTLStore())
}

// NewSessionRegistryWithTTL constructs a registry backed by the provided TTL store.
func NewSessionRegistryWithTTL(ttl store.TTLStore) *SessionRegistry {
	return &SessionRegistry{
		sessions:       make(map[runtime.SessionID]*sessionEntry),
		connections:    make(map[string]*sessionEntry),
		ttl:            ttl,
		touchObservers: make(map[*runtime.LiveSession]func()),
	}
}

// Put registers a session so future websocket joins can resume it.
func (r *SessionRegistry) Put(sess *runtime.LiveSession) {
	if sess == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[sess.ID()] = &sessionEntry{session: sess}
	r.attachSessionLocked(sess)
}

// Remove deletes a session from the registry, detaching any active transport.
func (r *SessionRegistry) Remove(id runtime.SessionID) {
	r.mu.Lock()
	release := r.removeSessionLocked(id, true)
	r.mu.Unlock()

	release.release()
}

// Attach wires the given transport to the session and associates it with the connection id.
func (r *SessionRegistry) Attach(id runtime.SessionID, connID string, transport Transport) (*runtime.LiveSession, error) {
	if connID == "" || transport == nil {
		return nil, errors.New("live: missing connection or transport")
	}

	var (
		session  *runtime.LiveSession
		releases []transportRelease
	)

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
	session = entry.session
	r.attachSessionLocked(session)

	r.mu.Unlock()

	for _, rel := range releases {
		rel.release()
	}

	if session == nil {
		_ = transport.Close()
		return nil, errors.New("live: session unavailable")
	}

	session.AttachTransport(transport)

	return session, nil
}

// Lookup returns the session for the provided id, if present.
func (r *SessionRegistry) Lookup(id runtime.SessionID) (*runtime.LiveSession, bool) {
	r.mu.RLock()
	entry, ok := r.sessions[id]
	r.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return entry.session, true
}

// LookupWithConnection returns the session for the provided id and verifies it is bound to the given connection id.
// The returned boolean reports whether the connection currently owns the session.
func (r *SessionRegistry) LookupWithConnection(id runtime.SessionID, connID string) (*runtime.LiveSession, Transport, bool) {
	if id == "" {
		return nil, nil, false
	}

	r.mu.RLock()
	entry, ok := r.sessions[id]
	r.mu.RUnlock()

	if !ok || entry == nil || entry.session == nil {
		return nil, nil, false
	}

	if connID == "" || entry.connID != connID {
		return entry.session, nil, false
	}

	return entry.session, entry.transport, entry.transport != nil
}

// LookupByConnection returns the session and transport currently bound to the connection id.
func (r *SessionRegistry) LookupByConnection(connID string) (*runtime.LiveSession, Transport, bool) {
	r.mu.RLock()
	entry, ok := r.connections[connID]
	r.mu.RUnlock()
	if !ok {
		return nil, nil, false
	}
	return entry.session, entry.transport, true
}

// ConnectionForSession returns the connection id and transport currently attached to the session, if any.
func (r *SessionRegistry) ConnectionForSession(id runtime.SessionID) (string, Transport, bool) {
	r.mu.RLock()
	entry, ok := r.sessions[id]
	r.mu.RUnlock()
	if !ok || entry == nil || entry.connID == "" || entry.transport == nil {
		return "", nil, false
	}
	return entry.connID, entry.transport, true
}

// DetachConnection clears the transport bound to the provided connection id.
func (r *SessionRegistry) DetachConnection(connID string) {
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

// SweepExpired prunes sessions whose TTL has elapsed and returns their ids.
func (r *SessionRegistry) SweepExpired() []runtime.SessionID {
	if r.ttl == nil {
		r.mu.Lock()
		var (
			expired  []runtime.SessionID
			releases []transportRelease
		)
		for id, entry := range r.sessions {
			if entry.session == nil || !entry.session.Expired() {
				continue
			}
			release := r.removeSessionLocked(id, true)
			releases = append(releases, release)
			expired = append(expired, id)
		}
		r.mu.Unlock()
		for _, rel := range releases {
			rel.release()
		}
		return expired
	}
	var (
		result   []runtime.SessionID
		releases []transportRelease
	)
	if ids, err := r.ttl.Expired(time.Now()); err == nil && len(ids) > 0 {
		r.mu.Lock()
		for _, id := range ids {
			release := r.removeSessionLocked(id, true)
			releases = append(releases, release)
		}
		r.mu.Unlock()
		result = append(result, ids...)
	}
	r.mu.Lock()
	for id, entry := range r.sessions {
		if entry.session == nil || !entry.session.Expired() {
			continue
		}
		release := r.removeSessionLocked(id, true)
		releases = append(releases, release)
		result = append(result, id)
	}
	r.mu.Unlock()
	for _, rel := range releases {
		rel.release()
	}
	return result
}

func (r *SessionRegistry) attachSessionLocked(sess *runtime.LiveSession) {
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

func (r *SessionRegistry) removeSessionLocked(id runtime.SessionID, dropTTL bool) transportRelease {
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
	release := transportRelease{session: entry.session, transport: entry.transport}
	entry.transport = nil
	entry.connID = ""
	delete(r.sessions, id)
	return release
}
