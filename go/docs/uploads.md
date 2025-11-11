# Upload & Ref-Backed Inputs

This document explains the upload hook pipeline end-to-end: how `UseUpload`
creates slots, how refs/inputs are wired, how the client UploadManager streams
files, server-side staging/validation, and lifecycle events.

## 1. Server Hook (`UseUpload`)

```go
upload := live.UseUpload(ctx)

upload.Accept("image/png")
upload.MaxSize(5 << 20)

upload.OnChange(func(meta live.FileMeta) { ... })
upload.OnComplete(func(file live.UploadedFile) h.Updates { ... })
upload.OnError(func(err error) h.Updates { ... })

node := h.Input(
    h.Type("file"),
    h.Attach(upload), // or upload.BindInput(...) for backwards compatibility
)

progress := upload.Progress()
```

- `UseUpload` registers an `uploadSlot` for the component (one per hook index)
  and tracks accept/multiple/max size plus lifecycle callbacks.
- `UseUpload` implements the attachment interface (see `docs/attach.md`), so
  components can call `h.Attach(upload)`. `BindInput` remains as a helper that
  returns `[]h.Prop{h.Attach(upload), ...}` for backwards compatibility.
- Slots persist across renders until the component unmounts
  (`ComponentSession.releaseUploadSlots`); unmounting cancels in-flight uploads.

## 2. Client Wiring

1. `UploadManager` registers itself with `registerUploadDelegate` so it receives
   all DOM events delegated through `events.ts`.
2. On `change` events:
   - Resolve the actual `<input type="file" data-pond-upload="...">`.
   - If no connection or no session ID, abort and emit an `upload:error`.
   - Send `UploadClient` message `{ t:"upload", op:"change", meta: {name,size,type} }`
     so the server can update progress and run `OnChange`.
   - Start an `XMLHttpRequest` to `uploadEndpoint/<sid>/<uploadId>`, streaming
     the selected file via `FormData`.
3. `XMLHttpRequest` hooks:
   - `upload.onprogress` → send `op:"progress", loaded, total`.
   - `onload` → if status >= 300, send `op:"error"`.
   - `onerror` → send network error.
   - `onabort` → send `op:"cancelled"`.
4. UploadManager tracks active uploads (`Map<uploadId, {xhr,input}>`) so
   `handleControl` can abort them when the server requests cancellation.

## 3. Server Upload Lifecycle

1. **Change** (`HandleUploadChange`):
   - `slot.beginUpload(meta)` sets status to `uploading`, resets progress.
   - Invokes `onChange` callback (if provided). Good place to validate or update
     UI immediately.
2. **Progress** (`HandleUploadProgress`):
   - Updates `slot.progress` (marks component dirty so `upload.Progress()` rerenders).
3. **Error** (`HandleUploadError`):
   - Sets status to `error`, stores the error, runs `onError`.
4. **Cancelled** (`HandleUploadCancelled`):
   - Status → `cancelled`.
5. **Complete**:
   - After the HTTP upload finishes, the transport handler stages the raw bytes
     via `StageUploadedFile`. Once ready, it calls
     `ComponentSession.CompleteUpload(id, UploadedFile)`:
     - `slot.processing()` → status `processing`.
     - Runs `onComplete` callback with `UploadedFile` (contains metadata,
       temp file path, `io.ReadSeekCloser`).
     - `onComplete` returns `h.Updates`; component marked dirty so any updates
       rerender.
     - Slot status becomes `complete`.

## 4. Server Staging (`StageUploadedFile`)

- Writes the uploaded data to a temp file (`os.CreateTemp`).
- Enforces `MaxSize` (limiting `io.Copy`), returning `ErrUploadTooLarge` if
  exceeded.
- Returns `UploadedFile` with `.Reader` and `.TempPath`. Callers must move the
  file to permanent storage (`os.Rename`, `io.Copy`, etc.) and close/remove it.

## 5. Cancellation

- Server can call `LiveSession.CancelUpload(id)` to:
  1. Notify component slot (`HandleUploadCancelled`).
  2. Send `UploadControl{op:"cancel"}` to the client so UploadManager aborts the XHR.
- Slots have `UploadHandle.Cancel()` which calls `requestCancel()` → forwards to
  LiveSession (so server + client stay in sync).

## 6. Upload Bindings & Caching (Clean Metadata)

- Instead of emitting `data-pond-upload`, template payloads include upload
  binding entries in `bindings.uploads`:
  ```json
  {
    "componentId": "c123",
    "path": [4,0],
    "uploadId": "u1",
    "accept": ["image/png"],
    "multiple": false,
    "maxSize": 5242880
  }
  ```
- During hydration the client resolves `{componentId,path}` to the actual
  `<input type="file">`, stores `uploadId`/config in a `WeakMap`, and registers
  it with `UploadManager`. Event delegation no longer scans DOM attributes; it
  reads the binding map instead.
- Component boot templates include upload bindings for their subtree, so newly
  mounted components hydrate inputs the same way as boot/init.
- Because templates can be cached (see `docs/hydration-flow.md`), upload binding
  entries are small descriptors; when a cached template is reused, the hydrator
  simply reattaches the upload metadata to the cloned DOM nodes.

## 7. Validation & Errors

- **Client-side**: Accept/multiple attributes inform the browser; beyond that,
  we rely on the server to validate.
- **Server-side**:
  - `MaxSize` prevents large uploads at staging time.
  - Developers can inspect `FileMeta` during `OnChange` or `OnComplete` to
    enforce content types, file names, etc.
  - If validation fails, call `upload.Cancel()` or return an error via
    `OnError`.
- All errors should be surfaced through `OnError` callback so components can
  show feedback. Additionally, the runtime logs upload errors and can report
  diagnostics if callbacks panic.

## 8. Improvement Ideas

- **Clean metadata** – integrate upload slots with the structured bindings
  system to remove `data-pond-upload` from HTML.
- **Multiple files** – support multi-file uploads per slot (currently `files[0]`
  only). Would require server API changes (`[]UploadedFile`) and per-file
  progress tracking.
- **Chunked uploads** – current implementation streams the whole file via one
  POST. Consider resumable uploads or chunking for very large files.
- **Security hooks** – provide middleware to scan uploads (virus scanning,
  content policy) before invoking `OnComplete`.
- **Client progress UX** – expose hooks/events so developers can build custom
  progress bars without polling `upload.Progress()` manually.

Understanding this pipeline helps when modifying uploads, building custom file
workflows, or debugging issues between input change events and server staging.
