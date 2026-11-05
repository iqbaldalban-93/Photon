package resumebatch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"app/internal/fsx"
)

type Batch struct {
	Paths   []string `json:"paths"`
	Album   string   `json:"album"`
	Total   int      `json:"total"`
	SavedAt int64    `json:"savedAt"`
}

func Save(path string, batch Batch) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	return fsx.WriteFileAtomic(path, data, 0o600)
}

func Clear(path string) error {
	return os.Remove(path)
}

func LoadAndClear(path string) *Batch {
	data, err := os.ReadFile(path)
	_ = os.Remove(path)
	if err != nil || len(data) == 0 {
		return nil
	}
	var batch Batch
	if err := json.Unmarshal(data, &batch); err != nil {
		return nil
	}
	if len(batch.Paths) == 0 {
		return nil
	}
	return &batch
}

func Peek(path string) *Batch {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var batch Batch
	if err := json.Unmarshal(data, &batch); err != nil || len(batch.Paths) == 0 {
		return nil
	}
	return &batch
}

type Tracker struct {
	mu        sync.Mutex
	path      string
	allPaths  []string
	album     string
	completed map[string]bool
	dirty     bool
	stopCh    chan struct{}
}

func (t *Tracker) Start(path string, paths []string, album string) {
	t.mu.Lock()
	t.path = path
	t.allPaths = paths
	t.album = album
	t.completed = make(map[string]bool, len(paths))
	t.dirty = true
	stopCh := make(chan struct{})
	t.stopCh = stopCh
	t.mu.Unlock()

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				t.Persist()
			}
		}
	}()
}

func (t *Tracker) MarkCompleted(paths ...string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.completed == nil {
		return
	}
	for _, p := range paths {
		if p != "" && !t.completed[p] {
			t.completed[p] = true
			t.dirty = true
		}
	}
}

func (t *Tracker) Persist() {
	t.mu.Lock()
	if !t.dirty {
		t.mu.Unlock()
		return
	}
	t.dirty = false
	remaining := make([]string, 0, len(t.allPaths))
	for _, p := range t.allPaths {
		if !t.completed[p] {
			remaining = append(remaining, p)
		}
	}
	album := t.album
	total := len(t.allPaths)
	path := t.path
	t.mu.Unlock()

	if len(remaining) == 0 {
		return
	}
	_ = Save(path, Batch{
		Paths:   remaining,
		Album:   album,
		Total:   total,
		SavedAt: time.Now().Unix(),
	})
}

func (t *Tracker) Stop() {
	t.mu.Lock()
	path := t.path
	if t.stopCh != nil {
		close(t.stopCh)
		t.stopCh = nil
	}
	t.allPaths = nil
	t.completed = nil
	t.dirty = false
	t.mu.Unlock()
	if path != "" {
		_ = os.Remove(path)
	}
}
