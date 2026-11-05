// Package ctxrelay wires a channel-based cancellation source into a context,
// so code built around a cancel channel (upload managers) can hand derived
// contexts to code built around context.Context (API calls).
package ctxrelay

import "context"

// RelayCancel derives a context that is cancelled as soon as src is closed.
// A nil src yields an unrelayed context. The returned cancel func must still
// be called (usually via defer) to release the relay goroutine.
func RelayCancel(src <-chan struct{}) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	if src != nil {
		go func() {
			select {
			case <-src:
				cancel()
			case <-ctx.Done():
			}
		}()
	}
	return ctx, cancel
}
