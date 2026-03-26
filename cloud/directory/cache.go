// cloud/directory/cache.go
package directory

import "context"

// OnlineCache tracks agent online status.
type OnlineCache interface {
	MarkOnline(ctx context.Context, agentID, endpoint string) error
	IsOnline(ctx context.Context, agentID string) (bool, string, error)
	MarkOffline(ctx context.Context, agentID string) error
}
