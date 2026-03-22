package qq

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
)

func init() {
	channels.RegisterFactory("qq", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewQQChannel(cfg.Channels.QQ, b)
	})
}
