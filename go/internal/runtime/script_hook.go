package runtime

import (
	"fmt"
	"sync"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

// ScriptHandle exposes lifecycle controls for the script hook.
type ScriptHandle struct {
	slot *scriptSlot
}

// AttachTo implements the Attachment interface, allowing ScriptHandle to be used with h.Attach.
func (h ScriptHandle) AttachTo(node *dom.StructuredNode) {
	if h.slot == nil || node == nil {
		return
	}
	h.slot.registerBinding(node)
}

// OnMessage registers a callback invoked when the client sends a message via transport.send.
func (h ScriptHandle) OnMessage(fn func(map[string]any)) {
	if h.slot != nil {
		h.slot.setOnMessage(fn)
	}
}

// Send sends an event to the client script via transport.on.
func (h ScriptHandle) Send(event string, data map[string]any) {
	if h.slot != nil {
		h.slot.send(event, data)
	}
}

// UseScript registers a script slot for the current component.
func UseScript(ctx Ctx, script string) ScriptHandle {
	if ctx.frame == nil {
		panic("runtime2: UseScript called outside render")
	}

	idx := ctx.frame.idx
	ctx.frame.idx++

	if idx >= len(ctx.frame.cells) {
		cell := &scriptCell{}
		ctx.frame.cells = append(ctx.frame.cells, cell)
	}

	raw := ctx.frame.cells[idx]
	cell, ok := raw.(*scriptCell)
	if !ok {
		panicHookMismatch(ctx.comp, idx, "UseScript", raw)
	}

	if ctx.sess != nil {
		if cell.slot == nil || cell.slot.sess == nil {
			cell.slot = ctx.sess.registerScriptSlot(ctx.comp, idx, script)
		} else {
			cell.slot.updateScript(script)
		}
	}

	return ScriptHandle{slot: cell.slot}
}

type scriptCell struct {
	slot *scriptSlot
}

// scriptSlot tracks a single script instance.
type scriptSlot struct {
	id        string
	sess      *ComponentSession
	component *component
	hookIndex int

	script    string
	scriptMu  sync.RWMutex
	onMessage func(map[string]any)

	eventQueue []scriptEvent
	eventMu    sync.Mutex
}

type scriptEvent struct {
	scriptID string
	event    string
	data     map[string]any
}

func (slot *scriptSlot) registerBinding(node *dom.StructuredNode) {
	if slot == nil || node == nil || slot.id == "" {
		return
	}

	slot.scriptMu.RLock()
	script := slot.script
	slot.scriptMu.RUnlock()

	node.Script = &dom.ScriptMeta{
		ScriptID: slot.id,
		Script:   script,
	}
}

func (s *ComponentSession) registerScriptSlot(comp *component, index int, script string) *scriptSlot {
	if s == nil || comp == nil {
		return nil
	}

	s.scriptMu.Lock()
	defer s.scriptMu.Unlock()

	id := fmt.Sprintf("%s:s%d", comp.id, index)

	slot := &scriptSlot{
		id:        id,
		sess:      s,
		component: comp,
		hookIndex: index,
		script:    script,
	}

	s.scripts[id] = slot
	return slot
}

func (s *ComponentSession) findScriptSlot(id string) *scriptSlot {
	if s == nil || id == "" {
		return nil
	}
	s.scriptMu.Lock()
	slot := s.scripts[id]
	s.scriptMu.Unlock()
	return slot
}

func (slot *scriptSlot) updateScript(script string) {
	slot.scriptMu.Lock()
	slot.script = script
	slot.scriptMu.Unlock()
}

func (slot *scriptSlot) setOnMessage(fn func(map[string]any)) {
	slot.scriptMu.Lock()
	slot.onMessage = fn
	slot.scriptMu.Unlock()
}

func (slot *scriptSlot) send(event string, data map[string]any) {
	if slot == nil {
		return
	}

	slot.eventMu.Lock()
	slot.eventQueue = append(slot.eventQueue, scriptEvent{
		scriptID: slot.id,
		event:    event,
		data:     data,
	})
	slot.eventMu.Unlock()

	if slot.sess != nil && slot.component != nil {
		slot.sess.markDirty(slot.component)
	}
}

func (slot *scriptSlot) handleMessage(data map[string]any) {
	if slot == nil {
		return
	}

	slot.scriptMu.RLock()
	handler := slot.onMessage
	slot.scriptMu.RUnlock()

	if handler != nil {
		handler(data)
	}
}

// ComponentSession script event handlers

func (s *ComponentSession) HandleScriptMessage(id string, data map[string]any) {
	slot := s.findScriptSlot(id)
	if slot == nil {
		return
	}

	s.withRecovery("script:message", func() error {
		slot.handleMessage(data)
		return nil
	})
}

func (s *ComponentSession) CollectScriptEvents() []scriptEvent {
	if s == nil {
		return nil
	}

	s.scriptMu.Lock()
	defer s.scriptMu.Unlock()

	var events []scriptEvent
	for _, slot := range s.scripts {
		slot.eventMu.Lock()
		if len(slot.eventQueue) > 0 {
			events = append(events, slot.eventQueue...)
			slot.eventQueue = nil
		}
		slot.eventMu.Unlock()
	}

	return events
}
