package library

import "app/libweb/gphotos"

// Local mutation API: applying remote sync pages and mutating items/albums
// in response to user actions from the web UI.

func (ix *Index) MarkWatched(mediaKeys []string, watched bool) (int, error) {
	ix.mu.RLock()
	ids := make([]string, 0, len(mediaKeys))
	for _, mk := range mediaKeys {
		if it, ok := ix.items[mk]; ok && it.Type == "video" {
			ids = append(ids, identityKey(it))
		}
	}
	store := ix.watched
	ix.mu.RUnlock()
	if store == nil {
		return 0, nil
	}
	n, err := store.Set(ids, watched)
	if n > 0 {
		ix.mu.Lock()
		ix.watchedGen++
		ix.mu.Unlock()
	}
	return n, err
}

func (ix *Index) ApplyPage(page *gphotos.SyncPage) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	for _, it := range page.Items {
		ix.items[it.MediaKey] = it
	}
	for _, al := range page.Albums {
		ix.albums[al.Key] = al
	}
	for _, key := range page.Deletions {
		delete(ix.items, key)
		if _, wasAlbum := ix.albums[key]; wasAlbum {
			for _, it := range ix.items {
				if len(it.CollectionIDs) == 0 {
					continue
				}
				kept := it.CollectionIDs[:0]
				for _, id := range it.CollectionIDs {
					if id != key {
						kept = append(kept, id)
					}
				}
				it.CollectionIDs = kept
			}
			delete(ix.albums, key)
		}
	}
	ix.bump()
}

func (ix *Index) ApplyLocalMutations(fn func(map[string]*Item)) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	fn(ix.items)
	ix.bump()
}

func (ix *Index) TrashItems(dedupKeys []string, trashTS int64) {
	ix.ApplyLocalMutations(func(items map[string]*Item) {
		for _, dk := range dedupKeys {
			for _, it := range items {
				if it.DedupKey == dk {
					it.TrashTimestamp = trashTS
				}
			}
		}
	})
}

func (ix *Index) SetArchivedLocal(dedupKeys []string, archived bool) {
	ix.ApplyLocalMutations(func(items map[string]*Item) {
		for _, dk := range dedupKeys {
			for _, it := range items {
				if it.DedupKey == dk {
					it.IsArchived = archived
				}
			}
		}
	})
}

func (ix *Index) SetFavoriteLocal(dedupKey string, favorite bool) {
	ix.ApplyLocalMutations(func(items map[string]*Item) {
		for _, it := range items {
			if it.DedupKey == dedupKey {
				it.IsFavorite = favorite
			}
		}
	})
}

func (ix *Index) DeleteLocal(dedupKeys []string) {
	ix.ApplyLocalMutations(func(items map[string]*Item) {
		for _, dk := range dedupKeys {
			for key, it := range items {
				if it.DedupKey == dk {
					delete(items, key)
				}
			}
		}
	})
}

func (ix *Index) DedupKeyFor(mediaKey string) string {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	if it, ok := ix.items[mediaKey]; ok {
		return it.DedupKey
	}
	return ""
}

func (ix *Index) UpsertAlbum(al *Album) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.albums[al.Key] = al
	ix.bump()
}

func (ix *Index) RemoveAlbum(key string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	delete(ix.albums, key)
	ix.bump()
}

func (ix *Index) RemoveFromCollection(mediaKeys []string, collectionKey string) int {
	want := make(map[string]struct{}, len(mediaKeys))
	for _, k := range mediaKeys {
		want[k] = struct{}{}
	}
	var count int64
	ix.mu.Lock()
	for key, it := range ix.items {
		if _, ok := want[key]; !ok {
			continue
		}
		idx := -1
		for i, id := range it.CollectionIDs {
			if id == collectionKey {
				idx = i
				break
			}
		}
		if idx < 0 {
			continue
		}
		it.CollectionIDs = append(it.CollectionIDs[:idx], it.CollectionIDs[idx+1:]...)
		count++
	}
	if count > 0 {
		if al, ok := ix.albums[collectionKey]; ok {
			al.ItemCount -= count
			if al.ItemCount < 0 {
				al.ItemCount = 0
			}
		}
	}
	ix.bump()
	ix.mu.Unlock()
	return int(count)
}

func (ix *Index) AddToCollection(mediaKeys []string, collectionKey string) int {
	want := make(map[string]struct{}, len(mediaKeys))
	for _, k := range mediaKeys {
		want[k] = struct{}{}
	}
	var count int64
	ix.mu.Lock()
	for key, it := range ix.items {
		if _, ok := want[key]; !ok {
			continue
		}
		found := false
		for _, id := range it.CollectionIDs {
			if id == collectionKey {
				found = true
				break
			}
		}
		if found {
			continue
		}
		it.CollectionIDs = append(it.CollectionIDs, collectionKey)
		count++
	}
	if count > 0 {
		if al, ok := ix.albums[collectionKey]; ok {
			al.ItemCount += count
		}
	}
	ix.bump()
	ix.mu.Unlock()
	return int(count)
}

func (ix *Index) GetItem(mediaKey string) (*Item, bool) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	it, ok := ix.items[mediaKey]
	if !ok {
		return nil, false
	}
	cp := *it
	return &cp, true
}

func (ix *Index) GetAlbum(key string) (*Album, bool) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	al, ok := ix.albums[key]
	if !ok {
		return nil, false
	}
	cp := *al
	return &cp, true
}

func (ix *Index) DedupKeysFor(mediaKeys []string) []string {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	out := make([]string, 0, len(mediaKeys))
	for _, mk := range mediaKeys {
		if it, ok := ix.items[mk]; ok {
			out = append(out, it.DedupKey)
		}
	}
	return out
}
