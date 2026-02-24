package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Limiter provides per-operation rate limiting to avoid account bans.
type Limiter struct {
	mu       sync.Mutex
	lastCall map[string]time.Time
	minGap   map[string]time.Duration
}

// DefaultLimiter returns a limiter with safe default intervals.
func DefaultLimiter() *Limiter {
	return &Limiter{
		lastCall: make(map[string]time.Time),
		minGap: map[string]time.Duration{
			"publish": 30 * time.Second,
			"like":    2 * time.Second,
			"comment": 5 * time.Second,
			"follow":  3 * time.Second,
			"search":  1 * time.Second,
			"default": 500 * time.Millisecond,
		},
	}
}

// Wait blocks until the operation can safely proceed.
func (l *Limiter) Wait(ctx context.Context, op string) error {
	l.mu.Lock()
	gap, ok := l.minGap[op]
	if !ok {
		gap = l.minGap["default"]
	}
	last, seen := l.lastCall[op]
	l.mu.Unlock()

	if seen {
		wait := gap - time.Since(last)
		if wait > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}
	}

	l.mu.Lock()
	l.lastCall[op] = time.Now()
	l.mu.Unlock()
	return nil
}
