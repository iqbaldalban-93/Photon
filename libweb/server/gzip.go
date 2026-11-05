package server

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

var gzipPool = sync.Pool{
	New: func() any { return gzip.NewWriter(nil) },
}

var compressiblePrefixes = []string{
	"text/",
	"application/json",
	"application/javascript",
	"image/svg+xml",
}

func wantsGzip(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	if ct == "" || ct == "text/event-stream" {
		return false
	}
	for _, p := range compressiblePrefixes {
		if strings.HasPrefix(ct, p) {
			return true
		}
	}
	return false
}

func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		enc := strings.TrimSpace(strings.ToLower(part))
		if q := strings.Index(enc, ";"); q >= 0 {
			enc = strings.TrimSpace(enc[:q])
		}
		if enc == "gzip" || enc == "*" {
			return true
		}
	}
	return false
}

func (s *Server) gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !acceptsGzip(r) {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Add("Vary", "Accept-Encoding")
		gw := &gzipResponseWriter{ResponseWriter: w}
		defer func() {
			if gw.zw != nil {
				_ = gw.zw.Close()
				gzipPool.Put(gw.zw)
				gw.zw = nil
			}
		}()
		next.ServeHTTP(gw, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	zw          *gzip.Writer
	wroteHeader bool
	compress    bool
}

func (g *gzipResponseWriter) WriteHeader(status int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true
	h := g.Header()
	if wantsGzip(h.Get("Content-Type")) &&
		h.Get("Content-Encoding") == "" &&
		status != http.StatusNoContent &&
		status != http.StatusNotModified {
		h.Del("Content-Length")
		h.Set("Content-Encoding", "gzip")
		if zw, ok := gzipPool.Get().(*gzip.Writer); ok {
			zw.Reset(g.ResponseWriter)
			g.zw = zw
			g.compress = true
		}
	}
	g.ResponseWriter.WriteHeader(status)
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}
	if g.compress {
		return g.zw.Write(b)
	}
	return g.ResponseWriter.Write(b)
}

func (g *gzipResponseWriter) Flush() {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}
	if g.compress && g.zw != nil {
		_ = g.zw.Flush()
	}
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
