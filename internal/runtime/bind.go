package runtime

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"

	twmerge "github.com/Oudwins/tailwind-merge-go/pkg/twmerge"
	"github.com/eleven-am/pondlive/internal/htmlutil"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

type bindKind uint8

const (
	bindText bindKind = iota
	bindAttr
	bindAttrOptional
	bindPresence
	bindClass
	bindStyle
	bindRaw
)

type binding struct {
	s      *Session
	o      *reactive.Owner
	eff    *reactive.Effect
	kind   bindKind
	id     uint32
	name   string
	src    reactive.Readable[string]
	pres   reactive.Readable[bool]
	parts  []work.ClassPart
	tag    string
	first  bool
	failed bool
	on     bool
	init   string
}

func (b *binding) compute() string {
	switch b.kind {
	case bindClass:
		return joinClass(b.parts, true)
	case bindRaw:
		return normalizeRaw(b.tag, b.src.Get())
	default:
		return b.src.Get()
	}
}

func (b *binding) run() {
	defer func() {
		if r := recover(); r != nil {
			b.failed = true
			b.s.bindingPanic(b.o, r)
			if b.eff != nil {
				b.eff.Dispose()
			}
		}
	}()
	if b.kind == bindPresence {
		on := b.pres.Get()
		if b.first {
			b.on, b.first = on, false
			return
		}
		if b.s.isEcho(b.id, b.name, strconv.FormatBool(on)) {
			return
		}
		if on {
			b.s.edit(protocol.AttrEdit(b.id, b.name, ""))
		} else {
			b.s.edit(protocol.RemoveAttrEdit(b.id, b.name))
		}
		return
	}
	v := b.compute()
	if b.first {
		b.init, b.first = v, false
		return
	}
	switch b.kind {
	case bindText:
		b.s.edit(protocol.TextEdit(b.id, v))
	case bindAttr:
		if b.s.isEcho(b.id, b.name, v) {
			return
		}
		b.s.edit(protocol.AttrEdit(b.id, b.name, v))
	case bindAttrOptional, bindClass:
		if v == "" {
			b.s.edit(protocol.RemoveAttrEdit(b.id, b.name))
			return
		}
		b.s.edit(protocol.AttrEdit(b.id, b.name, v))
	case bindStyle:
		b.s.edit(protocol.StyleEdit(b.id, b.name, v))
	case bindRaw:
		b.s.edit(protocol.HTMLEdit(b.id, v))
	}
}

func (s *Session) bindingPanic(o *reactive.Owner, r any) {
	s.panicToBoundary(o, r, false)
}

func (s *Session) panicToBoundary(o *reactive.Owner, r any, late bool) {
	stack := string(debug.Stack())
	phase := "binding"
	err := s.buildError(ErrCodeRender, fmt.Sprintf("%v", r), o, phase, stack, r)
	s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: phase,
		Message: fmt.Sprintf("panic: %v", r),
		Stack:   stack,
		Meta:    map[string]any{"component": o.Info.Name, "panic_value": r},
	})
	if late {
		s.late = append(s.late, func() { s.deliver(o, err) })
		return
	}
	s.deliver(o, err)
}

func (s *Session) startBinding(o *reactive.Owner, b *binding) *binding {
	b.s, b.o, b.first = s, o, true
	b.eff = reactive.NewEffect(o, reactive.KindRender, b.run)
	if b.failed {
		b.eff.Dispose()
	}
	return b
}

func peekString(src reactive.Readable[string]) (v string) {
	defer func() {
		if r := recover(); r != nil {
			v = ""
		}
	}()
	return src.Peek()
}

func peekBool(src reactive.Readable[bool]) (v bool) {
	defer func() {
		if r := recover(); r != nil {
			v = false
		}
	}()
	return src.Peek()
}

func joinClass(parts []work.ClassPart, live bool) string {
	var sb strings.Builder
	for _, p := range parts {
		token := p.Static
		if p.Source != nil {
			if live {
				token = p.Source.Get()
			} else {
				token = peekString(p.Source)
			}
		}
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(token)
	}
	joined := sb.String()
	if strings.IndexAny(joined, " \t\n\r") < 0 {
		return joined
	}
	return htmlutil.UniqueClasses(twmerge.Merge(joined))
}
