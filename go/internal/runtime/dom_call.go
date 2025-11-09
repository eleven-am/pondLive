package runtime

import (
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	internalhtml "github.com/eleven-am/pondlive/go/internal/html"
)

// DOMCall instructs the client to invoke a method on the referenced element.
func DOMCall[T dom.ElementDescriptor](ctx Ctx, ref *internalhtml.ElementRef[T], method string, args ...any) {
	if ctx.sess == nil || ctx.sess.owner == nil {
		return
	}
	if ref == nil {
		return
	}
	method = strings.TrimSpace(method)
	if method == "" {
		return
	}
	refID := ref.ID()
	if refID == "" {
		return
	}
	effect := DOMCallEffect{
		Type:   "domcall",
		Ref:    refID,
		Method: method,
	}
	if len(args) > 0 {
		effect.Args = append([]any(nil), args...)
	}
	ctx.sess.owner.enqueueFrameEffect(effect)
}
