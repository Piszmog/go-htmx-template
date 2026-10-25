package log_test

import (
	"testing"

	"go-htmx-template/internal/log"
)

func TestGetLevel(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  log.Level
	}{
		{name: "default", value: "", want: log.LevelInfo},
		{name: "lowercase", value: "debug", want: log.LevelDebug},
		{name: "uppercase", value: "DEBUG", want: log.LevelDebug},
		{name: "whitespace and mixed case", value: " Warn ", want: log.LevelWarn},
		{name: "error", value: "error", want: log.LevelError},
		{name: "unknown falls back to info", value: "verbose", want: log.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tt.value)

			if got := log.GetLevel(); got != tt.want {
				t.Fatalf("GetLevel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetOutput(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  log.Output
	}{
		{name: "default", value: "", want: log.OutputText},
		{name: "json", value: "json", want: log.OutputJSON},
		{name: "uppercase json", value: "JSON", want: log.OutputJSON},
		{name: "whitespace", value: " text ", want: log.OutputText},
		{name: "unknown falls back to text", value: "xml", want: log.OutputText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LOG_OUTPUT", tt.value)

			if got := log.GetOutput(); got != tt.want {
				t.Fatalf("GetOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}
