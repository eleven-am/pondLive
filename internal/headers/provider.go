package headers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eleven-am/pondlive/internal/runtime"
)

const cookieTokenTTL = 2 * time.Minute

const cookieTokenBodyLimit = 1024

const cookieTokenCap = 32

type pendingCookie struct {
	name    string
	value   string
	options *CookieOptions
}

func (c *pendingCookie) deletes() bool {
	return c.options != nil && c.options.MaxAge < 0
}

func (c *pendingCookie) received() (*http.Cookie, bool) {
	pair := cookiePair(c.httpCookie())
	if pair == "" {
		return nil, false
	}
	parsed, err := http.ParseCookie(pair)
	if err != nil || len(parsed) != 1 {
		return nil, false
	}
	return parsed[0], true
}

func (c *pendingCookie) httpCookie() *http.Cookie {
	cookie := &http.Cookie{
		Name:  c.name,
		Value: c.value,
	}

	if c.options != nil {
		cookie.Path = c.options.Path
		cookie.Domain = c.options.Domain
		cookie.MaxAge = c.options.MaxAge
		cookie.Expires = c.options.Expires
		cookie.Secure = c.options.Secure
		cookie.HttpOnly = c.options.HttpOnly
		cookie.SameSite = c.options.SameSite
	}

	if cookie.Path == "" {
		cookie.Path = "/"
	}
	return cookie
}

type tokenSlot struct {
	cookie  *pendingCookie
	seq     uint64
	expires time.Time
}

type tokenItem struct {
	token string
	seq   uint64
}

type tokenQueue struct {
	slots map[string]tokenSlot
	order []tokenItem
	head  int
	seq   uint64
}

func (q *tokenQueue) live(item tokenItem) bool {
	slot, ok := q.slots[item.token]
	return ok && slot.seq == item.seq
}

func (q *tokenQueue) sweep(now time.Time) {
	for q.head < len(q.order) {
		item := q.order[q.head]
		if q.live(item) && now.Before(q.slots[item.token].expires) {
			break
		}
		if q.live(item) {
			delete(q.slots, item.token)
		}
		q.head++
	}
	q.compact()
}

func (q *tokenQueue) evictOldest() bool {
	for q.head < len(q.order) {
		item := q.order[q.head]
		q.head++
		if q.live(item) {
			delete(q.slots, item.token)
			return true
		}
	}
	return false
}

func (q *tokenQueue) compact() {
	remaining := len(q.order) - q.head
	switch {
	case remaining == 0:
		q.order = q.order[:0]
		q.head = 0
	case remaining > 4*cookieTokenCap:
		kept := q.order[:0]
		for _, item := range q.order[q.head:] {
			if q.live(item) {
				kept = append(kept, item)
			}
		}
		q.order = kept
		q.head = 0
	case q.head > 4*cookieTokenCap:
		copy(q.order, q.order[q.head:])
		q.order = q.order[:remaining]
		q.head = 0
	}
}

func (q *tokenQueue) add(token string, cookie *pendingCookie, now time.Time) bool {
	q.sweep(now)
	if q.slots == nil {
		q.slots = make(map[string]tokenSlot)
	}
	evicted := false
	if _, exists := q.slots[token]; !exists && len(q.slots) >= cookieTokenCap {
		evicted = q.evictOldest()
	}
	q.seq++
	q.slots[token] = tokenSlot{cookie: cookie, seq: q.seq, expires: now.Add(cookieTokenTTL)}
	q.order = append(q.order, tokenItem{token: token, seq: q.seq})
	q.compact()
	return evicted
}

func (q *tokenQueue) take(token string, now time.Time) *pendingCookie {
	q.sweep(now)
	slot, ok := q.slots[token]
	if !ok {
		return nil
	}
	delete(q.slots, token)
	return slot.cookie
}

type cookieTokens struct {
	mu       sync.Mutex
	now      func() time.Time
	onEvict  func(queue string)
	pending  tokenQueue
	awaiting tokenQueue
}

func newCookieTokens() *cookieTokens {
	return &cookieTokens{now: time.Now}
}

func (c *cookieTokens) store(token string, cookie *pendingCookie) {
	c.mu.Lock()
	evicted := c.pending.add(token, cookie, c.now())
	c.mu.Unlock()
	if evicted && c.onEvict != nil {
		c.onEvict("pending")
	}
}

func (c *cookieTokens) consume(token string) *pendingCookie {
	c.mu.Lock()
	now := c.now()
	cookie := c.pending.take(token, now)
	evicted := cookie != nil && c.awaiting.add(token, cookie, now)
	c.mu.Unlock()
	if evicted && c.onEvict != nil {
		c.onEvict("awaiting-ack")
	}
	return cookie
}

func (c *cookieTokens) acknowledge(token string) *pendingCookie {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.awaiting.take(token, c.now())
}

func evictionReporter(report func(runtime.Report)) func(string) {
	if report == nil {
		return nil
	}
	return func(queue string) {
		report(runtime.Report{
			Severity: runtime.SeverityDiagnostic,
			Phase:    "headers:cookie-token-evicted",
			Message:  "outstanding cookie token limit reached; the oldest token was evicted and its cookie will not be applied",
			Meta: map[string]any{
				"queue": queue,
				"limit": cookieTokenCap,
			},
		})
	}
}

type tokenHolder struct {
	tokens  atomic.Pointer[cookieTokens]
	onEvict func(queue string)
}

func (h *tokenHolder) get() *cookieTokens {
	if t := h.tokens.Load(); t != nil {
		return t
	}
	fresh := newCookieTokens()
	fresh.onEvict = h.onEvict
	h.tokens.CompareAndSwap(nil, fresh)
	return h.tokens.Load()
}

type providerState struct {
	requestState  *RequestState
	script        runtime.ScriptHandle
	handler       runtime.HandlerHandle
	handlerURL    string
	ackHandlerURL string
	holder        *tokenHolder
}

func (p *providerState) storeCookie(token string, cookie *pendingCookie) {
	p.holder.get().store(token, cookie)
}

type tokenPayload struct {
	Token string `json:"token"`
}

func readToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	defer r.Body.Close()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, cookieTokenBodyLimit))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		return "", false
	}

	var payload tokenPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return "", false
	}
	return payload.Token, true
}

func (p *providerState) serveCookieSet(w http.ResponseWriter, r *http.Request) {
	token, ok := readToken(w, r)
	if !ok {
		return
	}

	pending := p.holder.get().consume(token)
	if pending == nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	cookie := pending.httpCookie()

	http.SetCookie(w, cookie)
	w.WriteHeader(http.StatusNoContent)
}

func (p *providerState) serveCookieAck(w http.ResponseWriter, r *http.Request) {
	token, ok := readToken(w, r)
	if !ok {
		return
	}

	issued := p.holder.get().acknowledge(token)
	if issued == nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	p.requestState.ApplyCookie(issued.httpCookie(), issued.deletes())
	p.requestState.NotifyChange()
	w.WriteHeader(http.StatusNoContent)
}
