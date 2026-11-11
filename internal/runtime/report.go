package runtime

import (
	"runtime/debug"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
)

type Severity uint8

const (
	SeverityWarning Severity = iota + 1
	SeverityError
	SeverityDiagnostic
)

type Report struct {
	Severity  Severity
	Phase     string
	Code      string
	Component string
	Message   string
	Stack     string
	Meta      map[string]any
}

func (s *Session) Report(o *reactive.Owner, r Report) {
	if s == nil {
		return
	}
	if r.Severity == SeverityDiagnostic {
		s.diagnose(r.Phase, r.Message, r.Stack, r.Meta)
		return
	}
	if s.graph.Mode() == reactive.ModeSerialize {
		return
	}
	name := r.Component
	if o != nil {
		name = o.Info.Name
	}
	if r.Severity == SeverityError {
		s.reportError(o, name, r)
		return
	}
	s.warn(&Warning{Code: WarningCode(r.Code), Phase: r.Phase, Component: name, Message: r.Message, Stack: r.Stack, Meta: r.Meta})
}

func (s *Session) diagnose(phase, message, stack string, meta map[string]any) {
	s.report(protocol.Diagnostic{Phase: phase, Message: message, StackTrace: stack, Metadata: protocol.DiagMeta(s.SessionID, meta)})
}

func (s *Session) reportError(o *reactive.Owner, name string, r Report) {
	meta := withComponent(name, r.Meta)
	phase := r.Phase
	if phase == "" {
		phase = "error:" + r.Code
	}
	s.diagnose(phase, r.Message, r.Stack, meta)
	if s.devMode {
		s.deliver(o, s.buildError(ErrCodeApp, r.Message, o, phase, r.Stack, nil))
	}
}

type ReportLevel struct{ severity Severity }

var (
	ReportWarning = ReportLevel{SeverityWarning}
	ReportError   = ReportLevel{SeverityError}
)

func (c *Ctx) Report(level ReportLevel, code, message string) {
	ReportOn(c, Report{Severity: level.severity, Code: code, Message: message, Stack: string(debug.Stack())})
}

func ReportOn(c *Ctx, r Report) {
	if c == nil || c.s == nil {
		return
	}
	s, o := c.s, c.o
	if s.graph.OnActor() {
		s.Report(o, r)
		return
	}
	s.RunOnActor(func() { s.Report(o, r) })
}
