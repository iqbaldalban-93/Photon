package backend

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watched-folder lifecycle and CRUD: attach/shutdown, add/remove/configure
// folders (persisted in the app config), and the global pause switch.

type WatcherBatchEvent struct {
	Files []string `json:"Files"`
	Album string   `json:"Album"`
}

type WatchService struct {
	mu       sync.Mutex
	app      AppInterface
	uploads  *UploadManager
	watcher  *fsnotify.Watcher
	started  bool
	states   map[string]*watchedFolderState
	watching map[string]bool
	pending  map[string]*pendingFile
	recent   map[string]time.Time
	paused   bool
}

type watchedFolderState struct {
	folder     WatchedFolder
	buffer     []string
	flushTimer *time.Timer
}

type pendingFile struct {
	folder string
	timer  *time.Timer
}

func AttachWatchService(s *WatchService, app AppInterface, uploads *UploadManager) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	s.mu.Lock()
	s.app = app
	s.uploads = uploads
	s.watcher = watcher
	s.states = make(map[string]*watchedFolderState)
	s.watching = make(map[string]bool)
	s.pending = make(map[string]*pendingFile)
	s.recent = make(map[string]time.Time)
	for _, f := range CurrentConfig().WatchedFolders {
		state := &watchedFolderState{folder: f}
		s.states[s.stateKey(f.Path)] = state
		if f.Enabled {
			if err := s.watchTreeLocked(f.Path); err != nil {
				slog.Warn("watched folder could not be watched", "path", f.Path, "error", err)
			}
		}
	}
	s.started = true
	var scanKeys []string
	for k, st := range s.states {
		if st.folder.Enabled {
			scanKeys = append(scanKeys, k)
		}
	}
	s.mu.Unlock()

	go s.loop()
	for _, k := range scanKeys {
		go s.scanExisting(k)
	}
	return nil
}

func (s *WatchService) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return
	}
	s.started = false
	for p := range s.watching {
		_ = s.watcher.Remove(p)
	}
	for _, pf := range s.pending {
		pf.timer.Stop()
	}
	for _, st := range s.states {
		if st.flushTimer != nil {
			st.flushTimer.Stop()
		}
	}
	_ = s.watcher.Close()
}

func (s *WatchService) AddWatchedFolder(path, album string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("folder path cannot be empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid folder path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("folder not accessible: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a folder: %s", abs)
	}
	album = strings.TrimSpace(album)

	key := s.stateKey(abs)

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range CurrentConfig().WatchedFolders {
		if s.stateKey(existing.Path) == key {
			return fmt.Errorf("folder is already being watched")
		}
	}

	folder := WatchedFolder{Path: abs, Album: album, Enabled: true}
	if s.started {
		s.states[key] = &watchedFolderState{folder: folder}
		if err := s.watchTreeLocked(abs); err != nil {
			s.unwatchTreeLocked(key)
			delete(s.states, key)
			return fmt.Errorf("could not watch folder: %w", err)
		}
	}
	if err := setConfig(func(c *Config) {
		c.WatchedFolders = append(c.WatchedFolders, folder)
	}); err != nil {
		if s.started {
			s.unwatchTreeLocked(key)
			delete(s.states, key)
		}
		return err
	}
	if s.started {
		go s.scanExisting(key)
	}
	return nil
}

func (s *WatchService) RemoveWatchedFolder(path string) error {
	key := s.stateKey(path)
	if key == "" || key == "." {
		return fmt.Errorf("folder path cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	folders := CurrentConfig().WatchedFolders
	updated := slices.DeleteFunc(slices.Clone(folders), func(f WatchedFolder) bool {
		return s.stateKey(f.Path) == key
	})
	if len(updated) == len(folders) {
		return fmt.Errorf("folder is not being watched")
	}
	if err := setConfig(func(c *Config) { c.WatchedFolders = updated }); err != nil {
		return err
	}
	if s.started {
		s.unwatchTreeLocked(key)
		s.cancelPendingForLocked(key)
		state := s.states[key]
		delete(s.states, key)
		if state != nil && state.flushTimer != nil {
			state.flushTimer.Stop()
		}
	}
	return nil
}

func (s *WatchService) SetWatchedFolderAlbum(path, album string) error {
	key := s.stateKey(path)
	album = strings.TrimSpace(album)

	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	updated := slices.Clone(CurrentConfig().WatchedFolders)
	for i := range updated {
		if s.stateKey(updated[i].Path) == key {
			found = true
			updated[i].Album = album
		}
	}
	if !found {
		return fmt.Errorf("folder is not being watched")
	}
	if err := setConfig(func(c *Config) { c.WatchedFolders = updated }); err != nil {
		return err
	}
	if state := s.states[key]; state != nil {
		state.folder.Album = album
	}
	return nil
}

func (s *WatchService) SetWatchedFolderEnabled(path string, enabled bool) error {
	key := s.stateKey(path)
	if key == "" || key == "." {
		return fmt.Errorf("folder path cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	updated := slices.Clone(CurrentConfig().WatchedFolders)
	for i := range updated {
		if s.stateKey(updated[i].Path) == key {
			found = true
			updated[i].Enabled = enabled
		}
	}
	if !found {
		return fmt.Errorf("folder is not being watched")
	}
	state := s.states[key]
	if enabled && s.started && state != nil {
		if err := s.watchTreeLocked(state.folder.Path); err != nil {
			return fmt.Errorf("could not watch folder: %w", err)
		}
	}
	if err := setConfig(func(c *Config) { c.WatchedFolders = updated }); err != nil {
		if enabled && s.started && state != nil {
			s.unwatchTreeLocked(key)
		}
		return err
	}
	if !s.started {
		return nil
	}
	if enabled {
		if state != nil {
			state.folder.Enabled = true
			go s.scanExisting(key)
		}
		return nil
	}
	s.unwatchTreeLocked(key)
	s.cancelPendingForLocked(key)
	if state != nil {
		state.folder.Enabled = false
		if state.flushTimer != nil {
			state.flushTimer.Stop()
			state.flushTimer = nil
		}
		state.buffer = nil
	}
	return nil
}

func (s *WatchService) IsPaused() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.paused
}

func (s *WatchService) SetPaused(paused bool) {
	s.mu.Lock()
	if !s.started || s.paused == paused {
		s.mu.Unlock()
		return
	}
	s.paused = paused
	var scanKeys []string
	if paused {
		for p, pf := range s.pending {
			pf.timer.Stop()
			delete(s.pending, p)
		}
		for _, st := range s.states {
			if st.flushTimer != nil {
				st.flushTimer.Stop()
				st.flushTimer = nil
			}
			st.buffer = nil
		}
	} else {
		for k, st := range s.states {
			if st.folder.Enabled {
				scanKeys = append(scanKeys, k)
			}
		}
	}
	s.mu.Unlock()
	for _, k := range scanKeys {
		go s.scanExisting(k)
	}
}
