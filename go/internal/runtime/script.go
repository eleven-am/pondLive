package runtime

import (
	"fmt"
	"strings"
	"sync"

	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type ScriptHandle struct {
	slot *scriptSlot
}

func (h ScriptHandle) ID() string {
	if h.slot != nil {
		return h.slot.id
	}
	return ""
}

func (h ScriptHandle) RefID() string {
	return h.ID()
}

func (h ScriptHandle) On(event string, fn func(interface{})) {
	if h.slot != nil {
		h.slot.setEventHandler(event, fn)
	}
}

func (h ScriptHandle) Send(event string, data interface{}) {
	if h.slot != nil {
		h.slot.send(event, data)
	}
}

func (h ScriptHandle) AttachTo(elem *work.Element) {
	if h.slot == nil || elem == nil {
		return
	}

	h.slot.attachTo(elem)
}

type scriptCell struct {
	slot *scriptSlot
}

type scriptSlot struct {
	id        string
	sess      *Session
	instance  *Instance
	hookIndex int

	script   string
	scriptMu sync.RWMutex

	subs              map[string]*protocol.Subscription
	cleanupRegistered bool
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
		slot.sess.Bus = protocol.NewBus()
	}

	if slot.subs == nil {
		slot.subs = make(map[string]*protocol.Subscription)
	}

	if oldSub := slot.subs[event]; oldSub != nil {
		oldSub.Unsubscribe()
	}

	slot.subs[event] = slot.sess.Bus.SubscribeToScriptMessages(slot.id, func(evt string, data interface{}) {
		if evt == event {
			fn(data)
		}
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

	if slot.sess.Bus == nil {
		return
	}

	slot.sess.Bus.PublishScriptSend(slot.id, event, data)
}

func (slot *scriptSlot) handleMessage(event string, data interface{}) {
	if slot == nil || slot.sess == nil {
		return
	}

	if slot.sess.Bus != nil {
		slot.sess.Bus.PublishScriptMessage(slot.id, event, data)
	}
}

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

func minifyJS(script string) string {
	var result strings.Builder
	result.Grow(len(script))

	n := len(script)
	lastTokenCanPrecedeDivision := false

	for i := 0; i < n; i++ {
		ch := script[i]

		if ch == '"' || ch == '\'' || ch == '`' {
			start := i
			i = skipString(script, i)
			result.WriteString(script[start : i+1])
			lastTokenCanPrecedeDivision = true
			continue
		}

		if ch == '/' && i+1 < n {
			next := script[i+1]

			if next == '/' {
				for i < n && script[i] != '\n' {
					i++
				}
				i--
				continue
			}

			if next == '*' {
				i += 2
				for i+1 < n {
					if script[i] == '*' && script[i+1] == '/' {
						i++
						break
					}
					i++
				}
				continue
			}

			if !lastTokenCanPrecedeDivision {
				start := i
				i = skipRegex(script, i)
				result.WriteString(script[start : i+1])
				lastTokenCanPrecedeDivision = true
				continue
			}
		}

		if isWhitespace(ch) {
			for i+1 < n && isWhitespace(script[i+1]) {
				i++
			}

			if result.Len() > 0 && i+1 < n {
				prevByte := result.String()[result.Len()-1]
				nextByte := script[i+1]
				if isIdentifierChar(prevByte) && isIdentifierChar(nextByte) {
					result.WriteByte(' ')
				}
			}
			continue
		}

		result.WriteByte(ch)
		lastTokenCanPrecedeDivision = canPrecedeDivision(ch)
	}

	return result.String()
}

func skipString(s string, i int) int {
	quote := s[i]
	n := len(s)
	i++

	for i < n {
		ch := s[i]

		if ch == '\\' && i+1 < n {
			i += 2
			continue
		}

		if ch == quote {
			return i
		}

		if quote == '`' && ch == '$' && i+1 < n && s[i+1] == '{' {
			i += 2
			braceDepth := 1
			for i < n && braceDepth > 0 {
				c := s[i]
				if c == '{' {
					braceDepth++
				} else if c == '}' {
					braceDepth--
				} else if c == '"' || c == '\'' || c == '`' {
					i = skipString(s, i)
				}
				i++
			}
			continue
		}

		i++
	}

	return n - 1
}

func skipRegex(s string, i int) int {
	n := len(s)
	i++

	for i < n {
		ch := s[i]

		if ch == '\\' && i+1 < n {
			i += 2
			continue
		}

		if ch == '[' {
			i++
			for i < n && s[i] != ']' {
				if s[i] == '\\' && i+1 < n {
					i += 2
				} else {
					i++
				}
			}
			i++
			continue
		}

		if ch == '/' {
			i++
			for i < n && isRegexFlag(s[i]) {
				i++
			}
			return i - 1
		}

		i++
	}

	return n - 1
}

func isRegexFlag(ch byte) bool {
	return ch == 'g' || ch == 'i' || ch == 'm' || ch == 's' || ch == 'u' || ch == 'y' || ch == 'd' || ch == 'v'
}

func canPrecedeDivision(ch byte) bool {
	if ch == ')' || ch == ']' {
		return true
	}
	if isIdentifierChar(ch) {
		return true
	}
	return false
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isIdentifierChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '$'
}
