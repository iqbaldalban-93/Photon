package server

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"time"
)

const maxUploadFileBytes = 64 << 30

var errUploadTooLarge = errors.New("uploaded file is too large")

func savePart(part *multipart.Part, dest string) (err error) {
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
		if err != nil {
			_ = os.Remove(dest)
		}
	}()

	br := newPeekReader(part)
	var src io.Reader = br
	if head, _ := br.peek(2); len(head) == 2 && head[0] == 0x1f && head[1] == 0x8b {
		zr, zerr := gzip.NewReader(br)
		if zerr != nil {
			return zerr
		}
		defer func() { _ = zr.Close() }()
		src = zr
	}
	n, err := io.Copy(f, io.LimitReader(src, maxUploadFileBytes+1))
	if err != nil {
		return err
	}
	if n > maxUploadFileBytes {
		return errUploadTooLarge
	}
	return nil
}

func sleepSeconds(n int) { time.Sleep(time.Duration(n) * time.Second) }

type peekReader struct {
	src  io.Reader
	buf  []byte
	used bool
}

func newPeekReader(src io.Reader) *peekReader { return &peekReader{src: src} }

func (p *peekReader) peek(n int) ([]byte, error) {
	if p.used || p.buf != nil {
		return p.buf, nil
	}
	buf := make([]byte, n)
	m, err := io.ReadFull(p.src, buf)
	if err != nil && m == 0 {
		return nil, err
	}
	p.buf = buf[:m]
	p.used = true
	return p.buf, nil
}

func (p *peekReader) Read(b []byte) (int, error) {
	if p.buf != nil {
		n := copy(b, p.buf)
		p.buf = p.buf[n:]
		if n > 0 {
			return n, nil
		}
	}
	return p.src.Read(b)
}

func chunkStrings(in []string, size int) [][]string {
	if len(in) == 0 {
		return nil
	}
	var out [][]string
	for i := 0; i < len(in); i += size {
		end := min(i+size, len(in))
		out = append(out, in[i:end])
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func urlPathEscape(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r < 0x20 || r == '"' || r == '\\':
			continue
		default:
			sb.WriteRune(r)
		}
	}
	return strings.ReplaceAll(sb.String(), " ", "_")
}

func apiErr(err error) error {
	return fmt.Errorf("google photos API: %w", err)
}
