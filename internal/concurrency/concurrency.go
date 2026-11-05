package concurrency

import (
	"context"
	"sync"
	"sync/atomic"
)

type AdaptiveConcurrency struct {
	mu     sync.Mutex
	active int
	limit  int
	max    int
	min    int
	wake   chan struct{}

	cleanStreak    int32
	onLimitChanged func(limit, max int)
}

func NewAdaptiveConcurrency(maxWorkers int, onLimitChanged func(limit, max int)) *AdaptiveConcurrency {
	if maxWorkers < 1 {
		maxWorkers = 1
	}
	return &AdaptiveConcurrency{
		limit:          maxWorkers,
		max:            maxWorkers,
		min:            1,
		wake:           make(chan struct{}),
		onLimitChanged: onLimitChanged,
	}
}

func (c *AdaptiveConcurrency) broadcast() {
	close(c.wake)
	c.wake = make(chan struct{})
}

func (c *AdaptiveConcurrency) Acquire(ctx context.Context) bool {
	for {
		if ctx.Err() != nil {
			return false
		}
		c.mu.Lock()
		if c.active < c.limit {
			c.active++
			c.mu.Unlock()
			return true
		}
		wake := c.wake
		c.mu.Unlock()

		select {
		case <-wake:
		case <-ctx.Done():
			return false
		}
	}
}

func (c *AdaptiveConcurrency) Release() {
	c.mu.Lock()
	if c.active > 0 {
		c.active--
	}
	c.broadcast()
	c.mu.Unlock()
}

func (c *AdaptiveConcurrency) ReportRetry() {
	atomic.StoreInt32(&c.cleanStreak, 0)
	c.mu.Lock()
	if c.limit <= c.min {
		c.mu.Unlock()
		return
	}
	c.limit--
	c.broadcast()
	limit, max := c.limit, c.max
	c.mu.Unlock()
	if c.onLimitChanged != nil {
		c.onLimitChanged(limit, max)
	}
}

func (c *AdaptiveConcurrency) ReportSuccess() {
	const recoverAfter = 5
	streak := atomic.AddInt32(&c.cleanStreak, 1)
	if streak < recoverAfter {
		return
	}
	atomic.StoreInt32(&c.cleanStreak, 0)
	c.mu.Lock()
	if c.limit >= c.max {
		c.mu.Unlock()
		return
	}
	c.limit++
	c.broadcast()
	limit, max := c.limit, c.max
	c.mu.Unlock()
	if c.onLimitChanged != nil {
		c.onLimitChanged(limit, max)
	}
}

func (c *AdaptiveConcurrency) CurrentLimit() (limit, max int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.limit, c.max
}
