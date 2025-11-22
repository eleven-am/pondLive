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

// On registers a callback invoked when the client sends a named event via transport.send.
func (h ScriptHandle) On(event string, fn func(interface{})) {
	if h.slot != nil {
		h.slot.setEventHandler(event, fn)
	}
}

// Send sends an event to the client script via transport.on.
func (h ScriptHandle) Send(event string, data interface{}) {
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

	script        string
	scriptMu      sync.RWMutex
	eventHandlers map[string]func(interface{})
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
		id:            id,
		sess:          s,
		component:     comp,
		hookIndex:     index,
		script:        script,
		eventHandlers: make(map[string]func(interface{})),
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

func (slot *scriptSlot) setEventHandler(event string, fn func(interface{})) {
	slot.scriptMu.Lock()
	slot.eventHandlers[event] = fn
	slot.scriptMu.Unlock()
}

func (slot *scriptSlot) send(event string, data interface{}) {
	if slot == nil || slot.sess == nil {
		return
	}

	slot.sess.mu.Lock()
	sender := slot.sess.scriptEventSender
	slot.sess.mu.Unlock()

	if sender != nil {
		_ = sender(slot.id, event, data)
	}
}

func (slot *scriptSlot) handleMessage(event string, data interface{}) {
	if slot == nil {
		return
	}

	slot.scriptMu.RLock()
	handler := slot.eventHandlers[event]
	slot.scriptMu.RUnlock()

	if handler != nil {
		handler(data)
	}
}

func (s *ComponentSession) HandleScriptMessage(id string, event string, data interface{}) {
	slot := s.findScriptSlot(id)
	if slot == nil {
		return
	}

	s.withRecovery("script:message", func() error {
		slot.handleMessage(event, data)
		return nil
	})
}
