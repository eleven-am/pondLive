package router

import "github.com/eleven-am/pondlive/go/internal/runtime"

// UseMetadata merges meta into the session-level metadata for the current render.
func UseMetadata(ctx runtime.Ctx, meta *runtime.Meta) {
	if meta == nil {
		return
	}
	sess := ctx.Session()
	if sess == nil {
		return
	}
	sess.SetMetadata(meta)
}
