package live

import "github.com/eleven-am/pondlive/go/internal/runtime"

type (
	UploadStatus   = runtime.UploadStatus
	FileMeta       = runtime.FileMeta
	UploadProgress = runtime.UploadProgress
	UploadedFile   = runtime.UploadedFile
	UploadHandle   = runtime.UploadHandle
)

func UseUpload(ctx Ctx) UploadHandle { return runtime.UseUpload(ctx) }
