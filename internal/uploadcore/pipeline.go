// Package uploadcore holds the upload pipeline pieces shared by the desktop
// (Wails) and browser-library managers: the per-file token→upload→commit
// sequence and the album batch planning. Control state machines, event
// payloads and pre-upload dedupe logic stay with each manager.
package uploadcore

import (
	"context"
	"fmt"

	gphotoclient "app/internal/gphotos"
)

// APIClient is the slice of the Google Photos client the per-file pipeline
// needs. Both layer Api types satisfy it.
type APIClient interface {
	GetUploadToken(ctx context.Context, shaHashB64 string, fileSize int64) (string, error)
	UploadFileWithProgress(ctx context.Context, filePath string, uploadToken string, onProgress gphotoclient.UploadProgressCallback) (gphotoclient.ScottyFinalizeToken, error)
	CommitUploadAs(ctx context.Context, finalizeToken gphotoclient.ScottyFinalizeToken, fileName string, sha1Hash []byte, uploadTimestamp int64, device gphotoclient.CommitDevice) (string, error)
}

// ConcurrencyReporter lets the pipeline feed the adaptive concurrency limiter.
type ConcurrencyReporter interface {
	ReportRetry()
	ReportSuccess()
}

// StatusSink receives per-file progress in the shared shape. Each layer maps
// it onto its own event payload (Wails ThreadStatus vs SSE ThreadStatus).
type StatusSink func(status, message string, uploaded, total int64, attempt int)

// Options carries everything the pipeline needs for one file. The caller has
// already hashed the file and decided it is not a dedupe skip.
type Options struct {
	API             APIClient
	Concurrency     ConcurrencyReporter // optional
	Sink            StatusSink
	Device          gphotoclient.CommitDevice
	FileName        string // commit name (base name)
	FileSize        int64
	SHA1Hash        []byte
	SHA1B64         string
	UploadTimestamp int64
	DeleteFromHost  bool
	DeleteFile      func(path string) error

	// OnPause, when set, is consulted before emitting upload progress; while
	// paused the emission is suppressed (desktop behavior). Nil for libweb.
	OnPause func() bool
}

// UploadFile runs token acquisition, the resumable upload with progress and
// retry reporting, the retried commit, and optional host deletion. It returns
// the committed media key. Status flow: uploading → finalizing.
func UploadFile(ctx context.Context, filePath string, opts Options) (string, error) {
	if opts.API == nil || opts.Sink == nil {
		return "", fmt.Errorf("uploadcore: API and Sink are required")
	}
	sink := opts.Sink
	retried := false

	sink("uploading", "Uploading...", 0, opts.FileSize, 0)

	token, err := opts.API.GetUploadToken(ctx, opts.SHA1B64, opts.FileSize)
	if err != nil {
		return "", fmt.Errorf("error uploading file: %w", err)
	}

	progressCb := func(bytesUploaded, bytesTotal int64, attempt int) {
		if opts.OnPause != nil && opts.OnPause() {
			return
		}
		message := "Uploading..."
		if attempt > 1 {
			message = fmt.Sprintf("Retrying... (attempt %d)", attempt)
			if !retried && opts.Concurrency != nil {
				retried = true
				opts.Concurrency.ReportRetry()
			}
		}
		sink("uploading", message, bytesUploaded, bytesTotal, attempt)
	}

	finalizeToken, err := opts.API.UploadFileWithProgress(ctx, filePath, token, progressCb)
	if err != nil {
		return "", fmt.Errorf("error uploading file: %w", err)
	}

	sink("finalizing", "Committing upload...", 0, 0, 0)

	mediaKey, err := opts.API.CommitUploadAs(ctx, finalizeToken, opts.FileName, opts.SHA1Hash, opts.UploadTimestamp, opts.Device)
	if err != nil {
		return "", fmt.Errorf("error committing file: %w", err)
	}
	if mediaKey == "" {
		return "", fmt.Errorf("media key not received")
	}

	if !retried && opts.Concurrency != nil {
		opts.Concurrency.ReportSuccess()
	}

	if opts.DeleteFromHost && opts.DeleteFile != nil {
		if err := opts.DeleteFile(filePath); err != nil {
			return mediaKey, fmt.Errorf("uploaded successfully but failed to delete file: %w", err)
		}
	}

	return mediaKey, nil
}
