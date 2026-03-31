// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - OneBot Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package onebot

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelsonebot "github.com/RealityLink-Tech/MoonHub/pkg/channels/onebot"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&OneBotPlugin{})
}

// OneBotPlugin implements ChannelPlugin interface for OneBot
type OneBotPlugin struct{}

// Metadata returns plugin metadata
func (p *OneBotPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-onebot",
		Name:        "OneBot",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "OneBot protocol channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *OneBotPlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

// Validate checks if the plugin can run with current config
func (p *OneBotPlugin) Validate(cfg *config.Config) error {
	if cfg.Channels.OneBot.Enabled && cfg.Channels.OneBot.WSUrl == "" {
		return fmt.Errorf("onebot ws_url required when enabled")
	}
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *OneBotPlugin) ChannelPrefix() string {
	return "onebot"
}

// IsEnabled checks if the channel is enabled in config
func (p *OneBotPlugin) IsEnabled(cfg *config.Config) bool {
	return cfg.Channels.OneBot.Enabled && cfg.Channels.OneBot.WSUrl != ""
}

// CreateChannel instantiates the channel implementation
func (p *OneBotPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	return channelsonebot.NewOneBotChannel(cfg.Channels.OneBot, bus)
}
