package live

import runtime "github.com/eleven-am/pondlive/go/internal/runtime"

type (
	UploadStatus   = runtime.UploadStatus
	FileMeta       = runtime.FileMeta
	UploadProgress = runtime.UploadProgress
	UploadedFile   = runtime.UploadedFile
	UploadHandle   = runtime.UploadHandle
)

// UseUpload returns a handle for managing file uploads with progress tracking and validation.
// The upload automatically begins when the user selects a file through the bound input element.
//
// Example - Basic file upload:
//
//	func FileUploader(ctx live.Ctx) h.Node {
//	    upload := live.UseUpload(ctx)
//	    fileName, setFileName := live.UseState(ctx, "")
//
//	    upload.OnChange(func(meta live.FileMeta) {
//	        fmt.Printf("File selected: %s (%d bytes)\n", meta.Name, meta.Size)
//	    })
//
//	    upload.OnComplete(func(file live.UploadedFile) h.Updates {
//	        setFileName(file.Name)
//	        // Process the uploaded file
//	        // file.Reader provides access to the file contents
//	        // file.TempPath is the temporary server path
//	        return nil
//	    })
//
//	    upload.OnError(func(err error) h.Updates {
//	        fmt.Printf("Upload failed: %v\n", err)
//	        return nil
//	    })
//
//	    progress := upload.Progress()
//
//	    return h.Div(
//	        h.Input(upload.BindInput(h.Type("file"))...),
//	        renderProgress(progress, fileName()),
//	    )
//	}
//
//	func renderProgress(p live.UploadProgress, fileName string) h.Node {
//	    switch p.Status {
//	    case live.UploadStatusUploading:
//	        return h.Div(
//	            h.Text(fmt.Sprintf("Uploading... %.1f%%", p.Percent)),
//	            h.Progress(h.Value(fmt.Sprintf("%.0f", p.Percent)), h.Max("100")),
//	        )
//	    case live.UploadStatusComplete:
//	        return h.Div(h.Text(fmt.Sprintf("Uploaded: %s", fileName)))
//	    case live.UploadStatusError:
//	        return h.Div(h.Text(fmt.Sprintf("Error: %v", p.Error)))
//	    default:
//	        return h.Div(h.Text("Select a file to upload"))
//	    }
//	}
//
// Example - Multiple files with validation:
//
//	func MultiFileUploader(ctx live.Ctx) h.Node {
//	    upload := live.UseUpload(ctx)
//
//	    // Configure upload constraints
//	    upload.MaxSize(5 * 1024 * 1024)  // 5MB limit
//	    upload.AllowMultiple(true)       // Enable multiple file selection
//	    upload.Accept("image/png", "image/jpeg")  // Only accept images
//
//	    upload.OnComplete(func(file live.UploadedFile) h.Updates {
//	        fmt.Printf("Uploaded: %s (%s, %d bytes)\n", file.Name, file.Type, file.Size)
//	        // Process each file
//	        return nil
//	    })
//
//	    return h.Div(
//	        h.Input(upload.BindInput(h.Type("file"))...),
//	        h.Button(
//	            h.OnClick(func() h.Updates {
//	                upload.Cancel()  // Cancel in-progress upload
//	                return nil
//	            }),
//	            h.Text("Cancel Upload"),
//	        ),
//	    )
//	}
func UseUpload(ctx Ctx) UploadHandle { return runtime.UseUpload(ctx) }
