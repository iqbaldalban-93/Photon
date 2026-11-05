package server

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"app/libweb/authx"
	"app/libweb/events"
	"app/libweb/library"
	"app/libweb/proxy"
	"app/libweb/uploadx"
)

var ErrUploadsRunning = errors.New("web uploads are running")

type Server struct {
	log     *slog.Logger
	webFS   fs.FS
	index   *library.Index
	sync    *library.SyncManager
	uploads *uploadx.Manager
	thumbs  *proxy.ThumbnailCache
	files   *proxy.FileProxy
	hls     *proxy.HLSProxy
	media   *http.Client
	Version string

	Events *events.Hub

	AccountEpoch func() uint64

	SwitchAccount func(email string, force bool) error

	AccountsChanged func()
}

func (s *Server) Thumbs() *proxy.ThumbnailCache { return s.thumbs }

func New(
	log *slog.Logger,
	webFS embed.FS,
	index *library.Index,
	sync *library.SyncManager,
	uploads *uploadx.Manager,
	mediaClient *http.Client,
	bearer func() (string, error),
	ua string,
	thumbDir string,
	thumbCacheMB int,
) (*Server, error) {
	fetcher := proxy.Fetcher{Client: mediaClient, Bearer: bearer, UA: ua}
	thumbs, err := proxy.NewThumbnailCache(fetcher, thumbDir, thumbCacheMB)
	if err != nil {
		return nil, err
	}
	root, err := fs.Sub(webFS, "web")
	if err != nil {
		return nil, err
	}
	return &Server{
		log:     log,
		webFS:   root,
		index:   index,
		sync:    sync,
		uploads: uploads,
		media:   mediaClient,
		thumbs:  thumbs,
		files:   &proxy.FileProxy{Fetcher: fetcher},
		hls: &proxy.HLSProxy{
			Fetcher: fetcher,
			SegmentPath: func(upstream string) string {
				return "/api/media/seg?u=" + encodeSeg(upstream)
			},
		},
	}, nil
}

func encodeSeg(u string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(u))
}

func base64RawDecode(s string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/auth/status", s.handleAuthStatus)
	mux.HandleFunc("POST /api/auth/setup", s.handleAuthSetup)
	mux.HandleFunc("POST /api/auth/login", s.handleAuthLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleAuthLogout)

	mux.Handle("GET /api/state", s.guard(http.HandlerFunc(s.handleState)))
	mux.Handle("GET /api/events", s.guard(http.HandlerFunc(s.handleEvents)))
	mux.Handle("GET /api/media", s.guard(http.HandlerFunc(s.handleMediaList)))
	mux.Handle("GET /api/library/duplicates", s.guard(http.HandlerFunc(s.handleLibraryDuplicates)))
	mux.Handle("POST /api/library/sync", s.guard(http.HandlerFunc(s.handleLibrarySync)))
	mux.Handle("GET /api/library/status", s.guard(http.HandlerFunc(s.handleLibraryStatus)))
	mux.Handle("POST /api/library/autosync", s.guard(http.HandlerFunc(s.handleAutoSyncSet)))

	mux.Handle("GET /api/albums", s.guard(http.HandlerFunc(s.handleAlbumList)))
	mux.Handle("POST /api/albums", s.guard(http.HandlerFunc(s.handleAlbumCreate)))
	mux.Handle("POST /api/albums/{key}/rename", s.guard(http.HandlerFunc(s.handleAlbumRename)))
	mux.Handle("DELETE /api/albums/{key}", s.guard(http.HandlerFunc(s.handleAlbumDelete)))
	mux.Handle("POST /api/albums/{key}/add", s.guard(http.HandlerFunc(s.handleAlbumAdd)))
	mux.Handle("POST /api/albums/{key}/remove", s.guard(http.HandlerFunc(s.handleAlbumRemove)))

	mux.Handle("POST /api/media/trash", s.guard(http.HandlerFunc(s.handleMediaTrash)))
	mux.Handle("POST /api/media/restore", s.guard(http.HandlerFunc(s.handleMediaRestore)))
	mux.Handle("POST /api/media/delete", s.guard(http.HandlerFunc(s.handleMediaDelete)))
	mux.Handle("POST /api/media/archive", s.guard(http.HandlerFunc(s.handleMediaArchive)))
	mux.Handle("POST /api/media/watched", s.guard(http.HandlerFunc(s.handleMediaWatched)))
	mux.Handle("POST /api/media/{key}/favorite", s.guard(http.HandlerFunc(s.handleMediaFavorite)))
	mux.Handle("POST /api/media/{key}/caption", s.guard(http.HandlerFunc(s.handleMediaCaption)))
	mux.Handle("GET /api/media/{key}/download", s.guard(http.HandlerFunc(s.handleMediaDownload)))
	mux.Handle("GET /api/media/{key}/video", s.guard(http.HandlerFunc(s.handleMediaVideo)))
	mux.Handle("GET /api/media/{key}/hls.m3u8", s.guard(http.HandlerFunc(s.handleMediaHLS)))
	mux.Handle("GET /api/media/seg", s.guard(http.HandlerFunc(s.handleMediaSegment)))

	mux.Handle("GET /api/upload/status", s.guard(http.HandlerFunc(s.handleUploadStatus)))
	mux.Handle("POST /api/upload/files", s.guard(http.HandlerFunc(s.handleUploadFiles)))
	mux.Handle("POST /api/upload/paths", s.guard(http.HandlerFunc(s.handleUploadPaths)))
	mux.Handle("POST /api/upload/pause", s.guard(http.HandlerFunc(s.handleUploadControl("pause"))))
	mux.Handle("POST /api/upload/resume", s.guard(http.HandlerFunc(s.handleUploadControl("resume"))))
	mux.Handle("POST /api/upload/cancel", s.guard(http.HandlerFunc(s.handleUploadControl("cancel"))))
	mux.Handle("POST /api/upload/skip", s.guard(http.HandlerFunc(s.handleUploadSkip)))
	mux.Handle("POST /api/upload/resume-batch", s.guard(http.HandlerFunc(s.handleUploadResumeBatch)))

	mux.Handle("GET /api/account", s.guard(http.HandlerFunc(s.handleAccountGet)))
	mux.Handle("POST /api/account/credentials", s.guard(http.HandlerFunc(s.handleAccountAdd)))
	mux.Handle("DELETE /api/account/credentials", s.guard(http.HandlerFunc(s.handleAccountRemove)))
	mux.Handle("POST /api/account/select", s.guard(http.HandlerFunc(s.handleAccountSelect)))
	mux.Handle("POST /api/account/token-binding", s.guard(http.HandlerFunc(s.handleAccountTokenBinding)))

	mux.Handle("GET /api/settings", s.guard(http.HandlerFunc(s.handleSettingsGet)))
	mux.Handle("PUT /api/settings", s.guard(http.HandlerFunc(s.handleSettingsPut)))

	mux.Handle("GET /api/thumb/{key}", s.guard(http.HandlerFunc(s.handleThumb)))

	mux.Handle("GET /", http.HandlerFunc(s.handleStatic))

	return s.gzip(s.securityHeaders(mux))
}

func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authx.NeedsSetup() {
			if strings.HasPrefix(r.URL.Path, "/api/auth/") {
				next.ServeHTTP(w, r)
				return
			}
			jsonError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		if !authx.Authorized(r) {
			jsonError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		h.Set("Content-Security-Policy", "script-src 'self'; worker-src 'self' blob:; connect-src 'self'; frame-ancestors 'self'; base-uri 'self'; object-src 'none'")
		if !sameOriginRequest(r) {
			jsonError(w, http.StatusForbidden, "cross-origin request blocked")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sameOriginRequest(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func jsonWrite(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	jsonWrite(w, status, map[string]string{"error": msg})
}

type apiError struct {
	status int
	msg    string
}

func (e apiError) Error() string { return e.msg }

func writeErr(w http.ResponseWriter, err error) {
	var ae apiError
	if ok := asAPIError(err, &ae); ok {
		jsonError(w, ae.status, ae.msg)
		return
	}
	jsonError(w, http.StatusInternalServerError, err.Error())
}

func asAPIError(err error, target *apiError) bool {
	if e, ok := err.(apiError); ok {
		*target = e
		return true
	}
	return false
}

func decodeJSON(r *http.Request, v any) error {
	defer func() { _ = r.Body.Close() }()
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4<<20))
	if err := dec.Decode(v); err != nil {
		return apiError{http.StatusBadRequest, "invalid JSON body"}
	}
	return nil
}
