package mdns

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("expected client instance")
	}

	if client.timeout != DefaultDiscoverTimeout {
		t.Errorf("expected default timeout %v, got %v", DefaultDiscoverTimeout, client.timeout)
	}
}

func TestNewClientWithConfig(t *testing.T) {
	config := ClientConfig{
		Timeout: 10 * time.Second,
	}

	client := NewClient(config)
	if client.timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", client.timeout)
	}
}

func TestDiscover(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewClient(ClientConfig{
		Timeout: 2 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	devices, err := client.Discover(ctx)
	if err != nil {
		t.Logf("discover returned error (expected if no devices): %v", err)
	}

	// It's OK if no devices are found in test environment
	t.Logf("discovered %d devices", len(devices))
}
