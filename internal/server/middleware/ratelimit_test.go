package middleware_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go-htmx-template/internal/server/middleware"
)

func newHandler(t *testing.T, rpm, maxEntries int) http.Handler {
	t.Helper()
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	ipCfg := middleware.IPConfig{TrustProxyHeaders: false}
	m := middleware.RateLimit(t.Context(), slog.Default(), rpm, maxEntries, ipCfg)
	return m(ok)
}

func request(handler http.Handler, ip string) int {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip + ":9999"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr.Code
}

// TestAllowDeny verifies requests within burst pass and excess are rejected.
func TestAllowDeny(t *testing.T) {
	t.Parallel()
	h := newHandler(t, 3, 1000)

	for i := range 3 {
		if code := request(h, "1.2.3.4"); code != http.StatusOK {
			t.Fatalf("request %d: want 200, got %d", i+1, code)
		}
	}
	if code := request(h, "1.2.3.4"); code != http.StatusTooManyRequests {
		t.Fatalf("4th request: want 429, got %d", code)
	}
}

// TestDifferentIPsAreIndependent confirms separate IPs have independent buckets.
func TestDifferentIPsAreIndependent(t *testing.T) {
	t.Parallel()
	h := newHandler(t, 1, 1000)

	if code := request(h, "1.1.1.1"); code != http.StatusOK {
		t.Fatalf("first IP first request: want 200, got %d", code)
	}
	if code := request(h, "2.2.2.2"); code != http.StatusOK {
		t.Fatalf("second IP first request: want 200, got %d", code)
	}
	if code := request(h, "1.1.1.1"); code != http.StatusTooManyRequests {
		t.Fatalf("first IP second request: want 429, got %d", code)
	}
	if code := request(h, "2.2.2.2"); code != http.StatusTooManyRequests {
		t.Fatalf("second IP second request: want 429, got %d", code)
	}
}

// TestLRUEviction confirms the LRU entry is evicted at capacity and receives a
// fresh token bucket on its next request.
func TestLRUEviction(t *testing.T) {
	t.Parallel()
	h := newHandler(t, 10, 2)

	// Exhaust ip-A's tokens (burst=10).
	for range 10 {
		request(h, "10.0.0.1")
	}
	if code := request(h, "10.0.0.1"); code != http.StatusTooManyRequests {
		t.Fatalf("ip-A should be rate-limited before eviction, got %d", code)
	}

	// ip-B: LRU order = [ip-B, ip-A]
	request(h, "10.0.0.2")
	// ip-C: evicts ip-A (least recently seen), LRU order = [ip-C, ip-B]
	request(h, "10.0.0.3")

	// ip-A was evicted; its next request creates a fresh bucket — should be allowed.
	if code := request(h, "10.0.0.1"); code != http.StatusOK {
		t.Fatalf("evicted ip-A should get a fresh bucket, want 200, got %d", code)
	}
}

// TestMaxEntriesOne confirms that only one IP is tracked at a time.
func TestMaxEntriesOne(t *testing.T) {
	t.Parallel()
	h := newHandler(t, 10, 1)

	// Exhaust ip-A.
	for range 10 {
		request(h, "10.0.0.1")
	}
	if code := request(h, "10.0.0.1"); code != http.StatusTooManyRequests {
		t.Fatalf("ip-A should be rate-limited, got %d", code)
	}

	// ip-B displaces ip-A (only 1 slot).
	request(h, "10.0.0.2")

	// ip-A re-enters with a fresh bucket.
	if code := request(h, "10.0.0.1"); code != http.StatusOK {
		t.Fatalf("evicted ip-A should get a fresh bucket, want 200, got %d", code)
	}
}

// TestConcurrentAccess races many goroutines through allow() to surface data races.
// Run with: go test -race ./internal/server/middleware/...
func TestConcurrentAccess(t *testing.T) {
	t.Parallel()
	h := newHandler(t, 1000, 500)

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ip := "10.0.0." + itoa(id%50)
			for range 10 {
				request(h, ip)
			}
		}(i)
	}
	wg.Wait()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
