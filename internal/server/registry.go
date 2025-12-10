package server

import (
	"embed"
	"errors"
	"io/fs"
	"sync"
	"time"

	"github.com/eleven-am/pondlive/internal/server/store"
	"github.com/eleven-am/pondlive/internal/session"
)

var (
	ErrSessionNotFound = errors.New("server: session not found")
)

//go:embed static/pondlive.js static/pondlive-dev.js static/pondlive-dev.js.map
var assetsEmbed embed.FS

var Assets, _ = fs.Sub(assetsEmbed, "static")

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

type SessionRegistry struct {
	mu             sync.RWMutex
	sessions       map[session.SessionID]*sessionEntry
	connections    map[string]*sessionEntry
	ttl            store.TTLStore
	touchObservers map[*session.LiveSession]func()
}

func NewSessionRegistry() *SessionRegistry {
	return NewSessionRegistryWithTTL(store.NewInMemoryTTLStore())
}

func NewSessionRegistryWithTTL(ttl store.TTLStore) *SessionRegistry {
	return &SessionRegistry{
		sessions:       make(map[session.SessionID]*sessionEntry),
		connections:    make(map[string]*sessionEntry),
		ttl:            ttl,
		touchObservers: make(map[*session.LiveSession]func()),
	}
}

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

func (r *SessionRegistry) Remove(id session.SessionID) {
	r.mu.Lock()
	release := r.removeSessionLocked(id, true)
	r.mu.Unlock()
	release.release()
}

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
	entry.session.SetTransport(transport)
	r.attachSessionLocked(entry.session)
	r.mu.Unlock()

	for _, rel := range releases {
		rel.release()
	}

	return entry.session, nil
}

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

func (r *SessionRegistry) Lookup(id session.SessionID) (*session.LiveSession, bool) {
	r.mu.RLock()
	entry, ok := r.sessions[id]
	r.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return entry.session, true
}

func (r *SessionRegistry) LookupWithConnection(id session.SessionID, connID string) (*session.LiveSession, session.Transport, bool) {
	r.mu.RLock()
	entry, ok := r.sessions[id]
	if !ok || entry == nil {
		r.mu.RUnlock()
		return nil, nil, false
	}
	sess := entry.session
	entryConnID := entry.connID
	transport := entry.transport
	r.mu.RUnlock()

	if connID == "" || entryConnID != connID {
		return sess, nil, false
	}
	return sess, transport, transport != nil
}

func (r *SessionRegistry) LookupByConnection(connID string) (*session.LiveSession, session.Transport, bool) {
	if connID == "" {
		return nil, nil, false
	}
	r.mu.RLock()
	entry, ok := r.connections[connID]
	if !ok || entry == nil {
		r.mu.RUnlock()
		return nil, nil, false
	}
	sess := entry.session
	transport := entry.transport
	r.mu.RUnlock()

	return sess, transport, transport != nil
}

func (r *SessionRegistry) ConnectionForSession(id session.SessionID) (string, session.Transport, bool) {
	r.mu.RLock()
	entry, ok := r.sessions[id]
	if !ok || entry == nil {
		r.mu.RUnlock()
		return "", nil, false
	}
	connID := entry.connID
	transport := entry.transport
	r.mu.RUnlock()

	return connID, transport, connID != "" && transport != nil
}

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
	remove := sess.OnTouch(func(time.Time) {
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

func (r *SessionRegistry) Range(fn func(*session.LiveSession) bool) {
	r.mu.RLock()
	sessions := make([]*session.LiveSession, 0, len(r.sessions))
	for _, entry := range r.sessions {
		if entry.session != nil {
			sessions = append(sessions, entry.session)
		}
	}
	r.mu.RUnlock()

	for _, sess := range sessions {
		if !fn(sess) {
			return
		}
	}
}
