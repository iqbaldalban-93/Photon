package uploadx

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"app/internal/progress"
	"app/internal/resumebatch"
	"app/libweb/configx"
	"app/libweb/gphotos"
)

// Manager: the web-upload lifecycle — control surface (pause/cancel/skip),
// status aggregation for SSE, resumable-batch tracking and batch queueing.

type Manager struct {
	log  *logger
	data string

	mu       sync.Mutex
	status   Status
	cancel   chan struct{}
	canceled bool
	running  bool
	wg       sync.WaitGroup

	pauseMu  sync.Mutex
	paused   bool
	pauseSig chan struct{}

	skipMu  sync.Mutex
	skipSet map[string]bool

	resume resumebatch.Tracker

	queueMu sync.Mutex
	queue   []queuedUpload

	onBatchDone func()

	dataMu sync.RWMutex

	speed progress.Meter
	sent  map[int]sentMark
}

func NewManager(dataDir string, onBatchDone func()) *Manager {
	return &Manager{
		log:         &logger{},
		data:        dataDir,
		onBatchDone: onBatchDone,
	}
}

func (m *Manager) resumePath() string {
	m.dataMu.RLock()
	defer m.dataMu.RUnlock()
	return filepath.Join(m.data, "resumable.json")
}

func (m *Manager) SetDataDir(dir string) {
	m.dataMu.Lock()
	m.data = dir
	m.dataMu.Unlock()
}

func (m *Manager) saveResumable(b ResumableBatch) {
	_ = resumebatch.Save(m.resumePath(), b)
}

func (m *Manager) clearResumable() { _ = resumebatch.Clear(m.resumePath()) }

func (m *Manager) LoadResumable() *ResumableBatch {
	return resumebatch.LoadAndClear(m.resumePath())
}

func (m *Manager) PeekResumable() *ResumableBatch {
	return resumebatch.Peek(m.resumePath())
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.status
	s.Workers = append([]ThreadStatus{}, m.status.Workers...)
	s.Results = append([]FileUploadResult{}, m.status.Results...)
	s.Warnings = append([]Warning{}, m.status.Warnings...)
	s.Queued = append([]QueuedBatch{}, m.status.Queued...)
	return s
}

func (m *Manager) Progress() (running, paused bool, total, done, failed int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := &m.status
	return st.Running, st.Paused, st.Total, st.Done + st.Skipped + st.Failed, st.Failed
}

// Speed returns the current upload speed in bytes per second (0 when idle or
// paused). Call it on a steady tick; it feeds a short sliding window.
func (m *Manager) Speed() float64 {
	m.mu.Lock()
	active := m.status.Running && !m.status.Paused
	m.mu.Unlock()
	if !active {
		return 0
	}
	return m.speed.Rate()
}

// countSent converts a worker's cumulative BytesUploaded into deltas for the
// speed meter. Caller holds m.mu.
func (m *Manager) countSent(ts ThreadStatus) {
	if ts.Status != "uploading" || ts.BytesTotal <= 0 {
		delete(m.sent, ts.WorkerID)
		return
	}
	if m.sent == nil {
		m.sent = map[int]sentMark{}
	}
	prev := m.sent[ts.WorkerID]
	if prev.path != ts.FilePath {
		prev = sentMark{path: ts.FilePath}
	}
	if ts.BytesUploaded > prev.bytes {
		m.speed.Add(ts.BytesUploaded - prev.bytes)
	}
	m.sent[ts.WorkerID] = sentMark{path: ts.FilePath, bytes: ts.BytesUploaded}
}

func (m *Manager) setStatus(fn func(*Status)) {
	m.mu.Lock()
	fn(&m.status)
	m.mu.Unlock()
}

func (m *Manager) setWorker(ts ThreadStatus) {
	m.mu.Lock()
	m.countSent(ts)
	for i := range m.status.Workers {
		if m.status.Workers[i].WorkerID == ts.WorkerID {
			m.status.Workers[i] = ts
			m.mu.Unlock()
			return
		}
	}
	m.status.Workers = append(m.status.Workers, ts)
	m.mu.Unlock()
}

func (m *Manager) Cancel() {
	m.mu.Lock()
	if m.cancel != nil && !m.canceled {
		close(m.cancel)
		m.canceled = true
	}
	m.mu.Unlock()

	m.queueMu.Lock()
	m.queue = nil
	m.queueMu.Unlock()
	m.setStatus(func(s *Status) { s.Queued = nil })
}

func (m *Manager) Pause() {
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
	m.setStatus(func(s *Status) { s.Paused = true })
}

func (m *Manager) Resume() {
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
	m.setStatus(func(s *Status) { s.Paused = false })
}

func (m *Manager) SkipFile(path string) {
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

func (m *Manager) isSkipRequested(path string) bool {
	m.skipMu.Lock()
	defer m.skipMu.Unlock()
	return m.skipSet != nil && m.skipSet[path]
}

func (m *Manager) CancelQueuedFile(path string) {
	if path == "" {
		return
	}
	m.queueMu.Lock()
	for i := range m.queue {
		for j, p := range m.queue[i].Paths {
			if p != path {
				continue
			}
			m.queue[i].Paths = append(m.queue[i].Paths[:j], m.queue[i].Paths[j+1:]...)
			if len(m.queue[i].Paths) == 0 {
				m.queue = append(m.queue[:i], m.queue[i+1:]...)
			}
			m.queueMu.Unlock()
			m.refreshQueueStatus()
			return
		}
	}
	m.queueMu.Unlock()
}

func (m *Manager) refreshQueueStatus() {
	m.queueMu.Lock()
	q := make([]QueuedBatch, 0, len(m.queue))
	for i, b := range m.queue {
		q = append(q, QueuedBatch{Position: i + 1, Total: len(b.Paths), Paths: b.Paths, Album: b.Album})
	}
	m.queueMu.Unlock()
	m.setStatus(func(s *Status) { s.Queued = q })
}

func (m *Manager) Upload(paths []string, albumTarget string) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		m.queueMu.Lock()
		m.queue = append(m.queue, queuedUpload{Paths: paths, Album: albumTarget})
		m.queueMu.Unlock()
		m.refreshQueueStatus()
		return
	}
	m.running = true
	m.cancel = make(chan struct{})
	m.canceled = false
	m.mu.Unlock()

	m.setStatus(func(s *Status) {
		s.Running = true
		s.Paused = false
		s.Total = 0
		s.TotalBytes = 0
		s.Done = 0
		s.Failed = 0
		s.Skipped = 0
		s.Scanning = 0
		s.Workers = nil
		s.Results = nil
		s.Warnings = nil
		s.Album = nil
		s.StartedAt = time.Now()
		s.FinishedAt = time.Time{}
		s.LastErrorMsg = ""
	})

	m.skipMu.Lock()
	m.skipSet = make(map[string]bool)
	m.skipMu.Unlock()

	cancelled := func() bool {
		m.mu.Lock()
		c := m.canceled
		m.mu.Unlock()
		return c
	}

	targetPaths, fileSizes, err := filterFiles(paths, cancelled, func() {
		m.mu.Lock()
		m.status.Scanning++
		m.mu.Unlock()
	})
	if err != nil {
		if errors.Is(err, contextCanceled) {
			m.finishBatch()
			return
		}
		m.setStatus(func(s *Status) { s.LastErrorMsg = err.Error() })
		m.finishBatch()
		return
	}

	m.setStatus(func(s *Status) {
		s.Scanning = 0
		s.Total = len(targetPaths)
	})

	if len(targetPaths) == 0 {
		m.finishBatch()
		return
	}

	sharedApi, err := gphotos.NewApi()
	if err != nil {
		for _, p := range targetPaths {
			m.addResult(FileUploadResult{Path: p, FileName: filepath.Base(p), IsError: true, ErrorMessage: err.Error()})
		}
		m.setStatus(func(s *Status) { s.LastErrorMsg = err.Error() })
		m.finishBatch()
		return
	}

	m.startResumeTracking(targetPaths, albumTarget)

	var totalBytes int64
	for _, p := range targetPaths {
		if meta, ok := fileSizes[p]; ok {
			totalBytes += meta.Size
		}
	}
	m.setStatus(func(s *Status) { s.TotalBytes = totalBytes })

	cfg := configx.Get()
	threads := min(cfg.Photos.UploadThreads, configx.MaxUploadThreads)
	if threads < 1 {
		threads = 1
	}
	numWorkers := min(threads, len(targetPaths))

	concurrency := newAdaptiveConcurrency(numWorkers, func(limit, max int) {
		m.setStatus(func(s *Status) { s.Concurrency = [2]int{limit, max} })
	})
	m.setStatus(func(s *Status) { s.Concurrency = [2]int{numWorkers, numWorkers} })

	dep := workerDeps{
		m:           m,
		api:         sharedApi,
		concurrency: concurrency,
		fileSizes:   fileSizes,
		cancelled:   cancelled,
	}

	workChan := make(chan string, len(targetPaths))
	results := make(chan FileUploadResult, len(targetPaths))

	for i := range numWorkers {
		m.wg.Add(1)
		go m.worker(i, dep, workChan, results, m.getCancelChan())
	}

	go func() {
	LOOP:
		for _, p := range targetPaths {
			select {
			case <-m.getCancelChan():
				break LOOP
			case workChan <- p:
			}
		}
		close(workChan)
	}()

	go func() {
		go func() {
			m.wg.Wait()
			close(results)
		}()

		var mediaKeys, mediaPaths []string
		for result := range results {
			m.addResult(result)
			if !result.IsError && result.MediaKey != "" {
				mediaKeys = append(mediaKeys, result.MediaKey)
				mediaPaths = append(mediaPaths, result.Path)
			}
			m.markResumeCompleted(result.Path)
		}

		albumTarget = strings.TrimSpace(albumTarget)
		if albumTarget != "" && len(mediaKeys) > 0 {
			m.pinToAlbum(sharedApi, albumTarget, mediaKeys)
		}

		m.finishBatch()
	}()
}

func (m *Manager) getCancelChan() <-chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cancel
}

func (m *Manager) addResult(r FileUploadResult) {
	r.FileName = filepath.Base(r.Path)
	m.mu.Lock()
	m.status.Results = append(m.status.Results, r)
	if len(m.status.Results) > 500 {
		m.status.Results = m.status.Results[len(m.status.Results)-500:]
	}
	switch {
	case r.IsError:
		m.status.Failed++
		m.status.LastErrorMsg = r.ErrorMessage
	case r.Skipped:
		m.status.Skipped++
	default:
		m.status.Done++
	}
	m.mu.Unlock()
}

func (m *Manager) finishBatch() {
	m.mu.Lock()
	m.running = false
	m.status.Running = false
	m.status.FinishedAt = time.Now()
	m.mu.Unlock()

	m.stopResumeTracking()

	m.queueMu.Lock()
	if len(m.queue) == 0 {
		m.queueMu.Unlock()
		if m.onBatchDone != nil {
			m.onBatchDone()
		}
		return
	}
	next := m.queue[0]
	m.queue = m.queue[1:]
	m.queueMu.Unlock()
	m.refreshQueueStatus()

	m.Upload(next.Paths, next.Album)
}

func (m *Manager) startResumeTracking(targetPaths []string, album string) {
	m.resume.Start(m.resumePath(), targetPaths, album)
}

func (m *Manager) markResumeCompleted(paths ...string) {
	m.resume.MarkCompleted(paths...)
}

func (m *Manager) stopResumeTracking() {
	m.resume.Stop()
	m.clearResumable()
}
