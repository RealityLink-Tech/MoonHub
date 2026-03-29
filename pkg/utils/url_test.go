package utils

import (
	"context"
	"net"
	"testing"
)

func TestIsBlockedTargetIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		// Loopback
		{"127.0.0.1", true},
		{"::1", true},
		// RFC1918 private
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		// CGNAT (RFC 6598)
		{"100.64.0.1", true},
		{"100.127.255.255", true},
		// Link-local
		{"169.254.1.1", true},
		{"fe80::1", true},
		// Multicast
		{"224.0.0.1", true},
		{"ff02::1", true},
		// Unspecified
		{"0.0.0.0", true},
		{"::", true},
		// Public IPs (should NOT be blocked)
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"203.0.113.1", false},
		// CGNAT boundary (not blocked)
		{"100.63.255.255", false},
		{"100.128.0.0", false},
		// nil
		{"", true}, // net.ParseIP("") returns nil
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			var ip net.IP
			if tt.ip != "" {
				ip = net.ParseIP(tt.ip)
			}
			if got := IsBlockedTargetIP(ip); got != tt.want {
				t.Errorf("IsBlockedTargetIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestValidateURLForRequest_Scheme(t *testing.T) {
	ctx := context.Background()

	_, err := ValidateURLForRequest(ctx, "ftp://example.com/file")
	if err == nil {
		t.Error("expected error for ftp scheme")
	}

	_, err = ValidateURLForRequest(ctx, "file:///etc/passwd")
	if err == nil {
		t.Error("expected error for file scheme")
	}

	_, err = ValidateURLForRequest(ctx, "gopher://localhost")
	if err == nil {
		t.Error("expected error for gopher scheme")
	}
}

func TestValidateURLForRequest_UserInfo(t *testing.T) {
	ctx := context.Background()

	_, err := ValidateURLForRequest(ctx, "http://user:pass@example.com/")
	if err == nil {
		t.Error("expected error for URL with user info")
	}
}

func TestValidateURLForRequest_Localhost(t *testing.T) {
	ctx := context.Background()

	_, err := ValidateURLForRequest(ctx, "http://localhost/")
	if err == nil {
		t.Error("expected error for localhost")
	}

	_, err = ValidateURLForRequest(ctx, "http://my.localhost/")
	if err == nil {
		t.Error("expected error for .localhost subdomain")
	}

	_, err = ValidateURLForRequest(ctx, "http://127.0.0.1/")
	if err == nil {
		t.Error("expected error for loopback IP")
	}
}

func TestValidateURLForRequest_MissingHost(t *testing.T) {
	ctx := context.Background()

	_, err := ValidateURLForRequest(ctx, "http://")
	if err == nil {
		t.Error("expected error for missing host")
	}
}

func TestValidateURLForRequest_InvalidURL(t *testing.T) {
	ctx := context.Background()

	_, err := ValidateURLForRequest(ctx, "://bad")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestValidateURLForRequest_ReturnsURL(t *testing.T) {
	ctx := context.Background()

	u, err := ValidateURLForRequest(ctx, "https://example.com/path?q=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Scheme != "https" {
		t.Errorf("scheme = %q, want https", u.Scheme)
	}
	if u.Host != "example.com" {
		t.Errorf("host = %q, want example.com", u.Host)
	}
}
