package utils

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// ValidateURLForRequest validates a URL for safe use in outgoing HTTP requests.
// It rejects non-http(s) schemes, URLs with credentials, localhost targets,
// and resolves the hostname to block private/loopback/CGNAT/link-local/multicast IPs.
// Returns a sanitized *url.URL on success (safe to use with http.NewRequest).
func ValidateURLForRequest(ctx context.Context, raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("only http and https URLs are allowed")
	}
	if u.User != nil {
		return nil, fmt.Errorf("user info in URL is not allowed")
	}
	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("missing host")
	}
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return nil, fmt.Errorf("localhost targets are not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		if IsBlockedTargetIP(ip) {
			return nil, fmt.Errorf("target address is not allowed")
		}
		return u, nil
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(lookupCtx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve host: %w", err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses for host")
	}
	for _, a := range addrs {
		if IsBlockedTargetIP(a.IP) {
			return nil, fmt.Errorf("resolved address is not allowed")
		}
	}
	return u, nil
}

// IsBlockedTargetIP returns true if the IP falls into a non-public range:
// loopback, private (RFC1918), link-local unicast, multicast, unspecified,
// or CGNAT (RFC 6598).
func IsBlockedTargetIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// RFC 6598 CGNAT — not classified as private in all Go versions.
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}
	return false
}
