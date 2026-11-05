package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"app/libweb/gphotos"
)

// Index: the per-account in-memory library store with JSON persistence and
// sync-token bookkeeping. Mutations and queries live in sibling files.

type Item = gphotos.Item

type Album = gphotos.Album

type Index struct {
	mu     sync.RWMutex
	items  map[string]*Item
	albums map[string]*Album
	path   string

	syncToken   string
	resumeToken string
	initDone    bool

	lastSave time.Time

	version     uint64
	queryCache  map[string]QueryResult
	countsCache map[string]int
	statsCache  map[string]any
	statsAt     uint64
	countsAt    uint64

	watched    *WatchedStore
	watchedGen uint64
	countsWG   uint64
	countsAcct string
}

func (ix *Index) bump() {
	ix.version++
	ix.queryCache = nil
	ix.countsCache = nil
	ix.statsCache = nil
}

func (ix *Index) cacheQuery(key string, res QueryResult) {
	if ix.queryCache == nil {
		ix.queryCache = map[string]QueryResult{}
	}
	if len(ix.queryCache) > 64 {
		ix.queryCache = nil
	}
	ix.queryCache[key] = res
}

func (ix *Index) SetWatchedStore(w *WatchedStore) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.watched = w
	ix.watchedGen++
}

func isLibraryVideo(it *Item) bool {
	return it.Type == "video" && it.TrashTimestamp == 0
}

func (ix *Index) isWatched(it *Item) bool {
	return it.Type == "video" && ix.watched.Has(identityKey(it))
}

func identityKey(it *Item) string {
	if it.DedupKey != "" {
		return it.DedupKey
	}
	return it.MediaKey
}

func NewIndexAt(path string) (*Index, error) {
	ix := &Index{
		items:  map[string]*Item{},
		albums: map[string]*Album{},
		path:   path,
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	ix.loadFrom(path)
	return ix, nil
}

func (ix *Index) loadFrom(path string) {
	ix.items = map[string]*Item{}
	ix.albums = map[string]*Album{}
	ix.syncToken, ix.resumeToken, ix.initDone = "", "", false
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var snap snapshot
	if json.Unmarshal(raw, &snap) != nil {
		return
	}
	for _, it := range snap.Items {
		ix.items[it.MediaKey] = it
	}
	for _, al := range snap.Albums {
		ix.albums[al.Key] = al
	}
	ix.syncToken = snap.SyncToken
	ix.resumeToken = snap.ResumeToken
	ix.initDone = snap.InitDone
}

func (ix *Index) SwitchTo(path string) error {
	if err := ix.Save(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.path = path
	ix.loadFrom(path)
	ix.lastSave = time.Now()
	ix.watchedGen++
	ix.bump()
	return nil
}

func (ix *Index) Version() uint64 {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.version
}

type snapshot struct {
	Items       []*Item  `json:"items"`
	Albums      []*Album `json:"albums"`
	SyncToken   string   `json:"syncToken"`
	ResumeToken string   `json:"resumeToken"`
	InitDone    bool     `json:"initDone"`
}

func (ix *Index) Save() error {
	ix.mu.Lock()
	snap := snapshot{
		Items:       make([]*Item, 0, len(ix.items)),
		Albums:      make([]*Album, 0, len(ix.albums)),
		SyncToken:   ix.syncToken,
		ResumeToken: ix.resumeToken,
		InitDone:    ix.initDone,
	}
	for _, it := range ix.items {
		snap.Items = append(snap.Items, it)
	}
	for _, al := range ix.albums {
		snap.Albums = append(snap.Albums, al)
	}
	ix.lastSave = time.Now()
	path := ix.path
	ix.mu.Unlock()

	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (ix *Index) SaveIfStale(d time.Duration) {
	ix.mu.RLock()
	stale := time.Since(ix.lastSave) >= d
	ix.mu.RUnlock()
	if stale {
		_ = ix.Save()
	}
}

func (ix *Index) Tokens() (syncToken, resumeToken string, initDone bool) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.syncToken, ix.resumeToken, ix.initDone
}

func (ix *Index) SetTokens(syncToken, resumeToken string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.syncToken = syncToken
	if resumeToken != "" || syncToken != "" {
		ix.resumeToken = resumeToken
	}
	if syncToken != "" && resumeToken == "" {
		ix.resumeToken = ""
	}
}

func (ix *Index) SetInitDone() {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.initDone = true
}

func (ix *Index) ResetForFullSync() {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.syncToken = ""
	ix.resumeToken = ""
	ix.initDone = false
	ix.bump()
}
