package runtime

import (
	"fmt"

	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

const (
	WarnAttachIncompatible  WarningCode = "REF_INCOMPATIBLE_TAG"
	WarnAttachedTwiceReport WarningCode = "REF_ATTACHED_TWICE"
)

type attachment struct {
	s         *Session
	o         *reactive.Owner
	name      string
	id        uint32
	accepts   func(tag string) bool
	writeAttr func(w *writer)
	onBind    func(el *work.Element)
	onUnbind  func()
}

type attachable interface{ base() *attachment }

func (a *attachment) base() *attachment { return a }

func (a *attachment) IsAttached() bool { return a != nil && a.id != 0 }

func (a *attachment) ID() uint32 {
	if a == nil {
		return 0
	}
	return a.id
}

type AttachedTwiceError struct{ Name string }

func (e AttachedTwiceError) Error() string {
	return "runtime: " + e.Name + " is attached to two live elements"
}

type IncompatibleTagError struct{ Name, Tag string }

func (e IncompatibleTagError) Error() string {
	return fmt.Sprintf("runtime: %s cannot be attached to <%s>", e.Name, e.Tag)
}

func AttachItem(target any) work.Item {
	var a attachable
	switch t := target.(type) {
	case *Ref:
		if t != nil {
			a = t
		}
	case ScriptHandle:
		if t.slot != nil {
			a = t.slot
		}
	case *UploadHandle:
		if t != nil && t.script.slot != nil {
			a = t.script.slot
		}
	}
	if a == nil {
		return work.Skip()
	}
	return work.ItemFunc(func(el *work.Element) { attachTo(el, a) })
}

func attachTo(el *work.Element, a attachable) {
	if el == nil || a == nil {
		return
	}
	el.Attachments = append(el.Attachments, a)
}

func writeAttachmentAttrs(w *writer, el *work.Element) {
	for _, item := range el.Attachments {
		if a, ok := item.(attachable); ok && a.base().writeAttr != nil {
			a.base().writeAttr(w)
		}
	}
}

func (s *Session) bindAttachments(o *reactive.Owner, el *work.Element, id uint32) {
	for _, item := range el.Attachments {
		if a, ok := item.(attachable); ok {
			s.bindAttachment(o, a.base(), el, id)
		}
	}
}

func (s *Session) bindAttachment(o *reactive.Owner, a *attachment, el *work.Element, id uint32) {
	if a.accepts != nil && !a.accepts(el.Tag) {
		s.attachProblem(a.o, WarnAttachIncompatible, IncompatibleTagError{Name: a.name, Tag: el.Tag})
		return
	}
	if a.id != 0 {
		s.attachProblem(a.o, WarnAttachedTwiceReport, AttachedTwiceError{Name: a.name})
		return
	}
	a.id = id
	if a.onBind != nil {
		a.onBind(el)
	}
	o.OnCleanup(func() {
		if a.id != id {
			return
		}
		if a.onUnbind != nil {
			a.onUnbind()
		}
		a.id = 0
	})
}

func (s *Session) attachProblem(o *reactive.Owner, code WarningCode, err error) {
	if s.devMode {
		panic(err)
	}
	s.Report(o, Report{Severity: SeverityWarning, Code: string(code), Message: err.Error()})
}
