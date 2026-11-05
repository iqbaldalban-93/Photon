package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"app/libweb/configx"
	"app/libweb/gphotos"
	"app/libweb/library"
	"app/libweb/proxy"
)

// Media and library endpoints: listing/sync control, item mutations, media
// streaming (download/video/HLS), thumbnails, and watched-state.

func (s *Server) handleMediaList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	res := s.index.Query(library.QuerySpec{
		Filter:   q.Get("filter"),
		AlbumKey: q.Get("album"),
		Query:    q.Get("q"),
		Sort:     q.Get("sort"),
		Offset:   offset,
		Limit:    limit,
	})
	jsonWrite(w, http.StatusOK, res)
}

func (s *Server) handleLibrarySync(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Full bool `json:"full"`
	}
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
	}
	started := s.sync.Trigger(body.Full)
	if !started {
		jsonError(w, http.StatusConflict, "a sync is already running")
		return
	}
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleLibraryStatus(w http.ResponseWriter, r *http.Request) {
	jsonWrite(w, http.StatusOK, map[string]any{
		"sync":        s.sync.Status(),
		"counts":      s.index.Counts(),
		"stats":       s.index.Stats(),
		"autosyncSec": configx.Get().LibrarySyncSec,
	})
}

func (s *Server) handleAutoSyncSet(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IntervalSec int `json:"intervalSec"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	sec := body.IntervalSec
	switch {
	case sec == 0:
		sec = 120
	case sec < 0:
		sec = -1
	case sec < 15:
		sec = 15
	case sec > 86400:
		sec = 86400
	}
	if err := configx.Update(func(c *configx.Config) { c.LibrarySyncSec = sec }); err != nil {
		writeErr(w, err)
		return
	}
	jsonWrite(w, http.StatusOK, map[string]any{"ok": true, "intervalSec": sec})
}

func (s *Server) handleLibraryDuplicates(w http.ResponseWriter, r *http.Request) {
	jsonWrite(w, http.StatusOK, map[string]any{"groups": s.index.DuplicateGroups()})
}

type keysBody struct {
	Keys []string `json:"keys"`
}

func (s *Server) handleMediaTrash(w http.ResponseWriter, r *http.Request) {
	s.mediaMutation(w, r, func(api *gphotos.Api, dedupKeys []string) error {
		return api.TrashAction(r.Context(), dedupKeys)
	}, func(keys []string) {
		s.index.TrashItems(keys, time.Now().Unix())
	})
}

func (s *Server) handleMediaRestore(w http.ResponseWriter, r *http.Request) {
	s.mediaMutation(w, r, func(api *gphotos.Api, dedupKeys []string) error {
		return api.RestoreFromTrash(r.Context(), dedupKeys)
	}, func(keys []string) {
		s.index.TrashItems(keys, 0)
	})
}

func (s *Server) handleMediaDelete(w http.ResponseWriter, r *http.Request) {
	s.mediaMutation(w, r, func(api *gphotos.Api, dedupKeys []string) error {
		return api.DeletePermanently(r.Context(), dedupKeys)
	}, func(keys []string) {
		s.index.DeleteLocal(keys)
	})
}

func (s *Server) handleMediaArchive(w http.ResponseWriter, r *http.Request) {
	var body struct {
		keysBody
		Archived bool `json:"archived"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	api, err := gphotos.Shared()
	if err != nil {
		writeErr(w, apiErr(err))
		return
	}
	dedupKeys := s.index.DedupKeysFor(body.Keys)
	for _, batch := range chunkStrings(dedupKeys, 500) {
		if err := api.SetArchived(r.Context(), batch, body.Archived); err != nil {
			writeErr(w, apiErr(err))
			return
		}
	}
	s.index.SetArchivedLocal(dedupKeys, body.Archived)
	_ = s.index.Save()
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMediaFavorite(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var body struct {
		Favorite bool `json:"favorite"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	api, err := gphotos.Shared()
	if err != nil {
		writeErr(w, apiErr(err))
		return
	}
	dk := s.index.DedupKeyFor(key)
	if dk == "" {
		jsonError(w, http.StatusNotFound, "item not found in index; sync the library first")
		return
	}
	if err := api.SetFavorite(r.Context(), dk, body.Favorite); err != nil {
		writeErr(w, apiErr(err))
		return
	}
	s.index.SetFavoriteLocal(dk, body.Favorite)
	_ = s.index.Save()
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMediaCaption(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var body struct {
		Caption string `json:"caption"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	api, err := gphotos.Shared()
	if err != nil {
		writeErr(w, apiErr(err))
		return
	}
	dk := s.index.DedupKeyFor(key)
	if dk == "" {
		jsonError(w, http.StatusNotFound, "item not found in index; sync the library first")
		return
	}
	if err := api.SetCaption(r.Context(), dk, body.Caption); err != nil {
		writeErr(w, apiErr(err))
		return
	}
	s.index.ApplyLocalMutations(func(items map[string]*gphotos.Item) {
		if it2, ok := items[key]; ok {
			it2.Caption = body.Caption
		}
	})
	_ = s.index.Save()
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) mediaMutation(
	w http.ResponseWriter,
	r *http.Request,
	call func(*gphotos.Api, []string) error,
	local func([]string),
) {
	var body keysBody
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	api, err := gphotos.Shared()
	if err != nil {
		writeErr(w, apiErr(err))
		return
	}
	dedupKeys := s.index.DedupKeysFor(body.Keys)
	if len(dedupKeys) == 0 {
		jsonError(w, http.StatusNotFound, "no matching items in index; sync the library first")
		return
	}
	for _, batch := range chunkStrings(dedupKeys, 500) {
		if err := call(api, batch); err != nil {
			writeErr(w, apiErr(err))
			return
		}
	}
	local(dedupKeys)
	_ = s.index.Save()
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMediaDownload(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	variant := r.URL.Query().Get("variant")

	item, ok := s.index.GetItem(key)
	if !ok {
		jsonError(w, http.StatusNotFound, "item not found in index; sync the library first")
		return
	}
	info, err := s.downloadInfo(r, key)
	if err != nil {
		writeErr(w, err)
		return
	}
	url := info.OriginalURL
	if variant == "edited" && info.EditedURL != "" {
		url = info.EditedURL
	}
	if item.Type == "video" && info.VideoURL != "" {
		url = info.VideoURL
	}
	if url == "" {
		url = firstNonEmpty(info.EditedURL, info.OriginalURL, info.VideoURL)
	}
	name := info.FileName
	if name == "" {
		name = item.FileName
	}
	s.files.Handler(w, r, url, "attachment; filename*=UTF-8''"+urlPathEscape(name))
}

func (s *Server) handleMediaVideo(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	info, err := s.downloadInfo(r, key)
	if err != nil {
		writeErr(w, err)
		return
	}
	url := firstNonEmpty(info.VideoURL, info.OriginalURL, info.EditedURL)
	if url == "" {
		jsonError(w, http.StatusNotFound, "no streamable URL for this item")
		return
	}
	s.files.Handler(w, r, url, "")
}

func (s *Server) handleMediaHLS(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !proxy.ValidMediaKey(key) {
		jsonError(w, http.StatusBadRequest, "invalid media key")
		return
	}
	if _, err := gphotos.Shared(); err != nil {
		writeErr(w, apiErr(err))
		return
	}
	var contentVersion int64
	if it, ok := s.index.GetItem(key); ok {
		contentVersion = it.ContentVersion
	}
	manifestURL := fmt.Sprintf("https://lh3.googleusercontent.com/p/%s%%3Dmm,hls-vm", key)
	if contentVersion > 0 {
		manifestURL = fmt.Sprintf("https://lh3.googleusercontent.com/p/%s%%3Dmm,hls-vm-iv%d", key, contentVersion)
	}
	s.hls.ManifestHandler(w, r, manifestURL)
}

func (s *Server) handleMediaSegment(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Query().Get("u")
	if u == "" {
		jsonError(w, http.StatusBadRequest, "missing u")
		return
	}
	decoded, err := base64RawDecode(u)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid u")
		return
	}
	s.hls.SegmentHandler(w, r, decoded)
}

func (s *Server) handleThumb(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	q := r.URL.Query()
	width, _ := strconv.ParseInt(q.Get("w"), 10, 64)
	height, _ := strconv.ParseInt(q.Get("h"), 10, 64)
	version, _ := strconv.ParseInt(q.Get("v"), 10, 64)
	crop := q.Get("sq") == "1"
	if width <= 0 {
		width = 400
	}
	if height <= 0 {
		height = 400
	}
	s.thumbs.Handler(w, r, key, width, height, version, crop)
}

func (s *Server) handleMediaWatched(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Keys    []string `json:"keys"`
		Watched bool     `json:"watched"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	if len(body.Keys) == 0 || len(body.Keys) > 100000 {
		jsonError(w, http.StatusBadRequest, "keys required")
		return
	}
	n, err := s.index.MarkWatched(body.Keys, body.Watched)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "could not save watched state: "+err.Error())
		return
	}
	jsonWrite(w, http.StatusOK, map[string]any{"ok": true, "changed": n, "counts": s.index.Counts()})
}
