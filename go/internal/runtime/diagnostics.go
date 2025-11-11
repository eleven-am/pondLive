package runtime

import (
	"strings"
	"time"

	"github.com/eleven-am/pondlive/go/internal/protocol"
)

const defaultDiagnosticHistory = 32

// Diagnostic captures structured information about a recovered runtime panic.
type Diagnostic struct {
	Code          string
	Phase         string
	ComponentID   string
	ComponentName string
	Message       string
	Hook          string
	HookIndex     int
	Suggestion    string
	Stack         string
	Panic         string
	Metadata      map[string]any
	CapturedAt    time.Time
}

// ToErrorDetails converts the diagnostic into a protocol payload suitable for
// delivery to the client.
func (d Diagnostic) ToErrorDetails() protocol.ErrorDetails {
	details := protocol.ErrorDetails{
		Phase:         d.Phase,
		ComponentID:   d.ComponentID,
		ComponentName: d.ComponentName,
		Hook:          d.Hook,
		HookIndex:     d.HookIndex,
		Suggestion:    d.Suggestion,
		Stack:         d.Stack,
		Panic:         d.Panic,
		Metadata:      cloneDiagnosticMetadata(d.Metadata),
	}
	if !d.CapturedAt.IsZero() {
		details.CapturedAt = d.CapturedAt.Format(time.RFC3339Nano)
	}
	return details
}

// ToServerError encodes the diagnostic as a protocol.ServerError message.
func (d Diagnostic) ToServerError(id SessionID) protocol.ServerError {
	details := d.ToErrorDetails()
	code := d.Code
	if code == "" {
		code = "runtime_panic"
	}
	message := d.Message
	if message == "" {
		message = "live: runtime panic recovered"
	}
	return protocol.ServerError{
		T:       "error",
		SID:     string(id),
		Code:    code,
		Message: message,
		Details: &details,
	}
}

// ToProtocolDiagnostic encodes the diagnostic as a protocol.Diagnostic message.
func (d Diagnostic) ToProtocolDiagnostic(id SessionID) protocol.Diagnostic {
	details := d.ToErrorDetails()
	code := d.Code
	if code == "" {
		code = "runtime_panic"
	}
	message := d.Message
	if message == "" {
		message = "live: runtime panic recovered"
	}
	return protocol.Diagnostic{
		T:       "diagnostic",
		SID:     string(id),
		Code:    code,
		Message: message,
		Details: &details,
	}
}

// DiagnosticError wraps a Diagnostic to implement the error interface.
type DiagnosticError struct {
	diag Diagnostic
}

// Error implements the error interface.
func (e DiagnosticError) Error() string {
	return e.diag.Message
}

// Diagnostic exposes the captured diagnostic.
func (e DiagnosticError) Diagnostic() Diagnostic {
	return e.diag
}

// AsError converts the diagnostic into a DiagnosticError.
func (d Diagnostic) AsError() DiagnosticError {
	return DiagnosticError{diag: d}
}

// AsDiagnosticError extracts the diagnostic payload from the provided error if present.
func AsDiagnosticError(err error) (Diagnostic, bool) {
	switch v := err.(type) {
	case DiagnosticError:
		return v.Diagnostic(), true
	case *DiagnosticError:
		if v == nil {
			return Diagnostic{}, false
		}
		return v.Diagnostic(), true
	default:
		return Diagnostic{}, false
	}
}

func cloneDiagnosticMetadata(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func normalizeDiagnosticCode(phase string) string {
	if phase == "" {
		return "runtime_panic"
	}
	slug := strings.ToLower(phase)
	slug = strings.ReplaceAll(slug, " ", "_")
	slug = strings.ReplaceAll(slug, ":", "_")
	slug = strings.ReplaceAll(slug, "/", "_")
	slug = strings.ReplaceAll(slug, "-", "_")
	slug = strings.Trim(slug, "_")
	if slug == "" {
		return "runtime_panic"
	}
	return slug + "_panic"
}

type metadataCarrier interface {
	Metadata() map[string]any
}

type suggestionCarrier interface {
	Suggestion() string
}

type hookCarrier interface {
	HookName() string
	HookIndex() int
}
