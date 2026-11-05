package hashx

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"sync"
)

const copyBufferSize = 1 * 1024 * 1024

var hashBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, copyBufferSize)
		return &buf
	},
}

type ctxWriter struct {
	ctx context.Context
	w   io.Writer
}

func (cw *ctxWriter) Write(p []byte) (int, error) {
	if err := cw.ctx.Err(); err != nil {
		return 0, err
	}
	return cw.w.Write(p)
}

type readerOnly struct{ io.Reader }

func CalculateSHA1(ctx context.Context, filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer func() { _ = file.Close() }()

	hash := sha1.New()
	cw := &ctxWriter{ctx: ctx, w: hash}

	bufPtr := hashBufferPool.Get().(*[]byte)
	defer hashBufferPool.Put(bufPtr)
	_, err = io.CopyBuffer(cw, readerOnly{file}, *bufPtr)
	if err != nil {
		return nil, fmt.Errorf("error calculating hash: %w", err)
	}

	return hash.Sum(nil), nil
}
