package upload

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tus/tusd/v2/pkg/handler"
)

const (
	TokenHeader          = "X-Upload-Token"
	DefaultGrantTTL      = 15 * time.Minute
	MaxGrantFiles        = 100
	DefaultMaxGrants     = 8
	DefaultSessionBudget = 256 << 20
)

var (
	ErrGrantUnknown     = errors.New("upload: unknown upload grant")
	ErrGrantExpired     = errors.New("upload: upload grant expired")
	ErrGrantExhausted   = errors.New("upload: upload grant has no files remaining")
	ErrUploadNotGranted = errors.New("upload: upload does not belong to this grant")
	ErrBudgetExceeded   = errors.New("upload: session upload budget exceeded")
	ErrUploadFinished   = errors.New("upload: upload is already complete and owned by the application")
)

type FileInfo = handler.FileInfo

type GrantSpec struct {
	Files      int
	MaxSize    int64
	Accept     []string
	OnComplete func(FileInfo) error
}

type uploadPhase uint8

const (
	phaseIncomplete uploadPhase = iota
	phaseFinishing
)

type uploadState struct {
	size  int64
	phase uploadPhase
}

type grant struct {
	sessionID string
	spec      GrantSpec
	remaining int
	uploads   map[string]*uploadState
	active    int
	expires   time.Time
	seq       uint64
}

type Limits struct {
	MaxGrants     int
	SessionBudget int64
}

type Pool struct {
	mu   sync.Mutex
	used int64
	max  int64
}

func NewPool(max int64) *Pool {
	return &Pool{max: max}
}

func (p *Pool) acquire(n int64) bool {
	if p == nil {
		return true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.max > 0 && p.used+n > p.max {
		return false
	}
	p.used += n
	return true
}

func (p *Pool) release(n int64) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.used -= n
	p.mu.Unlock()
}

func (p *Pool) Used() int64 {
	if p == nil {
		return 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.used
}

type BindOptions struct {
	Limits  Limits
	Reap    func(uploadIDs []string)
	Pool    *Pool
	OnClose func()
}

type Registry struct {
	mu       sync.Mutex
	grants   map[string]*grant
	ttl      time.Duration
	now      func() time.Time
	seq      uint64
	limits   Limits
	bound    bool
	reap     func(uploadIDs []string)
	pool     *Pool
	onClose  func()
	inflight int64
	closed   bool
	dropped  []string
}

func NewRegistry() *Registry {
	return &Registry{
		grants: make(map[string]*grant),
		ttl:    DefaultGrantTTL,
		now:    time.Now,
		limits: Limits{MaxGrants: DefaultMaxGrants, SessionBudget: DefaultSessionBudget},
	}
}

func SessionIDFromToken(token string) (string, bool) {
	i := strings.LastIndexByte(token, '.')
	if i <= 0 || i == len(token)-1 {
		return "", false
	}
	return token[:i], true
}

func ClampFiles(files int) int {
	if files < 1 {
		return 1
	}
	if files > MaxGrantFiles {
		return MaxGrantFiles
	}
	return files
}

func (r *Registry) Bind(opts BindOptions) bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	if r.bound {
		r.mu.Unlock()
		return false
	}
	r.bound = true
	r.reap = opts.Reap
	r.pool = opts.Pool
	r.onClose = opts.OnClose
	if opts.Limits.MaxGrants != 0 {
		r.limits.MaxGrants = opts.Limits.MaxGrants
	}
	if opts.Limits.SessionBudget != 0 {
		r.limits.SessionBudget = opts.Limits.SessionBudget
	}
	r.evictLocked()
	r.unlockAndReap()
	return true
}

func (r *Registry) Issue(sessionID string, spec GrantSpec) string {
	if r == nil || sessionID == "" {
		return ""
	}
	var secret [16]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return ""
	}
	token := sessionID + "." + base64.RawURLEncoding.EncodeToString(secret[:])

	spec.Files = ClampFiles(spec.Files)
	spec.Accept = append([]string(nil), spec.Accept...)

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return ""
	}
	now := r.now()
	r.sweepLocked(now)
	r.seq++
	r.grants[token] = &grant{
		sessionID: sessionID,
		spec:      spec,
		remaining: spec.Files,
		uploads:   make(map[string]*uploadState),
		expires:   now.Add(r.ttl),
		seq:       r.seq,
	}
	r.evictLocked()
	r.unlockAndReap()
	return token
}

func (r *Registry) Revoke(token string) {
	if r == nil || token == "" {
		return
	}
	r.mu.Lock()
	r.dropLocked(token)
	r.unlockAndReap()
}

func (r *Registry) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.closed = true
	for token := range r.grants {
		r.dropLocked(token)
	}
	onClose := r.onClose
	r.onClose = nil
	r.unlockAndReap()
	if onClose != nil {
		onClose()
	}
}

func (r *Registry) Expire() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.sweepLocked(r.now())
	r.unlockAndReap()
}

func (r *Registry) LiveUploadIDs() []string {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var ids []string
	for _, g := range r.grants {
		for id := range g.uploads {
			ids = append(ids, id)
		}
	}
	return ids
}

func (r *Registry) Reserve(token string, size int64, check func(GrantSpec) error) (string, GrantSpec, error) {
	if r == nil {
		return "", GrantSpec{}, ErrGrantUnknown
	}
	r.mu.Lock()
	id, spec, err := r.reserveLocked(token, size, check)
	r.unlockAndReap()
	return id, spec, err
}

func (r *Registry) reserveLocked(token string, size int64, check func(GrantSpec) error) (string, GrantSpec, error) {
	g, err := r.liveLocked(token)
	if err != nil {
		return "", GrantSpec{}, err
	}
	if g.remaining < 1 {
		return "", GrantSpec{}, ErrGrantExhausted
	}
	if check != nil {
		if err := check(g.spec); err != nil {
			return "", GrantSpec{}, err
		}
	}
	if size < 0 {
		size = 0
	}
	if r.limits.SessionBudget > 0 && r.inflight+size > r.limits.SessionBudget {
		return "", GrantSpec{}, ErrBudgetExceeded
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", GrantSpec{}, err
	}
	if !r.pool.acquire(size) {
		return "", GrantSpec{}, ErrBudgetExceeded
	}
	id := hex.EncodeToString(raw[:])
	g.remaining--
	g.uploads[id] = &uploadState{size: size}
	r.inflight += size
	return id, g.spec, nil
}

func (r *Registry) Acquire(token, uploadID string) (func(), error) {
	if r == nil {
		return nil, ErrGrantUnknown
	}
	r.mu.Lock()
	g, err := r.liveLocked(token)
	var release func()
	if err == nil {
		if _, ok := g.uploads[uploadID]; ok {
			g.active++
			release = func() { r.releaseRequest(g) }
		} else {
			err = ErrUploadNotGranted
		}
	}
	r.unlockAndReap()
	return release, err
}

func (r *Registry) releaseRequest(g *grant) {
	r.mu.Lock()
	if g.active > 0 {
		g.active--
	}
	g.expires = r.now().Add(r.ttl)
	r.mu.Unlock()
}

func (r *Registry) Finish(token, uploadID string) (GrantSpec, error) {
	if r == nil {
		return GrantSpec{}, ErrGrantUnknown
	}
	r.mu.Lock()
	g, err := r.liveLocked(token)
	var spec GrantSpec
	if err == nil {
		u, ok := g.uploads[uploadID]
		switch {
		case !ok:
			err = ErrUploadNotGranted
		default:
			if u.phase == phaseIncomplete {
				r.release(u.size)
				u.phase = phaseFinishing
			}
			spec = g.spec
		}
	}
	r.unlockAndReap()
	return spec, err
}

func (r *Registry) release(size int64) {
	r.inflight -= size
	r.pool.release(size)
}

func (r *Registry) Settle(token, uploadID string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	if g, ok := r.grants[token]; ok {
		if u, owned := g.uploads[uploadID]; owned {
			if u.phase == phaseIncomplete {
				r.release(u.size)
			}
			delete(g.uploads, uploadID)
		}
		if g.remaining == 0 && len(g.uploads) == 0 {
			delete(r.grants, token)
		}
	}
	r.mu.Unlock()
}

func (r *Registry) Terminate(token, uploadID string) error {
	if r == nil {
		return ErrGrantUnknown
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.grants[token]
	if !ok {
		return ErrGrantUnknown
	}
	u, owned := g.uploads[uploadID]
	if !owned {
		return ErrUploadNotGranted
	}
	if u.phase != phaseIncomplete {
		return ErrUploadFinished
	}
	r.release(u.size)
	delete(g.uploads, uploadID)
	if g.remaining == 0 && len(g.uploads) == 0 {
		delete(r.grants, token)
	}
	return nil
}

func (r *Registry) Len() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.grants)
}

func (r *Registry) Inflight() int64 {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inflight
}

func (r *Registry) liveLocked(token string) (*grant, error) {
	g, ok := r.grants[token]
	if !ok {
		return nil, ErrGrantUnknown
	}
	now := r.now()
	if g.active == 0 && !now.Before(g.expires) {
		r.dropLocked(token)
		return nil, ErrGrantExpired
	}
	g.expires = now.Add(r.ttl)
	return g, nil
}

func (r *Registry) dropLocked(token string) {
	g, ok := r.grants[token]
	if !ok {
		return
	}
	delete(r.grants, token)
	for id, u := range g.uploads {
		if u.phase == phaseIncomplete {
			r.release(u.size)
			r.dropped = append(r.dropped, id)
		}
	}
}

func (r *Registry) sweepLocked(now time.Time) {
	for token, g := range r.grants {
		if g.active == 0 && !now.Before(g.expires) {
			r.dropLocked(token)
		}
	}
}

func (r *Registry) evictLocked() {
	limit := r.limits.MaxGrants
	if limit <= 0 || len(r.grants) <= limit {
		return
	}
	type entry struct {
		token string
		seq   uint64
	}
	entries := make([]entry, 0, len(r.grants))
	for token, g := range r.grants {
		entries = append(entries, entry{token, g.seq})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].seq < entries[j].seq })
	for _, e := range entries[:len(entries)-limit] {
		r.dropLocked(e.token)
	}
}

func (r *Registry) unlockAndReap() {
	ids := r.dropped
	r.dropped = nil
	reap := r.reap
	r.mu.Unlock()
	if len(ids) > 0 && reap != nil {
		reap(ids)
	}
}
