package backend

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"app/internal/ctxrelay"
)

// Worker-pool plumbing: dependency bundle, the worker loop, and the panic
// guard around per-file processing.

type ProgressCallback func(event string, data any)

type pauseGate func() bool

type workerDeps struct {
	api               *Api
	app               AppInterface
	paused            pauseGate
	skipRequested     func(path string) bool
	concurrency       *AdaptiveConcurrency
	fileMetaFor       func(path string) (fileMeta, bool)
	cfg               Config
	AlwaysDeduplicate bool
}

func startUploadWorker(workerID int, deps workerDeps, workChan <-chan UploadWorkItem, results chan<- FileUploadResult, cancel <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	app := deps.app
	paused := deps.paused

	app.EmitEvent(EventThreadStatus, ThreadStatus{
		WorkerID: workerID,
		Status:   "idle",
		Message:  "Waiting for files...",
	})

	callback := func(event string, data any) {
		app.EmitEvent(event, data)
	}

	for item := range workChan {
		select {
		case <-cancel:
			app.EmitEvent(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "idle",
				Message:  "Cancelled",
			})
			return
		default:
		}

		if paused != nil && paused() {
			app.EmitEvent(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "idle",
				Message:  "Cancelled",
			})
			return
		}

		path := uploadWorkPrimaryPath(item)

		if deps.skipRequested != nil && deps.skipRequested(path) {
			results <- FileUploadResult{Path: path, Skipped: true, SkipCode: "user-skip", SkipReason: "Skipped by user"}
			app.EmitEvent(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "completed",
				FilePath: path,
				FileName: filepath.Base(path),
				Message:  "Skipped by user",
			})
			app.EmitEvent(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "idle",
				Message:  "Waiting for next file...",
			})
			continue
		}

		ctx, cancelUpload := ctxrelay.RelayCancel(cancel)

		if deps.concurrency != nil && !deps.concurrency.Acquire(ctx) {
			cancelUpload()
			app.EmitEvent(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "idle",
				Message:  "Cancelled",
			})
			return
		}

		mediaKey, existing, err := safeUploadWorkItem(ctx, deps, item, workerID, callback)
		if deps.concurrency != nil {
			deps.concurrency.Release()
		}
		if err != nil && mediaKey != "" {
			results <- uploadSuccessResult(path, mediaKey, existing)
			app.EmitEvent(EventUploadWarning, PreflightWarning{
				Paths:   []string{path},
				Code:    "local-cleanup-failed",
				Message: err.Error(),
			})
			app.EmitEvent(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "completed",
				FilePath: path,
				FileName: filepath.Base(path),
				Message:  fmt.Sprintf("Uploaded, but local cleanup failed: %v", err),
			})
		} else if err != nil {
			results <- FileUploadResult{IsError: true, Error: err, ErrorMessage: err.Error(), Path: path}
			app.EmitEvent(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "error",
				FilePath: path,
				FileName: filepath.Base(path),
				Message:  fmt.Sprintf("Error: %v", err),
			})
		} else {
			results <- uploadSuccessResult(path, mediaKey, existing)
			app.EmitEvent(EventThreadStatus, ThreadStatus{
				WorkerID: workerID,
				Status:   "completed",
				FilePath: path,
				FileName: filepath.Base(path),
				Message:  "Completed",
			})
		}
		cancelUpload()

		app.EmitEvent(EventThreadStatus, ThreadStatus{
			WorkerID: workerID,
			Status:   "idle",
			Message:  "Waiting for next file...",
		})
	}

	app.EmitEvent(EventThreadStatus, ThreadStatus{
		WorkerID: workerID,
		Status:   "idle",
		Message:  "Finished",
	})
}

func uploadSuccessResult(path, mediaKey string, alreadyInLibrary bool) FileUploadResult {
	if alreadyInLibrary {
		return FileUploadResult{Path: path, MediaKey: mediaKey, Skipped: true, SkipCode: "already-in-library", SkipReason: "Already in library"}
	}
	return FileUploadResult{Path: path, MediaKey: mediaKey}
}

func safeUploadWorkItem(ctx context.Context, deps workerDeps, item UploadWorkItem, workerID int, callback ProgressCallback) (mediaKey string, existing bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			mediaKey, existing = "", false
			err = fmt.Errorf("internal error while processing file: %v", r)
		}
	}()
	return uploadWorkItem(ctx, deps, item, workerID, callback)
}

func uploadWorkItem(ctx context.Context, deps workerDeps, item UploadWorkItem, workerID int, callback ProgressCallback) (string, bool, error) {
	if item.Single == nil {
		return "", false, fmt.Errorf("invalid upload work item")
	}
	return uploadFileWithCallback(ctx, deps, item.Single.Path, workerID, callback)
}
