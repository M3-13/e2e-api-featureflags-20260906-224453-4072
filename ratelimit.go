package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	rateLimitPerSecond = 10
	rateLimitBurst     = 10
)

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{buckets: make(map[string]*tokenBucket)}
}

// allow reports whether a request from the given client IP may proceed. A
// token bucket of rateLimitBurst capacity is refilled at rateLimitPerSecond
// tokens per second.
func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &tokenBucket{tokens: rateLimitBurst, lastRefill: now}
		rl.buckets[ip] = b
	} else {
		elapsed := now.Sub(b.lastRefill).Seconds()
		b.tokens += elapsed * rateLimitPerSecond
		if b.tokens > rateLimitBurst {
			b.tokens = rateLimitBurst
		}
		b.lastRefill = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// RateLimit limits POST/PUT/DELETE requests to rateLimitPerSecond per client
// IP (the host portion of r.RemoteAddr). Requests over the limit answer
// 429 {"error":"rate limit exceeded"}. GET requests are not limited.
func RateLimit(next http.Handler) http.Handler {
	rl := newRateLimiter()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete:
			if !rl.allow(clientIP(r)) {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
