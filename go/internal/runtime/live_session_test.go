package runtime

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	render "github.com/eleven-am/pondlive/go/internal/render"
	"github.com/eleven-am/pondlive/go/internal/testh"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type stubTransport struct {
	inits       []protocol.Init
	resumes     []protocol.ResumeOK
	frames      []protocol.Frame
	templates   []protocol.TemplateFrame
	errors      []protocol.ServerError
	diagnostics []protocol.Diagnostic
	controls    []protocol.PubsubControl
	uploads     []protocol.UploadControl
	dom         []protocol.DOMRequest
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

func (s *stubTransport) SendTemplate(frame protocol.TemplateFrame) error {
	s.templates = append(s.templates, frame)
	return nil
}

func (s *stubTransport) SendServerError(err protocol.ServerError) error {
	s.errors = append(s.errors, err)
	return nil
}

func (s *stubTransport) SendDiagnostic(diag protocol.Diagnostic) error {
	s.diagnostics = append(s.diagnostics, diag)
	return nil
}

func (s *stubTransport) SendPubsubControl(ctrl protocol.PubsubControl) error {
	s.controls = append(s.controls, ctrl)
	return nil
}

func (s *stubTransport) SendUploadControl(ctrl protocol.UploadControl) error {
	s.uploads = append(s.uploads, ctrl)
	return nil
}

func (s *stubTransport) SendDOMRequest(req protocol.DOMRequest) error {
	s.dom = append(s.dom, req)
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

func awaitDOMRequest(t *testing.T, transport *stubTransport) protocol.DOMRequest {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		if len(transport.dom) > 0 {
			return transport.dom[len(transport.dom)-1]
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for dom request")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func findClickHandler(structured render.Structured) string {
	return findHandlerAttr(structured, "data-onclick")
}

func findHandlerAttr(structured render.Structured, attr string) string {
	event := strings.TrimPrefix(attr, "data-on")
	if idx := strings.IndexByte(event, '-'); idx != -1 {
		event = event[:idx]
	}
	event = strings.TrimSpace(event)
	if event == "" {
		return ""
	}
	for _, binding := range structured.Bindings {
		if binding.Event == event && binding.Handler != "" {
			return binding.Handler
		}
	}
	return ""
}

func boolPtr(v bool) *bool { return &v }

func containsString(list []string, needle string) bool {
	for _, item := range list {
		if item == needle {
			return true
		}
	}
	return false
}

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

	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}, 1); err != nil {
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
	if len(resume.Templates) != 0 {
		t.Fatalf("expected no template replay, got %d", len(resume.Templates))
	}
	if len(resume.Frames) != 1 || resume.Frames[0].Seq != 2 {
		t.Fatalf("expected replay of seq 2, got %+v", resume.Frames)
	}
	if len(transport.resumes) != 0 {
		t.Fatalf("expected join to remain side-effect free, saw %d resume sends", len(transport.resumes))
	}

	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}, 2); err != nil {
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
	if len(resume.Templates) != 0 {
		t.Fatalf("expected no template replay for stale client, got %d", len(resume.Templates))
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
	if len(resume.Templates) != 0 {
		t.Fatalf("expected no template replay when up to date, got %d", len(resume.Templates))
	}
}

func TestLiveSessionRefDeltaOnlyMetadata(t *testing.T) {
	var (
		setListen func([]string)
		setProps  func([]string)
	)
	component := func(ctx Ctx, _ struct{}) h.Node {
		getListen, updateListen := UseState(ctx, []string{"click"})
		getProps, updateProps := UseState(ctx, []string{"detail"})
		if setListen == nil {
			setListen = func(next []string) { updateListen(next) }
		}
		if setProps == nil {
			setProps = func(next []string) { updateProps(next) }
		}
		ref := UseElement[h.HTMLDivElement](ctx)
		ref.Bind("focus", h.EventBinding{
			Listen: append([]string(nil), getListen()...),
			Props:  append([]string(nil), getProps()...),
		})
		return h.Div(h.Attach(ref))
	}

	transport := &stubTransport{}
	sess := NewLiveSession("sid-ref-meta", 1, component, struct{}{}, &LiveSessionConfig{Transport: transport})

	if setListen == nil || setProps == nil {
		t.Fatal("expected state setters to be captured")
	}
	if len(sess.snapshot.Refs) != 1 {
		t.Fatalf("expected one ref in initial snapshot, got %d", len(sess.snapshot.Refs))
	}
	setListen([]string{"focus", "blur"})
	setProps([]string{"target.value"})

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	if len(transport.frames) != 0 {
		t.Fatalf("expected no frames since refs no longer include event metadata, got %d", len(transport.frames))
	}
}

func TestLiveSessionEmitsTemplateFrame(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession("tpl", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport})

	sess.component.setTemplateUpdate(templateUpdate{
		structured: sess.component.prev,
		html:       "<div>fresh</div>",
	})

	if err := sess.onPatch(nil); err != nil {
		t.Fatalf("onPatch returned error: %v", err)
	}

	if len(transport.templates) != 1 {
		t.Fatalf("expected one template frame, got %d", len(transport.templates))
	}
	tpl := transport.templates[0]
	if tpl.TemplatePayload.HTML != "<div>fresh</div>" {
		t.Fatalf("unexpected template html: %q", tpl.TemplatePayload.HTML)
	}
	if tpl.SID != string(sess.id) {
		t.Fatalf("template sid mismatch: %q", tpl.SID)
	}

	if len(transport.frames) != 1 {
		t.Fatalf("expected one subsequent frame, got %d", len(transport.frames))
	}
	frame := transport.frames[0]
	if len(frame.Patch) != 0 {
		t.Fatalf("expected no patch operations after template, got %d", len(frame.Patch))
	}
	if frame.Seq == 0 {
		t.Fatalf("expected frame sequence to be assigned")
	}
}

func TestLiveSessionReplaysTemplateFramesOnResume(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession("tpl-replay", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport})

	join := sess.Join(1, 0)
	if join.Init == nil {
		t.Fatal("expected init on first join")
	}

	sess.component.setTemplateUpdate(templateUpdate{
		structured: sess.component.prev,
		html:       "<div>fresh</div>",
	})

	if err := sess.onPatch(nil); err != nil {
		t.Fatalf("onPatch returned error: %v", err)
	}

	if len(transport.templates) != 1 {
		t.Fatalf("expected template frame to be sent, got %d", len(transport.templates))
	}
	if len(transport.frames) != 1 {
		t.Fatalf("expected follow-up frame to be sent, got %d", len(transport.frames))
	}

	frameSeq := transport.frames[0].Seq
	if frameSeq == 0 {
		t.Fatal("expected frame sequence to be assigned")
	}

	resume := sess.Join(1, join.Init.Seq)
	if resume.Resume == nil {
		t.Fatal("expected resume payload when acking init")
	}
	if len(resume.Templates) != 1 {
		t.Fatalf("expected one template frame to replay, got %d", len(resume.Templates))
	}
	tpl := resume.Templates[0]
	if tpl.TemplatePayload.HTML != "<div>fresh</div>" {
		t.Fatalf("unexpected template html replay: %q", tpl.TemplatePayload.HTML)
	}
	if len(resume.Frames) != 1 || resume.Frames[0].Seq != frameSeq {
		t.Fatalf("expected frame %d to replay, got %+v", frameSeq, resume.Frames)
	}

	sess.Ack(frameSeq)

	resume = sess.Join(1, frameSeq)
	if resume.Resume == nil {
		t.Fatal("expected resume payload when up to date")
	}
	if len(resume.Templates) != 0 {
		t.Fatalf("expected template replay to be pruned, got %d", len(resume.Templates))
	}
}

func TestLiveSessionStreamsDiagnostics(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession("diag-stream", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport, DevMode: boolPtr(true)})

	sess.ReportDiagnostic(Diagnostic{Code: "template_mismatch", Message: "boom"})

	if len(transport.diagnostics) != 1 {
		t.Fatalf("expected diagnostic stream to emit frame, got %d", len(transport.diagnostics))
	}
	diag := transport.diagnostics[0]
	if diag.Code != "template_mismatch" {
		t.Fatalf("unexpected diagnostic code: %q", diag.Code)
	}
	if diag.SID != string(sess.id) {
		t.Fatalf("expected diagnostic sid to match session, got %q", diag.SID)
	}
}

func TestDiagnosticsIncludeComponentScopeMetadata(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession("diag-scope", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport, DevMode: boolPtr(true)})

	root := sess.ComponentSession().root
	if root == nil {
		t.Fatal("expected root component to exist")
	}

	sess.ReportDiagnostic(Diagnostic{
		Code:        "comp_failure",
		Message:     "component panic",
		ComponentID: root.id,
	})

	if len(transport.diagnostics) != 1 {
		t.Fatalf("expected diagnostic frame, got %d", len(transport.diagnostics))
	}
	details := transport.diagnostics[0].Details
	if details == nil {
		t.Fatalf("expected diagnostic details to be populated")
	}
	if details.Metadata == nil {
		t.Fatalf("expected diagnostic metadata to include component scope")
	}
	scopeVal, ok := details.Metadata["componentScope"]
	if !ok {
		t.Fatalf("expected componentScope metadata, got %v", details.Metadata)
	}
	scope, ok := scopeVal.(map[string]any)
	if !ok {
		t.Fatalf("componentScope metadata should be map, got %T", scopeVal)
	}
	if scope["componentId"] != root.id {
		t.Fatalf("expected componentScope componentId %q, got %v", root.id, scope["componentId"])
	}
}

func TestEnsureSnapshotStaticsOwnedCopiesOnWrite(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession("sid-copy", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport})
	if len(sess.snapshot.Statics) == 0 {
		t.Fatalf("expected initial snapshot statics to be populated")
	}
	firstPtr := reflect.ValueOf(sess.snapshot.Statics).Pointer()
	sess.ensureSnapshotStaticsOwned()
	ownedPtr := reflect.ValueOf(sess.snapshot.Statics).Pointer()
	if firstPtr == ownedPtr {
		t.Fatalf("expected ensureSnapshotStaticsOwned to clone statics slice")
	}
	sess.ensureSnapshotStaticsOwned()
	if reflect.ValueOf(sess.snapshot.Statics).Pointer() != ownedPtr {
		t.Fatalf("expected second ensureSnapshotStaticsOwned call to avoid additional copies")
	}
}

func TestLiveSessionRefDeltaRemovesRef(t *testing.T) {
	var setAttached func(bool)
	component := func(ctx Ctx, _ struct{}) h.Node {
		getAttached, updateAttached := UseState(ctx, true)
		if setAttached == nil {
			setAttached = func(next bool) { updateAttached(next) }
		}
		if !getAttached() {
			return h.Div()
		}
		ref := UseElement[h.HTMLDivElement](ctx)
		return h.Div(h.Attach(ref))
	}

	transport := &stubTransport{}
	sess := NewLiveSession("sid-ref-remove", 1, component, struct{}{}, &LiveSessionConfig{Transport: transport})

	if setAttached == nil {
		t.Fatal("expected setter to be captured")
	}
	if len(sess.snapshot.Refs) != 1 {
		t.Fatalf("expected one ref in snapshot, got %d", len(sess.snapshot.Refs))
	}
	var id string
	for refID := range sess.snapshot.Refs {
		id = refID
		break
	}

	setAttached(false)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	var (
		refTemplate *protocol.TemplateFrame
		refFrame    *protocol.Frame
	)
	for i := len(transport.templates) - 1; i >= 0; i-- {
		tpl := transport.templates[i]
		if len(tpl.TemplatePayload.Refs.Del) > 0 {
			refTemplate = &tpl
			break
		}
	}
	if refTemplate == nil {
		for i := len(transport.frames) - 1; i >= 0; i-- {
			frame := transport.frames[i]
			if len(frame.Refs.Del) > 0 {
				refFrame = &frame
				break
			}
		}
	}
	switch {
	case refTemplate != nil:
		if len(refTemplate.TemplatePayload.Refs.Del) != 1 || refTemplate.TemplatePayload.Refs.Del[0] != id {
			t.Fatalf("expected ref deletion for %q, got %v", id, refTemplate.TemplatePayload.Refs.Del)
		}
	case refFrame != nil:
		if len(refFrame.Refs.Del) != 1 || refFrame.Refs.Del[0] != id {
			t.Fatalf("expected ref deletion for %q, got %v", id, refFrame.Refs.Del)
		}
	default:
		t.Fatalf("expected ref delta to be present in template or patch frames (templates=%+v, frames=%+v)", transport.templates, transport.frames)
	}
	if _, ok := sess.snapshot.Refs[id]; ok {
		t.Fatalf("expected ref %q removed from snapshot", id)
	}
}

func TestExtractHandlerMetaIncludesListenAndProps(t *testing.T) {
	handler := func(h.Event) h.Updates { return h.Rerender() }
	tree := h.Video(
		h.OnWith("timeupdate", h.EventOptions{
			Listen: []string{"play", "pause"},
			Props:  []string{"target.currentTime", "event.detail"},
		}, handler),
		h.On("play", handler),
	)

	structured, err := render.ToStructuredWithHandlers(tree, render.StructuredOptions{
		Components: testh.NewMockComponentLookup(),
	})
	if err != nil {
		t.Fatalf("ToStructuredWithHandlers failed: %v", err)
	}
	meta := extractHandlerMeta(structured)
	if len(meta) != 2 {
		t.Fatalf("expected two handler metas, got %d", len(meta))
	}
	var timeupdateMeta protocol.HandlerMeta
	var foundTimeupdate, foundPlay bool
	for _, m := range meta {
		if m.Event == "timeupdate" {
			timeupdateMeta = m
			foundTimeupdate = true
		} else if m.Event == "play" {
			foundPlay = true
		}
	}
	if !foundTimeupdate || !foundPlay {
		t.Fatalf("expected both timeupdate and play handlers, got metas: %+v", meta)
	}

	if !containsString(timeupdateMeta.Props, "target.currentTime") || !containsString(timeupdateMeta.Props, "event.detail") {
		t.Fatalf("props were not captured for timeupdate: %+v", timeupdateMeta.Props)
	}
	if !containsString(timeupdateMeta.Listen, "pause") || !containsString(timeupdateMeta.Listen, "play") {
		t.Fatalf("expected pause and play in listen metadata for timeupdate: %+v", timeupdateMeta.Listen)
	}
}

func TestLiveSessionAckPrunesHistory(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession("sid2", 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport})
	handlerID := findClickHandler(sess.component.prev)
	if handlerID == "" {
		t.Fatal("missing handler id")
	}

	sess.Join(1, 0)
	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}, 1); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}, 2); err != nil {
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
	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}, 1); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if len(transport.frames) != 1 {
		t.Fatalf("expected first frame sent, got %d", len(transport.frames))
	}

	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}, 1); err != nil {
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

	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}, 1); err != nil {
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
	html := render.RenderHTML(sess.RenderRoot())

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
	html := render.RenderHTML(sess.RenderRoot())

	boot := sess.BuildBoot(html)
	if boot.Client == nil {
		t.Fatal("expected boot client config to be populated")
	}
	if boot.Client.Endpoint != "/ws" {
		t.Fatalf("expected endpoint /ws, got %q", boot.Client.Endpoint)
	}
}

func TestLiveSessionBootEnablesClientDebugInDevMode(t *testing.T) {
	sess := NewLiveSession("boot-client-debug", 1, counterComponent, struct{}{}, &LiveSessionConfig{DevMode: boolPtr(true)})
	html := render.RenderHTML(sess.RenderRoot())

	boot := sess.BuildBoot(html)
	if boot.Client == nil {
		t.Fatal("expected boot client config to be populated in dev mode")
	}
	if boot.Client.Debug == nil || !*boot.Client.Debug {
		t.Fatalf("expected client debug to be enabled in dev mode, got %+v", boot.Client.Debug)
	}
}

func TestLiveSessionBootHonorsConfiguredClientDebug(t *testing.T) {
	debug := false
	cfg := &LiveSessionConfig{
		DevMode:      boolPtr(true),
		ClientConfig: &protocol.ClientConfig{Debug: &debug},
	}
	sess := NewLiveSession("boot-client-debug-override", 1, counterComponent, struct{}{}, cfg)
	html := render.RenderHTML(sess.RenderRoot())

	boot := sess.BuildBoot(html)
	if boot.Client == nil {
		t.Fatal("expected boot client config to be populated when provided")
	}
	if boot.Client.Debug == nil {
		t.Fatal("expected boot client debug flag to be present")
	}
	if *boot.Client.Debug {
		t.Fatalf("expected boot client debug to remain false, got true")
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

	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}, 1); err != nil {
		t.Fatalf("dispatch after recovery failed: %v", err)
	}
}

func TestLiveSessionDOMGetLifecycle(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession(SessionID("dom-get"), 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport})

	done := make(chan struct{})
	go func() {
		values, err := sess.DOMGet("ref:7", "element.value")
		if err != nil {
			t.Errorf("DOMGet returned error: %v", err)
		} else if values["element.value"] != "ready" {
			t.Errorf("unexpected DOMGet values: %#v", values)
		}
		close(done)
	}()

	req := awaitDOMRequest(t, transport)
	if req.Ref != "ref:7" {
		t.Fatalf("expected ref ref:7, got %q", req.Ref)
	}
	if len(req.Props) != 1 || req.Props[0] != "element.value" {
		t.Fatalf("unexpected selectors: %#v", req.Props)
	}

	sess.HandleDOMResponse(protocol.DOMResponse{ID: req.ID, Values: map[string]any{"element.value": "ready"}})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("DOMGet did not complete")
	}
}

func TestLiveSessionDOMGetTimeout(t *testing.T) {
	transport := &stubTransport{}
	sess := NewLiveSession(SessionID("dom-timeout"), 1, counterComponent, struct{}{}, &LiveSessionConfig{Transport: transport})
	sess.domGetTimeout = 20 * time.Millisecond

	start := time.Now()
	values, err := sess.DOMGet("ref:9", "element.value")
	if !errors.Is(err, errDOMGetTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if values != nil {
		t.Fatalf("expected nil values on timeout, got %#v", values)
	}
	if len(transport.dom) != 1 {
		t.Fatalf("expected exactly one dom request, got %d", len(transport.dom))
	}
	if time.Since(start) < sess.domGetTimeout {
		t.Fatalf("DOMGet returned before timeout elapsed")
	}
}

func TestLiveSessionHandleRouterReset(t *testing.T) {
	var outletID string

	routeComponent := func(ctx Ctx, _ Match) h.Node {
		return h.Div(h.Text("route"))
	}

	outletComponent := func(ctx Ctx, _ struct{}) h.Node {
		outletID = ctx.ComponentID()
		return Routes(ctx,
			Route(ctx, RouteProps{Path: "/", Component: routeComponent}),
		)
	}

	app := func(ctx Ctx, _ struct{}) h.Node {
		return Router(ctx,
			Render(ctx, outletComponent, struct{}{}),
		)
	}

	transport := &stubTransport{}
	dev := true
	sess := NewLiveSession(SessionID("router-reset"), 1, app, struct{}{}, &LiveSessionConfig{Transport: transport, DevMode: &dev})

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}
	if outletID == "" {
		t.Fatal("expected router outlet id to be captured")
	}
	t.Logf("captured outlet id: %s", outletID)
	if len(transport.templates) > 0 {
		tpl := transport.templates[len(transport.templates)-1]
		if tpl.Scope != nil {
			t.Logf("initial template scope: component=%s parent=%s", tpl.Scope.ComponentID, tpl.Scope.ParentID)
		} else {
			t.Log("initial template has no scope")
		}
	}
	for _, path := range sess.snapshot.ComponentPaths {
		t.Logf("snapshot component path: id=%s parent=%s", path.ComponentID, path.ParentID)
	}

	comp := sess.ComponentSession()
	if comp == nil {
		t.Fatal("expected component session")
	}
	comp.componentsMu.RLock()
	comp.componentsMu.RUnlock()
	if comp.componentByID(outletID) == nil {
		t.Fatalf("expected outlet component %s to be located", outletID)
	}

	transport.templates = nil
	transport.frames = nil

	if err := sess.HandleRouterReset(outletID); err != nil {
		t.Fatalf("router reset failed: %v", err)
	}
	if len(transport.templates) == 0 {
		t.Fatal("expected router reset to emit template frame")
	}
	tpl := transport.templates[len(transport.templates)-1]
	if tpl.Scope == nil || tpl.Scope.ComponentID != outletID {
		t.Fatalf("unexpected template scope: %+v", tpl.Scope)
	}
	if tpl.TemplatePayload.HTML == "" && len(tpl.TemplatePayload.S) == 0 {
		t.Fatalf("expected template payload to contain content, got %+v", tpl.TemplatePayload)
	}
	if len(transport.frames) == 0 {
		t.Fatal("expected diff frame alongside template")
	}
	if len(transport.frames[len(transport.frames)-1].Patch) != 0 {
		t.Fatalf("expected router reset frame without diff operations, got %d", len(transport.frames[len(transport.frames)-1].Patch))
	}

	transport.templates = nil
	transport.frames = nil
	transport.diagnostics = nil

	err := sess.HandleRouterReset("missing-component")
	if err == nil {
		t.Fatal("expected router reset to fail for unknown component")
	}
	if _, ok := AsDiagnosticError(err); !ok {
		t.Fatalf("expected diagnostic error, got %v", err)
	}
	if len(transport.diagnostics) == 0 {
		t.Fatal("expected diagnostic message for failed router reset")
	}
	diag := transport.diagnostics[len(transport.diagnostics)-1]
	if diag.Code != "router_reset_failed" {
		t.Fatalf("expected router_reset_failed diagnostic, got %q", diag.Code)
	}
	if len(transport.errors) == 0 {
		t.Fatal("expected server error payload to accompany diagnostic")
	}
	errMsg := transport.errors[len(transport.errors)-1]
	if errMsg.Details == nil {
		t.Fatal("expected error details for router reset diagnostic")
	}
	if id, ok := errMsg.Details.Metadata["componentId"].(string); !ok || id != "missing-component" {
		t.Fatalf("expected diagnostic metadata to include requested component id, got %+v", errMsg.Details.Metadata)
	}
}
