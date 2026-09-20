package httpapi

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type limiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newLimiter() *limiter {
	return &limiter{hits: map[string][]time.Time{}}
}

func (l *limiter) wrap(perMinute int, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r) + ":" + r.Method + ":" + r.URL.Path
		if l != nil && !l.allow(key, perMinute) {
			writeErr(w, http.StatusTooManyRequests, "请求过多")
			return
		}
		next(w, r)
	}
}

func (l *limiter) allow(key string, perMinute int) bool {
	now := time.Now()
	cut := now.Add(-time.Minute)
	l.mu.Lock()
	defer l.mu.Unlock()
	prev := l.hits[key]
	kept := prev[:0]
	for _, t := range prev {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= perMinute {
		l.hits[key] = kept
		l.prune(cut)
		return false
	}
	l.hits[key] = append(kept, now)
	l.prune(cut)
	return true
}

func (l *limiter) prune(cut time.Time) {
	if len(l.hits) < 1024 {
		return
	}
	for key, times := range l.hits {
		alive := false
		for _, t := range times {
			if t.After(cut) {
				alive = true
				break
			}
		}
		if !alive {
			delete(l.hits, key)
		}
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
