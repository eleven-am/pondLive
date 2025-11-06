package runtime

// UseMetadata merges meta into the session-level metadata for the current render.
func UseMetadata(ctx Ctx, meta *Meta) {
	if meta == nil {
		return
	}
	sess := ctx.Session()
	if sess == nil {
		return
	}
	sess.SetMetadata(meta)
}
