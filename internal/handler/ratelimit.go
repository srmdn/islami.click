package handler

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimitBucket struct {
	started time.Time
	count   int
}

type rateLimiter struct {
	mu         sync.Mutex
	limit      int
	window     time.Duration
	maxEntries int
	buckets    map[string]rateLimitBucket
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		limit:      limit,
		window:     window,
		maxEntries: 10_000,
		buckets:    make(map[string]rateLimitBucket),
	}
}

func (l *rateLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	for existingKey, bucket := range l.buckets {
		if now.Sub(bucket.started) >= l.window {
			delete(l.buckets, existingKey)
		}
	}

	bucket, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= l.maxEntries {
			return false
		}
		l.buckets[key] = rateLimitBucket{started: now, count: 1}
		return true
	}
	if now.Sub(bucket.started) >= l.window {
		l.buckets[key] = rateLimitBucket{started: now, count: 1}
		return true
	}
	if bucket.count >= l.limit {
		return false
	}
	bucket.count++
	l.buckets[key] = bucket
	return true
}

func quizClientKey(r *http.Request) string {
	remoteHost := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		remoteHost = host
	}
	if remoteIP := net.ParseIP(remoteHost); remoteIP != nil && remoteIP.IsLoopback() {
		if forwardedIP := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); forwardedIP != nil {
			return forwardedIP.String()
		}
	}
	if remoteHost != "" {
		return remoteHost
	}
	return "unknown"
}

func allowQuizRequest(w http.ResponseWriter, r *http.Request, limiter *rateLimiter) bool {
	if limiter.Allow(quizClientKey(r), time.Now().UTC()) {
		return true
	}
	w.Header().Set("Retry-After", "60")
	http.Error(w, "too many requests", http.StatusTooManyRequests)
	return false
}
