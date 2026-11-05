package backend

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// fsnotify event loop plus the debounce/settle/confirm/flush state machine
// that turns filesystem churn into stable upload batches.

const (
	watchDebounceDelay = 3 * time.Second
	watchStableDelay   = 500 * time.Millisecond
	watchFlushDelay    = 1 * time.Second
	watchRecentTTL     = 10 * time.Minute
)

func (s *WatchService) scanExisting(key string) {
	s.mu.Lock()
	state := s.states[key]
	if !s.started || state == nil || !state.folder.Enabled {
		s.mu.Unlock()
		return
	}
	var files []string
	s.walkLocked(state.folder.Path, func(string) {}, func(f string) {
		if s.filterFileLocked(f) {
			files = append(files, f)
		}
	})
	s.mu.Unlock()

	for _, f := range files {
		s.scheduleFile(f)
	}
}

func (s *WatchService) loop() {
	for {
		select {
		case ev, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			s.handleEvent(ev)
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			slog.Warn("watched folder error", "error", err)
		}
	}
}

func (s *WatchService) handleEvent(ev fsnotify.Event) {
	path := filepath.Clean(ev.Name)

	if ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
		s.cancelPendingPath(path)
		s.forgetWatchesUnder(path)
		return
	}

	if ev.Has(fsnotify.Create) {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			s.discoverDir(path)
			return
		}
	}

	if ev.Has(fsnotify.Create) || ev.Has(fsnotify.Write) {
		s.scheduleFile(path)
	}
}

func (s *WatchService) discoverDir(path string) {
	s.mu.Lock()
	state := s.ownerLocked(path)
	if state == nil || !state.folder.Enabled {
		s.mu.Unlock()
		return
	}
	var files []string
	s.walkLocked(path, func(dir string) { _ = s.addWatchLocked(dir) }, func(f string) { files = append(files, f) })
	s.mu.Unlock()

	for _, f := range files {
		s.scheduleFile(f)
	}
}

func (s *WatchService) scheduleFile(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.app == nil {
		return
	}
	if s.paused {
		return
	}
	state := s.ownerLocked(path)
	if state == nil || !state.folder.Enabled {
		return
	}
	if pf := s.pending[path]; pf != nil {
		pf.timer.Stop()
	}
	pf := &pendingFile{folder: state.folder.Path}
	pf.timer = time.AfterFunc(watchDebounceDelay, func() { s.settleFile(path, pf) })
	s.pending[path] = pf
}

func (s *WatchService) settleFile(path string, pf *pendingFile) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pending[path] != pf {
		return
	}
	state := s.ownerLocked(path)
	if state == nil || !state.folder.Enabled || state.folder.Path != pf.folder {
		delete(s.pending, path)
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() == 0 {
		delete(s.pending, path)
		return
	}
	if !s.filterFileLocked(path) {
		delete(s.pending, path)
		return
	}
	if t, ok := s.recent[path]; ok && time.Since(t) < watchRecentTTL {
		delete(s.pending, path)
		return
	}
	pf.timer = time.AfterFunc(watchStableDelay, func() {
		s.confirmStable(path, pf, info.Size(), info.ModTime())
	})
}

func (s *WatchService) confirmStable(path string, pf *pendingFile, size int64, mtime time.Time) {
	s.mu.Lock()

	if s.pending[path] != pf {
		s.mu.Unlock()
		return
	}
	now, err := os.Stat(path)
	if err != nil {
		delete(s.pending, path)
		s.mu.Unlock()
		return
	}
	if now.Size() != size || !now.ModTime().Equal(mtime) {
		pf.timer = time.AfterFunc(watchDebounceDelay, func() { s.settleFile(path, pf) })
		s.mu.Unlock()
		return
	}

	delete(s.pending, path)
	state := s.ownerLocked(path)
	if state == nil || !state.folder.Enabled {
		s.mu.Unlock()
		return
	}
	s.recent[path] = time.Now()
	s.pruneRecentLocked()
	state.buffer = append(state.buffer, path)
	if state.flushTimer == nil {
		state.flushTimer = time.AfterFunc(watchFlushDelay, func() { s.flush(state) })
	}
	s.mu.Unlock()
}

func (s *WatchService) flush(state *watchedFolderState) {
	s.mu.Lock()
	if s.paused {
		state.buffer = nil
		state.flushTimer = nil
		s.mu.Unlock()
		return
	}
	files := state.buffer
	state.buffer = nil
	state.flushTimer = nil
	album := state.folder.Album
	s.mu.Unlock()

	if len(files) == 0 || s.app == nil || s.uploads == nil {
		return
	}
	s.app.EmitEvent(EventWatcherBatchStarted, WatcherBatchEvent{Files: files, Album: album})
	s.uploads.UploadWatched(s.app, files, album)
}

func (s *WatchService) cancelPendingPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pf := s.pending[path]; pf != nil {
		pf.timer.Stop()
		delete(s.pending, path)
	}
}

func (s *WatchService) cancelPendingForLocked(folderKey string) {
	for p, pf := range s.pending {
		if s.stateKey(pf.folder) == folderKey {
			pf.timer.Stop()
			delete(s.pending, p)
		}
	}
}

func (s *WatchService) walkLocked(root string, onDir func(dir string), onFile func(path string)) {
	onDir(root)
	entries, err := os.ReadDir(root)
	if err != nil {
		slog.Warn("could not read watched directory", "path", root, "error", err)
		return
	}
	for _, e := range entries {
		p := filepath.Join(root, e.Name())
		if e.IsDir() {
			if s.excludeDirLocked(e.Name()) {
				continue
			}
			s.walkLocked(p, onDir, onFile)
			continue
		}
		if onFile != nil {
			onFile(p)
		}
	}
}

func (s *WatchService) watchTreeLocked(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a folder: %s", root)
	}
	if err := s.addWatchLocked(root); err != nil {
		return err
	}
	failed := 0
	s.walkLocked(root, func(dir string) {
		if dir == root {
			return
		}
		if err := s.addWatchLocked(dir); err != nil {
			failed++
		}
	}, nil)
	if failed > 0 {
		slog.Warn("some subfolders could not be watched (OS watch limit?)", "root", root, "count", failed)
	}
	return nil
}

func (s *WatchService) unwatchTreeLocked(folderKey string) {
	state := s.states[folderKey]
	if state == nil {
		return
	}
	for p := range s.watching {
		if matchesRoot(state.folder.Path, p) && s.ownerLocked(p) == state {
			_ = s.watcher.Remove(p)
			delete(s.watching, p)
		}
	}
}

func (s *WatchService) forgetWatchesUnder(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return
	}
	for p := range s.watching {
		if matchesRoot(path, p) {
			_ = s.watcher.Remove(p)
			delete(s.watching, p)
		}
	}
}

func (s *WatchService) addWatchLocked(path string) error {
	if s.watching[path] {
		return nil
	}
	if err := s.watcher.Add(path); err != nil {
		slog.Warn("could not watch directory", "path", path, "error", err)
		return err
	}
	s.watching[path] = true
	return nil
}

func (s *WatchService) filterFileLocked(path string) bool {
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") || strings.HasPrefix(base, "~$") {
		return false
	}
	cfg := CurrentConfig()
	return cfg.DisableUnsupportedFilesFilter || looksLikeSupportedMedia(path)
}

func (s *WatchService) excludeDirLocked(name string) bool {
	pattern := CurrentConfig().ExcludePattern
	return pattern != "" && name == pattern
}

func (s *WatchService) pruneRecentLocked() {
	for p, t := range s.recent {
		if time.Since(t) >= watchRecentTTL {
			delete(s.recent, p)
		}
	}
}

func (s *WatchService) ownerLocked(path string) *watchedFolderState {
	var best *watchedFolderState
	bestLen := -1
	for _, st := range s.states {
		if !matchesRoot(st.folder.Path, path) {
			continue
		}
		if l := len(st.folder.Path); l > bestLen {
			best, bestLen = st, l
		}
	}
	return best
}

func (s *WatchService) stateKey(path string) string {
	p := filepath.Clean(strings.TrimSpace(path))
	if p == "." {
		return ""
	}
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}

func matchesRoot(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		root = strings.ToLower(root)
		path = strings.ToLower(path)
	}
	if root == path {
		return true
	}
	return strings.HasPrefix(path, root+string(filepath.Separator))
}
