package middleware

import (
	"net"
	"net/http"
	"slices"
	"strings"
)

// IPConfig controls whether proxy headers (X-Forwarded-For, X-Real-IP) are
// trusted for client IP extraction. Set TrustProxyHeaders to true only when
// running behind a single trusted reverse proxy (e.g., Caddy) that appends the
// observed client address to X-Forwarded-For. A directly exposed server must
// leave this false, otherwise clients can spoof their address.
type IPConfig struct {
	TrustProxyHeaders bool
}

// GetClientIP extracts the client IP address from the request.
func GetClientIP(r *http.Request, cfg IPConfig) string {
	if cfg.TrustProxyHeaders {
		if ip := parseXForwardedFor(r); ip != "" {
			return ip
		}

		if ip := parseXRealIP(r); ip != "" {
			return ip
		}
	}

	return stripPort(r.RemoteAddr)
}

// parseXForwardedFor extracts and validates the rightmost IP from the
// X-Forwarded-For header. The header format is "client, proxy1, proxy2"; the
// rightmost entry is the address observed by the nearest trusted proxy, while
// anything to its left can be supplied by the client. This assumes a single
// trusted proxy that appends to the header rather than passing it through.
func parseXForwardedFor(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return ""
	}

	parts := strings.Split(xff, ",")
	for _, part := range slices.Backward(parts) {
		ip := stripPort(strings.TrimSpace(part))
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	return ""
}

// parseXRealIP extracts and validates the IP from the X-Real-IP header.
func parseXRealIP(r *http.Request) string {
	xri := r.Header.Get("X-Real-IP")
	if xri == "" {
		return ""
	}

	ip := stripPort(strings.TrimSpace(xri))

	if net.ParseIP(ip) == nil {
		return ""
	}

	return ip
}

func stripPort(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}
