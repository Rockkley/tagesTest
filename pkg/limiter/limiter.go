package limiter

import (
	"context"
	"errors"
	"sync"
)

type Limiter struct {
	limit int
	count int
	mu    sync.Mutex
}

func NewLimiter(limit int) *Limiter {
	if limit <= 0 {
		limit = 1
	}
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
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				l.mu.Lock()
			}
		}
	}

	l.count++
	return nil
}

func (l *Limiter) Release() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.count <= 0 {
		return errors.New("release called more times than acquire")
	}
	l.count--
	return nil
}
