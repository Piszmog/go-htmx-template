package middleware

import "net/http"

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	http.ResponseWriter

	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

var _ http.ResponseWriter = (*responseWriter)(nil)

// newResponseWriter creates a new responseWriter.
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		bytesWritten:   0,
		wroteHeader:    false,
	}
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.wroteHeader {
		rw.statusCode = statusCode
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write captures bytes written and calls underlying Write.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Unwrap returns the underlying ResponseWriter for http.ResponseController.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
