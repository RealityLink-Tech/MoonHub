package dynamictools

import (
	"context"
	"testing"
)

func TestValidateFetchURL_Blocked(t *testing.T) {
	ctx := context.Background()
	cases := []string{
		"http://127.0.0.1/api",
		"http://localhost/foo",
		"https://192.168.1.1/",
		"ftp://example.com/",
		"http://user:pass@8.8.8.8/",
	}
	for _, raw := range cases {
		if err := validateFetchURL(ctx, raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

func TestValidateFetchURL_AllowsPublicIP(t *testing.T) {
	ctx := context.Background()
	// No server required; only validation path for literal public IP.
	if err := validateFetchURL(ctx, "https://8.8.8.8/"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPFetch_RejectsLoopback(t *testing.T) {
	hf := NewHostFunctions()
	_, err := hf.HTTPFetch(context.Background(), "GET", "http://127.0.0.1:9/nope", nil, "")
	if err == nil {
		t.Fatal("expected error for loopback URL")
	}
}
