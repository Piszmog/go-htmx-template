package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseWriter_DefaultStatusCode(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	assert.Equal(t, http.StatusOK, rw.statusCode)
	assert.Equal(t, 0, rw.bytesWritten)
	assert.False(t, rw.wroteHeader)
}

func TestResponseWriter_WriteHeaderCapturesCode(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)
	rw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, rw.statusCode)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.True(t, rw.wroteHeader)
}

func TestResponseWriter_WriteHeaderIsIdempotent(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)
	rw.WriteHeader(http.StatusNotFound)
	rw.WriteHeader(http.StatusInternalServerError)

	assert.Equal(t, http.StatusNotFound, rw.statusCode, "second WriteHeader call should be ignored")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestResponseWriter_WriteCountsBytesAcrossMultipleCalls(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	n1, err := rw.Write([]byte("Hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n1)

	n2, err := rw.Write([]byte(" World"))
	require.NoError(t, err)
	assert.Equal(t, 6, n2)

	assert.Equal(t, 11, rw.bytesWritten)
	assert.Equal(t, "Hello World", rec.Body.String())
}

func TestResponseWriter_WriteImplicitlyCallsWriteHeader(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	assert.False(t, rw.wroteHeader, "header should not be written before Write")

	_, err := rw.Write([]byte("body"))
	require.NoError(t, err)

	assert.True(t, rw.wroteHeader, "Write should implicitly call WriteHeader")
	assert.Equal(t, http.StatusOK, rw.statusCode)
}

func TestResponseWriter_UnwrapReturnsUnderlying(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	assert.Equal(t, rec, rw.Unwrap())
}
