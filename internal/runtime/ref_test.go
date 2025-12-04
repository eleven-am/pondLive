package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestElementRefAddHandler(t *testing.T) {
	ref := &ElementRef{
		id:       "test-ref",
		handlers: make(map[string][]work.Handler),
	}

	handler := work.Handler{
		EventOptions: metadata.EventOptions{
			Prevent: true,
		},
		Fn: func(evt work.Event) work.Updates {
			return nil
		},
	}

	ref.AddHandler("click", handler)

	if len(ref.handlers["click"]) != 1 {
		t.Errorf("expected 1 handler, got %d", len(ref.handlers["click"]))
	}
}

func TestElementRefAddMultipleHandlersSameEvent(t *testing.T) {
	ref := &ElementRef{
		id:       "test-ref",
		handlers: make(map[string][]work.Handler),
	}

	handler1 := work.Handler{
		EventOptions: metadata.EventOptions{
			Prevent: true,
		},
		Fn: func(evt work.Event) work.Updates {
			return nil
		},
	}

	handler2 := work.Handler{
		EventOptions: metadata.EventOptions{
			Stop: true,
		},
		Fn: func(evt work.Event) work.Updates {
			return nil
		},
	}

	ref.AddHandler("click", handler1)
	ref.AddHandler("click", handler2)

	if len(ref.handlers["click"]) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(ref.handlers["click"]))
	}
}

func TestElementRefAddHandlersDifferentEvents(t *testing.T) {
	ref := &ElementRef{
		id:       "test-ref",
		handlers: make(map[string][]work.Handler),
	}

	clickHandler := work.Handler{
		Fn: func(evt work.Event) work.Updates {
			return nil
		},
	}

	inputHandler := work.Handler{
		EventOptions: metadata.EventOptions{
			Debounce: 300,
		},
		Fn: func(evt work.Event) work.Updates {
			return nil
		},
	}

	ref.AddHandler("click", clickHandler)
	ref.AddHandler("input", inputHandler)

	if len(ref.handlers["click"]) != 1 {
		t.Errorf("expected 1 click handler, got %d", len(ref.handlers["click"]))
	}

	if len(ref.handlers["input"]) != 1 {
		t.Errorf("expected 1 input handler, got %d", len(ref.handlers["input"]))
	}
}

func TestElementRefAddHandlerIgnoresEmptyEvent(t *testing.T) {
	ref := &ElementRef{
		id:       "test-ref",
		handlers: make(map[string][]work.Handler),
	}

	handler := work.Handler{
		Fn: func(evt work.Event) work.Updates {
			return nil
		},
	}

	ref.AddHandler("", handler)

	if len(ref.handlers) != 0 {
		t.Error("expected no handlers when empty event name")
	}
}

func TestElementRefAddHandlerOnNilRef(t *testing.T) {
	var ref *ElementRef

	handler := work.Handler{
		Fn: func(evt work.Event) work.Updates {
			return nil
		},
	}

	ref.AddHandler("click", handler)
}

func TestElementRefProxyHandler(t *testing.T) {
	ref := &ElementRef{
		id:         "test-ref",
		handlers:   make(map[string][]work.Handler),
		generation: 1,
	}

	callCount := 0
	handler := work.Handler{
		Fn: func(evt work.Event) work.Updates {
			callCount++
			return nil
		},
	}

	ref.AddHandler("click", handler)

	proxy := ref.ProxyHandler("click")
	if proxy.Fn == nil {
		t.Fatal("expected proxy handler to have Fn")
	}

	evt := work.Event{Name: "click"}
	proxy.Fn(evt)

	if callCount != 1 {
		t.Errorf("expected handler to be called once, got %d", callCount)
	}
}

func TestElementRefProxyHandlerMultipleHandlers(t *testing.T) {
	ref := &ElementRef{
		id:         "test-ref",
		handlers:   make(map[string][]work.Handler),
		generation: 1,
	}

	callCount := 0
	handler1 := work.Handler{
		Fn: func(evt work.Event) work.Updates {
			callCount++
			return nil
		},
	}
	handler2 := work.Handler{
		Fn: func(evt work.Event) work.Updates {
			callCount++
			return nil
		},
	}

	ref.AddHandler("click", handler1)
	ref.AddHandler("click", handler2)

	proxy := ref.ProxyHandler("click")
	evt := work.Event{Name: "click"}
	proxy.Fn(evt)

	if callCount != 2 {
		t.Errorf("expected both handlers to be called, got %d calls", callCount)
	}
}

func TestElementRefProxyHandlerGenerationMismatch(t *testing.T) {
	ref := &ElementRef{
		id:         "test-ref",
		handlers:   make(map[string][]work.Handler),
		generation: 1,
	}

	callCount := 0
	handler := work.Handler{
		Fn: func(evt work.Event) work.Updates {
			callCount++
			return nil
		},
	}

	ref.AddHandler("click", handler)
	proxy := ref.ProxyHandler("click")

	ref.generation = 2

	evt := work.Event{Name: "click"}
	proxy.Fn(evt)

	if callCount != 0 {
		t.Errorf("expected handler not to be called, got %d calls", callCount)
	}
}

func TestElementRefMergedOptions(t *testing.T) {
	ref := &ElementRef{
		id:       "test-ref",
		handlers: make(map[string][]work.Handler),
	}

	handler1 := work.Handler{
		EventOptions: metadata.EventOptions{
			Prevent:  true,
			Props:    []string{"clientX"},
			Listen:   []string{"mousemove"},
			Debounce: 300,
		},
		Fn: func(evt work.Event) work.Updates {
			return nil
		},
	}

	handler2 := work.Handler{
		EventOptions: metadata.EventOptions{
			Stop:     true,
			Props:    []string{"clientY"},
			Listen:   []string{"mouseup"},
			Debounce: 500,
		},
		Fn: func(evt work.Event) work.Updates {
			return nil
		},
	}

	ref.AddHandler("click", handler1)
	ref.AddHandler("click", handler2)

	merged := ref.MergedOptions("click")

	if !merged.Prevent {
		t.Error("expected Prevent to be true")
	}
	if !merged.Stop {
		t.Error("expected Stop to be true")
	}

	if merged.Debounce != 300 {
		t.Errorf("expected Debounce to be 300, got %d", merged.Debounce)
	}

	if len(merged.Props) != 2 {
		t.Errorf("expected 2 props, got %d", len(merged.Props))
	}

	if len(merged.Listen) != 2 {
		t.Errorf("expected 2 listen events, got %d", len(merged.Listen))
	}
}

func TestElementRefMergedOptionsEmpty(t *testing.T) {
	ref := &ElementRef{
		id:       "test-ref",
		handlers: make(map[string][]work.Handler),
	}

	merged := ref.MergedOptions("click")

	if merged.Prevent || merged.Stop || merged.Passive {
		t.Error("expected all boolean options to be false")
	}
	if merged.Debounce != 0 || merged.Throttle != 0 {
		t.Error("expected timing options to be 0")
	}
	if merged.Props != nil || merged.Listen != nil {
		t.Error("expected Props and Listen to be nil")
	}
}

func TestElementRefResetAttachment(t *testing.T) {
	ref := &ElementRef{
		id:         "test-ref",
		handlers:   make(map[string][]work.Handler),
		attached:   true,
		generation: 1,
	}

	handler := work.Handler{
		Fn: func(evt work.Event) work.Updates {
			return nil
		},
	}

	ref.AddHandler("click", handler)
	initialGeneration := ref.generation

	if !ref.attached {
		t.Error("expected ref to be attached initially")
	}

	ref.ResetAttachment()

	if ref.attached {
		t.Error("expected ref to be detached after reset")
	}

	if ref.generation != initialGeneration+1 {
		t.Errorf("expected generation to increment, got %d", ref.generation)
	}

	if len(ref.handlers["click"]) != 0 {
		t.Error("expected handlers to be cleared after reset")
	}
}

func TestElementRefResetAttachmentOnNilRef(t *testing.T) {
	var ref *ElementRef

	ref.ResetAttachment()
}
