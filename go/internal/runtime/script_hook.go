package runtime

import (
	"fmt"
	"strings"
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

// minifyJS performs basic JavaScript minification by removing unnecessary whitespace.
// It preserves spaces where syntactically required (between keywords, identifiers).
func minifyJS(script string) string {
	var result strings.Builder
	result.Grow(len(script))

	inString := false
	stringChar := byte(0)
	inRegex := false
	prevChar := byte(0)

	for i := 0; i < len(script); i++ {
		ch := script[i]

		if !inRegex && (ch == '"' || ch == '\'' || ch == '`') {
			if !inString {
				inString = true
				stringChar = ch
			} else if ch == stringChar && prevChar != '\\' {
				inString = false
				stringChar = 0
			}
			result.WriteByte(ch)
			prevChar = ch
			continue
		}

		if inString {
			result.WriteByte(ch)
			prevChar = ch
			continue
		}

		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {

			if i > 0 && i < len(script)-1 {
				prev := script[i-1]
				next := script[i+1]
				if isIdentifierChar(prev) && isIdentifierChar(next) {
					result.WriteByte(' ')
				}
			}
			prevChar = ch
			continue
		}

		result.WriteByte(ch)
		prevChar = ch
	}

	return result.String()
}

// isIdentifierChar returns true if the character can be part of a JS identifier or keyword.
func isIdentifierChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '$'
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
		script:        minifyJS(script),
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
	slot.script = minifyJS(script)
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
