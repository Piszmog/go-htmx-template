package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimit returns a middleware that rate limits requests per IP address.
// Uses token bucket algorithm with in-memory storage (suitable for single instance).
// For distributed systems, consider using Redis-backed rate limiter.
//
// Parameters:
//   - requestsPerMinute: Maximum requests allowed per IP per minute (default: 50)
func RateLimit(logger *slog.Logger, requestsPerMinute int) Handler {
	limiter := &ipRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(float64(requestsPerMinute) / 60.0), // Convert to per-second
		burst:    requestsPerMinute,                             // Allow burst up to limit
		logger:   logger,
	}

	// Cleanup stale entries every 10 minutes to prevent memory growth
	go limiter.cleanupLoop()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract IP address (handle proxy headers from Caddy)
			ip := getClientIP(r)

			// Check rate limit
			if !limiter.allow(ip) {
				logger.Warn("rate limit exceeded",
					slog.String("ip", ip),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)

				w.Header().Set("Retry-After", "60") // Suggest retry after 60 seconds
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ipRateLimiter manages rate limiters per IP address.
type ipRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
	logger   *slog.Logger
}

// allow checks if a request from the given IP should be allowed.
func (i *ipRateLimiter) allow(ip string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(i.rate, i.burst)
		i.limiters[ip] = limiter
	}

	return limiter.Allow()
}

// cleanupLoop removes stale rate limiters every 10 minutes.
func (i *ipRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		i.mu.Lock()
		// Remove limiters that haven't been used recently
		// This prevents memory growth for IPs that only visit once
		for ip, limiter := range i.limiters {
			// If limiter has full tokens, it hasn't been used recently
			if limiter.Tokens() == float64(i.burst) {
				delete(i.limiters, ip)
			}
		}
		count := len(i.limiters)
		i.mu.Unlock()

		i.logger.Debug("rate limiter cleanup", slog.Int("active_ips", count))
	}
}

// getClientIP extracts the real client IP, handling reverse proxy headers.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by Caddy/nginx)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP in the chain (original client)
		if ip, _, err := net.SplitHostPort(xff); err == nil {
			return ip
		}
		return xff
	}

	// Check X-Real-IP header (alternative proxy header)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	return r.RemoteAddr
}
