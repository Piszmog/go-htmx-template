package handler_test

import (
	"io"
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
	return &handler.Handler{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestHome_StatusCode(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHome_ContentType(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
}

func TestHome_ContainsWelcomeText(t *testing.T) {
	t.Parallel()

	h := newHomeHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	assert.Contains(t, rec.Body.String(), "Welcome!")
}

func TestCount_StatusCode(t *testing.T) {
	h := newHomeHandler()
	req := httptest.NewRequest(http.MethodPost, "/count", nil)
	rec := httptest.NewRecorder()

	h.Count(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCount_ContentType(t *testing.T) {
	h := newHomeHandler()
	req := httptest.NewRequest(http.MethodPost, "/count", nil)
	rec := httptest.NewRecorder()

	h.Count(rec, req)

	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
}

func TestCount_ContainsCounterFragment(t *testing.T) {
	h := newHomeHandler()
	req := httptest.NewRequest(http.MethodPost, "/count", nil)
	rec := httptest.NewRecorder()

	h.Count(rec, req)

	body := rec.Body.String()
	assert.Contains(t, body, `id="counter"`)
	assert.Contains(t, body, "Count: ")
}

func TestCount_IncrementsOnEachCall(t *testing.T) {
	h := newHomeHandler()

	req1 := httptest.NewRequest(http.MethodPost, "/count", nil)
	rec1 := httptest.NewRecorder()
	h.Count(rec1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/count", nil)
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
	idx := strings.Index(body, prefix)
	if idx < 0 {
		return 0, false
	}
	rest := body[idx+len(prefix):]
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
