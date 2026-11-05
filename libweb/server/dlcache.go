package server

import (
	"net/http"
	"sync"
	"time"

	"app/libweb/gphotos"
)

// Download-URL cache: Google download URLs are short-lived; cache the
// prepared info per media key for a few minutes to avoid re-requesting.

var dlCache = struct {
	sync.Mutex
	m map[string]dlCacheEntry
}{m: map[string]dlCacheEntry{}}

type dlCacheEntry struct {
	info *gphotos.DownloadInfo
	exp  time.Time
}

const dlCacheTTL = 10 * time.Minute

func (s *Server) downloadInfo(r *http.Request, key string) (*gphotos.DownloadInfo, error) {
	dlCache.Lock()
	if e, ok := dlCache.m[key]; ok && time.Now().Before(e.exp) {
		dlCache.Unlock()
		return e.info, nil
	}
	dlCache.Unlock()

	api, err := gphotos.Shared()
	if err != nil {
		return nil, apiErr(err)
	}
	info, err := api.GetDownloadURLs(r.Context(), key)
	if err != nil {
		return nil, apiErr(err)
	}
	dlCache.Lock()
	if len(dlCache.m) > 512 {
		dlCache.m = map[string]dlCacheEntry{}
	}
	dlCache.m[key] = dlCacheEntry{info: info, exp: time.Now().Add(dlCacheTTL)}
	dlCache.Unlock()
	return info, nil
}
