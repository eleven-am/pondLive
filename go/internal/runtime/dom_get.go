package runtime

import (
	"errors"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	html "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// DOMGet requests DOM properties for the provided element ref by delegating to the session.
func DOMGet[T dom.ElementDescriptor](ctx Ctx, ref *html.ElementRef[T], selectors ...string) (map[string]any, error) {
	if ctx.sess == nil || ctx.sess.owner == nil {
		return nil, errors.New("runtime: domget requires live session context")
	}
	if ref == nil {
		return nil, errors.New("runtime: domget requires a non-nil element ref")
	}
	id := strings.TrimSpace(ref.ID())
	if id == "" {
		return nil, errors.New("runtime: domget requires a bound element ref")
	}
	return ctx.sess.owner.DOMGet(id, selectors...)
}
