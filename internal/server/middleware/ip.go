package middleware

import (
	"net"
	"net/http"
	"strings"

	"go-htmx-template/internal/version"
)

// GetClientIP extracts the client IP address from the request. In production
// (behind a reverse proxy like Caddy), it parses X-Forwarded-For and X-Real-IP
// headers. In dev mode (version.Value == "dev"), proxy headers are ignored
// since there is no reverse proxy, and they could be spoofed by the client.
func GetClientIP(r *http.Request) string {
	if version.Value != "dev" {
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

	// Take the first IP (leftmost = original client).
	ip, _, _ := strings.Cut(xff, ",")
	ip = strings.TrimSpace(ip)

	// The value may include a port (e.g., "[::1]:8080"), strip it.
	ip = stripPort(ip)

	// Validate that it's actually an IP address.
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

// stripPort removes the port from an address string. It handles both
// IPv4 ("192.168.1.1:8080") and IPv6 ("[::1]:8080") formats.
func stripPort(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}
