package httpx

import (
	"math"
	"sync"
	"time"
)

const sweepInterval = time.Minute

type bucket struct {
	tokens float64
	last   time.Time
}

type Limiter struct {
	mu        sync.Mutex
	rate      float64
	burst     float64
	now       func() time.Time
	buckets   map[string]*bucket
	lastSweep time.Time
}

func NewLimiter(perSecond float64, burst int) *Limiter {
	return &Limiter{
		rate:    perSecond,
		burst:   float64(burst),
		now:     time.Now,
		buckets: make(map[string]*bucket),
	}
}

func NewLimiterPer(attempts int, window time.Duration) *Limiter {
	return NewLimiter(float64(attempts)/window.Seconds(), attempts)
}

func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.refilled(key)
	if b.tokens < 1 {
		return false, l.waitFor(b)
	}
	b.tokens--
	return true, 0
}

func (l *Limiter) Check(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.refilled(key)
	if b.tokens < 1 {
		return false, l.waitFor(b)
	}
	return true, 0
}

func (l *Limiter) Hit(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.refilled(key)
	b.tokens = math.Max(0, b.tokens-1)
}

func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}

func (l *Limiter) refilled(key string) *bucket {
	now := l.now()
	l.sweep(now)
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
		return b
	}
	if elapsed := now.Sub(b.last).Seconds(); elapsed > 0 {
		b.tokens = math.Min(l.burst, b.tokens+elapsed*l.rate)
		b.last = now
	}
	return b
}

func (l *Limiter) waitFor(b *bucket) time.Duration {
	if l.rate <= 0 {
		return time.Hour
	}
	return time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
}

func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < sweepInterval {
		return
	}
	l.lastSweep = now
	for key, b := range l.buckets {
		if b.tokens+now.Sub(b.last).Seconds()*l.rate >= l.burst {
			delete(l.buckets, key)
		}
	}
}
