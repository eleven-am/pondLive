package runtime

import (
	"strconv"
	"strings"

	"github.com/eleven-am/pondlive/internal/effectguard"
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
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

type scriptSlot struct {
	attachment
	id     string
	sess   *Session
	o      *reactive.Owner
	script string
	subs   map[string]*protocol.Subscription
}

func (slot *scriptSlot) setEventHandler(event string, fn func(interface{})) {
	if slot == nil || slot.sess == nil || slot.o == nil || event == "" || fn == nil {
		return
	}
	if slot.sess.graph.Mode() == reactive.ModeSerialize {
		return
	}
	if slot.subs == nil {
		slot.subs = make(map[string]*protocol.Subscription)
		slot.o.OnCleanup(func() {
			for _, sub := range slot.subs {
				if sub != nil {
					sub.Unsubscribe()
				}
			}
			slot.subs = nil
		})
	}
	if old := slot.subs[event]; old != nil {
		old.Unsubscribe()
	}
	slot.subs[event] = slot.sess.Bus.SubscribeToScriptMessages(slot.id, event, func(data interface{}) {
		slot.sess.invokeHandler(slot.o, func() { fn(data) })
	})
}

func (slot *scriptSlot) send(event string, data interface{}) {
	if effectguard.Refuse("Script.Send") {
		return
	}
	if slot == nil || slot.sess == nil || slot.sess.Bus == nil {
		return
	}
	slot.sess.Bus.PublishScriptSend(slot.id, event, data)
}

func UseScript(ctx *Ctx, script string) ScriptHandle {
	slot := reactive.Cell(ctx.o, func() *scriptSlot {
		return ctx.s.newScriptSlot(ctx.o, script)
	})
	return ScriptHandle{slot: slot}
}

func (s *Session) newScriptSlot(o *reactive.Owner, script string) *scriptSlot {
	s.scriptSeq++
	slot := &scriptSlot{
		id:     "s" + strconv.FormatUint(uint64(s.scriptSeq), 10),
		sess:   s,
		o:      o,
		script: minifyJS(script),
	}
	slot.attachment = attachment{s: s, o: o, name: "script " + slot.id}
	meta := &metadata.ScriptMeta{ScriptID: slot.id, Script: slot.script}
	slot.writeAttr = func(w *writer) { w.attr(protocol.AttrScript, scriptSpecJSON(meta)) }
	if s.scripts == nil {
		s.scripts = make(map[string]*scriptSlot)
	}
	s.scripts[slot.id] = slot
	o.OnCleanup(func() { delete(s.scripts, slot.id) })
	return slot
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
