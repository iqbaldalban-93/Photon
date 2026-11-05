package backend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"app/internal/resumebatch"
)

// UploadManager owns the desktop upload lifecycle: batch chains, queueing,
// pause/cancel/skip control, resumable-batch tracking and auto-retry.

type UploadManager struct {
	mu       sync.Mutex
	cancel   chan struct{}
	canceled bool
	running  bool
	app      AppInterface

	pauseMu  sync.Mutex
	paused   bool
	pauseSig chan struct{}

	skipMu  sync.Mutex
	skipSet map[string]bool

	resume resumebatch.Tracker

	chainMu      sync.Mutex
	chainSummary batchSummary
	chainRetried bool

	queueMu sync.Mutex
	queue   []queuedUpload
}

type failedEntry struct {
	path   string
	album  string
	dedupe bool
}

type batchSummary struct {
	uploaded      int
	failed        int
	skipped       int
	failedEntries []failedEntry
}

func (s batchSummary) total() int {
	return s.uploaded + s.failed + s.skipped
}

type queuedUpload struct {
	Paths        []string
	Album        string
	AlwaysDedupe bool
}

func NewUploadManager(app AppInterface) *UploadManager {
	return &UploadManager{
		app: app,
	}
}

func (m *UploadManager) Cancel() {
	m.mu.Lock()
	if m.cancel != nil && !m.canceled {
		close(m.cancel)
		m.canceled = true
	}
	m.mu.Unlock()

	m.queueMu.Lock()
	m.queue = nil
	m.queueMu.Unlock()
}

func (m *UploadManager) enqueue(app AppInterface, paths []string, albumTarget string, alwaysDeduplicate bool) {
	m.queueMu.Lock()
	m.queue = append(m.queue, queuedUpload{Paths: paths, Album: albumTarget, AlwaysDedupe: alwaysDeduplicate})
	position := len(m.queue)
	m.queueMu.Unlock()
	app.EmitEvent(EventUploadQueued, UploadQueuedEvent{Position: position, Total: len(paths), Paths: paths})
}

func (m *UploadManager) cancelQueuedFile(path string) (bool, int) {
	m.queueMu.Lock()
	defer m.queueMu.Unlock()

	for i := range m.queue {
		for j, p := range m.queue[i].Paths {
			if p != path {
				continue
			}
			m.queue[i].Paths = append(m.queue[i].Paths[:j], m.queue[i].Paths[j+1:]...)
			if len(m.queue[i].Paths) == 0 {
				m.queue = append(m.queue[:i], m.queue[i+1:]...)
			}
			return true, len(m.queue)
		}
	}
	return false, len(m.queue)
}

func (m *UploadManager) CancelQueuedFile(app AppInterface, path string) {
	if path == "" {
		return
	}
	removed, remaining := m.cancelQueuedFile(path)
	if !removed {
		return
	}
	app.EmitEvent(EventQueuedFileCancelled, QueuedFileCancelledEvent{
		Path:            path,
		RemainingQueued: remaining,
	})
}

func (m *UploadManager) dequeueNext() (queuedUpload, bool) {
	m.queueMu.Lock()
	defer m.queueMu.Unlock()
	if len(m.queue) == 0 {
		return queuedUpload{}, false
	}
	next := m.queue[0]
	m.queue = m.queue[1:]
	return next, true
}

func (m *UploadManager) finishBatch(app AppInterface) {
	m.stopResumeTracking()

	cancelled := m.isCancelled()

	m.mu.Lock()
	if next, ok := m.dequeueNext(); ok && !cancelled {
		m.mu.Unlock()
		m.startBatch(app, next.Paths, next.Album, next.AlwaysDedupe)
		return
	}

	summary := m.peekChainSummary()
	if !cancelled && summary.failed > 0 && !m.chainRetried && CurrentConfig().AutoRetryFailed {
		m.chainRetried = true
		failed := m.dropChainFailures()
		m.mu.Unlock()

		if len(failed) == 0 {
			m.finalizeChain(app, false)
			return
		}
		app.EmitEvent(EventUploadAutoRetry, summary.failed)
		app.GetLogger().Info(fmt.Sprintf("auto-retrying %d failed file(s) in %v", summary.failed, autoRetryCooldown))
		time.AfterFunc(autoRetryCooldown, func() {
			if m.isCancelled() {
				m.finalizeChain(app, true)
				return
			}
			type retryKey struct {
				album  string
				dedupe bool
			}
			groups := make(map[retryKey][]string)
			for _, e := range failed {
				k := retryKey{album: e.album, dedupe: e.dedupe}
				groups[k] = append(groups[k], e.path)
			}
			first := true
			for k, paths := range groups {
				if first {
					first = false
					m.startBatch(app, paths, k.album, k.dedupe)
				} else {
					m.enqueue(app, paths, k.album, k.dedupe)
				}
			}
		})
		return
	}
	m.mu.Unlock()

	m.finalizeChain(app, false)
}

func (m *UploadManager) finalizeChain(app AppInterface, cancelled bool) {
	m.mu.Lock()
	m.running = false
	m.chainRetried = false
	m.mu.Unlock()

	summary := m.takeChainSummary()
	app.EmitEvent(EventUploadStop, nil)
	if summary.total() > 0 {
		notifyBatchChainComplete(summary, cancelled || m.isCancelled())
	}
}

func (m *UploadManager) addChainSummary(s batchSummary) {
	m.chainMu.Lock()
	m.chainSummary.uploaded += s.uploaded
	m.chainSummary.failed += s.failed
	m.chainSummary.skipped += s.skipped
	m.chainSummary.failedEntries = append(m.chainSummary.failedEntries, s.failedEntries...)
	m.chainMu.Unlock()
}

func (m *UploadManager) peekChainSummary() batchSummary {
	m.chainMu.Lock()
	defer m.chainMu.Unlock()
	return m.chainSummary
}

func (m *UploadManager) dropChainFailures() []failedEntry {
	m.chainMu.Lock()
	defer m.chainMu.Unlock()
	entries := m.chainSummary.failedEntries
	m.chainSummary.failed = 0
	m.chainSummary.failedEntries = nil
	return entries
}

func (m *UploadManager) takeChainSummary() batchSummary {
	m.chainMu.Lock()
	defer m.chainMu.Unlock()
	s := m.chainSummary
	m.chainSummary = batchSummary{}
	return s
}

func (m *UploadManager) Pause() {
	m.pauseMu.Lock()
	if m.paused {
		m.pauseMu.Unlock()
		return
	}
	m.paused = true
	if m.pauseSig == nil {
		m.pauseSig = make(chan struct{})
	}
	m.pauseMu.Unlock()
	m.app.EmitEvent(EventUploadPauseState, true)
}

func (m *UploadManager) Resume() {
	m.pauseMu.Lock()
	if !m.paused {
		m.pauseMu.Unlock()
		return
	}
	m.paused = false
	if m.pauseSig != nil {
		close(m.pauseSig)
		m.pauseSig = nil
	}
	m.pauseMu.Unlock()
	m.app.EmitEvent(EventUploadPauseState, false)
}

func (m *UploadManager) waitIfPaused() bool {
	for {
		m.pauseMu.Lock()
		if !m.paused {
			m.pauseMu.Unlock()
			return false
		}
		sig := m.pauseSig
		m.pauseMu.Unlock()

		// Resume closes sig; cancel closes the cancel channel. Pause() never
		// replaces sig while paused, so no polling fallback is needed.
		select {
		case <-m.getCancelChan():
			return true
		case <-sig:
		}
	}
}

func (m *UploadManager) isCancelled() bool {
	m.mu.Lock()
	cancel := m.cancel
	m.mu.Unlock()
	if cancel == nil {
		return false
	}
	select {
	case <-cancel:
		return true
	default:
		return false
	}
}

func (m *UploadManager) getCancelChan() <-chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cancel
}

func (m *UploadManager) SkipFile(path string) {
	if path == "" {
		return
	}
	m.skipMu.Lock()
	if m.skipSet == nil {
		m.skipSet = make(map[string]bool)
	}
	m.skipSet[path] = true
	m.skipMu.Unlock()
}

func (m *UploadManager) isSkipRequested(path string) bool {
	m.skipMu.Lock()
	defer m.skipMu.Unlock()
	return m.skipSet != nil && m.skipSet[path]
}

func (m *UploadManager) startResumeTracking(targetPaths []string, album string) {
	m.resume.Start(resumeFilePath(), targetPaths, album)
}

func (m *UploadManager) markResumeCompleted(paths ...string) {
	m.resume.MarkCompleted(paths...)
}

func (m *UploadManager) stopResumeTracking() {
	m.resume.Stop()
	clearResumableBatch()
}

const autoRetryCooldown = 10 * time.Second

func (m *UploadManager) Upload(app AppInterface, paths []string, albumTarget string) {
	m.upload(app, paths, albumTarget, false)
}

func (m *UploadManager) UploadWatched(app AppInterface, paths []string, albumTarget string) {
	m.upload(app, paths, albumTarget, true)
}

func (m *UploadManager) upload(app AppInterface, paths []string, albumTarget string, alwaysDeduplicate bool) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		m.enqueue(app, paths, albumTarget, alwaysDeduplicate)
		return
	}
	m.running = true
	m.cancel = make(chan struct{})
	m.canceled = false
	m.mu.Unlock()

	m.chainMu.Lock()
	m.chainSummary = batchSummary{}
	m.chainMu.Unlock()

	m.startBatch(app, paths, albumTarget, alwaysDeduplicate)
}

func (m *UploadManager) startBatch(app AppInterface, paths []string, albumTarget string, alwaysDeduplicate bool) {
	m.pauseMu.Lock()
	m.paused = false
	m.pauseSig = nil
	m.pauseMu.Unlock()

	app.EmitEvent(EventUploadStart, UploadBatchStart{Phase: "preflight"})

	m.skipMu.Lock()
	m.skipSet = make(map[string]bool)
	m.skipMu.Unlock()

	var scannedCount int64
	var lastScanEmit int64
	onScanned := func() {
		scannedCount++
		if scannedCount-lastScanEmit >= 200 {
			lastScanEmit = scannedCount
			app.EmitEvent(EventUploadPreflightProgress, UploadPreflightProgress{Scanned: int(scannedCount)})
		}
	}

	var unreadable []string
	onSkip := func(dir string, _ error) { unreadable = append(unreadable, dir) }
	targetPaths, fileSizes, err := filterGooglePhotosFilesWithCancel(CurrentConfig(), paths, m.isCancelled, onScanned, onSkip)
	if len(unreadable) > 0 {
		app.EmitEvent(EventUploadWarning, PreflightWarning{
			Paths:   unreadable,
			Code:    "folder-unreadable",
			Message: fmt.Sprintf("%d folder(s) could not be read and were not scanned", len(unreadable)),
		})
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			m.finishPreflight(app)
			return
		}
		app.EmitEvent(EventFileStatus, FileUploadResult{
			IsError:      true,
			Error:        err,
			ErrorMessage: err.Error(),
		})
		m.finishPreflight(app)
		return
	}

	workItems := make([]UploadWorkItem, 0, len(targetPaths))
	for _, path := range targetPaths {
		workItems = append(workItems, UploadWorkItem{Kind: UploadWorkSingle, Single: &SingleMedia{Path: path}})
	}

	emitUploadPreflight(app, targetPaths)

	if len(workItems) == 0 {
		m.finishPreflight(app)
		return
	}

	sharedApi, err := NewApi()
	if err != nil {
		for _, item := range workItems {
			app.EmitEvent(EventFileStatus, FileUploadResult{
				IsError:      true,
				Error:        err,
				ErrorMessage: err.Error(),
				Path:         uploadWorkPrimaryPath(item),
			})
		}
		m.finishBatch(app)
		return
	}

	m.startResumeTracking(targetPaths, albumTarget)

	go func() {
		var totalBytes int64
		for _, item := range workItems {
			path := uploadWorkPrimaryPath(item)
			if path == "" {
				continue
			}
			if meta, ok := fileSizes[path]; ok {
				totalBytes += meta.Size
			} else if info, err := os.Stat(path); err == nil {
				totalBytes += info.Size()
			}
		}
		app.EmitEvent(EventUploadTotalBytes, totalBytes)
	}()

	cfg := CurrentConfig()
	if cfg.UploadThreads < 1 {
		cfg.UploadThreads = 1
	}

	numWorkers := min(cfg.UploadThreads, len(workItems))

	concurrency := NewAdaptiveConcurrency(numWorkers, func(limit, max int) {
		app.EmitEvent(EventUploadConcurrency, UploadConcurrencyStatus{Limit: limit, Max: max})
	})

	deps := workerDeps{
		api:           sharedApi,
		app:           app,
		paused:        m.waitIfPaused,
		skipRequested: m.isSkipRequested,
		concurrency:   concurrency,
		fileMetaFor: func(path string) (fileMeta, bool) {
			meta, ok := fileSizes[path]
			return meta, ok
		},
		cfg:               cfg,
		AlwaysDeduplicate: alwaysDeduplicate,
	}

	cancelCh := m.getCancelChan()

	workChan := make(chan UploadWorkItem, len(workItems))
	results := make(chan FileUploadResult, len(workItems))

	var wg sync.WaitGroup
	for i := range numWorkers {
		wg.Add(1)
		go startUploadWorker(i, deps, workChan, results, cancelCh, &wg)
	}

	go func() {
	LOOP:
		for _, item := range workItems {
			select {
			case <-cancelCh:
				break LOOP
			case workChan <- item:
			}
		}
		close(workChan)
	}()

	go func() {
		go func() {
			wg.Wait()
			close(results)
		}()

		var mediaKeys []string
		var mediaPaths []string
		var outcomes batchSummary
		for result := range results {
			app.EmitEvent(EventFileStatus, result)
			paths := result.Paths
			if len(paths) == 0 && result.Path != "" {
				paths = []string{result.Path}
			}
			if !result.IsError {
				m.markResumeCompleted(paths...)
			}
			if !result.IsError && result.MediaKey != "" {
				mediaKeys = append(mediaKeys, result.MediaKey)
				mediaPaths = append(mediaPaths, result.Path)
			}
			if result.IsError {
				outcomes.failed++
				if result.Path != "" {
					outcomes.failedEntries = append(outcomes.failedEntries, failedEntry{path: result.Path, album: albumTarget, dedupe: alwaysDeduplicate})
				}
				app.GetLogger().Error(fmt.Sprintf("upload error: %v", result.Error))
			} else if result.Skipped {
				outcomes.skipped++
				app.GetLogger().Info(fmt.Sprintf("upload skipped: %v", result.Path))
			} else {
				outcomes.uploaded++
				app.GetLogger().Info(fmt.Sprintf("upload success: %v", result.Path))
			}
		}
		m.addChainSummary(outcomes)

		if albumTarget = strings.TrimSpace(albumTarget); albumTarget != "" && len(mediaKeys) > 0 {
			m.addFilesToAlbum(app, sharedApi, nil, albumTarget, mediaKeys, mediaPaths)
		}

		m.finishBatch(app)
	}()
}

func (m *UploadManager) addFilesToAlbum(app AppInterface, api *Api, cancel <-chan struct{}, albumTarget string, mediaKeys []string, paths []string) {
	albumManager := NewAlbumManager(api, app, cancel)
	albumKeys, err := albumManager.AddToAlbum(mediaKeys, paths, albumTarget)
	if err != nil {
		app.EmitEvent(EventAlbumError, AlbumError{
			AlbumName: albumTarget,
			Error:     err.Error(),
			IsError:   true,
		})
		app.GetLogger().Error(fmt.Sprintf("album operation failed: %v", err))
		return
	}
	app.GetLogger().Info(fmt.Sprintf("album operation complete: keys=%v", albumKeys))
}

func (m *UploadManager) finishPreflight(app AppInterface) {
	m.finishBatch(app)
}

func emitUploadPreflight(app AppInterface, targetPaths []string) {
	app.EmitEvent(EventUploadStart, UploadBatchStart{
		Total:      len(targetPaths),
		TotalBytes: 0,
		Files:      targetPaths,
		Phase:      "ready",
	})
}
