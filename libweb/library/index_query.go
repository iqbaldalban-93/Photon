package library

import (
	"fmt"
	"sort"
	"strings"
)

// Query engine: filtering, sorting, identity-collapsed results, paging, and
// the cached counts/stats aggregates backing the library web UI.

type QuerySpec struct {
	Filter   string
	AlbumKey string
	Query    string
	Sort     string
	Offset   int
	Limit    int
}

type QueryResult struct {
	Items []*Item `json:"items"`
	Total int     `json:"total"`
}

func (ix *Index) Query(spec QuerySpec) QueryResult {
	cacheKey := fmt.Sprintf("%s\x00%s\x00%s\x00%s", spec.Filter, spec.AlbumKey, spec.Sort, strings.ToLower(spec.Query))
	if spec.Filter == "watched" || spec.Filter == "unwatched" {
		ix.mu.RLock()
		cacheKey += fmt.Sprintf("\x00%d\x00%s", ix.watchedGen, ix.watched.Account())
		ix.mu.RUnlock()
	}

	ix.mu.RLock()
	if ix.queryCache != nil {
		if res, ok := ix.queryCache[cacheKey]; ok {
			ix.mu.RUnlock()
			return ix.page(res, spec.Offset, spec.Limit)
		}
	}
	ix.mu.RUnlock()

	ix.mu.Lock()
	defer ix.mu.Unlock()
	if ix.queryCache != nil {
		if res, ok := ix.queryCache[cacheKey]; ok {
			return ix.page(res, spec.Offset, spec.Limit)
		}
	}

	q := strings.ToLower(spec.Query)
	matched := make([]*Item, 0, len(ix.items))
	for _, it := range ix.items {
		if !ix.matches(it, spec) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(it.FileName), q) {
			continue
		}
		matched = append(matched, it)
	}
	matched = collapseByIdentity(matched)

	switch spec.Sort {
	case "old":
		sort.SliceStable(matched, func(i, j int) bool {
			return matched[i].TimestampUTCMS < matched[j].TimestampUTCMS
		})
	case "name":
		sort.SliceStable(matched, func(i, j int) bool {
			return strings.ToLower(matched[i].FileName) < strings.ToLower(matched[j].FileName)
		})
	case "size":
		sort.SliceStable(matched, func(i, j int) bool {
			return matched[i].SizeBytes > matched[j].SizeBytes
		})
	case "dur":
		sort.SliceStable(matched, func(i, j int) bool {
			return matched[i].DurationMS > matched[j].DurationMS
		})
	default:
		sort.Slice(matched, func(i, j int) bool {
			if matched[i].TimestampUTCMS != matched[j].TimestampUTCMS {
				return matched[i].TimestampUTCMS > matched[j].TimestampUTCMS
			}
			return matched[i].MediaKey < matched[j].MediaKey
		})
	}

	res := QueryResult{Items: matched, Total: len(matched)}
	ix.cacheQuery(cacheKey, res)
	return ix.page(res, spec.Offset, spec.Limit)
}

func (ix *Index) page(res QueryResult, offset, limit int) QueryResult {
	total := res.Total
	if offset > total {
		offset = total
	}
	end := total
	if limit > 0 && offset+limit < total {
		end = offset + limit
	}
	page := make([]*Item, 0, end-offset)
	for _, it := range res.Items[offset:end] {
		cp := *it
		cp.Watched = ix.isWatched(it)
		page = append(page, &cp)
	}
	return QueryResult{Items: page, Total: total}
}

func (ix *Index) matches(it *Item, spec QuerySpec) bool {
	switch spec.Filter {
	case "video":
		return it.Type == "video" && it.TrashTimestamp == 0
	case "photo":
		return it.Type != "video" && it.TrashTimestamp == 0
	case "favorite":
		return it.IsFavorite && it.TrashTimestamp == 0
	case "archived":
		return it.IsArchived && it.TrashTimestamp == 0
	case "trash":
		return it.TrashTimestamp > 0
	case "watched":
		return isLibraryVideo(it) && ix.isWatched(it)
	case "unwatched":
		return isLibraryVideo(it) && !ix.isWatched(it)
	case "album":
		if spec.AlbumKey == "" {
			return false
		}
		if it.TrashTimestamp > 0 {
			return false
		}
		for _, id := range it.CollectionIDs {
			if id == spec.AlbumKey {
				return true
			}
		}
		return false
	default:
		return !it.IsArchived && it.TrashTimestamp == 0
	}
}

func (ix *Index) Counts() map[string]int {
	ix.mu.RLock()
	acct := ix.watched.Account()
	if ix.countsCache != nil && ix.countsAt == ix.version && ix.countsWG == ix.watchedGen && ix.countsAcct == acct {
		c := ix.countsCache
		ix.mu.RUnlock()
		return c
	}
	ix.mu.RUnlock()

	ix.mu.Lock()
	defer ix.mu.Unlock()
	acct = ix.watched.Account()
	if ix.countsCache != nil && ix.countsAt == ix.version && ix.countsWG == ix.watchedGen && ix.countsAcct == acct {
		return ix.countsCache
	}
	c := map[string]int{"all": 0, "video": 0, "photo": 0, "favorite": 0, "archived": 0, "trash": 0, "watched": 0, "unwatched": 0}
	seen := map[string]map[string]struct{}{}
	count := func(cat string, it *Item) {
		m := seen[cat]
		if m == nil {
			m = map[string]struct{}{}
			seen[cat] = m
		}
		k := identityKey(it)
		if _, dup := m[k]; dup {
			return
		}
		m[k] = struct{}{}
		c[cat]++
	}
	for _, it := range ix.items {
		if it.TrashTimestamp == 0 {
			count("all", it)
			if it.Type == "video" {
				count("video", it)
			} else {
				count("photo", it)
			}
			if it.IsFavorite {
				count("favorite", it)
			}
		}
		if it.IsArchived && it.TrashTimestamp == 0 {
			count("archived", it)
		}
		if it.TrashTimestamp > 0 {
			count("trash", it)
		}
		if isLibraryVideo(it) {
			if ix.isWatched(it) {
				count("watched", it)
			} else {
				count("unwatched", it)
			}
		}
	}
	ix.countsCache = c
	ix.countsAt = ix.version
	ix.countsWG = ix.watchedGen
	ix.countsAcct = acct
	return c
}

func (ix *Index) ListAlbums() []*Album {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	out := make([]*Album, 0, len(ix.albums))
	for _, al := range ix.albums {
		cp := *al
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ItemCount != out[j].ItemCount {
			return out[i].ItemCount > out[j].ItemCount
		}
		return out[i].Title < out[j].Title
	})
	return out
}

func (ix *Index) Stats() map[string]any {
	ix.mu.RLock()
	if ix.statsCache != nil && ix.statsAt == ix.version {
		s := ix.statsCache
		ix.mu.RUnlock()
		return s
	}
	ix.mu.RUnlock()

	ix.mu.Lock()
	defer ix.mu.Unlock()
	if ix.statsCache != nil && ix.statsAt == ix.version {
		return ix.statsCache
	}
	var totalSize, videoSize, videoDur int64
	videos, photos := 0, 0
	seenStats := make(map[string]struct{}, len(ix.items))
	for _, it := range ix.items {
		if it.TrashTimestamp > 0 {
			continue
		}
		k := identityKey(it)
		if _, dup := seenStats[k]; dup {
			continue
		}
		seenStats[k] = struct{}{}
		totalSize += it.SizeBytes
		if it.Type == "video" {
			videos++
			videoSize += it.SizeBytes
			videoDur += it.DurationMS
		} else {
			photos++
		}
	}
	s := map[string]any{
		"totalItems": len(seenStats),
		"photos":     photos,
		"videos":     videos,
		"totalSize":  totalSize,
		"videoSize":  videoSize,
		"videoHours": float64(videoDur) / 3600000.0,
		"albums":     len(ix.albums),
	}
	ix.statsCache = s
	ix.statsAt = ix.version
	return s
}
