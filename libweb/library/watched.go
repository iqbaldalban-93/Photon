package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"app/internal/fsx"
)

type WatchedStore struct {
	mu      sync.RWMutex
	path    string
	account func() string
	data    map[string]map[string]int64
}

type watchedFile struct {
	Version  int                         `json:"version"`
	Accounts map[string]map[string]int64 `json:"accounts"`
}

func NewWatchedStore(dataDir string, account func() string) (*WatchedStore, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	w := &WatchedStore{
		path:    filepath.Join(dataDir, "watched.json"),
		account: account,
		data:    map[string]map[string]int64{},
	}
	raw, err := os.ReadFile(w.path)
	if err != nil {
		return w, nil
	}
	var f watchedFile
	if json.Unmarshal(raw, &f) == nil && f.Accounts != nil {
		w.data = f.Accounts
	}
	return w, nil
}

func (w *WatchedStore) Account() string {
	if w == nil || w.account == nil {
		return ""
	}
	return w.account()
}

func (w *WatchedStore) Has(key string) bool {
	if w == nil {
		return false
	}
	acct := w.Account()
	w.mu.RLock()
	defer w.mu.RUnlock()
	_, ok := w.data[acct][key]
	return ok
}

func (w *WatchedStore) Count() int {
	if w == nil {
		return 0
	}
	acct := w.Account()
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.data[acct])
}

func (w *WatchedStore) Set(keys []string, watched bool) (int, error) {
	if w == nil || len(keys) == 0 {
		return 0, nil
	}
	acct := w.Account()
	now := time.Now().UnixMilli()

	w.mu.Lock()
	m := w.data[acct]
	changed := 0
	for _, k := range keys {
		if k == "" {
			continue
		}
		_, has := m[k]
		switch {
		case watched && !has:
			if m == nil {
				m = map[string]int64{}
				w.data[acct] = m
			}
			m[k] = now
			changed++
		case !watched && has:
			delete(m, k)
			changed++
		}
	}
	if changed == 0 {
		w.mu.Unlock()
		return 0, nil
	}
	raw, err := json.Marshal(watchedFile{Version: 1, Accounts: w.data})
	if err == nil {
		err = fsx.WriteFileAtomic(w.path, raw, 0o600)
	}
	w.mu.Unlock()
	return changed, err
}
