package runtime

import (
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestUseScript(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		Scripts: make(map[string]*scriptSlot),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	script := `
		function(element, transport) {
			console.log('Hello');
		}
	`

	handle := UseScript(ctx, script)
	if handle.slot == nil {
		t.Fatal("expected slot to be non-nil")
	}

	expectedID := "test-comp:s0"
	if handle.slot.id != expectedID {
		t.Errorf("expected script ID '%s', got '%s'", expectedID, handle.slot.id)
	}

	if _, exists := sess.Scripts[expectedID]; !exists {
		t.Error("expected script to be registered in session")
	}

	ctx.hookIndex = 0
	handle2 := UseScript(ctx, script)

	if handle.slot != handle2.slot {
		t.Error("expected same script slot across renders")
	}
}

func TestUseScriptMultiple(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		Scripts: make(map[string]*scriptSlot),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	script1 := "function(element, transport) { console.log('1'); }"
	script2 := "function(element, transport) { console.log('2'); }"

	handle1 := UseScript(ctx, script1)
	handle2 := UseScript(ctx, script2)

	if handle1.slot.id != "test-comp:s0" {
		t.Errorf("expected script1 ID 'test-comp:s0', got '%s'", handle1.slot.id)
	}
	if handle2.slot.id != "test-comp:s1" {
		t.Errorf("expected script2 ID 'test-comp:s1', got '%s'", handle2.slot.id)
	}

	if len(sess.Scripts) != 2 {
		t.Errorf("expected 2 scripts registered, got %d", len(sess.Scripts))
	}
}

func TestScriptHandleOn(t *testing.T) {
	inst := &Instance{
		ID:       "test-comp",
		cleanups: []func(){},
	}
	sess := &Session{
		Bus: protocol.NewBus(),
	}
	slot := &scriptSlot{
		id:       "test:s0",
		sess:     sess,
		instance: inst,
	}
	handle := ScriptHandle{slot: slot}

	callCount := 0
	var receivedData interface{}
	done := make(chan struct{})

	handle.On("test-event", func(data interface{}) {
		callCount++
		receivedData = data
		close(done)
	})

	channelID := protocol.Topic("script:test:s0")
	if sess.Bus.SubscriberCount(channelID) != 1 {
		t.Errorf("expected 1 subscriber on bus channel, got %d", sess.Bus.SubscriberCount(channelID))
	}

	slot.handleMessage("test-event", "test-data")

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for handler")
	}

	if callCount != 1 {
		t.Errorf("expected handler called once, got %d", callCount)
	}

	if receivedData != "test-data" {
		t.Errorf("expected data 'test-data', got %v", receivedData)
	}
}

func TestScriptHandleSend(t *testing.T) {
	inst := &Instance{
		ID: "test-comp",
	}
	sess := &Session{
		Bus: protocol.NewBus(),
	}
	slot := &scriptSlot{
		id:       "test:s0",
		sess:     sess,
		instance: inst,
	}
	handle := ScriptHandle{slot: slot}

	var receivedAction string
	var receivedPayload protocol.ScriptPayload
	done := make(chan struct{})

	sess.Bus.Subscribe(protocol.Topic("script:test:s0"), func(action string, data interface{}) {
		receivedAction = action
		if payload, ok := data.(protocol.ScriptPayload); ok {
			receivedPayload = payload
		}
		close(done)
	})

	handle.Send("client-event", map[string]string{"key": "value"})

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for callback")
	}

	if receivedAction != "send" {
		t.Errorf("expected action 'send', got '%s'", receivedAction)
	}

	if receivedPayload.ScriptID != "test:s0" {
		t.Errorf("expected script ID 'test:s0', got '%s'", receivedPayload.ScriptID)
	}

	if receivedPayload.Event != "client-event" {
		t.Errorf("expected event 'client-event', got '%s'", receivedPayload.Event)
	}

	if receivedPayload.Data == nil {
		t.Error("expected data to be non-nil")
	}
}

func TestScriptHandleAttachTo(t *testing.T) {
	slot := &scriptSlot{
		id:     "test:s0",
		script: "console.log('test')",
	}
	handle := ScriptHandle{slot: slot}

	elem := &work.Element{
		Tag: "div",
	}

	handle.AttachTo(elem)

	if elem.Script == nil {
		t.Fatal("expected Script to be non-nil")
	}

	if elem.Script.ScriptID != "test:s0" {
		t.Errorf("expected ScriptID 'test:s0', got '%s'", elem.Script.ScriptID)
	}

	if elem.Script.Script != "console.log('test')" {
		t.Errorf("expected script content, got '%s'", elem.Script.Script)
	}
}

func TestScriptHandleOnNilSlot(t *testing.T) {
	handle := ScriptHandle{slot: nil}

	handle.On("test", func(data interface{}) {})
	handle.Send("test", nil)
	handle.AttachTo(&work.Element{})
}

func TestScriptUpdateOnRerender(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		Scripts: make(map[string]*scriptSlot),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	script1 := "console.log('v1')"
	script2 := "console.log('v2')"

	handle1 := UseScript(ctx, script1)
	slot := handle1.slot

	ctx.hookIndex = 0
	handle2 := UseScript(ctx, script2)

	if handle2.slot != slot {
		t.Error("expected same slot")
	}

	slot.scriptMu.RLock()
	updatedScript := slot.script
	slot.scriptMu.RUnlock()

	if updatedScript == script1 {
		t.Error("expected script to be updated")
	}
}

func TestMinifyJS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes unnecessary spaces",
			input:    "function   test()   {  return  42;  }",
			expected: "function test(){return 42;}",
		},
		{
			name:     "preserves string literals",
			input:    `console.log("hello world")`,
			expected: `console.log("hello world")`,
		},
		{
			name:     "removes newlines",
			input:    "function test() {\n  return 42;\n}",
			expected: "function test(){return 42;}",
		},
		{
			name:     "preserves spaces between identifiers",
			input:    "return value",
			expected: "return value",
		},
		{
			name:     "handles multiple string types",
			input:    "var a = \"double\"; var b = 'single'; var c = `template`;",
			expected: "var a=\"double\";var b='single';var c=`template`;",
		},
		{
			name:     "reduces consecutive whitespace to single space",
			input:    "return     value",
			expected: "return value",
		},
		{
			name:     "handles mixed whitespace types",
			input:    "function\t\t\ntest() {\n\treturn\t42;\n}",
			expected: "function test(){return 42;}",
		},
		{
			name:     "removes single-line comments",
			input:    "var x = 1; // this is a comment\nvar y = 2;",
			expected: "var x=1;var y=2;",
		},
		{
			name:     "removes multi-line comments",
			input:    "var x = 1; /* comment */ var y = 2;",
			expected: "var x=1;var y=2;",
		},
		{
			name:     "removes multi-line comment spanning lines",
			input:    "var x = 1;\n/* this\nis\na\ncomment */\nvar y = 2;",
			expected: "var x=1;var y=2;",
		},
		{
			name:     "preserves regex literals",
			input:    "var re = /test/gi;",
			expected: "var re=/test/gi;",
		},
		{
			name:     "handles regex with escapes",
			input:    "var re = /\\/path\\//;",
			expected: "var re=/\\/path\\//;",
		},
		{
			name:     "handles regex with character class",
			input:    "var re = /[a-z]/;",
			expected: "var re=/[a-z]/;",
		},
		{
			name:     "distinguishes division from regex",
			input:    "var x = a / b;",
			expected: "var x=a/b;",
		},
		{
			name:     "regex after assignment",
			input:    "x = /pattern/;",
			expected: "x=/pattern/;",
		},
		{
			name:     "division after closing paren",
			input:    "(a + b) / c",
			expected: "(a+b)/c",
		},
		{
			name:     "preserves slash in string",
			input:    `var s = "a/b";`,
			expected: `var s="a/b";`,
		},
		{
			name:     "comment inside string preserved",
			input:    `var s = "// not a comment";`,
			expected: `var s="// not a comment";`,
		},
		{
			name:     "template literal with expression",
			input:    "var s = `hello ${name}`;",
			expected: "var s=`hello ${name}`;",
		},
		{
			name:     "regex with all flags",
			input:    "var re = /test/gimsuy;",
			expected: "var re=/test/gimsuy;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := minifyJS(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIsIdentifierChar(t *testing.T) {
	tests := []struct {
		ch       byte
		expected bool
	}{
		{'a', true},
		{'Z', true},
		{'5', true},
		{'_', true},
		{'$', true},
		{' ', false},
		{'+', false},
		{'(', false},
	}

	for _, tt := range tests {
		result := isIdentifierChar(tt.ch)
		if result != tt.expected {
			t.Errorf("isIdentifierChar(%c) = %v, expected %v", tt.ch, result, tt.expected)
		}
	}
}

func TestSessionHandleScriptMessage(t *testing.T) {
	inst := &Instance{
		ID:       "test-comp",
		cleanups: []func(){},
	}
	sess := &Session{
		Scripts: make(map[string]*scriptSlot),
		Bus:     protocol.NewBus(),
	}

	slot := &scriptSlot{
		id:       "test:s0",
		sess:     sess,
		instance: inst,
	}
	sess.Scripts["test:s0"] = slot

	callCount := 0
	done := make(chan struct{})
	slot.setEventHandler("test-event", func(data interface{}) {
		callCount++
		close(done)
	})

	sess.HandleScriptMessage("test:s0", "test-event", nil)

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for handler")
	}

	if callCount != 1 {
		t.Errorf("expected handler called once, got %d", callCount)
	}
}

func TestSessionHandleScriptMessageNotFound(t *testing.T) {
	sess := &Session{
		Scripts: make(map[string]*scriptSlot),
	}

	sess.HandleScriptMessage("nonexistent", "test-event", nil)
}

func TestHookScriptMismatch(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		Scripts: make(map[string]*scriptSlot),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	UseScript(ctx, "console.log('test')")

	ctx.hookIndex = 0
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for hook mismatch")
		}
	}()
	UseState[int](ctx, 0)
}

func TestScriptPanicRecovery(t *testing.T) {
	inst := &Instance{
		ID:       "test-comp",
		cleanups: []func(){},
	}
	sess := &Session{
		Scripts: make(map[string]*scriptSlot),
		Bus:     protocol.NewBus(),
	}

	slot := &scriptSlot{
		id:       "test:s0",
		sess:     sess,
		instance: inst,
	}
	sess.Scripts["test:s0"] = slot

	handle := ScriptHandle{slot: slot}
	handle.On("panic-event", func(data interface{}) {
		panic("intentional panic")
	})

	defer func() {
		if r := recover(); r != nil {
			t.Error("expected HandleScriptMessage to recover from panic, but it propagated")
		}
	}()

	sess.HandleScriptMessage("test:s0", "panic-event", nil)
}

func TestScriptCleanupAccumulationFix(t *testing.T) {
	inst := &Instance{
		ID:       "test-comp",
		cleanups: []func(){},
	}
	sess := &Session{
		Scripts: make(map[string]*scriptSlot),
		Bus:     protocol.NewBus(),
	}

	slot := &scriptSlot{
		id:       "test:s0",
		sess:     sess,
		instance: inst,
	}

	handle := ScriptHandle{slot: slot}

	for i := 0; i < 10; i++ {
		handle.On("test-event", func(data interface{}) {})
	}

	if len(inst.cleanups) != 1 {
		t.Errorf("expected 1 cleanup registered, got %d", len(inst.cleanups))
	}

	channelID := protocol.Topic("script:test:s0")
	if sess.Bus.SubscriberCount(channelID) != 1 {
		t.Errorf("expected 1 bus subscriber, got %d", sess.Bus.SubscriberCount(channelID))
	}
}

func TestScriptHandleOnConcurrentSafety(t *testing.T) {
	inst := &Instance{
		ID:       "test-comp",
		cleanups: []func(){},
	}
	sess := &Session{
		Scripts: make(map[string]*scriptSlot),
		Bus:     protocol.NewBus(),
	}

	slot := &scriptSlot{
		id:       "test:s0",
		sess:     sess,
		instance: inst,
	}

	handle := ScriptHandle{slot: slot}

	done := make(chan struct{})

	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				handle.On("event-"+string(rune('a'+n)), func(data interface{}) {})
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	slot.subsMu.Lock()
	subCount := len(slot.subs)
	slot.subsMu.Unlock()

	if subCount != 10 {
		t.Errorf("expected 10 unique event handlers, got %d", subCount)
	}
}

func TestScriptHandleIDNilSlot(t *testing.T) {
	handle := ScriptHandle{slot: nil}
	if handle.ID() != "" {
		t.Errorf("expected empty string for nil slot, got %q", handle.ID())
	}
}

func TestScriptHandleSendNilSlot(t *testing.T) {
	handle := ScriptHandle{slot: nil}
	handle.Send("event", "data")
}

func TestScriptHandleAttachToNilSlot(t *testing.T) {
	handle := ScriptHandle{slot: nil}
	elem := &work.Element{Tag: "div"}
	handle.AttachTo(elem)
}

func TestScriptHandleAttachToNilElement(t *testing.T) {
	inst := &Instance{ID: "test", cleanups: []func(){}}
	sess := &Session{Scripts: make(map[string]*scriptSlot), Bus: protocol.NewBus()}
	slot := &scriptSlot{id: "test:s0", sess: sess, instance: inst}
	handle := ScriptHandle{slot: slot}
	handle.AttachTo(nil)
}
