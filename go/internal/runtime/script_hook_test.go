package runtime

import (
	"sync"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

type scriptEvent struct {
	scriptID string
	event    string
	data     interface{}
}

type testScriptEventCollector struct {
	mu     sync.Mutex
	events []scriptEvent
}

func (c *testScriptEventCollector) send(scriptID, event string, data interface{}) error {
	c.mu.Lock()
	c.events = append(c.events, scriptEvent{
		scriptID: scriptID,
		event:    event,
		data:     data,
	})
	c.mu.Unlock()
	return nil
}

func (c *testScriptEventCollector) collect() []scriptEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	collected := append([]scriptEvent(nil), c.events...)
	c.events = nil
	return collected
}

func TestUseScriptBasic(t *testing.T) {
	var scriptHandle ScriptHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		scriptHandle = UseScript(ctx, "(element, transport) => { element.textContent = 'Hello'; }")
		node := &dom.StructuredNode{Tag: "div"}
		scriptHandle.AttachTo(node)
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if scriptHandle.slot == nil {
		t.Error("expected script slot to be created")
	}
	if scriptHandle.slot.script != "(element, transport) => { element.textContent = 'Hello'; }" {
		t.Errorf("unexpected script content: %s", scriptHandle.slot.script)
	}
}

func TestUseScriptAttachment(t *testing.T) {
	var scriptHandle ScriptHandle
	var createdNode *dom.StructuredNode

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		scriptHandle = UseScript(ctx, "(element, transport) => {}")
		node := &dom.StructuredNode{Tag: "div"}
		scriptHandle.AttachTo(node)
		createdNode = node
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if createdNode == nil || createdNode.Script == nil {
		t.Fatal("expected script to be attached to node")
	}
	if createdNode.Script.ScriptID == "" {
		t.Error("expected script ID to be set")
	}
	if createdNode.Script.Script != "(element, transport) => {}" {
		t.Errorf("unexpected script: %s", createdNode.Script.Script)
	}
}

func TestUseScriptOnMessage(t *testing.T) {
	messageReceived := false
	var receivedData interface{}
	var scriptHandle ScriptHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		scriptHandle = UseScript(ctx, "(element, transport) => {}")
		scriptHandle.On("test-event", func(data interface{}) {
			messageReceived = true
			receivedData = data
		})
		node := &dom.StructuredNode{Tag: "div"}
		scriptHandle.AttachTo(node)
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	testData := map[string]any{"foo": "bar", "count": 42}
	sess.HandleScriptMessage(scriptHandle.slot.id, "test-event", testData)

	if !messageReceived {
		t.Error("expected message handler to be called")
	}
	dataMap := receivedData.(map[string]any)
	if dataMap["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", dataMap["foo"])
	}
	if dataMap["count"] != 42 {
		t.Errorf("expected count=42, got %v", dataMap["count"])
	}
}

func TestUseScriptSend(t *testing.T) {
	var scriptHandle ScriptHandle
	collector := &testScriptEventCollector{}

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		scriptHandle = UseScript(ctx, "(element, transport) => {}")
		node := &dom.StructuredNode{Tag: "div"}
		scriptHandle.AttachTo(node)
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.SetScriptEventSender(collector.send)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	scriptHandle.Send("update", map[string]any{"text": "Updated"})

	events := collector.collect()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].event != "update" {
		t.Errorf("expected event 'update', got '%s'", events[0].event)
	}
	dataMap := events[0].data.(map[string]any)
	if dataMap["text"] != "Updated" {
		t.Errorf("expected data text='Updated', got %v", dataMap["text"])
	}
}

func TestUseScriptMultipleSends(t *testing.T) {
	var scriptHandle ScriptHandle
	collector := &testScriptEventCollector{}

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		scriptHandle = UseScript(ctx, "(element, transport) => {}")
		node := &dom.StructuredNode{Tag: "div"}
		scriptHandle.AttachTo(node)
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.SetScriptEventSender(collector.send)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	scriptHandle.Send("event1", map[string]any{"value": "first"})
	scriptHandle.Send("event2", map[string]any{"value": "second"})
	scriptHandle.Send("event3", map[string]any{"value": "third"})

	events := collector.collect()
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	expected := []struct {
		event string
		value string
	}{
		{"event1", "first"},
		{"event2", "second"},
		{"event3", "third"},
	}

	for i, exp := range expected {
		if events[i].event != exp.event {
			t.Errorf("event %d: expected '%s', got '%s'", i, exp.event, events[i].event)
		}
		dataMap := events[i].data.(map[string]any)
		if dataMap["value"] != exp.value {
			t.Errorf("event %d: expected value '%s', got %v", i, exp.value, dataMap["value"])
		}
	}
}

func TestUseScriptEventsClearedAfterCollect(t *testing.T) {
	collector := &testScriptEventCollector{}
	var scriptHandle ScriptHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		scriptHandle = UseScript(ctx, "(element, transport) => {}")
		node := &dom.StructuredNode{Tag: "div"}
		scriptHandle.AttachTo(node)
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.SetScriptEventSender(collector.send)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	scriptHandle.Send("test", map[string]any{"count": 1})

	events1 := collector.collect()
	if len(events1) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events1))
	}

	events2 := collector.collect()
	if len(events2) != 0 {
		t.Fatalf("expected 0 events after collect, got %d", len(events2))
	}
}

func TestUseScriptUpdate(t *testing.T) {
	var scriptHandle ScriptHandle
	var triggerUpdate func()

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		version, setVersion := UseState(ctx, 1)
		triggerUpdate = func() { setVersion(version() + 1) }

		script := "(element, transport) => { element.dataset.version = '" + string(rune(version()+'0')) + "'; }"
		scriptHandle = UseScript(ctx, script)
		node := &dom.StructuredNode{Tag: "div"}
		scriptHandle.AttachTo(node)
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	initialScript := scriptHandle.slot.script

	triggerUpdate()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	updatedScript := scriptHandle.slot.script
	if initialScript == updatedScript {
		t.Error("expected script to be updated")
	}
}

func TestUseScriptHandleNilCases(t *testing.T) {
	var nilHandle ScriptHandle

	nilHandle.AttachTo(&dom.StructuredNode{Tag: "div"})
	nilHandle.On("test", func(data interface{}) {})
	nilHandle.Send("test", nil)
}

func TestUseScriptMultipleScriptsInComponent(t *testing.T) {
	var handle1, handle2 ScriptHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		handle1 = UseScript(ctx, "(element, transport) => { /* script 1 */ }")
		handle2 = UseScript(ctx, "(element, transport) => { /* script 2 */ }")

		node1 := &dom.StructuredNode{Tag: "div"}
		node2 := &dom.StructuredNode{Tag: "span"}
		handle1.AttachTo(node1)
		handle2.AttachTo(node2)

		return &dom.StructuredNode{
			Tag:      "section",
			Children: []*dom.StructuredNode{node1, node2},
		}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if handle1.slot.id == handle2.slot.id {
		t.Error("expected different script IDs")
	}
	if handle1.slot.script == handle2.slot.script {
		t.Error("expected different scripts")
	}
}

func TestUseScriptMessageHandlerUpdate(t *testing.T) {
	var scriptHandle ScriptHandle
	message1Received := false
	message2Received := false

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		scriptHandle = UseScript(ctx, "(element, transport) => {}")
		node := &dom.StructuredNode{Tag: "div"}
		scriptHandle.AttachTo(node)
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	scriptHandle.On("test", func(data interface{}) {
		message1Received = true
	})

	sess.HandleScriptMessage(scriptHandle.slot.id, "test", map[string]any{"test": 1})
	if !message1Received {
		t.Error("expected first handler to be called")
	}

	scriptHandle.On("test2", func(data interface{}) {
		message2Received = true
	})

	sess.HandleScriptMessage(scriptHandle.slot.id, "test2", map[string]any{"test": 2})
	if !message2Received {
		t.Error("expected second handler to be called")
	}
}

func TestUseScriptSendImmediately(t *testing.T) {
	collector := &testScriptEventCollector{}
	var scriptHandle ScriptHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		scriptHandle = UseScript(ctx, "(element, transport) => {}")
		node := &dom.StructuredNode{Tag: "div"}
		scriptHandle.AttachTo(node)
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.SetScriptEventSender(collector.send)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	scriptHandle.Send("update", map[string]any{"value": "test"})

	events := collector.collect()
	if len(events) != 1 {
		t.Fatalf("expected 1 event to be sent immediately, got %d", len(events))
	}
	if events[0].event != "update" {
		t.Errorf("expected event 'update', got '%s'", events[0].event)
	}
}

func TestUseScriptHandleNonExistentScript(t *testing.T) {
	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	sess.HandleScriptMessage("non-existent-id", "test", map[string]any{"test": "data"})
}

func TestUseScriptComplexData(t *testing.T) {
	var receivedData interface{}
	var scriptHandle ScriptHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		scriptHandle = UseScript(ctx, "(element, transport) => {}")
		scriptHandle.On("complex", func(data interface{}) {
			receivedData = data
		})
		node := &dom.StructuredNode{Tag: "div"}
		scriptHandle.AttachTo(node)
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	complexData := map[string]any{
		"string":  "test",
		"number":  42,
		"boolean": true,
		"null":    nil,
		"array":   []any{1, 2, 3},
		"nested": map[string]any{
			"key":  "value",
			"deep": map[string]any{"level": 3},
		},
	}

	sess.HandleScriptMessage(scriptHandle.slot.id, "complex", complexData)

	dataMap := receivedData.(map[string]any)
	if dataMap["string"] != "test" {
		t.Errorf("expected string='test', got %v", dataMap["string"])
	}
	if dataMap["number"] != 42 {
		t.Errorf("expected number=42, got %v", dataMap["number"])
	}
	if dataMap["boolean"] != true {
		t.Errorf("expected boolean=true, got %v", dataMap["boolean"])
	}
	if dataMap["null"] != nil {
		t.Errorf("expected null=nil, got %v", dataMap["null"])
	}
}

func TestUseScriptPersistenceAcrossRenders(t *testing.T) {
	var scriptHandle ScriptHandle
	var triggerRender func()
	messageCount := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		count, setCount := UseState(ctx, 0)
		triggerRender = func() { setCount(count() + 1) }

		scriptHandle = UseScript(ctx, "(element, transport) => {}")
		scriptHandle.On("persist", func(data interface{}) {
			messageCount++
		})
		node := &dom.StructuredNode{Tag: "div"}
		scriptHandle.AttachTo(node)
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	firstSlotID := scriptHandle.slot.id

	triggerRender()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	secondSlotID := scriptHandle.slot.id
	if firstSlotID != secondSlotID {
		t.Error("expected script slot to persist across renders")
	}

	sess.HandleScriptMessage(scriptHandle.slot.id, "persist", map[string]any{"test": 1})
	if messageCount != 1 {
		t.Errorf("expected 1 message, got %d", messageCount)
	}
}
