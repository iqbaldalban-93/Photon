package progress

import (
	"io"
	"sync/atomic"
	"time"
)

type Reader struct {
	reader       io.Reader
	total        int64
	read         atomic.Int64
	lastEmit     time.Time
	emitInterval time.Duration
	onProgress   func(bytesRead, totalBytes int64)
}

func New(reader io.Reader, total int64, onProgress func(bytesRead, totalBytes int64)) *Reader {
	return &Reader{
		reader:       reader,
		total:        total,
		emitInterval: 100 * time.Millisecond,
		onProgress:   onProgress,
	}
}

func (pr *Reader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	if n > 0 || err == io.EOF {
		currentRead := pr.read.Load()
		if n > 0 {
			currentRead = pr.read.Add(int64(n))
		}

		now := time.Now()
		if now.Sub(pr.lastEmit) >= pr.emitInterval || err == io.EOF {
			pr.lastEmit = now
			if pr.onProgress != nil {
				pr.onProgress(currentRead, pr.total)
			}
		}
	}
	return n, err
}

func (pr *Reader) BytesRead() int64 {
	return pr.read.Load()
}
