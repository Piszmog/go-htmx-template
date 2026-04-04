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

const (
	secondsPerMinute = 60.0
	cleanupInterval  = 10 * time.Minute
	staleEntryCutoff = time.Hour
)

// DefaultMaxEntries is the default cap on the number of IPs tracked by RateLimit.
const DefaultMaxEntries = 10000

// RateLimit returns a middleware that rate limits requests per IP address using
// a token bucket algorithm with in-memory storage. maxEntries caps the number
// of IPs tracked simultaneously; use defaultMaxEntries if unsure.
func RateLimit(ctx context.Context, logger *slog.Logger, requestsPerMinute int, maxEntries int, ipCfg IPConfig) Handler {
	rl := &ipRateLimiter{
		limiters:   make(map[string]*list.Element),
		order:      list.New(),
		rate:       rate.Limit(float64(requestsPerMinute) / secondsPerMinute),
		burst:      requestsPerMinute,
		maxEntries: maxEntries,
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
		entry, ok := elem.Value.(*ipEntry)
		if !ok {
			rl.logger.Error("rate limiter: unexpected type in list element")
			return true
		}
		entry.lastSeen = now
		rl.order.MoveToFront(elem)
		return entry.limiter.Allow()
	}

	// At capacity: evict the least recently seen entry.
	if len(rl.limiters) >= rl.maxEntries {
		back := rl.order.Back()
		if back != nil {
			evicted, ok := rl.order.Remove(back).(*ipEntry)
			if ok {
				delete(rl.limiters, evicted.ip)
				rl.logger.Warn("rate limiter evicted entry at capacity",
					slog.String("evicted_ip", evicted.ip),
					slog.Int("max_entries", rl.maxEntries),
				)
			}
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
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-staleEntryCutoff)
			// Iterate the entire list — MoveToFront provides approximate ordering
			// but does not guarantee strict lastSeen order, so breaking early could
			// miss stale entries positioned before recently-seen ones.
			for elem := rl.order.Back(); elem != nil; {
				prev := elem.Prev()
				entry, ok := elem.Value.(*ipEntry)
				if !ok {
					rl.logger.Error("rate limiter cleanup: unexpected type in list element")
					elem = prev
					continue
				}
				if entry.lastSeen.Before(cutoff) {
					rl.order.Remove(elem)
					delete(rl.limiters, entry.ip)
				}
				elem = prev
			}
			count := len(rl.limiters)
			rl.mu.Unlock()

			rl.logger.Debug("rate limiter cleanup", slog.Int("active_ips", count))
		}
	}
}
