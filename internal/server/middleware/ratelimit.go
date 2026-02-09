package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimit returns a middleware that rate limits requests per IP address using
// a token bucket algorithm with in-memory storage.
func RateLimit(ctx context.Context, logger *slog.Logger, requestsPerMinute int, ipCfg IPConfig) Handler {
	limiter := &ipRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(float64(requestsPerMinute) / 60.0),
		burst:    requestsPerMinute,
		logger:   logger,
		ipCfg:    ipCfg,
	}

	go limiter.cleanupLoop(ctx)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetClientIP(r, limiter.ipCfg)

			if !limiter.allow(ip) {
				logger.Warn("rate limit exceeded",
					slog.String("ip", ip),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)

				w.Header().Set("Retry-After", "60")
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type ipRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
	logger   *slog.Logger
	ipCfg    IPConfig
}

func (i *ipRateLimiter) allow(ip string) bool {
	i.mu.RLock()
	limiter, exists := i.limiters[ip]
	i.mu.RUnlock()

	if exists {
		return limiter.Allow()
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists = i.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(i.rate, i.burst)
		i.limiters[ip] = limiter
	}

	return limiter.Allow()
}

func (i *ipRateLimiter) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			i.mu.Lock()
			for ip, limiter := range i.limiters {
				// Epsilon avoids floating-point precision issues that could
				// prevent cleanup of idle IPs when using exact equality.
				if limiter.Tokens() >= float64(i.burst)-0.01 {
					delete(i.limiters, ip)
				}
			}
			count := len(i.limiters)
			i.mu.Unlock()

			i.logger.Debug("rate limiter cleanup", slog.Int("active_ips", count))
		}
	}
}
