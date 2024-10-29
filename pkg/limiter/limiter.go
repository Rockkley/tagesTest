package limiter

import (
	"context"
	"sync"
)

type Limiter struct {
	limit int
	count int
	mu    sync.Mutex
}

func NewLimiter(limit int) *Limiter {
	return &Limiter{limit: limit}
}

func (l *Limiter) Acquire(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for l.count >= l.limit {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			l.mu.Unlock()
			l.mu.Lock()
		}
	}

	l.count++
	return nil
}

func (l *Limiter) Release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.count--
}
