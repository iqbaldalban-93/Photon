package backend

import (
	"sync"

	"app/internal/progress"
)

// TrayTracker is the event reducer behind the tray: it consumes the same
// events the frontend sees and maintains a compact upload-progress snapshot.

type TrayTracker struct {
	mu       sync.Mutex
	active   bool
	paused   bool
	phase    string
	scanned  int
	total    int
	done     int
	failed   int
	hadError bool

	totalBytes     int64
	completedBytes int64
	batchReported  int64
	fileBytes      map[string]int64
	workers        map[int]ThreadStatus

	// Bytes actually pushed over the network, fed by per-worker progress
	// events, so the speed readout keeps working across files and retries.
	speed progress.Meter
	sent  map[int]sentMark

	// Batches that were added while an upload was running. The backend
	// holds them in a queue and only emits uploadStart once they begin, so
	// the tray has to count them itself to stay in step with the main window.
	queued [][]string
}

type sentMark struct {
	path  string
	bytes int64
}

// countSent turns a worker's cumulative BytesUploaded into byte deltas for
// the speed meter. Caller holds t.mu.
func (t *TrayTracker) countSent(d ThreadStatus) {
	if d.Status != "uploading" || d.BytesTotal <= 0 {
		delete(t.sent, d.WorkerID)
		return
	}
	if t.sent == nil {
		t.sent = map[int]sentMark{}
	}
	prev := t.sent[d.WorkerID]
	if prev.path != d.FilePath {
		prev = sentMark{path: d.FilePath}
	}
	if d.BytesUploaded > prev.bytes {
		t.speed.Add(d.BytesUploaded - prev.bytes)
	}
	t.sent[d.WorkerID] = sentMark{path: d.FilePath, bytes: d.BytesUploaded}
}

func (t *TrayTracker) queuedFiles() int {
	n := 0
	for _, b := range t.queued {
		n += len(b)
	}
	return n
}

func (t *TrayTracker) resetBytes() {
	t.totalBytes, t.completedBytes, t.batchReported = 0, 0, 0
	t.fileBytes = map[string]int64{}
	t.workers = map[int]ThreadStatus{}
	t.sent = nil
	t.speed.Reset()
}

func (t *TrayTracker) activeBytes() int64 {
	var n int64
	for _, w := range t.workers {
		switch {
		case w.Status == "uploading" && w.BytesTotal > 0:
			n += w.BytesUploaded
		case w.Status == "finalizing" && w.FilePath != "":
			n += t.fileBytes[w.FilePath]
		}
	}
	return n
}

func (t *TrayTracker) Handle(event string, data any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	switch event {
	case "uploadStart":
		d, ok := data.(UploadBatchStart)
		if !ok {
			return
		}
		if d.Phase == "preflight" {
			if !t.active {
				// First batch of a new chain: start from zero.
				t.active, t.paused, t.phase = true, false, "scanning"
				t.scanned, t.total, t.done, t.failed, t.hadError = 0, 0, 0, 0, false
				t.queued = nil
				t.resetBytes()
				return
			}
			// A queued batch is starting inside a running chain. Keep the
			// chain totals; only the per-batch counters restart.
			t.scanned = 0
			t.batchReported = 0
			if len(t.queued) > 0 {
				t.queued = t.queued[1:]
			}
			return
		}
		t.active, t.phase = true, "uploading"
		t.total += d.Total
		if t.fileBytes == nil {
			t.resetBytes()
		}
		t.totalBytes += d.TotalBytes
	case "uploadQueued":
		if d, ok := data.(UploadQueuedEvent); ok && t.active {
			t.queued = append(t.queued, d.Paths)
		}
	case "queuedFileCancelled":
		if d, ok := data.(QueuedFileCancelledEvent); ok {
			t.dropQueuedPath(d.Path)
		}
	case "uploadPreflightProgress":
		if d, ok := data.(UploadPreflightProgress); ok {
			t.scanned = d.Scanned
		}
	case "uploadTotalBytes":
		if n, ok := data.(int64); ok {
			if t.fileBytes == nil {
				t.resetBytes()
			}
			t.totalBytes += n - t.batchReported
			t.batchReported = n
		}
	case "uploadTotalBytesDelta":
		if n, ok := data.(int64); ok {
			t.totalBytes += n
		}
	case "ThreadStatus":
		d, ok := data.(ThreadStatus)
		if !ok || !t.active {
			return
		}
		if t.fileBytes == nil {
			t.resetBytes()
		}
		if d.Status == "uploading" && d.FilePath != "" && d.BytesTotal > 0 {
			t.fileBytes[d.FilePath] = d.BytesTotal
		}
		t.workers[d.WorkerID] = d
		t.countSent(d)
	case "FileStatus":
		d, ok := data.(FileUploadResult)
		if !ok || !t.active {
			return
		}
		t.done++
		if d.IsError {
			t.failed++
		}
		if t.fileBytes == nil {
			t.resetBytes()
		}
		if !d.IsError {
			if n := t.fileBytes[d.Path]; n > 0 {
				t.completedBytes += n
			}
		}
		paths := d.Paths
		if len(paths) == 0 {
			paths = []string{d.Path}
		}
		for _, p := range paths {
			delete(t.fileBytes, p)
		}
	case "uploadPauseState":
		if p, ok := data.(bool); ok {
			t.paused = p
		}
	case "uploadStop":
		t.hadError = t.failed > 0
		t.active, t.paused = false, false
		t.workers = map[int]ThreadStatus{}
		t.sent = nil
		t.speed.Reset()
		t.queued = nil
	}
}

func (t *TrayTracker) dropQueuedPath(path string) {
	for i := range t.queued {
		for j, p := range t.queued[i] {
			if p != path {
				continue
			}
			b := append([]string(nil), t.queued[i][:j]...)
			b = append(b, t.queued[i][j+1:]...)
			if len(b) == 0 {
				t.queued = append(t.queued[:i], t.queued[i+1:]...)
			} else {
				t.queued[i] = b
			}
			return
		}
	}
}

type trayUploadSnapshot struct {
	Active, Paused, HadError bool
	Phase                    string
	Scanned, Total, Done     int
	Failed                   int
	TotalBytes, DoneBytes    int64
	SpeedBps                 float64
}

func (t *TrayTracker) snapshot() trayUploadSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	total := t.total + t.queuedFiles()
	done := t.done
	if total > 0 && done > total {
		done = total
	}
	doneBytes := t.completedBytes + t.activeBytes()
	if t.totalBytes > 0 && doneBytes > t.totalBytes {
		doneBytes = t.totalBytes
	}
	return trayUploadSnapshot{
		Active: t.active, Paused: t.paused, HadError: t.hadError, Phase: t.phase,
		Scanned: t.scanned, Total: total, Done: done, Failed: t.failed,
		TotalBytes: t.totalBytes, DoneBytes: doneBytes,
		SpeedBps: t.speed.Rate(),
	}
}
