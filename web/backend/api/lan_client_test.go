package api

import (
	"net"
	"net/http"
	"testing"
)

func TestIsLANScopeIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.32.0.1", false},
		{"100.64.0.1", true},
		{"169.254.1.1", true},
		{"fe80::1", true},
		{"8.8.8.8", false},
		{"", false},
	}
	for _, tc := range tests {
		var ip net.IP
		if tc.ip != "" {
			ip = net.ParseIP(tc.ip)
		}
		if got := IsLANScopeIP(ip); got != tc.want {
			t.Errorf("IsLANScopeIP(%q) = %v, want %v", tc.ip, got, tc.want)
		}
	}
}

func TestClientIPFromRequest_RemoteAddrIPv4(t *testing.T) {
	r := &http.Request{RemoteAddr: "192.168.1.10:54321"}
	if got := ClientIPFromRequest(r); !got.Equal(net.ParseIP("192.168.1.10")) {
		t.Fatalf("ClientIPFromRequest RemoteAddr = %v", got)
	}
}

func TestClientIPFromRequest_IgnoresXFFByDefault(t *testing.T) {
	t.Setenv(envTrustProxy, "")
	r := &http.Request{
		RemoteAddr: "192.168.1.10:1",
		Header:     http.Header{"X-Forwarded-For": []string{"8.8.8.8"}},
	}
	if got := ClientIPFromRequest(r); !got.Equal(net.ParseIP("192.168.1.10")) {
		t.Fatalf("expected RemoteAddr IP, got %v", got)
	}
}

func TestClientIPFromRequest_TrustProxyUsesXFF(t *testing.T) {
	t.Setenv(envTrustProxy, "1")

	r := &http.Request{
		RemoteAddr: "10.0.0.1:1",
		Header:     http.Header{"X-Forwarded-For": []string{"192.168.2.2, 8.8.8.8"}},
	}
	if got := ClientIPFromRequest(r); !got.Equal(net.ParseIP("192.168.2.2")) {
		t.Fatalf("expected first XFF IP, got %v", got)
	}
}
