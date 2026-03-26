// cloud/directory/cache_test.go
package directory

import (
	"context"
	"testing"
)

// memoryCache is a test double implementing OnlineCache in memory.
type memoryCache struct {
	data map[string]string // agentID -> endpoint
}

func newMemoryCache() *memoryCache {
	return &memoryCache{data: make(map[string]string)}
}

func (c *memoryCache) MarkOnline(ctx context.Context, agentID, endpoint string) error {
	c.data[agentID] = endpoint
	return nil
}

func (c *memoryCache) IsOnline(ctx context.Context, agentID string) (bool, string, error) {
	endpoint, ok := c.data[agentID]
	return ok, endpoint, nil
}

func (c *memoryCache) MarkOffline(ctx context.Context, agentID string) error {
	delete(c.data, agentID)
	return nil
}

func TestMemoryCache_OnlineFlow(t *testing.T) {
	cache := newMemoryCache()
	ctx := context.Background()

	cache.MarkOnline(ctx, "mh_aaaa1111bbbb2222", "wss://relay.example.com")

	online, endpoint, err := cache.IsOnline(ctx, "mh_aaaa1111bbbb2222")
	if err != nil {
		t.Fatal(err)
	}
	if !online {
		t.Error("expected online")
	}
	if endpoint != "wss://relay.example.com" {
		t.Errorf("endpoint = %q, want %q", endpoint, "wss://relay.example.com")
	}
}

func TestMemoryCache_OfflineFlow(t *testing.T) {
	cache := newMemoryCache()
	ctx := context.Background()

	cache.MarkOnline(ctx, "mh_aaaa1111bbbb2222", "wss://relay.example.com")
	cache.MarkOffline(ctx, "mh_aaaa1111bbbb2222")

	online, _, _ := cache.IsOnline(ctx, "mh_aaaa1111bbbb2222")
	if online {
		t.Error("expected offline after MarkOffline")
	}
}

func TestMemoryCache_NotFound(t *testing.T) {
	cache := newMemoryCache()
	online, endpoint, err := cache.IsOnline(context.Background(), "mh_nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if online {
		t.Error("expected offline for unknown agent")
	}
	if endpoint != "" {
		t.Errorf("endpoint = %q, want empty", endpoint)
	}
}
