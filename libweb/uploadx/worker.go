package uploadx

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"app/internal/concurrency"
	gclient "app/internal/gphotos"
	"app/internal/hashx"
	"app/internal/hostdelete"
	"app/internal/uploadcore"
	"app/libweb/configx"
	"app/libweb/gphotos"
)

var contextCanceled = context.Canceled

func contextBackground() context.Context { return context.Background() }

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type adaptiveConcurrency = concurrency.AdaptiveConcurrency

var newAdaptiveConcurrency = concurrency.NewAdaptiveConcurrency

type workerDeps struct {
	m           *Manager
	api         *gphotos.Api
	concurrency *adaptiveConcurrency
	fileSizes   map[string]fileMeta
	cancelled   func() bool
}

func (m *Manager) adjustTotalBytes(delta int64) {
	m.mu.Lock()
	m.status.TotalBytes += delta
	m.mu.Unlock()
}

func (m *Manager) worker(workerID int, dep workerDeps, workChan <-chan string, results chan<- FileUploadResult, cancel <-chan struct{}) {
	defer m.wg.Done()

	m.setWorker(ThreadStatus{WorkerID: workerID, Status: "idle", Message: "Waiting for files..."})

	for path := range workChan {
		select {
		case <-cancel:
			m.setWorker(ThreadStatus{WorkerID: workerID, Status: "idle", Message: "Cancelled"})
			return
		default:
		}

		if m.waitIfPaused(cancel) {
			m.setWorker(ThreadStatus{WorkerID: workerID, Status: "idle", Message: "Cancelled"})
			return
		}

		if m.isSkipRequested(path) {
			results <- FileUploadResult{Path: path, Skipped: true, SkipCode: "user-skip", SkipReason: "Skipped by user"}
			m.setWorker(ThreadStatus{WorkerID: workerID, Status: "completed", FilePath: path, FileName: filepath.Base(path), Message: "Skipped by user"})
			m.setWorker(ThreadStatus{WorkerID: workerID, Status: "idle", Message: "Waiting for next file..."})
			continue
		}

		ctx, cancelUpload := context.WithCancel(context.Background())
		go func() {
			select {
			case <-cancel:
				cancelUpload()
			case <-ctx.Done():
			}
		}()

		if !dep.concurrency.Acquire(ctx) {
			cancelUpload()
			m.setWorker(ThreadStatus{WorkerID: workerID, Status: "idle", Message: "Cancelled"})
			return
		}

		mediaKey, sizeDelta, err := m.uploadFile(ctx, dep, path, workerID)
		dep.concurrency.Release()
		if err != nil && mediaKey != "" {
			results <- FileUploadResult{Path: path, MediaKey: mediaKey}
			m.setStatus(func(s *Status) {
				s.Warnings = append(s.Warnings, Warning{Paths: []string{path}, Code: "local-cleanup-failed", Message: err.Error()})
			})
			m.setWorker(ThreadStatus{WorkerID: workerID, Status: "completed", FilePath: path, FileName: filepath.Base(path), Message: "Uploaded, but local cleanup failed"})
		} else if err != nil {
			results <- FileUploadResult{Path: path, IsError: true, ErrorMessage: err.Error()}
			m.setWorker(ThreadStatus{WorkerID: workerID, Status: "error", FilePath: path, FileName: filepath.Base(path), Message: "Error: " + err.Error()})
		} else {
			results <- FileUploadResult{Path: path, MediaKey: mediaKey}
			m.setWorker(ThreadStatus{WorkerID: workerID, Status: "completed", FilePath: path, FileName: filepath.Base(path), Message: "Completed"})
		}
		if sizeDelta != 0 {
			m.adjustTotalBytes(sizeDelta)
		}
		cancelUpload()

		m.setWorker(ThreadStatus{WorkerID: workerID, Status: "idle", Message: "Waiting for next file..."})
	}

	m.setWorker(ThreadStatus{WorkerID: workerID, Status: "idle", Message: "Finished"})
}

func (m *Manager) waitIfPaused(cancel <-chan struct{}) bool {
	for {
		m.pauseMu.Lock()
		if !m.paused {
			m.pauseMu.Unlock()
			return false
		}
		resume := m.pauseSig
		m.pauseMu.Unlock()

		select {
		case <-cancel:
			return true
		case <-resume:
		}
	}
}

func (m *Manager) uploadFile(ctx context.Context, dep workerDeps, filePath string, workerID int) (string, int64, error) {
	cfg := configx.Get().Photos
	fileName := filepath.Base(filePath)

	var uploadTimestamp int64
	var statSize int64
	haveStat := false
	if meta, ok := dep.fileSizes[filePath]; ok {
		uploadTimestamp = meta.ModTime.Unix()
		statSize = meta.Size
		haveStat = true
	}
	if !haveStat {
		if info, err := os.Stat(filePath); err == nil {
			uploadTimestamp = info.ModTime().Unix()
			statSize = info.Size()
			haveStat = true
		}
	}
	if cfg.SetDateFromFilename {
		if t, ok := gphotos.ParseTimestampFromFilename(filePath); ok {
			uploadTimestamp = t.Unix()
		}
	}

	report := func(status, message string, uploaded, total int64, attempt int) {
		m.setWorker(ThreadStatus{
			WorkerID: workerID, Status: status, FilePath: filePath, FileName: fileName,
			Message: message, BytesUploaded: uploaded, BytesTotal: total, Attempt: attempt,
		})
	}

	report("hashing", "Hashing...", 0, 0, 0)
	sha1Hash, err := hashx.CalculateSHA1(ctx, filePath)
	if err != nil {
		return "", 0, fmt.Errorf("error calculating file hash: %w", err)
	}
	sha1B64 := base64.StdEncoding.EncodeToString(sha1Hash)

	if !cfg.ForceUpload {
		report("checking", "Checking if file exists in library...", 0, 0, 0)
		mediaKey, err := dep.api.FindRemoteMediaByHash(ctx, sha1Hash)
		if err != nil {
			report("checking", "Hash check warning: "+err.Error()+", proceeding with upload", 0, 0, 0)
		}
		if mediaKey != "" {
			if cfg.DeleteFromHost {
				if rmErr := hostdelete.Trash(filePath); rmErr != nil {
					return mediaKey, 0, fmt.Errorf("file exists in library but failed to delete local copy: %w", rmErr)
				}
			}
			report("completed", "Already in library", 0, 0, 0)
			return mediaKey, -statSize, nil
		}
	}

	if !haveStat {
		return "", 0, fmt.Errorf("error getting file info")
	}

	report("uploading", "Uploading...", 0, statSize, 0)

	// Shared per-file pipeline: token → resumable upload → retried commit.
	mediaKey, err := uploadcore.UploadFile(ctx, filePath, uploadcore.Options{
		API:         dep.api,
		Concurrency: dep.concurrency,
		Sink: func(status, message string, uploaded, total int64, attempt int) {
			report(status, message, uploaded, total, attempt)
		},
		Device:          gclient.DeviceFromFlags(cfg.Saver, cfg.UseQuota),
		FileName:        fileName,
		FileSize:        statSize,
		SHA1Hash:        sha1Hash,
		SHA1B64:         sha1B64,
		UploadTimestamp: uploadTimestamp,
		DeleteFromHost:  cfg.DeleteFromHost,
		DeleteFile:      hostdelete.Trash,
	})
	if err != nil {
		return "", 0, err
	}
	return mediaKey, 0, nil
}
