# Attachment API Redesign

To unify how hooks bind to DOM elements, `h.Attach` is being generalized from a
ref-only helper into a generic attachment hook. This document outlines the new
API and how existing hooks migrate to it.

## 1. Motivation

- Today `h.Attach` only works with `ElementRef` from `UseElement`; it simply
  calls `dom.AttachElementRef`, which stamps ref metadata and merges ref-level
  event bindings.
- Hooks like `UseUpload` need to “attach” to DOM nodes too (to associate an
  input with an upload slot, accept list, etc.), but they aren’t refs.
- Rather than keep bespoke helpers (e.g., `upload.BindInput`) we want a uniform
  pattern: `h.Attach(...)` for anything that wires hook-level metadata into an
  element, without mutating HTML.

## 2. New Attachment Interface

```go
type Attachment interface {
    AttachTo(el *h.Element)
}

func Attach(target Attachment) h.Prop {
    if target == nil {
        return nil
    }
    return attachmentProp{target: target}
}

type attachmentProp struct {
    target Attachment
}

func (attachmentProp) isProp() {}

func (p attachmentProp) ApplyTo(el *h.Element) {
    if el == nil || p.target == nil {
        return
    }
    p.target.AttachTo(el)
}
```

- Hooks implement `Attachment` directly. Calling `h.Attach(hook)` wraps the
  target and lets it bind metadata to the element during render.
- Attachments run server-side during render; they can register bindings, enforce
  descriptor checks, etc., without touching HTML attributes.

## 3. ElementRef Attachment

`UseElement` returns an `ElementRef`. It now satisfies `Attachment`
directly:

```go
type ElementRef[T dom.ElementDescriptor] struct {
    *dom.ElementRef[T]
}

func (r *ElementRef[T]) AttachTo(el *h.Element) {
    if r == nil {
        return
    }
    dom.AttachElementRef[T](r.ElementRef, el)
}
```

Usage becomes:

```go
videoRef := live.UseElement[h.VideoDescriptor](ctx)
return h.Video(
    h.Attach(videoRef),
    // other props...
)
```

The attachment still merges ref event bindings, registers `ref:*` IDs, and feeds
the renderer so `bindings.attachments`/`refs` can be emitted without
`data-live-ref`.

## 4. Upload Attachment

`UseUpload` also implements `Attachment` so it can be passed directly to
`h.Attach`:

```go
func (handle UploadHandle) AttachTo(el *h.Element) {
    if el == nil || handle.slot == nil {
        return
    }
    handle.slot.registerBinding(el)
}
```

`registerBinding` (implementation detail) records `{componentId, path, uploadId,
accept, multiple, maxSize}` so the renderer emits a `bindings.uploads` entry.

Developers now wire upload inputs via:

```go
upload := live.UseUpload(ctx)
return h.Input(
    h.Type("file"),
    h.Attach(upload),
)
```

`upload.BindInput(...)` remains for backwards compatibility—it simply returns
`[]h.Prop{h.Attach(upload), ...}` so existing code continues to work while new
code uses the uniform API.

## 5. Future Attachments

Any hook that needs to associate metadata with a DOM node (e.g., focus traps,
gesture handlers, custom refs) can expose an `Attachment()` returning custom
logic. Attachments can:

- Register structured bindings (handlers/router/uploads/refs).
- Enforce invariants (descriptor checks, single attachment per render).
- Merge hook-specific event bindings.

## 6. Summary

- `h.Attach` now accepts any `Attachment`, not just refs.
- `UseElement` and `UseUpload` expose `Attachment()` helpers so developers use
  the same API for refs and uploads.
- Attachments run during render to register metadata; the renderer emits
  bindings in template payloads instead of mutating HTML attributes.
- `upload.BindInput` remains as a helper but is implemented in terms of
  `h.Attach(upload.Attachment())`, so components can adopt the new syntax at
  their own pace.
