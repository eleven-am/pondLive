package runtime

import (
	"errors"
	"testing"
	"time"

	handlers "github.com/eleven-am/go/pondlive/internal/handlers"
	"github.com/eleven-am/go/pondlive/internal/protocol"
	render "github.com/eleven-am/go/pondlive/internal/render"
	h "github.com/eleven-am/go/pondlive/pkg/live/html"
)

type stubTransport struct {
	inits    []protocol.Init
	resumes  []protocol.ResumeOK
	frames   []protocol.Frame
	errors   []protocol.ServerError
	controls []protocol.PubsubControl
}

func (s *stubTransport) SendInit(init protocol.Init) error {
	s.inits = append(s.inits, init)
	return nil
}

func (s *stubTransport) SendResume(res protocol.ResumeOK) error {
	s.resumes = append(s.resumes, res)
	return nil
}

func (s *stubTransport) SendFrame(frame protocol.Frame) error {
	s.frames = append(s.frames, frame)
	return nil
}

func (s *stubTransport) SendServerError(err protocol.ServerError) error {
	s.errors = append(s.errors, err)
	return nil
}

func (s *stubTransport) SendPubsubControl(ctrl protocol.PubsubControl) error {
	s.controls = append(s.controls, ctrl)
	return nil
}

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) advance(d time.Duration) {
	c.now = c.now.Add(d)
}

func (c *fakeClock) Now() time.Time { return c.now }

func counterComponent(ctx Ctx, _ struct{}) h.Node {
	value, set := UseState(ctx, 0)
	handler := func(h.Event) h.Updates {
		set(value() + 1)
		return h.Rerender()
	}
	return h.Div(
		h.Button(
			h.On("click", handler),
			h.Textf("%d", value()),
		),
	)
}

func findClickHandler(structured render.Structured) handlers.ID {
	for _, dyn := range structured.D {
		if dyn.Kind != render.DynAttrs {
			continue
		}
		if id, ok := dyn.Attrs["data-onclick"]; ok {
			return handlers.ID(id)
		}
	}
	return ""
}

func boolPtr(v bool) *bool { return &v }

func TestLiveSessionJoinAndReplay(t *testing.T) {
	transport := &stubTransport{}
	clock := &fakeClock{now: time.Now()}
	sess := NewLiveSession("sid1", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport, Clock: clock.Now})

	if sess.nextSeq != 1 {
		t.Fatalf("expected nextSeq=1, got %d", sess.nextSeq)
	}

	join := sess.Join(1, 0)
	if join.Init == nil {
		t.Fatal("expected init response on cold join")
	}
	if join.Init.Seq != 1 {
		t.Fatalf("expected init seq 1, got %d", join.Init.Seq)
	}
	if sess.nextSeq != 2 {
		t.Fatalf("expected nextSeq=2 after init, got %d", sess.nextSeq)
	}
	if len(transport.inits) != 0 {
		t.Fatalf("expected init not to be sent automatically, got %d", len(transport.inits))
	}
	if err := transport.SendInit(*join.Init); err != nil {
		t.Fatalf("manual init send failed: %v", err)
	}
	if len(transport.inits) != 1 {
		t.Fatalf("expected init to be recorded after manual send, got %d", len(transport.inits))
	}

	handlerID := findClickHandler(sess.component.prev)
	if handlerID == "" {
		t.Fatal("expected click handler id")
	}

	if err := sess.DispatchEvent(handlerID, handlers.Event{Name: "click"}, 1); err != nil {
		t.Fatalf("dispatch event: %v", err)
	}
	if len(transport.frames) != 1 {
		t.Fatalf("expected one frame sent, got %d", len(transport.frames))
	}
	if transport.frames[0].Seq != 2 {
		t.Fatalf("expected frame seq 2, got %d", transport.frames[0].Seq)
	}

	resume := sess.Join(1, 1)
	if resume.Resume == nil {
		t.Fatal("expected resume response when acking init")
	}
	if resume.Resume.From != 2 || resume.Resume.To != 3 {
		t.Fatalf("unexpected resume range: %+v", resume.Resume)
	}
	if len(resume.Frames) != 1 || resume.Frames[0].Seq != 2 {
		t.Fatalf("expected replay of seq 2, got %+v", resume.Frames)
	}
	if len(transport.resumes) != 0 {
		t.Fatalf("expected join to remain side-effect free, saw %d resume sends", len(transport.resumes))
	}

	if err := sess.DispatchEvent(handlerID, handlers.Event{Name: "click"}, 2); err != nil {
		t.Fatalf("dispatch event 2: %v", err)
	}
	if len(transport.frames) != 2 {
		t.Fatalf("expected second frame sent, got %d", len(transport.frames))
	}
	if transport.frames[1].Seq != 3 {
		t.Fatalf("expected frame seq 3, got %d", transport.frames[1].Seq)
	}

	resume = sess.Join(1, 1)
	if resume.Resume == nil {
		t.Fatal("expected resume for stale client")
	}
	if len(resume.Frames) != 2 {
		t.Fatalf("expected two frames in replay, got %d", len(resume.Frames))
	}
	if resume.Resume.To != sess.nextSeq {
		t.Fatalf("expected resume to equal nextSeq=%d, got %d", sess.nextSeq, resume.Resume.To)
	}

	resume = sess.Join(1, 3)
	if resume.Resume == nil {
		t.Fatal("expected resume for up-to-date client")
	}
	if resume.Resume.From != sess.nextSeq || resume.Resume.To != sess.nextSeq {
		t.Fatalf("expected resume range equal nextSeq, got %+v", resume.Resume)
	}
}

func TestLiveSessionAckPrunesHistory(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession("sid2", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport})
	handlerID := findClickHandler(sess.component.prev)
	if handlerID == "" {
		t.Fatal("missing handler id")
	}

	sess.Join(1, 0) // boot
	if err := sess.DispatchEvent(handlerID, handlers.Event{Name: "click"}, 1); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if err := sess.DispatchEvent(handlerID, handlers.Event{Name: "click"}, 2); err != nil {
		t.Fatalf("dispatch2: %v", err)
	}

	if len(sess.frames) != 2 {
		t.Fatalf("expected two frames retained, got %d", len(sess.frames))
	}

	sess.Ack(2)
	if len(sess.frames) != 1 || sess.frames[0].Seq != 3 {
		t.Fatalf("expected frame seq 3 to remain after ack, frames=%+v", sess.frames)
	}

	sess.Ack(3)
	if len(sess.frames) != 0 {
		t.Fatalf("expected all frames pruned after ack 3, got %d", len(sess.frames))
	}
}

func TestLiveSessionClientEventDedup(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession("sid3", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport})
	handlerID := findClickHandler(sess.component.prev)
	if handlerID == "" {
		t.Fatal("missing handler id")
	}

	sess.Join(1, 0)
	if err := sess.DispatchEvent(handlerID, handlers.Event{Name: "click"}, 1); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if len(transport.frames) != 1 {
		t.Fatalf("expected first frame sent, got %d", len(transport.frames))
	}

	if err := sess.DispatchEvent(handlerID, handlers.Event{Name: "click"}, 1); err != nil {
		t.Fatalf("dispatch duplicate: %v", err)
	}
	if len(transport.frames) != 1 {
		t.Fatalf("expected duplicate event to be ignored")
	}
}

func TestMetricsRecorderReceivesFrames(t *testing.T) {
	recorder := &stubRecorder{}
	RegisterMetricsRecorder(recorder)
	defer RegisterMetricsRecorder(nil)

	transport := &stubTransport{}
	sess := NewLiveSession("sid-metrics", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport})
	join := sess.Join(1, 0)
	if join.Init == nil {
		t.Fatal("expected init payload")
	}

	handlerID := findClickHandler(sess.component.prev)
	if handlerID == "" {
		t.Fatal("expected handler id")
	}

	if err := sess.DispatchEvent(handlerID, handlers.Event{Name: "click"}, 1); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if len(recorder.records) == 0 {
		t.Fatalf("expected recorder to capture frames")
	}
	last := recorder.records[len(recorder.records)-1]
	if last.Ops == 0 {
		t.Fatalf("expected ops count > 0, got %d", last.Ops)
	}
	if last.SessionID != "sid-metrics" {
		t.Fatalf("unexpected session id: %v", last.SessionID)
	}
}

func TestLiveSessionTTLExpiry(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1700000000, 0)}
	ttl := 5 * time.Second
	sess := NewLiveSession("sid4", 1, counterComponent, struct{}{}, &LiveSessionConfig{Clock: clock.Now, TTL: ttl})

	if sess.Expired() {
		t.Fatal("session should not be expired immediately")
	}

	clock.advance(6 * time.Second)
	if !sess.Expired() {
		t.Fatal("expected session to expire after ttl elapses")
	}

	sess.Join(1, 0)
	if sess.Expired() {
		t.Fatal("session should refresh timestamp after join")
	}
}

func TestLiveSessionBuildBoot(t *testing.T) {
	sess := NewLiveSession("boot-session", 2, counterComponent, struct{}{}, nil)
	html := render.RenderHTML(sess.RenderRoot(), sess.Registry())

	boot := sess.BuildBoot(html)
	if boot.T != "boot" {
		t.Fatalf("expected boot type, got %q", boot.T)
	}
	if boot.SID != "boot-session" {
		t.Fatalf("expected session id boot-session, got %q", boot.SID)
	}
	if boot.Ver != 2 {
		t.Fatalf("expected version 2, got %d", boot.Ver)
	}
	if boot.Seq == 0 {
		t.Fatal("expected non-zero boot seq")
	}
	if boot.HTML != html {
		t.Fatalf("expected boot html to equal rendered html")
	}
	if boot.Client != nil {
		t.Fatalf("expected boot client config to be nil by default, got %#v", boot.Client)
	}

	join := sess.Join(sess.Version(), boot.Seq)
	if join.Init != nil {
		t.Fatal("expected join after boot to skip init")
	}
	if join.Resume == nil {
		t.Fatal("expected resume payload after boot join")
	}
	if join.Resume.From != boot.Seq+1 {
		t.Fatalf("unexpected resume from: %d", join.Resume.From)
	}
	if join.Resume.To != sess.nextSeq {
		t.Fatalf("unexpected resume to: %d", join.Resume.To)
	}
}

func TestLiveSessionBootIncludesClientConfig(t *testing.T) {
	cfg := &LiveSessionConfig{ClientConfig: &protocol.ClientConfig{Endpoint: "/ws"}}
	sess := NewLiveSession("boot-client", 1, counterComponent, struct{}{}, cfg)
	html := render.RenderHTML(sess.RenderRoot(), sess.Registry())

	boot := sess.BuildBoot(html)
	if boot.Client == nil {
		t.Fatal("expected boot client config to be populated")
	}
	if boot.Client.Endpoint != "/ws" {
		t.Fatalf("expected endpoint /ws, got %q", boot.Client.Endpoint)
	}
}

type stubRecorder struct {
	records []FrameRecord
}

func (s *stubRecorder) RecordFrame(r FrameRecord) {
	s.records = append(s.records, r)
}

func TestLiveSessionInitIncludesDiagnostics(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession("diag-init", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport, DevMode: boolPtr(true)})

	sess.ReportDiagnostic(Diagnostic{Code: "test_code", Message: "boom"})

	join := sess.Join(1, 0)
	if join.Init == nil {
		t.Fatal("expected init payload")
	}
	if join.Init.Errors == nil || len(join.Init.Errors) != 1 {
		t.Fatalf("expected init errors to contain diagnostic, got %+v", join.Init.Errors)
	}
	if join.Init.Errors[0].Code != "test_code" {
		t.Fatalf("unexpected error code %q", join.Init.Errors[0].Code)
	}
	if len(transport.inits) != 0 {
		t.Fatalf("expected init diagnostics not to be dispatched automatically")
	}
	if err := transport.SendInit(*join.Init); err != nil {
		t.Fatalf("manual init send failed: %v", err)
	}
	if len(transport.inits) != 1 || len(transport.inits[0].Errors) != 1 {
		t.Fatalf("transport init should include diagnostic payload")
	}
}

func TestLiveSessionResumeIncludesDiagnostics(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession("diag-resume", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport, DevMode: boolPtr(true)})

	join := sess.Join(1, 0)
	if join.Init == nil {
		t.Fatal("expected init payload for cold join")
	}

	sess.ReportDiagnostic(Diagnostic{Code: "resume_code", Message: "resume boom"})

	resume := sess.Join(sess.Version(), join.Init.Seq)
	if resume.Resume == nil {
		t.Fatal("expected resume payload")
	}
	if resume.Resume.Errors == nil || len(resume.Resume.Errors) != 1 {
		t.Fatalf("expected resume errors to include diagnostic, got %+v", resume.Resume.Errors)
	}
	if resume.Resume.Errors[0].Code != "resume_code" {
		t.Fatalf("unexpected resume error code %q", resume.Resume.Errors[0].Code)
	}
	if len(transport.resumes) != 0 {
		t.Fatalf("expected join to remain side-effect free, saw %d resume sends", len(transport.resumes))
	}
	if err := transport.SendResume(*resume.Resume); err != nil {
		t.Fatalf("send resume: %v", err)
	}
	if len(transport.resumes) == 0 || len(transport.resumes[len(transport.resumes)-1].Errors) != 1 {
		t.Fatalf("transport resume should include diagnostic payload")
	}
}

func TestLiveSessionRecoverClearsError(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession("recover", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport, DevMode: boolPtr(true)})

	join := sess.Join(1, 0)
	if join.Init == nil {
		t.Fatal("expected init payload")
	}
	handlerID := findClickHandler(sess.component.prev)
	if handlerID == "" {
		t.Fatal("expected handler id")
	}

	sess.component.handlePanic("flush", errors.New("boom"))
	if !sess.component.errored {
		t.Fatal("expected component session to be errored after panic")
	}

	if err := sess.Recover(); err != nil {
		t.Fatalf("recover returned error: %v", err)
	}
	if sess.component.errored {
		t.Fatal("expected component session to clear errored after recover")
	}

	if err := sess.DispatchEvent(handlerID, handlers.Event{Name: "click"}, 1); err != nil {
		t.Fatalf("dispatch after recovery failed: %v", err)
	}
}
