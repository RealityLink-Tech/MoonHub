package channels

import (
	"sync"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
)

// ChannelFactory is a constructor function that creates a Channel from config and message bus.
// Each channel subpackage registers one or more factories via init().
type ChannelFactory func(cfg *config.Config, bus *bus.MessageBus) (Channel, error)

var (
	factoriesMu sync.RWMutex
	factories   = map[string]ChannelFactory{}
)

// RegisterFactory registers a named channel factory. Called from subpackage init() functions.
func RegisterFactory(name string, f ChannelFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[name] = f
}
