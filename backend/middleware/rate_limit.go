package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimitStore tracks request counts per IP address
type RateLimitStore struct {
	mu     sync.RWMutex
	counts map[string]*RateLimit
}

// RateLimit tracks requests for a single IP
type RateLimit struct {
	count     int
	resetTime time.Time
}

// NewRateLimitStore creates a new rate limiter
func NewRateLimitStore() *RateLimitStore {
	store := &RateLimitStore{
		counts: make(map[string]*RateLimit),
	}

	// Cleanup old entries every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			store.cleanup()
		}
	}()

	return store
}

// RateLimitMiddleware enforces request rate limiting per IP
// requestsPerMinute: max requests allowed per minute
func (s *RateLimitStore) RateLimitMiddleware(requestsPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)

			s.mu.Lock()
			now := time.Now()
			limit, exists := s.counts[clientIP]

			if !exists || now.After(limit.resetTime) {
				// New time window or first request
				s.counts[clientIP] = &RateLimit{
					count:     1,
					resetTime: now.Add(1 * time.Minute),
				}
				s.mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			if limit.count >= requestsPerMinute {
				s.mu.Unlock()
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Rate limit exceeded: 100 requests per minute"))
				return
			}

			limit.count++
			s.mu.Unlock()

			// Add rate limit info to response headers
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", "")
			w.Header().Set("X-RateLimit-Reset", "")

			next.ServeHTTP(w, r)
		})
	}
}

// cleanup removes expired entries
func (s *RateLimitStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for ip, limit := range s.counts {
		if now.After(limit.resetTime) {
			delete(s.counts, ip)
		}
	}
}

// getClientIP extracts client IP from request, handling proxies
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (from load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can have multiple IPs, take the first one
		ips := net.ParseIP(xff)
		if ips != nil {
			return xff
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Fall back to remote address
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
