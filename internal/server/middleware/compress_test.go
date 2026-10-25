package middleware_test

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/server/middleware"
)

func TestCompress_GzipsHTML(t *testing.T) {
	t.Parallel()

	body := "<html><body>" + strings.Repeat("hello world ", 100) + "</body></html>"

	handler := middleware.Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	assert.Contains(t, rec.Header().Get("Vary"), "Accept-Encoding")
	assert.Empty(t, rec.Header().Get("Content-Length"), "Content-Length must not describe the compressed body")

	reader, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, body, string(got))
}

func TestCompress_SkipsIncompressibleContent(t *testing.T) {
	t.Parallel()

	handler := middleware.Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not really a png"))
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "not really a png", rec.Body.String())
}

func TestCompress_HonoursAcceptEncoding(t *testing.T) {
	t.Parallel()

	handler := middleware.Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))

	for _, accept := range []string{"", "deflate", "identity"} {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		if accept != "" {
			req.Header.Set("Accept-Encoding", accept)
		}
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Empty(t, rec.Header().Get("Content-Encoding"), "Accept-Encoding %q", accept)
		assert.Equal(t, "hello", rec.Body.String(), "Accept-Encoding %q", accept)
		assert.Contains(t, rec.Header().Get("Vary"), "Accept-Encoding")
	}
}

func TestCompress_SkipsEmptyBodies(t *testing.T) {
	t.Parallel()

	handler := middleware.Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Content-Encoding"))
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestCompress_DoesNotDoubleCompress(t *testing.T) {
	t.Parallel()

	handler := middleware.Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "br")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pre-compressed"))
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "br", rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "pre-compressed", strings.TrimSpace(rec.Body.String()))
}

// TestCompress_SkipsTinyBodies covers claims and favicons, where the gzip frame
// would cost more than it saves and the accurate Content-Length must survive.
func TestCompress_SkipsTinyBodies(t *testing.T) {
	t.Parallel()

	const body = "tiny"

	handler := middleware.Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Content-Encoding"))
	assert.Equal(t, strconv.Itoa(len(body)), rec.Header().Get("Content-Length"))
	assert.Equal(t, body, rec.Body.String())
}

// TestCompress_HonoursExplicitRefusal guards `gzip;q=0`, which lists the token
// but marks it unacceptable.
func TestCompress_HonoursExplicitRefusal(t *testing.T) {
	t.Parallel()

	handler := middleware.Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello"))
	}))

	for _, accept := range []string{"gzip;q=0", "gzip; q=0.0", "deflate, gzip;q=0"} {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", accept)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Empty(t, rec.Header().Get("Content-Encoding"), "Accept-Encoding %q", accept)
		assert.Equal(t, "hello", rec.Body.String(), "Accept-Encoding %q", accept)
	}
}

// TestCompress_SkipsPartialContent guards Range requests: the bytes are a slice
// of the uncompressed representation, so they cannot carry Content-Encoding.
func TestCompress_SkipsPartialContent(t *testing.T) {
	t.Parallel()

	handler := middleware.Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		w.Header().Set("Content-Range", "bytes 0-4/100")
		w.Header().Set("Content-Length", "5")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("body{"))
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusPartialContent, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("Content-Encoding"))
	assert.Equal(t, "bytes 0-4/100", resp.Header.Get("Content-Range"))
	assert.Equal(t, "5", resp.Header.Get("Content-Length"))
}
