package server

import (
	"fmt"
	"net/http"

	"app/libweb/gphotos"
	"app/libweb/library"
)

// Album endpoints: create/rename/delete remote albums and add/remove media,
// keeping the local index collections in step.

func (s *Server) handleAlbumList(w http.ResponseWriter, r *http.Request) {
	jsonWrite(w, http.StatusOK, map[string]any{"albums": s.index.ListAlbums()})
}

func (s *Server) handleAlbumCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string   `json:"name"`
		MediaKeys []string `json:"mediaKeys"`
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
	if len(body.MediaKeys) == 0 {
		writeErr(w, apiError{http.StatusBadRequest, "Google Photos can't create an empty album. Select some photos first, then use Add to album and type a new album name."})
		return
	}
	batches := chunkStrings(body.MediaKeys, 500)
	key, err := api.CreateAlbum(r.Context(), body.Name, batches[0])
	if err != nil {
		writeErr(w, apiErr(err))
		return
	}
	for _, batch := range batches[1:] {
		if err := api.AddMediaToAlbum(r.Context(), key, batch); err != nil {
			writeErr(w, apiErr(err))
			return
		}
	}
	s.index.UpsertAlbum(&library.Album{Key: key, Title: body.Name})
	s.index.AddToCollection(body.MediaKeys, key)
	_ = s.index.Save()
	jsonWrite(w, http.StatusOK, map[string]any{"key": key})
}

func (s *Server) handleAlbumRename(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var body struct {
		Name string `json:"name"`
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
	if err := api.RenameAlbum(r.Context(), key, body.Name); err != nil {
		writeErr(w, apiErr(err))
		return
	}
	if album, ok := s.index.GetAlbum(key); ok {
		album.Title = body.Name
		s.index.UpsertAlbum(album)
		_ = s.index.Save()
	}
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAlbumDelete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	api, err := gphotos.Shared()
	if err != nil {
		writeErr(w, apiErr(err))
		return
	}
	if err := api.DeleteAlbum(r.Context(), key); err != nil {
		writeErr(w, apiErr(err))
		return
	}
	s.index.RemoveAlbum(key)
	_ = s.index.Save()
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAlbumAdd(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var body struct {
		MediaKeys []string `json:"mediaKeys"`
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
	for _, batch := range chunkStrings(body.MediaKeys, 500) {
		if err := api.AddMediaToAlbum(r.Context(), key, batch); err != nil {
			writeErr(w, apiErr(fmt.Errorf("%w [%s]", err, s.albumAddContext(key, batch))))
			return
		}
	}
	s.index.AddToCollection(body.MediaKeys, key)
	_ = s.index.Save()
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAlbumRemove(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var body struct {
		MediaKeys []string `json:"mediaKeys"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	if len(body.MediaKeys) == 0 {
		jsonError(w, http.StatusBadRequest, "no media keys given")
		return
	}
	api, err := gphotos.Shared()
	if err != nil {
		writeErr(w, apiErr(err))
		return
	}
	for _, batch := range chunkStrings(body.MediaKeys, 500) {
		if err := api.RemoveMediaFromAlbum(r.Context(), key, batch); err != nil {
			writeErr(w, apiErr(fmt.Errorf("%w [%s]", err, s.albumAddContext(key, batch))))
			return
		}
	}
	removed := s.index.RemoveFromCollection(body.MediaKeys, key)
	_ = s.index.Save()
	jsonWrite(w, http.StatusOK, map[string]any{"ok": true, "removed": removed})
}

func (s *Server) albumAddContext(albumKey string, batch []string) string {
	title, typ := "not in local library", int64(-1)
	if al, ok := s.index.GetAlbum(albumKey); ok {
		title, typ = al.Title, al.Type
	}
	known := 0
	for _, k := range batch {
		if _, ok := s.index.GetItem(k); ok {
			known++
		}
	}
	short := albumKey
	if len(short) > 12 {
		short = short[:12] + "…"
	}
	return fmt.Sprintf("album %q type=%d key=%s; %d/%d items found locally", title, typ, short, known, len(batch))
}
