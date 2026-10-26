package middleware

import (
	"compress/gzip"
	"net/http"
	"strconv"
	"strings"
)

// Compress gzips compressible responses when the client accepts it. It sets
// Vary: Accept-Encoding on every response so shared caches keep the compressed
// and identity variants apart.
func Compress() Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Vary", "Accept-Encoding")

			if !acceptsGzip(r) {
				next.ServeHTTP(w, r)
				return
			}

			gzw := &gzipResponseWriter{ResponseWriter: w}
			defer gzw.close()

			next.ServeHTTP(gzw, r)
		})
	}
}

// acceptsGzip reports whether the client accepts a gzip response. An encoding
// listed with q=0 is a refusal, so `gzip;q=0` must not count as acceptance.
// Wildcard-only clients get identity: that is always a valid response.
func acceptsGzip(r *http.Request) bool {
	for part := range strings.SplitSeq(r.Header.Get("Accept-Encoding"), ",") {
		encoding, params, _ := strings.Cut(part, ";")
		if strings.EqualFold(strings.TrimSpace(encoding), "gzip") {
			return !zeroQuality(params)
		}
	}
	return false
}

// zeroQuality reports whether parameters such as "q=0" mark the encoding as
// not acceptable.
func zeroQuality(params string) bool {
	for param := range strings.SplitSeq(params, ";") {
		name, value, ok := strings.Cut(param, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(name), "q") {
			continue
		}
		quality, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return err == nil && quality == 0
	}
	return false
}

// compressible reports whether a response of the given Content-Type benefits
// from compression. Already-compressed formats (images, fonts, zip) are left
// alone.
func compressible(contentType string) bool {
	mediaType, _, _ := strings.Cut(contentType, ";")
	mediaType = strings.TrimSpace(strings.ToLower(mediaType))

	switch mediaType {
	case "application/javascript", "application/json", "application/manifest+json",
		"application/wasm", "application/xhtml+xml", "application/xml",
		"image/svg+xml":
		return true
	default:
		// Any other text/* subtype is worth compressing; the switch above lists
		// the non-text types that are.
		return strings.HasPrefix(mediaType, "text/")
	}
}

// minCompressBytes is the smallest body worth gzipping. Below it the gzip frame
// and dictionary overhead can make the response larger than the original.
const minCompressBytes = 512

// worthCompressing reports whether a body of the given Content-Length is big
// enough to benefit. An unknown length (chunked) is compressed, because the
// handler has already committed to producing the bytes.
func worthCompressing(contentLength string) bool {
	if contentLength == "" {
		return true
	}

	size, err := strconv.Atoi(contentLength)
	if err != nil {
		return true
	}
	return size >= minCompressBytes
}

// gzipResponseWriter decides whether to compress once the handler has set the
// response headers, so it can honour the handler's Content-Type and drop the
// Content-Length it computed for the uncompressed body.
type gzipResponseWriter struct {
	http.ResponseWriter

	writer      *gzip.Writer
	compressing bool
	wroteHeader bool
}

var _ http.ResponseWriter = (*gzipResponseWriter)(nil)

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	// A partial response is a byte range of the identity representation, so
	// compressing it would contradict the Content-Range it carries.
	w.compressing = statusCode != http.StatusNoContent &&
		statusCode != http.StatusNotModified &&
		statusCode != http.StatusPartialContent &&
		w.Header().Get("Content-Encoding") == "" &&
		compressible(w.Header().Get("Content-Type")) &&
		worthCompressing(w.Header().Get("Content-Length"))

	if w.compressing {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		w.writer = gzip.NewWriter(w.ResponseWriter)
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

//nolint:wrapcheck // proxying the underlying writer; wrapping adds no value
func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.compressing {
		return w.writer.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// Flush pushes any buffered compressed bytes to the client.
func (w *gzipResponseWriter) Flush() {
	if w.compressing {
		_ = w.writer.Flush()
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter for http.ResponseController.
func (w *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *gzipResponseWriter) close() {
	if w.compressing {
		_ = w.writer.Close()
	}
}
