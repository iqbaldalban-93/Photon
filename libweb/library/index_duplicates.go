package library

import (
	"sort"
	"strings"
)

// Duplicate detection: items sharing a filename prefix are grouped, keeping
// one representative per identity (dedup key, falling back to media key).

type DuplicateGroup struct {
	Prefix string  `json:"prefix"`
	Items  []*Item `json:"items"`
}

func (ix *Index) DuplicateGroups() []DuplicateGroup {
	ix.mu.RLock()
	groupsByPrefix := make(map[string][]*Item)
	seenDedupKeys := make(map[string]map[string]struct{})
	prefixes := make(map[string]string)
	for _, it := range ix.items {
		if it.TrashTimestamp > 0 {
			continue
		}
		prefix := it.FileName
		if space := strings.IndexByte(prefix, ' '); space >= 0 {
			prefix = prefix[:space]
		}
		if prefix == "" {
			continue
		}
		key := strings.ToLower(prefix)
		dedupKey := it.DedupKey
		if dedupKey == "" {
			dedupKey = it.MediaKey
		}
		seen := seenDedupKeys[key]
		if seen == nil {
			seen = make(map[string]struct{})
			seenDedupKeys[key] = seen
		}
		if _, ok := seen[dedupKey]; ok {
			continue
		}
		seen[dedupKey] = struct{}{}

		copy := *it
		groupsByPrefix[key] = append(groupsByPrefix[key], &copy)
		if _, ok := prefixes[key]; !ok {
			prefixes[key] = prefix
		}
	}
	ix.mu.RUnlock()

	groups := make([]DuplicateGroup, 0)
	for key, items := range groupsByPrefix {
		if len(items) < 2 {
			continue
		}
		sort.Slice(items, func(i, j int) bool {
			return strings.ToLower(items[i].FileName) < strings.ToLower(items[j].FileName)
		})
		groups = append(groups, DuplicateGroup{Prefix: prefixes[key], Items: items})
	}
	sort.Slice(groups, func(i, j int) bool {
		return strings.ToLower(groups[i].Prefix) < strings.ToLower(groups[j].Prefix)
	})
	return groups
}

func better(a, b *Item) bool {
	if len(a.CollectionIDs) != len(b.CollectionIDs) {
		return len(a.CollectionIDs) > len(b.CollectionIDs)
	}
	if a.TimestampUTCMS != b.TimestampUTCMS {
		return a.TimestampUTCMS > b.TimestampUTCMS
	}
	return a.MediaKey < b.MediaKey
}

func collapseByIdentity(items []*Item) []*Item {
	best := make(map[string]*Item, len(items))
	for _, it := range items {
		k := identityKey(it)
		if cur, ok := best[k]; !ok || better(it, cur) {
			best[k] = it
		}
	}
	if len(best) == len(items) {
		return items
	}
	out := make([]*Item, 0, len(best))
	for _, it := range best {
		out = append(out, it)
	}
	return out
}
