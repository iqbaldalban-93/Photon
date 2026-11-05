package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"app/libweb/configx"
)

const maxUploadRequestBytes = 256 << 30

func (s *Server) handleUploadStatus(w http.ResponseWriter, r *http.Request) {
	jsonWrite(w, http.StatusOK, s.uploads.Status())
}

func (s *Server) handleUploadFiles(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadRequestBytes)
	mr, err := r.MultipartReader()
	if err != nil {
		jsonError(w, http.StatusBadRequest, "expected multipart/form-data")
		return
	}

	album := ""
	var paths []string
	queued := false
	defer func() {
		if !queued {
			for _, p := range paths {
				_ = os.Remove(p)
			}
		}
	}()
	uploadsDir := filepath.Join(configx.Get().DataDir, "uploads")
	if err := os.MkdirAll(uploadsDir, 0o700); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for {
		part, err := mr.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			jsonError(w, http.StatusBadRequest, "malformed multipart body")
			return
		}
		if part.FormName() == "album" {
			buf := make([]byte, 512)
			n, _ := part.Read(buf)
			album = strings.TrimSpace(string(buf[:n]))
			_ = part.Close()
			continue
		}
		if part.FormName() != "files" {
			_ = part.Close()
			continue
		}

		name := filepath.Base(part.FileName())
		if name == "" || name == "." || name == "/" {
			name = "upload.bin"
		}
		tmpPath := filepath.Join(uploadsDir, randomHex(8)+"_"+sanitizeFilename(name))
		if err := savePart(part, tmpPath); err != nil {
			_ = part.Close()
			jsonError(w, http.StatusInternalServerError, "failed to store upload: "+err.Error())
			return
		}
		_ = part.Close()
		paths = append(paths, tmpPath)
	}

	if len(paths) == 0 {
		jsonError(w, http.StatusBadRequest, "no files received")
		return
	}

	queued = true
	go s.cleanupAfterBatch(paths)
	s.uploads.Upload(paths, album)
	jsonWrite(w, http.StatusOK, map[string]any{"queued": len(paths)})
}

func (s *Server) cleanupAfterBatch(paths []string) {
	for {
		sleepSeconds(2)
		st := s.uploads.Status()
		if !st.Running && len(st.Queued) == 0 {
			break
		}
	}
	for _, p := range paths {
		_ = os.Remove(p)
	}
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.Map(func(r rune) rune {
		if r < 0x20 {
			return '_'
		}
		return r
	}, name)
	return name
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) handleUploadPaths(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackRequest(r) {
		jsonError(w, http.StatusForbidden, "uploading by host path is only available on the machine running Photon")
		return
	}
	var body struct {
		Paths []string `json:"paths"`
		Album string   `json:"album"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	if len(body.Paths) == 0 {
		jsonError(w, http.StatusBadRequest, "no paths given")
		return
	}
	s.uploads.Upload(body.Paths, body.Album)
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleUploadControl(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch action {
		case "pause":
			s.uploads.Pause()
		case "resume":
			s.uploads.Resume()
		case "cancel":
			s.uploads.Cancel()
		}
		jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

func (s *Server) handleUploadSkip(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	s.uploads.SkipFile(body.Path)
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleUploadResumeBatch(w http.ResponseWriter, r *http.Request) {
	batch := s.uploads.LoadResumable()
	if batch == nil {
		jsonError(w, http.StatusNotFound, "no resumable batch saved")
		return
	}
	s.uploads.Upload(batch.Paths, batch.Album)
	jsonWrite(w, http.StatusOK, map[string]bool{"ok": true})
}
