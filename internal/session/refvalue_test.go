package session

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestBrowserValuesReachTheSignalAndAreNotEchoed(t *testing.T) {
	var ref *runtime.Ref
	var live runtime.ReadSignal[string]
	comp := func(ctx *runtime.Ctx) work.Node {
		ref = runtime.UseCell(ctx, func() *runtime.Ref { return runtime.NewRef(ctx, "MediaRef", nil) })
		live = runtime.RefSignal(ref, runtime.RefSpec[string]{
			Prop: "currentTime",
			Decode: func(v any) (string, bool) {
				s, ok := v.(string)
				return s, ok
			},
		})
		return work.BuildElement("div", work.BuildElement("input", runtime.AttachItem(ref)), work.BuildElement("p", work.TextOf(live)))
	}
	ls := NewLiveSession("value", 1, comp, nil, WithRequest(httptest.NewRequest(http.MethodGet, "/", nil)))
	t.Cleanup(func() { _ = ls.Close() })
	if _, err := ls.Render(context.Background()); err != nil {
		t.Fatal(err)
	}
	tr := newRecordingTransport()
	if _, err := ls.Attach(context.Background(), tr, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	var id uint32
	onSession(t, ls, func() { id = ref.ID() })

	observed := false
	for _, m := range tr.messages() {
		if m.Topic == string(protocol.DOMHandler) && m.Event == string(protocol.DOMObserveAction) {
			observed = true
		}
	}
	if !observed {
		t.Fatalf("no observe frame was sent when the session went live: %v", tr.messages())
	}

	before := len(tr.messages())
	err := deliver(ls, protocol.ClientEvt{
		Event:   protocol.Event{Type: protocol.DOMHandler, SID: "value"},
		Action:  string(protocol.DOMValueAction),
		Payload: map[string]any{"id": float64(id), "prop": "currentTime", "value": "typed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := ls.Flush(); err != nil {
		t.Fatal(err)
	}
	var got string
	onSession(t, ls, func() { got = live.Peek() })
	if got != "typed" {
		t.Fatalf("signal = %q", got)
	}
	frames := tr.messages()[before:]
	var sawEdit bool
	for _, m := range frames {
		if m.Topic == string(protocol.DOMHandler) {
			t.Fatalf("a client value was echoed back: %+v", m)
		}
		if m.Topic == string(protocol.TopicFrame) && strings.Contains(fmt.Sprint(m.Data), "typed") {
			sawEdit = true
		}
	}
	if !sawEdit {
		t.Fatalf("the bound text did not update: %v", frames)
	}
}

func TestObserversAreSentAgainAfterTheDocumentIsReset(t *testing.T) {
	var ref *runtime.Ref
	comp := func(ctx *runtime.Ctx) work.Node {
		ref = runtime.UseCell(ctx, func() *runtime.Ref { return runtime.NewRef(ctx, "MediaRef", nil) })
		live := runtime.RefSignal(ref, runtime.RefSpec[string]{Prop: "currentTime", Decode: func(v any) (string, bool) {
			s, ok := v.(string)
			return s, ok
		}})
		return work.BuildElement("div", work.BuildElement("input", runtime.AttachItem(ref)), work.BuildElement("p", work.TextOf(live)))
	}
	ls, tr := startLiveWith(t, comp, nil)
	count := func() (observes, resets int, order []string) {
		for _, m := range tr.messages() {
			switch {
			case m.Topic == string(protocol.DOMHandler) && m.Event == string(protocol.DOMObserveAction):
				observes++
				order = append(order, "observe")
			case m.Topic == "frame" && m.Event == "reset":
				resets++
				order = append(order, "reset")
			}
		}
		return
	}
	before, _, _ := count()
	if before == 0 {
		t.Fatal("no observe frame before the reset")
	}
	resync(t, ls, ls.out.headSeq())
	after, resets, order := count()
	if resets != 1 || after != before+1 {
		t.Fatalf("observes %d -> %d, resets %d", before, after, resets)
	}
	if order[len(order)-1] != "observe" || order[len(order)-2] != "reset" {
		t.Fatalf("the observe must follow the reset frame: %v", order)
	}
}
