package middleware

import (
	"container/list"
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
	rl := &ipRateLimiter{
		limiters:   make(map[string]*list.Element),
		order:      list.New(),
		rate:       rate.Limit(float64(requestsPerMinute) / 60.0),
		burst:      requestsPerMinute,
		maxEntries: defaultMaxEntries,
		logger:     logger,
		ipCfg:      ipCfg,
	}

	go rl.cleanupLoop(ctx)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetClientIP(r, rl.ipCfg)

			if !rl.allow(ip) {
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
	ip       string
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu         sync.Mutex
	limiters   map[string]*list.Element
	order      *list.List
	rate       rate.Limit
	burst      int
	maxEntries int
	logger     *slog.Logger
	ipCfg      IPConfig
}

func (rl *ipRateLimiter) allow(ip string) bool {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if elem, exists := rl.limiters[ip]; exists {
		entry := elem.Value.(*ipEntry)
		entry.lastSeen = now
		rl.order.MoveToFront(elem)
		return entry.limiter.Allow()
	}

	// At capacity: evict the least recently seen entry.
	if len(rl.limiters) >= rl.maxEntries {
		back := rl.order.Back()
		if back != nil {
			evicted := rl.order.Remove(back).(*ipEntry)
			delete(rl.limiters, evicted.ip)
		}
	}

	entry := &ipEntry{
		ip:       ip,
		limiter:  rate.NewLimiter(rl.rate, rl.burst),
		lastSeen: now,
	}
	elem := rl.order.PushFront(entry)
	rl.limiters[ip] = elem

	return entry.limiter.Allow()
}

func (rl *ipRateLimiter) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-1 * time.Hour)
			// Iterate from back (oldest) and stop early once we hit a recent entry.
			for elem := rl.order.Back(); elem != nil; {
				entry := elem.Value.(*ipEntry)
				if entry.lastSeen.After(cutoff) {
					break
				}
				prev := elem.Prev()
				rl.order.Remove(elem)
				delete(rl.limiters, entry.ip)
				elem = prev
			}
			count := len(rl.limiters)
			rl.mu.Unlock()

			rl.logger.Debug("rate limiter cleanup", slog.Int("active_ips", count))
		}
	}
}
