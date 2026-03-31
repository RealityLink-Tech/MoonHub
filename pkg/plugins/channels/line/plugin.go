// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - LINE Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package line

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelsline "github.com/RealityLink-Tech/MoonHub/pkg/channels/line"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&LINEPlugin{})
}

// LINEPlugin implements ChannelPlugin interface for LINE
type LINEPlugin struct{}

// Metadata returns plugin metadata
func (p *LINEPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-line",
		Name:        "LINE",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "LINE bot channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *LINEPlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

// Validate checks if the plugin can run with current config
func (p *LINEPlugin) Validate(cfg *config.Config) error {
	if cfg.Channels.LINE.Enabled && cfg.Channels.LINE.ChannelAccessToken == "" {
		return fmt.Errorf("line channel_access_token required when enabled")
	}
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *LINEPlugin) ChannelPrefix() string {
	return "line"
}

// IsEnabled checks if the channel is enabled in config
func (p *LINEPlugin) IsEnabled(cfg *config.Config) bool {
	return cfg.Channels.LINE.Enabled && cfg.Channels.LINE.ChannelAccessToken != ""
}

// CreateChannel instantiates the channel implementation
func (p *LINEPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	return channelsline.NewLINEChannel(cfg.Channels.LINE, bus)
}
