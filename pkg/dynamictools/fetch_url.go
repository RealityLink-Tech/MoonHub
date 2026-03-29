package dynamictools

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxHTTPFetchRedirects limits redirect chains for schema-driven API fetches.
const maxHTTPFetchRedirects = 8

// newFetchHTTPClient returns an HTTP client with redirect validation and SSRF-safe defaults.
func newFetchHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxHTTPFetchRedirects {
				return fmt.Errorf("too many redirects")
			}
			if err := validateFetchURL(req.Context(), req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}
}

// validateFetchURL rejects URLs that are not http(s), carry credentials, or resolve to
// non-public addresses (loopback, RFC1918, CGNAT, link-local, multicast, etc.).
// This mitigates SSRF when FetchConfig points at arbitrary URLs.
func validateFetchURL(ctx context.Context, raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http and https URLs are allowed")
	}
	if u.User != nil {
		return fmt.Errorf("user info in URL is not allowed")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("missing host")
	}
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return fmt.Errorf("localhost targets are not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedFetchTargetIP(ip) {
			return fmt.Errorf("target address is not allowed for fetch")
		}
		return nil
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(lookupCtx, host)
	if err != nil {
		return fmt.Errorf("resolve host: %w", err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("no addresses for host")
	}
	for _, a := range addrs {
		if isBlockedFetchTargetIP(a.IP) {
			return fmt.Errorf("resolved address is not allowed for fetch")
		}
	}
	return nil
}

func isBlockedFetchTargetIP(ip net.IP) bool {
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
