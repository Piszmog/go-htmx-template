package middleware_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-htmx-template/internal/server/middleware"
)

func TestBodyLimit(t *testing.T) {
	t.Parallel()

	handler := middleware.BodyLimit(4)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "within limit", body: "1234", want: http.StatusOK},
		{name: "over limit", body: "12345", want: http.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.want, rec.Code)
		})
	}
}
