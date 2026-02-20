package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-htmx-template/internal/server/middleware"
)

func TestChain_Empty(t *testing.T) {
	t.Parallel()

	mw := middleware.Chain()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestChain_SingleMiddleware(t *testing.T) {
	t.Parallel()

	var middlewareCalled bool
	mw := middleware.Chain(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec, req)

	assert.True(t, middlewareCalled, "single middleware should be called")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestChain_MultipleMiddlewareExecuteInOrder(t *testing.T) {
	t.Parallel()

	var order []string

	makeMiddleware := func(name string) middleware.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+":before")
				next.ServeHTTP(w, r)
				order = append(order, name+":after")
			})
		}
	}

	mw := middleware.Chain(
		makeMiddleware("first"),
		makeMiddleware("second"),
		makeMiddleware("third"),
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, []string{
		"first:before",
		"second:before",
		"third:before",
		"handler",
		"third:after",
		"second:after",
		"first:after",
	}, order, "first middleware should be the outermost wrapper")
}
