package middleware

import (
	"net"
	"net/http"
	"net/netip"
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

	if ip := normalizeIP(r.RemoteAddr); ip != "" {
		return ip
	}
	return stripPort(r.RemoteAddr)
}

// ipv6PrefixBits is the prefix length used to bucket IPv6 clients.
const ipv6PrefixBits = 64

// RateLimitKey normalizes a client IP for rate limiting. IPv6 addresses are
// bucketed by /64 because a single client usually controls a whole /64 and
// could otherwise rotate addresses to reset its budget. IPv4 and unparsable
// values are returned unchanged.
func RateLimitKey(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}

	// ::ffff:203.0.113.7 and 203.0.113.7 are the same client, so they must land
	// in the same bucket.
	addr = addr.Unmap()
	if !addr.Is6() {
		return addr.String()
	}

	prefix, err := addr.Prefix(ipv6PrefixBits)
	if err != nil {
		return ip
	}
	return prefix.String()
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

	// Only the address appended by the trusted proxy is trustworthy. If it is
	// malformed, fall back to the other headers/RemoteAddr rather than scanning
	// left into values the client may have supplied.
	parts := strings.Split(xff, ",")
	return normalizeIP(parts[len(parts)-1])
}

// parseXRealIP extracts and validates the IP from the X-Real-IP header.
func parseXRealIP(r *http.Request) string {
	xri := r.Header.Get("X-Real-IP")
	if xri == "" {
		return ""
	}

	return normalizeIP(xri)
}

func normalizeIP(raw string) string {
	ip := stripPort(strings.TrimSpace(raw))
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ""
	}
	return addr.Unmap().String()
}

func stripPort(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}
