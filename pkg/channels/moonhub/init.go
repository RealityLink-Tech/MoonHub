package moonhub

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
)

func init() {
	channels.RegisterFactory("moonhub", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewMoonHubChannel(cfg.Channels.MoonHub, nil, b)
	})
}
