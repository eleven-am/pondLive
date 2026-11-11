package runtime

import (
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/eleven-am/pondlive/internal/reactive"
)

type boundaryKey struct{}

func (s *Session) hasWork() bool {
	if len(s.edits) > 0 || len(s.late) > 0 || s.graph.Pending(reactive.QueueRender) {
		return true
	}
	return !s.deferred.Load() && s.graph.Pending(reactive.QueueWatch)
}

func (s *Session) Flush() error {
	if s == nil {
		return errors.New("runtime: session not initialized")
	}
	_, err := ask(s, flushAskBound, func(s *Session) (struct{}, error) {
		s.flush()
		if s.closed {
			return struct{}{}, ErrSessionClosed
		}
		return struct{}{}, nil
	})
	return err
}

func (s *Session) FlushCount() uint64 {
	if s == nil {
		return 0
	}
	return s.flushes.Load()
}

func (s *Session) flush() {
	if s.root == nil {
		return
	}
	s.flushes.Add(1)
	for round := 0; round <= reactive.CascadeBudget; round++ {
		err := s.graph.Flush(reactive.FlushHooks{
			Drain:    s.drainSetters,
			Publish:  s.publish,
			Deferred: s.deferred.Load,
		})
		if err != nil {
			s.publish()
			s.failBudget(err)
			return
		}
		if len(s.late) == 0 {
			return
		}
		late := s.late
		s.late = nil
		for _, fn := range late {
			fn()
		}
	}
	s.publish()
}

func (s *Session) publish() {
	s.echo = nil
	if len(s.edits) == 0 {
		return
	}
	out := s.edits
	s.edits = nil
	s.Bus.PublishFrameEdits(out)
}

func (s *Session) failBudget(err error) {
	var budget *reactive.BudgetError
	if !errors.As(err, &budget) {
		return
	}
	if budget.Kind == reactive.BudgetRounds {
		owner := budget.Owner
		if owner == nil {
			owner = s.root
		}
		s.failOwner(owner, ErrCodeRender, "render loop limit exceeded", "flush:render-loop", nil)
		return
	}
	owners := s.graph.Abandon()
	s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "flush:update-depth",
		Message: "maximum update depth exceeded",
		Meta: map[string]any{
			"owners": len(owners),
		},
	})
	for _, o := range owners {
		s.failOwner(o, ErrCodeEffect, "maximum update depth exceeded", "flush:update-depth", nil)
	}
}

func (s *Session) failOwner(o *reactive.Owner, code ErrorCode, message, phase string, panicValue any) {
	if o == nil {
		return
	}
	err := s.buildError(code, message, o, phase, "", panicValue)
	s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: phase,
		Message: message,
		Meta:    map[string]any{"component": o.Info.Name},
	})
	s.deliver(o, err)
}

func (s *Session) onViolation(e *reactive.DevError) {
	if s.devMode {
		panic(e)
	}
	s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "reactive:violation",
		Message: e.Error(),
		Meta: map[string]any{
			"violation": int(e.Violation),
		},
	})
}

func (s *Session) onCleanupPanic(o *reactive.Owner, v any) {
	stack := string(debug.Stack())
	err := s.buildError(ErrCodeEffectCleanup, fmt.Sprintf("%v", v), o, "effect:cleanup", stack, v)
	s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "effect:cleanup",
		Message: fmt.Sprintf("panic: %v", v),
		Stack:   stack,
		Meta:    map[string]any{"component": o.Info.Name, "panic_value": v},
	})
	s.deliver(o, err)
}

func (s *Session) allowCreate(what string) bool {
	if s.graph.Mode() != reactive.ModeSerialize {
		return true
	}
	if s.devMode {
		panic(&reactive.DevError{Violation: reactive.CreateInSerialize})
	}
	s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "serialize:create",
		Message: what + " called during a serialize pass and ignored",
	})
	return false
}

func (s *Session) boundaryOf(o *reactive.Owner) *Boundary {
	if o == nil {
		return nil
	}
	if v, ok := o.Parent().Lookup(boundaryKey{}); ok {
		return v.(*Boundary)
	}
	return nil
}

func (s *Session) deliver(o *reactive.Owner, err *Error) {
	if b := s.boundaryOf(o); b != nil {
		b.push(err)
		return
	}
	if s.crash == nil {
		s.crash = err
	}
}

func (s *Session) recoverUser(code ErrorCode, phase string, o *reactive.Owner, fn func()) (compErr *Error) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			name := ""
			if o != nil {
				name = o.Info.Name
			}
			fullPhase := phase + ":" + name
			compErr = s.buildError(code, fmt.Sprintf("%v", r), o, fullPhase, stack, r)
			s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: fullPhase,
				Message: fmt.Sprintf("panic: %v", r),
				Stack:   stack,
				Meta: map[string]any{
					"component":   name,
					"panic_value": r,
				},
			})
		}
	}()
	fn()
	return nil
}

func (s *Session) ownerPath(o *reactive.Owner) []string {
	var names []string
	for p := o; p != nil; p = p.Parent() {
		names = append(names, p.Info.Name)
	}
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return names
}

func (s *Session) buildError(code ErrorCode, message string, o *reactive.Owner, phase, stack string, panicValue any) *Error {
	ectx := ErrorContext{
		SessionID: s.SessionID,
		Phase:     phase,
		HookIndex: -1,
		DevMode:   s.devMode,
	}
	if o != nil {
		ectx.ComponentID = fmt.Sprintf("%s#%d", o.Info.Name, o.Base)
		ectx.ComponentName = o.Info.Name
		if p := o.Parent(); p != nil {
			ectx.ParentID = fmt.Sprintf("%s#%d", p.Info.Name, p.Base)
		}
		ectx.ComponentNamePath = s.ownerPath(o)
		ectx.ComponentPath = ectx.ComponentNamePath
		ectx.HookCount = o.CellCount()
	}
	err := NewComponentErrorWithContext(code, message, stack, ectx)
	if panicValue != nil {
		err.Meta["panic_value"] = panicValue
	}
	return err
}

func (s *Session) invokeHandler(o *reactive.Owner, fn func()) {
	if o != nil && o.Disposed() {
		return
	}
	s.runAttributed(o, fn)
}

func (s *Session) runAttributed(o *reactive.Owner, fn func()) bool {
	err := s.recoverUser(ErrCodeHandler, "handler", o, fn)
	if err == nil {
		return true
	}
	s.deliver(o, err)
	return false
}
