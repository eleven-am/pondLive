package runtime

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/upload"
	"github.com/eleven-am/pondlive/internal/work"
)

type Stopper interface {
	Stop() bool
}

type Clock interface {
	Now() time.Time
	AfterFunc(d time.Duration, f func()) Stopper
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (realClock) AfterFunc(d time.Duration, f func()) Stopper { return time.AfterFunc(d, f) }

type Config struct {
	SessionID       string
	Root            func(*Ctx) work.Node
	DevMode         bool
	Bus             *protocol.Bus
	UploadRegistry  *upload.Registry
	UploadEndpoint  string
	UploadScriptURL string
	HandlerQueueCap int
	DOMTimeout      time.Duration
	Clock           Clock
}

type Session struct {
	Bus             *protocol.Bus
	SessionID       string
	UploadRegistry  *upload.Registry
	UploadEndpoint  string
	UploadScriptURL string

	graph  *reactive.Graph
	root   *reactive.Owner
	rootFn func(*Ctx) work.Node
	clock  Clock

	nextID        uint32
	edits         []protocol.Edit
	handlers      map[uint32]*handlerEntry
	refs          map[uint32]*Ref
	echo          map[echoKey]string
	exiting       map[*reactive.Owner]struct{}
	failed        map[*reactive.Owner]struct{}
	arming        []func()
	mounted       bool
	holdFeeds     bool
	docLen        int
	rootMounting  bool
	armRunning    bool
	late          []func()
	mountBodyRuns int
	feeds         []feedHandle
	portalSeq     uint32
	crash         *Error
	rootBoundary  *Boundary
	heldDrains    []func()
	heldWarnings  []*Warning
	warnSeq       uint64
	leakSeen      map[string]struct{}
	connected     *reactive.Signal[bool]

	externals map[externalSyncer]struct{}

	scripts   map[string]*scriptSlot
	scriptSeq uint32
	portals   *portalHost

	domReqMgr  atomic.Pointer[domRequestManager]
	domMu      sync.Mutex
	domClosed  bool
	domTimeout atomic.Int64

	httpHandlers map[string]*handlerEntryHTTP
	httpSeq      uint32

	devMode bool

	mb              mailbox
	starting        atomic.Bool
	running         atomic.Bool
	exited          chan struct{}
	actorGID        atomic.Int64
	clientEvents    atomic.Int32
	flushes         atomic.Uint64
	overflows       atomic.Int64
	deferred        atomic.Bool
	deferredTimers  []*Timer
	stepStart       atomic.Int64
	stepSeq         atomic.Uint64
	stepKind        atomic.Uint32
	onClosed        func()
	handlerSlots    atomic.Int32
	handlerQueueCap atomic.Int32
	completions     completionTracker
	closed          bool
	ctx             context.Context
	cancelCtx       context.CancelFunc
}

var goidOnce sync.Once

func NewSession(cfg Config) *Session {
	goidOnce.Do(func() {
		if err := reactive.VerifyGoid(); err != nil {
			panic(err)
		}
	})
	s := &Session{
		Bus:             cfg.Bus,
		SessionID:       cfg.SessionID,
		UploadRegistry:  cfg.UploadRegistry,
		UploadEndpoint:  cfg.UploadEndpoint,
		UploadScriptURL: cfg.UploadScriptURL,
		rootFn:          cfg.Root,
		clock:           cfg.Clock,
		devMode:         cfg.DevMode,
		graph:           reactive.NewGraph(),
		nextID:          1,
		handlers:        make(map[uint32]*handlerEntry),
	}
	if s.clock == nil {
		s.clock = realClock{}
	}
	if s.Bus == nil {
		s.Bus = protocol.NewBus()
	}
	s.Bus.SubscribeToDOMValues(s.applyDOMValue)
	if cfg.HandlerQueueCap > 0 {
		s.handlerQueueCap.Store(int32(cfg.HandlerQueueCap))
	}
	if cfg.DOMTimeout > 0 {
		s.domTimeout.Store(int64(cfg.DOMTimeout))
	}
	s.graph.SetViolationHandler(s.onViolation)
	s.graph.SetCleanupPanicHandler(s.onCleanupPanic)
	s.graph.SetPoster(func(fn func()) { _ = s.post(message{kind: msgSetter, thunk: fn}) })
	s.connected = reactive.NewSignal(s.graph, false)
	return s
}

func (s *Session) Graph() *reactive.Graph { return s.graph }

func (s *Session) getDOMRequestManager() *domRequestManager {
	if s == nil || s.Bus == nil {
		return nil
	}
	if mgr := s.domReqMgr.Load(); mgr != nil {
		return mgr
	}
	s.domMu.Lock()
	defer s.domMu.Unlock()
	if s.domClosed {
		return nil
	}
	if mgr := s.domReqMgr.Load(); mgr != nil {
		return mgr
	}
	timeout := time.Duration(s.domTimeout.Load())
	if timeout == 0 {
		timeout = defaultDOMTimeout
	}
	fresh := newDOMRequestManager(timeout)
	s.domReqMgr.Store(fresh)
	return fresh
}

func (s *Session) CloseDOM() {
	if s != nil {
		s.closeDOM()
	}
}

func (s *Session) closeDOM() {
	s.domMu.Lock()
	s.domClosed = true
	mgr := s.domReqMgr.Swap(nil)
	s.domMu.Unlock()
	if mgr != nil {
		mgr.close()
	}
}

func (s *Session) domIsClosed() bool {
	s.domMu.Lock()
	defer s.domMu.Unlock()
	return s.domClosed
}

func (s *Session) SetDOMTimeout(timeout time.Duration) {
	if s == nil {
		return
	}
	if timeout == 0 {
		timeout = defaultDOMTimeout
	}
	s.domTimeout.Store(int64(timeout))
	if mgr := s.domReqMgr.Load(); mgr != nil {
		mgr.setTimeout(timeout)
	}
}

func (s *Session) ResolveDOMResponse(resp protocol.DOMResponsePayload) {
	if s == nil {
		return
	}
	if s.domReqMgr.Load() == nil {
		return
	}
	s.RunOnActor(func() { s.finishDOM(resp.RequestID, resp, nil) })
}

func (s *Session) SetHandlerQueueCap(n int) {
	if s != nil {
		s.handlerQueueCap.Store(int32(n))
	}
}

func (s *Session) SetDevMode(enabled bool) {
	if s == nil {
		return
	}
	s.devMode = enabled
}

func (s *Session) DevMode() bool { return s != nil && s.devMode }

var cancelledContext = func() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}()

func (s *Session) allocIDs(n uint32) uint32 {
	base := s.nextID
	s.nextID += n
	return base
}

func (s *Session) edit(e protocol.Edit) {
	s.edits = append(s.edits, e)
}

func (s *Session) Connected() reactive.Readable[bool] { return s.connected }

func (s *Session) SetConnected(v bool) {
	if s == nil {
		return
	}
	s.connected.Set(v)
}
