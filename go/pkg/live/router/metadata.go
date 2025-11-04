package router

import ui "github.com/eleven-am/pondlive/go/pkg/live"

// UseMetadata merges meta into the session-level metadata for the current render.
func UseMetadata(ctx ui.Ctx, meta *ui.Meta) {
	if meta == nil {
		return
	}
	sess := ctx.Session()
	if sess == nil {
		return
	}
	current := sess.Metadata()
	sess.SetMetadata(ui.MergeMeta(current, meta))
}
