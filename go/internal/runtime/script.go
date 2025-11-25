package runtime

import (
	"fmt"
	"strings"
	"sync"

	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// ScriptHandle exposes lifecycle controls for the script hook.
type ScriptHandle struct {
	slot *scriptSlot
}

// ID returns the unique identifier for this script handle.
func (h ScriptHandle) ID() string {
	if h.slot != nil {
		return h.slot.id
	}
	return ""
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

// AttachTo attaches the script to a work element.
func (h ScriptHandle) AttachTo(elem *work.Element) {
	if h.slot == nil || elem == nil {
		return
	}

	h.slot.attachTo(elem)
}

type scriptCell struct {
	slot *scriptSlot
}

// scriptSlot tracks a single script instance.
type scriptSlot struct {
	id        string
	sess      *Session
	instance  *Instance
	hookIndex int

	script   string
	scriptMu sync.RWMutex

	subs              map[string]*Subscription // event -> bus subscription
	cleanupRegistered bool                     // ensure cleanup is added once
}

func (slot *scriptSlot) attachTo(elem *work.Element) {
	if slot == nil || elem == nil || slot.id == "" {
		return
	}

	slot.scriptMu.RLock()
	script := slot.script
	slot.scriptMu.RUnlock()

	elem.Script = &metadata.ScriptMeta{
		ScriptID: slot.id,
		Script:   script,
	}
}

func (slot *scriptSlot) updateScript(script string) {
	if slot == nil {
		return
	}

	minified := minifyJS(script)

	slot.scriptMu.Lock()

	if slot.script != minified {
		slot.script = minified
	}
	slot.scriptMu.Unlock()
}

func (slot *scriptSlot) setEventHandler(event string, fn func(interface{})) {
	if slot == nil || slot.sess == nil || slot.instance == nil || event == "" || fn == nil {
		return
	}

	if slot.sess.Bus == nil {
		slot.sess.Bus = NewBus()
	}

	channelID := fmt.Sprintf("%s:%s", slot.id, event)

	if slot.subs == nil {
		slot.subs = make(map[string]*Subscription)
	}
	slot.subs[event] = slot.sess.Bus.Upsert(channelID, func(eventName string, data interface{}) {
		fn(data)
	})

	if !slot.cleanupRegistered {
		slot.cleanupRegistered = true
		slot.instance.RegisterCleanup(func() {
			for _, sub := range slot.subs {
				if sub != nil {
					sub.Unsubscribe()
				}
			}
		})
	}
}

func (slot *scriptSlot) send(event string, data interface{}) {
	if slot == nil || slot.sess == nil {
		return
	}

	slot.sess.scriptSenderMu.RLock()
	sender := slot.sess.scriptEventSender
	slot.sess.scriptSenderMu.RUnlock()

	if sender != nil {
		if err := sender(slot.id, event, data); err != nil {

			_ = err
		}
	}
}

func (slot *scriptSlot) handleMessage(event string, data interface{}) {
	if slot == nil || slot.sess == nil {
		return
	}

	channelID := fmt.Sprintf("%s:%s", slot.id, event)
	if slot.sess.Bus != nil {
		slot.sess.Bus.Publish(channelID, event, data)
	}
}

// UseScript registers a script slot for the current component.
// Returns a handle for bidirectional communication with client-side JavaScript.
func UseScript(ctx *Ctx, script string) ScriptHandle {
	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx >= len(ctx.instance.HookFrame) {
		cell := &scriptCell{}
		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeScript,
			Value: cell,
		})
	}

	cell, ok := ctx.instance.HookFrame[idx].Value.(*scriptCell)
	if !ok {
		panic("runtime: UseScript hook mismatch")
	}

	if cell.slot == nil {
		cell.slot = ctx.session.registerScriptSlot(ctx.instance, idx, script)

		slotID := cell.slot.id
		ctx.instance.RegisterCleanup(func() {
			ctx.session.unregisterScriptSlot(slotID)
		})
	} else {
		cell.slot.updateScript(script)
	}

	return ScriptHandle{slot: cell.slot}
}

func (s *Session) registerScriptSlot(inst *Instance, index int, script string) *scriptSlot {
	if s == nil || inst == nil {
		return nil
	}

	s.scriptsMu.Lock()
	defer s.scriptsMu.Unlock()

	id := fmt.Sprintf("%s:s%d", inst.ID, index)

	slot := &scriptSlot{
		id:        id,
		sess:      s,
		instance:  inst,
		hookIndex: index,
		script:    minifyJS(script),
	}

	if s.Scripts == nil {
		s.Scripts = make(map[string]*scriptSlot)
	}
	s.Scripts[id] = slot

	return slot
}

func (s *Session) findScriptSlot(id string) *scriptSlot {
	if s == nil || id == "" {
		return nil
	}

	s.scriptsMu.RLock()
	slot := s.Scripts[id]
	s.scriptsMu.RUnlock()
	return slot
}

func (s *Session) unregisterScriptSlot(id string) {
	if s == nil || id == "" {
		return
	}

	s.scriptsMu.Lock()
	delete(s.Scripts, id)
	s.scriptsMu.Unlock()

}

// HandleScriptMessage handles a message from the client script.
// Recovers from panics in user callbacks to prevent session crashes.
func (s *Session) HandleScriptMessage(id string, event string, data interface{}) {
	slot := s.findScriptSlot(id)
	if slot == nil {
		return
	}

	phase := fmt.Sprintf("script:message:%s:%s", id, event)
	_ = s.withRecovery(phase, func() error {
		slot.handleMessage(event, data)
		return nil
	})
}

// SetScriptEventSender installs the callback for sending events to client scripts.
func (s *Session) SetScriptEventSender(sender func(scriptID, event string, data interface{}) error) {
	if s == nil {
		return
	}
	s.scriptSenderMu.Lock()
	s.scriptEventSender = sender
	s.scriptSenderMu.Unlock()
}

// minifyJS performs basic JavaScript minification by removing unnecessary whitespace.
// It preserves spaces where syntactically required (between keywords, identifiers).
// Consecutive whitespace (\s+) is reduced to a single space (\s) when needed.
func minifyJS(script string) string {
	var result strings.Builder
	result.Grow(len(script))

	inString := false
	stringChar := byte(0)
	prevChar := byte(0)

	for i := 0; i < len(script); i++ {
		ch := script[i]

		if ch == '"' || ch == '\'' || ch == '`' {
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
			wsStart := i
			for i+1 < len(script) && isWhitespace(script[i+1]) {
				i++
			}

			if wsStart > 0 && i < len(script)-1 {
				prev := script[wsStart-1]
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

// isWhitespace returns true if the character is whitespace.
func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

// isIdentifierChar returns true if the character can be part of a JS identifier or keyword.
func isIdentifierChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '$'
}
