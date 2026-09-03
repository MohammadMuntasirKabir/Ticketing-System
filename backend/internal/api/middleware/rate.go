package middleware

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/you/airline-ticketing/internal/api/helper"
)

// IPLimiter is a per-client token-bucket rate limiter. Each unique IP
// gets its own bucket, allocated on demand.
type IPLimiter struct {
	mu       sync.Mutex
	visitors map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func NewIPLimiter(r rate.Limit, b int) *IPLimiter {
	return &IPLimiter{visitors: make(map[string]*rate.Limiter), r: r, b: b}
}

func (l *IPLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lim, ok := l.visitors[ip]; ok {
		return lim
	}
	lim := rate.NewLimiter(l.r, l.b)
	l.visitors[ip] = lim
	return lim
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		if i := indexByte(ip, ','); i >= 0 {
			ip = ip[:i]
		}
		return trimSpace(ip)
	}
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// RateLimit returns middleware that rejects requests exceeding the
// configured per-IP rate with 429 Too Many Requests.
func RateLimit(l *IPLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.get(clientIP(r)).Allow() {
				helper.JSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
