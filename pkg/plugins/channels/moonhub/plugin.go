package moonhub

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	channelsmoonhub "github.com/RealityLink-Tech/MoonHub/pkg/channels/moonhub"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&MoonHubPlugin{})
}

// MoonHubPlugin implements ChannelPlugin interface for the MoonHub agent-to-agent channel.
type MoonHubPlugin struct{}

func (p *MoonHubPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-moonhub",
		Name:        "MoonHub",
		Type:        plugin.TypeChannel,
		Version:     "1.0.0",
		Description: "MoonHub agent-to-agent channel integration",
		Priority:    100,
	}
}

func (p *MoonHubPlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

func (p *MoonHubPlugin) Validate(cfg *config.Config) error {
	if cfg.Channels.MoonHub.Enabled && cfg.Channels.MoonHub.Token == "" {
		return fmt.Errorf("moonhub token required when enabled")
	}
	return nil
}

func (p *MoonHubPlugin) ChannelPrefix() string {
	return "moonhub"
}

func (p *MoonHubPlugin) IsEnabled(cfg *config.Config) bool {
	return cfg.Channels.MoonHub.Enabled && cfg.Channels.MoonHub.Token != ""
}

func (p *MoonHubPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (plugin.Channel, error) {
	return channelsmoonhub.NewMoonHubChannel(cfg.Channels.MoonHub, nil, bus)
}
