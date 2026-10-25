//nolint:testpackage // asserts unexported server configuration fields
package server

import (
	"log/slog"
	"net/http"
	"testing"
)

func TestNew_ConfiguresTimeouts(t *testing.T) {
	t.Parallel()

	s := New(slog.New(slog.DiscardHandler), ":0")

	tests := []struct {
		name string
		got  any
		want any
	}{
		{name: "read header timeout", got: s.srv.ReadHeaderTimeout, want: readHeaderTimeout},
		{name: "read timeout", got: s.srv.ReadTimeout, want: readTimeout},
		{name: "write timeout", got: s.srv.WriteTimeout, want: writeTimeout},
		{name: "idle timeout", got: s.srv.IdleTimeout, want: idleTimeout},
		{name: "max header bytes", got: s.srv.MaxHeaderBytes, want: maxHeaderBytes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Fatalf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}

	if s.srv.ErrorLog == nil {
		t.Fatal("ErrorLog should be wired to the structured logger")
	}
}

func TestWithRouter(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	s := New(slog.New(slog.DiscardHandler), ":0", WithRouter(handler))

	if s.srv.Handler != handler {
		t.Fatal("WithRouter should set the server handler")
	}
}
