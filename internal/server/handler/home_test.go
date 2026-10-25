package handler_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/server/handler"
)

func newHomeHandler() *handler.Handler {
	return handler.New(slog.New(slog.DiscardHandler), nil)
}

func TestHome_StatusCode(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHome_ContentType(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
}

func TestHome_ContainsWelcomeText(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	assert.Contains(t, rec.Body.String(), "Welcome!")
}

func TestHome_RendersHTMX4(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	body := rec.Body.String()
	assert.Regexp(t, `src="/assets/js/htmx@v4\.[0-9.]+min\.js"`, body)
	// The page needs the out-of-band target for error responses.
	assert.Contains(t, body, `id="error-region"`)

	// Config removed in htmx 4 must not creep back in.
	for _, removed := range []string{"responseHandling", "methodsThatUseUrlParams", "includeIndicatorStyles"} {
		assert.NotContains(t, body, removed)
	}
}

func TestCount_StatusCode(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", nil)
	rec := httptest.NewRecorder()

	h.Count(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCount_ContentType(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", nil)
	rec := httptest.NewRecorder()

	h.Count(rec, req)

	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
}

func TestCount_ContainsCounterFragment(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", nil)
	rec := httptest.NewRecorder()

	h.Count(rec, req)

	body := rec.Body.String()
	assert.Contains(t, body, `id="counter"`)
	assert.Contains(t, body, "Count: ")
}

func TestCount_ClearsErrorRegion(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", nil)
	rec := httptest.NewRecorder()

	h.Count(rec, req)

	body := rec.Body.String()
	assert.Contains(t, body, `id="error-region"`)
	assert.Contains(t, body, `hx-swap-oob="true"`)
}

//nolint:paralleltest // shares package-level counter; two sequential calls must not be interleaved
func TestCount_IncrementsOnEachCall(t *testing.T) {
	h := newHomeHandler()

	req1 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", nil)
	rec1 := httptest.NewRecorder()
	h.Count(rec1, req1)

	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", nil)
	rec2 := httptest.NewRecorder()
	h.Count(rec2, req2)

	count1, ok1 := extractCountFromBody(rec1.Body.String())
	count2, ok2 := extractCountFromBody(rec2.Body.String())

	require.True(t, ok1, "could not extract count from first response")
	require.True(t, ok2, "could not extract count from second response")
	assert.Equal(t, count1+1, count2, "each call should increment the counter by 1")
}

// extractCountFromBody parses the integer after "Count: " in the response body.
func extractCountFromBody(body string) (int, bool) {
	const prefix = "Count: "
	_, rest, found := strings.Cut(body, prefix)
	if !found {
		return 0, false
	}
	end := strings.IndexFunc(rest, func(r rune) bool {
		return r < '0' || r > '9'
	})
	if end <= 0 {
		return 0, false
	}
	n, err := strconv.Atoi(rest[:end])
	if err != nil {
		return 0, false
	}
	return n, true
}
