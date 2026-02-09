package middleware

import (
	"net"
	"net/http"
	"strings"
)

// IPConfig controls whether proxy headers (X-Forwarded-For, X-Real-IP) are
// trusted for client IP extraction. Set TrustProxyHeaders to true only when
// running behind a trusted reverse proxy (e.g., Caddy).
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

// parseXForwardedFor extracts and validates the first (client) IP from the
// X-Forwarded-For header. The header format is "client, proxy1, proxy2".
func parseXForwardedFor(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return ""
	}

	ip, _, _ := strings.Cut(xff, ",")
	ip = strings.TrimSpace(ip)
	ip = stripPort(ip)

	if net.ParseIP(ip) == nil {
		return ""
	}

	return ip
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
