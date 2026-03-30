// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - WeCom AI Bot Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package wecom_aibot

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelswecom "github.com/RealityLink-Tech/MoonHub/pkg/channels/wecom"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&WeComAIBotPlugin{})
}

// WeComAIBotPlugin implements ChannelPlugin interface for WeCom AI Bot
type WeComAIBotPlugin struct{}

// Metadata returns plugin metadata
func (p *WeComAIBotPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-wecom_aibot",
		Name:        "WeCom AI Bot",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "WeCom AI Bot channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *WeComAIBotPlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

// Validate checks if the plugin can run with current config
func (p *WeComAIBotPlugin) Validate(cfg *config.Config) error {
	if cfg.Channels.WeComAIBot.Enabled && cfg.Channels.WeComAIBot.Token == "" {
		return fmt.Errorf("wecom_aibot token required when enabled")
	}
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *WeComAIBotPlugin) ChannelPrefix() string {
	return "wecom_aibot"
}

// IsEnabled checks if the channel is enabled in config
func (p *WeComAIBotPlugin) IsEnabled(cfg *config.Config) bool {
	return cfg.Channels.WeComAIBot.Enabled && cfg.Channels.WeComAIBot.Token != ""
}

// CreateChannel instantiates the channel implementation
func (p *WeComAIBotPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	return channelswecom.NewWeComAIBotChannel(cfg.Channels.WeComAIBot, bus)
}
