package runtime

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// UploadedFile represents a staged file from a multipart upload.
type UploadedFile struct {
	FileMeta FileMeta
	TempPath string
	Reader   io.ReadCloser
}

// StageUploadedFile copies the uploaded content to a temporary file and returns a handle.
// If sizeLimit > 0, the upload is rejected if it exceeds the limit.
func StageUploadedFile(part io.Reader, filename, contentType string, sizeLimit int64) (UploadedFile, error) {
	if part == nil {
		return UploadedFile{}, errors.New("runtime2: missing upload payload")
	}

	dir := os.TempDir()
	file, err := os.CreateTemp(dir, "pond-upload-*")
	if err != nil {
		return UploadedFile{}, fmt.Errorf("runtime2: create temp file: %w", err)
	}

	var written int64
	if sizeLimit > 0 {
		written, err = io.Copy(file, io.LimitReader(part, sizeLimit+1))
		if err == nil && written > sizeLimit {
			err = ErrUploadTooLarge
		}
	} else {
		written, err = io.Copy(file, part)
	}

	if err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return UploadedFile{}, err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return UploadedFile{}, err
	}

	meta := FileMeta{
		Name: filepath.Base(filename),
		Size: written,
		Type: contentType,
	}

	return UploadedFile{
		FileMeta: meta,
		TempPath: file.Name(),
		Reader:   file,
	}, nil
}
