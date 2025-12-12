package runtime

import (
	"errors"
	"testing"
)

func TestError_Error(t *testing.T) {
	e := &Error{Message: "test message"}
	if e.Error() != "test message" {
		t.Errorf("expected 'test message', got %q", e.Error())
	}
}

func TestError_Error_Nil(t *testing.T) {
	var e *Error
	if e.Error() != "" {
		t.Errorf("expected empty string for nil error, got %q", e.Error())
	}
}

func TestError_Code(t *testing.T) {
	e := &Error{ErrorCode: ErrCodeRender}
	if e.Code() != ErrCodeRender {
		t.Errorf("expected %v, got %v", ErrCodeRender, e.Code())
	}
}

func TestError_Code_Nil(t *testing.T) {
	var e *Error
	if e.Code() != "" {
		t.Errorf("expected empty string for nil error, got %q", e.Code())
	}
}

func TestError_Stack(t *testing.T) {
	e := &Error{StackTrace: "stack trace here"}
	if e.Stack() != "stack trace here" {
		t.Errorf("expected 'stack trace here', got %q", e.Stack())
	}
}

func TestError_Stack_Nil(t *testing.T) {
	var e *Error
	if e.Stack() != "" {
		t.Errorf("expected empty string for nil error, got %q", e.Stack())
	}
}

func TestError_Metadata(t *testing.T) {
	meta := map[string]any{"key": "value"}
	e := &Error{Meta: meta}
	if e.Metadata()["key"] != "value" {
		t.Errorf("expected 'value', got %v", e.Metadata()["key"])
	}
}

func TestError_Metadata_Nil(t *testing.T) {
	var e *Error
	if e.Metadata() != nil {
		t.Errorf("expected nil for nil error, got %v", e.Metadata())
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("cause error")
	e := &Error{Cause: cause}
	if e.Unwrap() != cause {
		t.Errorf("expected cause error, got %v", e.Unwrap())
	}
}

func TestError_Unwrap_Nil(t *testing.T) {
	var e *Error
	if e.Unwrap() != nil {
		t.Errorf("expected nil for nil error, got %v", e.Unwrap())
	}
}

func TestNewError(t *testing.T) {
	e := NewError(ErrCodeApp, "app error")
	if e.ErrorCode != ErrCodeApp {
		t.Errorf("expected ErrCodeApp, got %v", e.ErrorCode)
	}
	if e.Message != "app error" {
		t.Errorf("expected 'app error', got %q", e.Message)
	}
	if e.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestNewErrorWithStack(t *testing.T) {
	e := NewErrorWithStack(ErrCodeHandler, "handler error", "stack trace")
	if e.ErrorCode != ErrCodeHandler {
		t.Errorf("expected ErrCodeHandler, got %v", e.ErrorCode)
	}
	if e.Message != "handler error" {
		t.Errorf("expected 'handler error', got %q", e.Message)
	}
	if e.StackTrace != "stack trace" {
		t.Errorf("expected 'stack trace', got %q", e.StackTrace)
	}
}

func TestNewComponentError(t *testing.T) {
	e := NewComponentError(ErrCodeRender, "render error", "stack", "comp-1", "render", 2)
	if e.ErrorCode != ErrCodeRender {
		t.Errorf("expected ErrCodeRender, got %v", e.ErrorCode)
	}
	if e.Message != "render error" {
		t.Errorf("expected 'render error', got %q", e.Message)
	}
	if e.StackTrace != "stack" {
		t.Errorf("expected 'stack', got %q", e.StackTrace)
	}
	if e.ComponentID != "comp-1" {
		t.Errorf("expected 'comp-1', got %q", e.ComponentID)
	}
	if e.Phase != "render" {
		t.Errorf("expected 'render', got %q", e.Phase)
	}
	if e.HookIndex != 2 {
		t.Errorf("expected 2, got %d", e.HookIndex)
	}
}

func TestNewComponentErrorWithContext(t *testing.T) {
	ctx := ErrorContext{
		SessionID:         "sess-1",
		ComponentID:       "comp-1",
		ComponentName:     "MyComponent",
		ParentID:          "parent-1",
		ComponentPath:     []string{"Root", "Parent"},
		ComponentNamePath: []string{"App", "Layout"},
		Phase:             "mount",
		HookIndex:         3,
		HookCount:         5,
		Props:             map[string]string{"prop": "value"},
		ProviderKeys:      []string{"key1", "key2"},
		DevMode:           true,
	}

	e := NewComponentErrorWithContext(ErrCodeMemo, "memo error", "stack", ctx)
	if e.ErrorCode != ErrCodeMemo {
		t.Errorf("expected ErrCodeMemo, got %v", e.ErrorCode)
	}
	if e.ComponentID != "comp-1" {
		t.Errorf("expected 'comp-1', got %q", e.ComponentID)
	}
	if e.Phase != "mount" {
		t.Errorf("expected 'mount', got %q", e.Phase)
	}
	if e.HookIndex != 3 {
		t.Errorf("expected 3, got %d", e.HookIndex)
	}
	if e.Meta["session_id"] != "sess-1" {
		t.Errorf("expected 'sess-1', got %v", e.Meta["session_id"])
	}
	if e.Meta["component_name"] != "MyComponent" {
		t.Errorf("expected 'MyComponent', got %v", e.Meta["component_name"])
	}
	if e.Meta["dev_mode"] != true {
		t.Errorf("expected true, got %v", e.Meta["dev_mode"])
	}
}

func TestError_ToServerError(t *testing.T) {
	e := &Error{
		ErrorCode:  ErrCodeApp,
		Message:    "app error",
		StackTrace: "stack",
		Meta:       map[string]any{"key": "value"},
	}

	se := e.ToServerError("session-123")
	if se.T != "error" {
		t.Errorf("expected 'error', got %q", se.T)
	}
	if se.SID != "session-123" {
		t.Errorf("expected 'session-123', got %q", se.SID)
	}
	if se.Code != string(ErrCodeApp) {
		t.Errorf("expected '%s', got %q", ErrCodeApp, se.Code)
	}
	if se.Message != "app error" {
		t.Errorf("expected 'app error', got %q", se.Message)
	}
	if se.StackTrace != "stack" {
		t.Errorf("expected 'stack', got %q", se.StackTrace)
	}
	if se.Meta["key"] != "value" {
		t.Errorf("expected 'value', got %v", se.Meta["key"])
	}
}

func TestError_ToServerError_Nil(t *testing.T) {
	var e *Error
	if e.ToServerError("session") != nil {
		t.Error("expected nil for nil error")
	}
}

func TestErrorBatch_Error_NilBatch(t *testing.T) {
	var b *ErrorBatch
	if b.Error() != "" {
		t.Errorf("expected empty string for nil batch, got %q", b.Error())
	}
}

func TestErrorBatch_Error_EmptyBatch(t *testing.T) {
	b := &ErrorBatch{errors: []*Error{}}
	if b.Error() != "" {
		t.Errorf("expected empty string for empty batch, got %q", b.Error())
	}
}

func TestErrorBatch_Error_SingleError(t *testing.T) {
	b := &ErrorBatch{errors: []*Error{{Message: "single error"}}}
	if b.Error() != "single error" {
		t.Errorf("expected 'single error', got %q", b.Error())
	}
}

func TestErrorBatch_Error_MultipleErrors(t *testing.T) {
	b := &ErrorBatch{errors: []*Error{
		{Message: "first error"},
		{Message: "second error"},
		{Message: "third error"},
	}}
	expected := "first error (and 2 more errors)"
	if b.Error() != expected {
		t.Errorf("expected %q, got %q", expected, b.Error())
	}
}

func TestErrorBatch_ByPhase(t *testing.T) {
	b := &ErrorBatch{errors: []*Error{
		{Message: "render1", Phase: "render"},
		{Message: "mount1", Phase: "mount"},
		{Message: "render2", Phase: "render"},
	}}

	renderErrors := b.ByPhase("render")
	if len(renderErrors) != 2 {
		t.Errorf("expected 2 render errors, got %d", len(renderErrors))
	}

	mountErrors := b.ByPhase("mount")
	if len(mountErrors) != 1 {
		t.Errorf("expected 1 mount error, got %d", len(mountErrors))
	}

	otherErrors := b.ByPhase("other")
	if len(otherErrors) != 0 {
		t.Errorf("expected 0 other errors, got %d", len(otherErrors))
	}
}

func TestErrorBatch_ByPhase_Nil(t *testing.T) {
	var b *ErrorBatch
	if b.ByPhase("render") != nil {
		t.Error("expected nil for nil batch")
	}
}

func TestErrorBatch_ByCode(t *testing.T) {
	b := &ErrorBatch{errors: []*Error{
		{Message: "render1", ErrorCode: ErrCodeRender},
		{Message: "app1", ErrorCode: ErrCodeApp},
		{Message: "render2", ErrorCode: ErrCodeRender},
	}}

	renderErrors := b.ByCode(ErrCodeRender)
	if len(renderErrors) != 2 {
		t.Errorf("expected 2 render errors, got %d", len(renderErrors))
	}

	appErrors := b.ByCode(ErrCodeApp)
	if len(appErrors) != 1 {
		t.Errorf("expected 1 app error, got %d", len(appErrors))
	}

	handlerErrors := b.ByCode(ErrCodeHandler)
	if len(handlerErrors) != 0 {
		t.Errorf("expected 0 handler errors, got %d", len(handlerErrors))
	}
}

func TestErrorBatch_ByCode_Nil(t *testing.T) {
	var b *ErrorBatch
	if b.ByCode(ErrCodeRender) != nil {
		t.Error("expected nil for nil batch")
	}
}

func TestErrorBatch_ToServerErrors(t *testing.T) {
	b := &ErrorBatch{errors: []*Error{
		{ErrorCode: ErrCodeApp, Message: "error1"},
		{ErrorCode: ErrCodeHandler, Message: "error2"},
	}}

	serverErrors := b.ToServerErrors("sess-1")
	if len(serverErrors) != 2 {
		t.Errorf("expected 2 server errors, got %d", len(serverErrors))
	}
	if serverErrors[0].Message != "error1" {
		t.Errorf("expected 'error1', got %q", serverErrors[0].Message)
	}
	if serverErrors[1].Message != "error2" {
		t.Errorf("expected 'error2', got %q", serverErrors[1].Message)
	}
}

func TestErrorBatch_ToServerErrors_Nil(t *testing.T) {
	var b *ErrorBatch
	if b.ToServerErrors("sess") != nil {
		t.Error("expected nil for nil batch")
	}
}

func TestErrorBatch_ToServerErrors_Empty(t *testing.T) {
	b := &ErrorBatch{errors: []*Error{}}
	if b.ToServerErrors("sess") != nil {
		t.Error("expected nil for empty batch")
	}
}

func TestErrorBatch_Count_Nil(t *testing.T) {
	var b *ErrorBatch
	if b.Count() != 0 {
		t.Errorf("expected 0 for nil batch, got %d", b.Count())
	}
}

func TestErrorBatch_First_Nil(t *testing.T) {
	var b *ErrorBatch
	if b.First() != nil {
		t.Error("expected nil for nil batch")
	}
}

func TestErrorBatch_First_Empty(t *testing.T) {
	b := &ErrorBatch{errors: []*Error{}}
	if b.First() != nil {
		t.Error("expected nil for empty batch")
	}
}

func TestErrorBatch_All_Nil(t *testing.T) {
	var b *ErrorBatch
	if b.All() != nil {
		t.Error("expected nil for nil batch")
	}
}

func TestNewErrorBatch_FiltersNil(t *testing.T) {
	b := NewErrorBatch(
		&Error{Message: "error1"},
		nil,
		&Error{Message: "error2"},
		nil,
	)
	if b.Count() != 2 {
		t.Errorf("expected 2 errors (nil filtered), got %d", b.Count())
	}
}

func TestErrorCodes(t *testing.T) {
	codes := []ErrorCode{
		ErrCodeRender,
		ErrCodeMemo,
		ErrCodeEffect,
		ErrCodeEffectCleanup,
		ErrCodeHandler,
		ErrCodeValidation,
		ErrCodeApp,
		ErrCodeSession,
		ErrCodeNetwork,
		ErrCodeTimeout,
	}

	for _, code := range codes {
		if code == "" {
			t.Error("expected non-empty error code")
		}
	}
}

func TestPondErrorInterface(t *testing.T) {
	var _ PondError = (*Error)(nil)

	e := &Error{
		ErrorCode:  ErrCodeApp,
		Message:    "test",
		StackTrace: "stack",
		Meta:       map[string]any{"k": "v"},
		Cause:      errors.New("cause"),
	}

	var pe PondError = e
	if pe.Code() != ErrCodeApp {
		t.Errorf("expected ErrCodeApp, got %v", pe.Code())
	}
	if pe.Error() != "test" {
		t.Errorf("expected 'test', got %q", pe.Error())
	}
	if pe.Stack() != "stack" {
		t.Errorf("expected 'stack', got %q", pe.Stack())
	}
	if pe.Metadata()["k"] != "v" {
		t.Errorf("expected 'v', got %v", pe.Metadata()["k"])
	}
	if pe.Unwrap() == nil {
		t.Error("expected non-nil Unwrap")
	}
}
