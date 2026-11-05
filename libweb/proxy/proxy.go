package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"app/internal/fsx"
)

const (
	maxThumbBytes    = 32 << 20
	maxManifestBytes = 8 << 20
)

var errTooLarge = errors.New("upstream response too large")

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errTooLarge
	}
	return data, nil
}

var googleHostSuffixes = []string{
	"googleusercontent.com",
	"googlevideo.com",
	"googleapis.com",
	"ggpht.com",
	"gstatic.com",
	"google.com",
}

func isGoogleHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, suffix := range googleHostSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

var ErrNotGoogleHost = errors.New("blocked: URL host is not a Google media host")

func ValidateGoogleURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("blocked: only https URLs are allowed")
	}
	if !isGoogleHost(u.Hostname()) {
		return ErrNotGoogleHost
	}
	return urlValidator(raw)
}

var urlValidator = ValidateURL

type Fetcher struct {
	Client *http.Client
	Bearer func() (string, error)
	UA     string
}

func (f *Fetcher) fetchGET(ctx stdCtx, rawURL string, rangeHeader string) (*http.Response, error) {
	do := func(token string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if f.UA != "" {
			req.Header.Set("User-Agent", f.UA)
		}
		req.Header.Set("Accept-Encoding", "identity")
		if rangeHeader != "" {
			req.Header.Set("Range", rangeHeader)
		}
		return f.Client.Do(req)
	}

	var token string
	if u, perr := url.Parse(rawURL); perr == nil && isGoogleHost(u.Hostname()) && f.Bearer != nil {
		token, _ = f.Bearer()
	}
	resp, err := do(token)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if token != "" {
			_ = resp.Body.Close()
			if fresh, berr := f.Bearer(); berr == nil {
				return do(fresh)
			}
			return do(token)
		}
	}
	return resp, nil
}

type stdCtx = context.Context

type ThumbnailCache struct {
	Fetcher
	dirMu    sync.RWMutex
	dir      string
	maxBytes int64

	mu        sync.Mutex
	size      int64
	written   int64
	lastEvict time.Time
}

func NewThumbnailCache(f Fetcher, dir string, maxMB int) (*ThumbnailCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &ThumbnailCache{Fetcher: f, dir: dir, maxBytes: int64(maxMB) * 1024 * 1024}, nil
}

func (tc *ThumbnailCache) getDir() string {
	tc.dirMu.RLock()
	defer tc.dirMu.RUnlock()
	return tc.dir
}

func (tc *ThumbnailCache) SetDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tc.dirMu.Lock()
	tc.dir = dir
	tc.dirMu.Unlock()
	tc.mu.Lock()
	tc.size, tc.written, tc.lastEvict = 0, 0, time.Time{}
	tc.mu.Unlock()
	return nil
}

func (tc *ThumbnailCache) Handler(w http.ResponseWriter, r *http.Request, mediaKey string, width, height, contentVersion int64, crop bool) {
	if !ValidMediaKey(mediaKey) {
		http.Error(w, "invalid media key", http.StatusBadRequest)
		return
	}
	url := buildThumbURL(mediaKey, width, height, contentVersion, crop)
	if err := urlValidator(url); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	cacheKey := fmt.Sprintf("%s_%d_%d_%d_%t", mediaKey, width, height, contentVersion, crop)
	sum := sha256.Sum256([]byte(cacheKey))
	path := filepath.Join(tc.getDir(), hex.EncodeToString(sum[:8])+".jpg")

	if data, err := os.ReadFile(path); err == nil {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "private, max-age=604800, immutable")
		_, _ = w.Write(data)
		return
	}

	resp, err := tc.fetchGET(r.Context(), url, "")
	if err != nil {
		http.Error(w, "thumbnail fetch failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("thumbnail upstream status %d", resp.StatusCode), http.StatusBadGateway)
		return
	}
	data, err := readLimited(resp.Body, maxThumbBytes)
	if err != nil {
		http.Error(w, "thumbnail read failed", http.StatusBadGateway)
		return
	}
	if writeFileAtomic(path, data, 0o600) == nil {
		tc.noteWrite(int64(len(data)))
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=604800, immutable")
	_, _ = w.Write(data)
}

func ValidMediaKey(key string) bool {
	if key == "" || len(key) > 256 {
		return false
	}
	for i := 0; i < len(key); i++ {
		c := key[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	return fsx.WriteFileAtomic(path, data, perm)
}

func (tc *ThumbnailCache) noteWrite(n int64) {
	tc.mu.Lock()
	tc.size += n
	tc.written += n
	due := tc.written >= 8*1024*1024 || time.Since(tc.lastEvict) > time.Minute
	tc.mu.Unlock()
	if due {
		tc.evict()
	}
}

func (tc *ThumbnailCache) evict() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.written = 0
	tc.lastEvict = time.Now()
	entries, err := os.ReadDir(tc.getDir())
	if err != nil {
		return
	}
	var total int64
	type fe struct {
		path string
		size int64
		mod  int64
	}
	files := make([]fe, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jpg") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fe{filepath.Join(tc.getDir(), e.Name()), info.Size(), info.ModTime().UnixNano()})
		total += info.Size()
	}
	tc.size = total
	if total <= tc.maxBytes {
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod < files[j].mod })
	for _, f := range files {
		if total <= tc.maxBytes {
			break
		}
		if os.Remove(f.path) == nil {
			total -= f.size
		}
	}
}

func buildThumbURL(mediaKey string, width, height, contentVersion int64, crop bool) string {
	var sb strings.Builder
	sb.WriteString("https://ap2.googleusercontent.com/gpa/")
	sb.WriteString(mediaKey)
	sb.WriteString("=k-sg")
	if width > 0 {
		fmt.Fprintf(&sb, "-w%d", width)
	}
	if height > 0 {
		fmt.Fprintf(&sb, "-h%d", height)
	}
	sb.WriteString("-rj")
	if contentVersion > 0 {
		fmt.Fprintf(&sb, "-iv%d", contentVersion)
	}
	sb.WriteString("-no")
	if crop {
		sb.WriteString("-c")
	}
	return sb.String()
}

type FileProxy struct {
	Fetcher
}

func (fp *FileProxy) Handler(w http.ResponseWriter, r *http.Request, upstreamURL, disposition string) {
	if err := urlValidator(upstreamURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	resp, err := fp.fetchGET(r.Context(), upstreamURL, r.Header.Get("Range"))
	if err != nil {
		http.Error(w, "fetch failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Error(w, fmt.Sprintf("upstream status %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "ETag", "Last-Modified"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	if disposition == "" && resp.Header.Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	if disposition != "" {
		w.Header().Set("Content-Disposition", disposition)
	}
	w.WriteHeader(resp.StatusCode)
	copyStream(w, resp.Body)
}

type HLSProxy struct {
	Fetcher
	SegmentPath func(upstreamURL string) string
}

func (hp *HLSProxy) ManifestHandler(w http.ResponseWriter, r *http.Request, manifestURL string) {
	if err := urlValidator(manifestURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	resp, err := hp.fetchGET(r.Context(), manifestURL, "")
	if err != nil {
		http.Error(w, "manifest fetch failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Error(w, fmt.Sprintf("manifest upstream status %d", resp.StatusCode), http.StatusBadGateway)
		return
	}
	body, err := readLimited(resp.Body, maxManifestBytes)
	if err != nil {
		http.Error(w, "manifest read failed", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(hp.rewrite(body, manifestURL))
}

func (hp *HLSProxy) rewrite(body []byte, baseURL string) []byte {
	base, err := url.Parse(baseURL)
	if err != nil {
		return body
	}
	lines := strings.Split(string(body), "\n")
	for i, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if idx := strings.Index(trimmed, "URI=\""); idx >= 0 {
				start := idx + 5
				if end := strings.Index(trimmed[start:], "\""); end >= 0 {
					raw := trimmed[start : start+end]
					lines[i] = trimmed[:start] + hp.proxyURI(raw, base) + trimmed[start+end:]
				}
			}
			continue
		}
		lines[i] = hp.proxyURI(trimmed, base)
	}
	return []byte(strings.Join(lines, "\n"))
}

func (hp *HLSProxy) proxyURI(raw string, base *url.URL) string {
	ref, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	abs := base.ResolveReference(ref)
	if abs.Scheme != "http" && abs.Scheme != "https" {
		return raw
	}
	if hp.SegmentPath == nil {
		return abs.String()
	}
	return hp.SegmentPath(abs.String())
}

func (hp *HLSProxy) SegmentHandler(w http.ResponseWriter, r *http.Request, upstreamURL string) {
	if err := ValidateGoogleURL(upstreamURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	isPlaylist := strings.Contains(strings.ToLower(upstreamURL), ".m3u8")
	if isPlaylist {
		resp, err := hp.fetchGET(r.Context(), upstreamURL, "")
		if err != nil {
			http.Error(w, "fetch failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			http.Error(w, fmt.Sprintf("upstream status %d", resp.StatusCode), http.StatusBadGateway)
			return
		}
		body, err := readLimited(resp.Body, maxManifestBytes)
		if err != nil {
			http.Error(w, "read failed", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(hp.rewrite(body, upstreamURL))
		return
	}

	fp := FileProxy{Fetcher: hp.Fetcher}
	fp.Handler(w, r, upstreamURL, "")
}

var copyBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 512*1024)
		return &buf
	},
}

func copyStream(w http.ResponseWriter, body io.Reader) {
	bufPtr := copyBufPool.Get().(*[]byte)
	defer copyBufPool.Put(bufPtr)
	_, _ = io.CopyBuffer(w, body, *bufPtr)
}
