// MoonHub - Ultra-lightweight personal AI agent
// Plugin Architecture - Discord Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package discord

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelsdiscord "github.com/RealityLink-Tech/MoonHub/pkg/channels/discord"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&DiscordPlugin{})
}

// DiscordPlugin implements ChannelPlugin interface for Discord
type DiscordPlugin struct{}

// Metadata returns plugin metadata
func (p *DiscordPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-discord",
		Name:        "Discord",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "Discord bot channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *DiscordPlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

// Validate checks if the plugin can run with current config
func (p *DiscordPlugin) Validate(cfg *config.Config) error {
	if cfg.Channels.Discord.Enabled && cfg.Channels.Discord.Token == "" {
		return fmt.Errorf("discord token required when enabled")
	}
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *DiscordPlugin) ChannelPrefix() string {
	return "discord"
}

// IsEnabled checks if the channel is enabled in config
func (p *DiscordPlugin) IsEnabled(cfg *config.Config) bool {
	return cfg.Channels.Discord.Enabled && cfg.Channels.Discord.Token != ""
}

// CreateChannel instantiates the channel implementation
func (p *DiscordPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	return channelsdiscord.NewDiscordChannel(cfg.Channels.Discord, bus)
}
