package server

import (
	"sync"
	"time"
)

// RateLimiter is a sliding-window-log limiter per key with a bounded number
// of tracked keys. When the table is full, new keys are rejected (fail
// closed) until entries expire; the janitor goroutine stops with Stop.
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	maxKeys int
	hits    map[string][]time.Time
	now     func() time.Time
	stop    chan struct{}
	once    sync.Once
}

func NewRateLimiter(limit int, window time.Duration, maxKeys int) *RateLimiter {
	l := &RateLimiter{limit: limit, window: window, maxKeys: maxKeys, hits: map[string][]time.Time{},
		now: time.Now, stop: make(chan struct{})}
	go l.janitor()
	return l
}

// Allow records a hit for key and reports whether it is within the limit;
// otherwise it returns the time until a slot frees up.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	cutoff := now.Add(-l.window)
	hits := l.hits[key]
	kept := hits[:0]
	for _, t := range hits {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.hits, key)
		if len(l.hits) >= l.maxKeys {
			return false, l.window
		}
	}
	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false, kept[0].Sub(cutoff)
	}
	l.hits[key] = append(kept, now)
	return true, 0
}

func (l *RateLimiter) janitor() {
	ticker := time.NewTicker(l.window)
	defer ticker.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-ticker.C:
			l.mu.Lock()
			cutoff := l.now().Add(-l.window)
			for key, hits := range l.hits {
				if len(hits) == 0 || !hits[len(hits)-1].After(cutoff) {
					delete(l.hits, key)
				}
			}
			l.mu.Unlock()
		}
	}
}

func (l *RateLimiter) Stop() { l.once.Do(func() { close(l.stop) }) }
