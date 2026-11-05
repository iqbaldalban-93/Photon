package progress

import (
	"sync"
	"time"
)

const (
	speedWindow  = 5 * time.Second
	speedMinSpan = time.Second
)

type speedSample struct {
	t time.Time
	b int64
}

// Meter turns a stream of "n more bytes were sent" calls into a smoothed
// bytes-per-second figure, using a short sliding window (the same idea as the
// speed readout in the main window). The zero value is ready to use.
type Meter struct {
	mu      sync.Mutex
	total   int64
	samples []speedSample
}

func (m *Meter) Add(n int64) {
	if n <= 0 {
		return
	}
	m.mu.Lock()
	m.total += n
	m.mu.Unlock()
}

func (m *Meter) Reset() {
	m.mu.Lock()
	m.total = 0
	m.samples = nil
	m.mu.Unlock()
}

// Rate samples the meter and returns the average speed over the last few
// seconds. Call it on a steady tick (about once a second); it returns 0 until
// there is enough history to be meaningful, and decays to 0 when nothing is
// being sent.
func (m *Meter) Rate() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if n := len(m.samples); n > 0 && now.Sub(m.samples[n-1].t) > speedWindow {
		m.samples = nil
	}
	m.samples = append(m.samples, speedSample{now, m.total})
	for len(m.samples) > 2 && now.Sub(m.samples[1].t) >= speedWindow {
		m.samples = m.samples[1:]
	}
	first := m.samples[0]
	span := now.Sub(first.t)
	if span < speedMinSpan {
		return 0
	}
	return max(0, float64(m.total-first.b)/span.Seconds())
}
