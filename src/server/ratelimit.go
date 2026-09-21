package server

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
)

type clientLimit struct {
	count     int
	resetTime time.Time
}

// RateLimiter provides IP-based sliding window rate limiting for sensitive endpoints.
type RateLimiter struct {
	mu          sync.Mutex
	limits      map[string]*clientLimit
	maxRequests int
	window      time.Duration
}

func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		limits:      make(map[string]*clientLimit),
		maxRequests: maxRequests,
		window:      window,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.window * 2)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, cl := range rl.limits {
			if now.After(cl.resetTime) {
				delete(rl.limits, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cl, exists := rl.limits[ip]
	if !exists || now.After(cl.resetTime) {
		rl.limits[ip] = &clientLimit{
			count:     1,
			resetTime: now.Add(rl.window),
		}
		return true
	}

	if cl.count >= rl.maxRequests {
		return false
	}

	cl.count++
	return true
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		ip := extractIP(r)
		if !rl.Allow(ip) {
			common.WriteErrorMessage(w, common.RATE_LIMIT_EXCEEDED_ERROR, "Too many requests. Please try again later.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return ip
	}
	return r.RemoteAddr
}
