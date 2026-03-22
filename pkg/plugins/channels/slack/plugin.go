// MoonHub - Ultra-lightweight personal AI agent
// Plugin Architecture - Slack Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package slack

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelsslack "github.com/RealityLink-Tech/MoonHub/pkg/channels/slack"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&SlackPlugin{})
}

// SlackPlugin implements ChannelPlugin interface for Slack
type SlackPlugin struct{}

// Metadata returns plugin metadata
func (p *SlackPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-slack",
		Name:        "Slack",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "Slack bot channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *SlackPlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

// Validate checks if the plugin can run with current config
func (p *SlackPlugin) Validate(cfg *config.Config) error {
	if cfg.Channels.Slack.Enabled && cfg.Channels.Slack.BotToken == "" {
		return fmt.Errorf("slack bot token required when enabled")
	}
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *SlackPlugin) ChannelPrefix() string {
	return "slack"
}

// IsEnabled checks if the channel is enabled in config
func (p *SlackPlugin) IsEnabled(cfg *config.Config) bool {
	return cfg.Channels.Slack.Enabled && cfg.Channels.Slack.BotToken != ""
}

// CreateChannel instantiates the channel implementation
// This calls the existing implementation from pkg/channels/slack
func (p *SlackPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	return channelsslack.NewSlackChannel(cfg.Channels.Slack, bus)
}
