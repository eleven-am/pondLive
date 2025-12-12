package runtime

import (
	"fmt"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
)

type ErrorCode string

const (
	ErrCodeRender        ErrorCode = "RENDER_PANIC"
	ErrCodeMemo          ErrorCode = "MEMO_PANIC"
	ErrCodeEffect        ErrorCode = "EFFECT_PANIC"
	ErrCodeEffectCleanup ErrorCode = "EFFECT_CLEANUP_PANIC"
	ErrCodeHandler       ErrorCode = "HANDLER_PANIC"
	ErrCodeValidation    ErrorCode = "VALIDATION_ERROR"
	ErrCodeApp           ErrorCode = "APP_ERROR"
	ErrCodeSession       ErrorCode = "SESSION_ERROR"
	ErrCodeNetwork       ErrorCode = "NETWORK_ERROR"
	ErrCodeTimeout       ErrorCode = "TIMEOUT_ERROR"
)

type PondError interface {
	error
	Code() ErrorCode
	Stack() string
	Metadata() map[string]any
	Unwrap() error
}

type Error struct {
	ErrorCode   ErrorCode
	Message     string
	StackTrace  string
	ComponentID string
	Phase       string
	HookIndex   int
	Timestamp   time.Time
	Meta        map[string]any
	Cause       error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *Error) Code() ErrorCode {
	if e == nil {
		return ""
	}
	return e.ErrorCode
}

func (e *Error) Stack() string {
	if e == nil {
		return ""
	}
	return e.StackTrace
}

func (e *Error) Metadata() map[string]any {
	if e == nil {
		return nil
	}
	return e.Meta
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *Error) ParsedStack() []StackFrame {
	if e == nil {
		return nil
	}
	return ParseStack(e.StackTrace)
}

func (e *Error) UserFrames() []StackFrame {
	if e == nil {
		return nil
	}
	return filterFrames(ParseStack(e.StackTrace), FrameUser)
}

func NewError(code ErrorCode, message string) *Error {
	return &Error{
		ErrorCode: code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

func NewErrorWithStack(code ErrorCode, message, stack string) *Error {
	return &Error{
		ErrorCode:  code,
		Message:    message,
		StackTrace: stack,
		Timestamp:  time.Now(),
	}
}

func NewComponentError(code ErrorCode, message, stack, componentID, phase string, hookIndex int) *Error {
	return &Error{
		ErrorCode:   code,
		Message:     message,
		StackTrace:  stack,
		ComponentID: componentID,
		Phase:       phase,
		HookIndex:   hookIndex,
		Timestamp:   time.Now(),
	}
}

var _ PondError = (*Error)(nil)

func (e *Error) ToServerError(sessionID string) *protocol.ServerError {
	if e == nil {
		return nil
	}
	return &protocol.ServerError{
		T:          "error",
		SID:        sessionID,
		Code:       string(e.ErrorCode),
		Message:    e.Message,
		StackTrace: e.StackTrace,
		Meta:       e.Meta,
	}
}

type ErrorContext struct {
	SessionID         string
	ComponentID       string
	ComponentName     string
	ParentID          string
	ComponentPath     []string
	ComponentNamePath []string
	Phase             string
	HookIndex         int
	HookCount         int
	Props             any
	ProviderKeys      []string
	DevMode           bool
}

func NewComponentErrorWithContext(code ErrorCode, message, stack string, ectx ErrorContext) *Error {
	return &Error{
		ErrorCode:   code,
		Message:     message,
		StackTrace:  stack,
		ComponentID: ectx.ComponentID,
		Phase:       ectx.Phase,
		HookIndex:   ectx.HookIndex,
		Timestamp:   time.Now(),
		Meta: map[string]any{
			"session_id":          ectx.SessionID,
			"component_name":      ectx.ComponentName,
			"parent_id":           ectx.ParentID,
			"component_path":      ectx.ComponentPath,
			"component_name_path": ectx.ComponentNamePath,
			"hook_count":          ectx.HookCount,
			"props":               ectx.Props,
			"provider_keys":       ectx.ProviderKeys,
			"dev_mode":            ectx.DevMode,
		},
	}
}

type ErrorBatch struct {
	errors []*Error
}

func NewErrorBatch(errors ...*Error) *ErrorBatch {
	filtered := make([]*Error, 0, len(errors))
	for _, e := range errors {
		if e != nil {
			filtered = append(filtered, e)
		}
	}
	return &ErrorBatch{errors: filtered}
}

func (b *ErrorBatch) HasErrors() bool {
	return b != nil && len(b.errors) > 0
}

func (b *ErrorBatch) Count() int {
	if b == nil {
		return 0
	}
	return len(b.errors)
}

func (b *ErrorBatch) First() *Error {
	if b == nil || len(b.errors) == 0 {
		return nil
	}
	return b.errors[0]
}

func (b *ErrorBatch) All() []*Error {
	if b == nil {
		return nil
	}
	return b.errors
}

func (b *ErrorBatch) Error() string {
	if b == nil || len(b.errors) == 0 {
		return ""
	}
	if len(b.errors) == 1 {
		return b.errors[0].Message
	}
	return fmt.Sprintf("%s (and %d more errors)", b.errors[0].Message, len(b.errors)-1)
}

func (b *ErrorBatch) ByPhase(phase string) []*Error {
	if b == nil {
		return nil
	}
	var result []*Error
	for _, e := range b.errors {
		if e.Phase == phase {
			result = append(result, e)
		}
	}
	return result
}

func (b *ErrorBatch) ByCode(code ErrorCode) []*Error {
	if b == nil {
		return nil
	}
	var result []*Error
	for _, e := range b.errors {
		if e.ErrorCode == code {
			result = append(result, e)
		}
	}
	return result
}

func (b *ErrorBatch) ToServerErrors(sessionID string) []*protocol.ServerError {
	if b == nil || len(b.errors) == 0 {
		return nil
	}
	result := make([]*protocol.ServerError, len(b.errors))
	for i, e := range b.errors {
		result[i] = e.ToServerError(sessionID)
	}
	return result
}
