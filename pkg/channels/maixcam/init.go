package maixcam

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
)

func init() {
	channels.RegisterFactory("maixcam", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewMaixCamChannel(cfg.Channels.MaixCam, b)
	})
}
