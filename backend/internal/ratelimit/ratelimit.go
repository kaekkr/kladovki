// Package ratelimit is a small in-memory fixed-window limiter.
//
// State lives in this process only: after a restart (or on a second Render
// instance) the counters start from zero. That is enough to stop password
// guessing and form spam from one client; it is not a distributed quota.
package ratelimit

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Limiter allows at most limit events per key in each window.
// A nil Limiter, or one with limit <= 0, never refuses anything.
type Limiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	hits      map[string]*bucket
	now       func() time.Time
	lastSweep time.Time
}

type bucket struct {
	start time.Time
	count int
}

func New(limit int, window time.Duration) *Limiter {
	return &Limiter{
		limit:     limit,
		window:    window,
		hits:      map[string]*bucket{},
		now:       time.Now,
		lastSweep: time.Now(),
	}
}

func (l *Limiter) enabled() bool { return l != nil && l.limit > 0 && l.window > 0 }

// current returns the live bucket for key (nil when none). Callers hold mu.
func (l *Limiter) current(key string, now time.Time) *bucket {
	l.sweep(now)
	b := l.hits[key]
	if b == nil || now.Sub(b.start) >= l.window {
		return nil
	}
	return b
}

// sweep drops expired buckets once per window, so memory stays bounded by
// the keys seen in roughly the last two windows. Callers hold mu.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}
	for k, b := range l.hits {
		if now.Sub(b.start) >= l.window {
			delete(l.hits, k)
		}
	}
	l.lastSweep = now
}

// Allow records one event for key and reports whether it is within the limit.
func (l *Limiter) Allow(key string) bool {
	if !l.enabled() {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b := l.current(key, now)
	if b == nil {
		l.hits[key] = &bucket{start: now, count: 1}
		return true
	}
	if b.count >= l.limit {
		return false
	}
	b.count++
	return true
}

// Add records one event (e.g. a failed login) without checking the limit.
func (l *Limiter) Add(key string) {
	if !l.enabled() {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if b := l.current(key, now); b != nil {
		b.count++
		return
	}
	l.hits[key] = &bucket{start: now, count: 1}
}

// Blocked reports whether key has used up its budget in the current window.
func (l *Limiter) Blocked(key string) bool {
	if !l.enabled() {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.current(key, l.now())
	return b != nil && b.count >= l.limit
}

// RetryAfter is how long until key's window ends (0 when it has no events).
func (l *Limiter) RetryAfter(key string) time.Duration {
	if !l.enabled() {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b := l.current(key, now)
	if b == nil {
		return 0
	}
	return b.start.Add(l.window).Sub(now)
}

// Reset forgets key (e.g. after a successful login).
func (l *Limiter) Reset(key string) {
	if !l.enabled() {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}

func (l *Limiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.hits)
}

// TooMany writes the 429 answer with a Retry-After header (whole seconds).
func TooMany(c *gin.Context, retry time.Duration, msg string) {
	secs := max(int(math.Ceil(retry.Seconds())), 1)
	c.Header("Retry-After", strconv.Itoa(secs))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": msg, "code": "rate_limited"})
}

// PerIP limits requests by c.ClientIP() — which is only the real client when
// trusted proxies are configured correctly (see docs/operations.md).
func PerIP(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !l.Allow(ip) {
			TooMany(c, l.RetryAfter(ip), "Слишком много запросов. Попробуйте через несколько минут.")
			return
		}
		c.Next()
	}
}
