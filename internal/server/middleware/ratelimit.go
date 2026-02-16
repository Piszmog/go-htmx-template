package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const defaultMaxEntries = 10000

// RateLimit returns a middleware that rate limits requests per IP address using
// a token bucket algorithm with in-memory storage.
func RateLimit(ctx context.Context, logger *slog.Logger, requestsPerMinute int, ipCfg IPConfig) Handler {
	limiter := &ipRateLimiter{
		limiters:   make(map[string]*ipEntry),
		rate:       rate.Limit(float64(requestsPerMinute) / 60.0),
		burst:      requestsPerMinute,
		maxEntries: defaultMaxEntries,
		logger:     logger,
		ipCfg:      ipCfg,
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

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu         sync.RWMutex
	limiters   map[string]*ipEntry
	rate       rate.Limit
	burst      int
	maxEntries int
	logger     *slog.Logger
	ipCfg      IPConfig
}

func (i *ipRateLimiter) allow(ip string) bool {
	now := time.Now()

	i.mu.RLock()
	entry, exists := i.limiters[ip]
	i.mu.RUnlock()

	if exists {
		allowed := entry.limiter.Allow()
		i.mu.Lock()
		entry.lastSeen = now
		i.mu.Unlock()
		return allowed
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	// Double-check after acquiring write lock.
	entry, exists = i.limiters[ip]
	if exists {
		entry.lastSeen = now
		return entry.limiter.Allow()
	}

	// At capacity: evict the oldest idle entry instead of rejecting.
	if len(i.limiters) >= i.maxEntries {
		i.evictOldest()
	}

	entry = &ipEntry{
		limiter:  rate.NewLimiter(i.rate, i.burst),
		lastSeen: now,
	}
	i.limiters[ip] = entry

	return entry.limiter.Allow()
}

// evictOldest removes the entry with the oldest lastSeen time.
// Must be called with i.mu held.
func (i *ipRateLimiter) evictOldest() {
	var oldestIP string
	var oldestTime time.Time

	for ip, entry := range i.limiters {
		if oldestIP == "" || entry.lastSeen.Before(oldestTime) {
			oldestIP = ip
			oldestTime = entry.lastSeen
		}
	}

	if oldestIP != "" {
		delete(i.limiters, oldestIP)
	}
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
			cutoff := time.Now().Add(-1 * time.Hour)
			for ip, entry := range i.limiters {
				if entry.lastSeen.Before(cutoff) {
					delete(i.limiters, ip)
				}
			}
			count := len(i.limiters)
			i.mu.Unlock()

			i.logger.Debug("rate limiter cleanup", slog.Int("active_ips", count))
		}
	}
}
