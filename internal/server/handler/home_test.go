package handler_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/server/handler"
	"go-htmx-template/internal/server/middleware"
)

func newHomeHandler() *handler.Handler {
	return handler.New(slog.New(slog.DiscardHandler), nil)
}

// TestHome_Headers asserts everything the handler decides before the body:
// status, cacheability, and the type and length of the buffered render.
func TestHome_Headers(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	newHomeHandler().Home(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Equal(t, strconv.Itoa(rec.Body.Len()), rec.Header().Get("Content-Length"))
}

func TestHome_UsesExplicitAccessiblePalette(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	body := rec.Body.String()
	assert.Contains(t, body, "bg-white")
	assert.Contains(t, body, "text-slate-900")
	assert.Contains(t, body, "dark:bg-slate-950")
	assert.NotContains(t, body, "text-indigo-200")
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

func TestHome_RendersStatelessCounterForm(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	body := rec.Body.String()
	assert.Contains(t, body, `name="count" value="0"`)
	assert.Contains(t, body, `hx-indicator="#counter-spinner"`)
	assert.Contains(t, body, `hx-disable="#counter-button"`)
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

func TestCount_IncrementsSubmittedValue(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	form := url.Values{"count": {"41"}}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.Count(rec, req)

	count, ok := extractCountFromBody(rec.Body.String())
	require.True(t, ok, "could not extract count from response")
	assert.Equal(t, 42, count)
}

func TestCount_RejectsOversizedBody(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	form := url.Values{"count": {strings.Repeat("1", 2<<20)}}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	middleware.BodyLimit(1<<20)(http.HandlerFunc(h.Count)).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Contains(t, rec.Body.String(), `id="error-region"`)
}

func TestCount_RejectsOutOfRangeValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{name: "negative", value: "-1"},
		{name: "max int", value: "9223372036854775807"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newHomeHandler()
			form := url.Values{"count": {tt.value}}
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			h.Count(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestCount_RejectsInvalidValue(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	form := url.Values{"count": {"nope"}}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.Count(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
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
