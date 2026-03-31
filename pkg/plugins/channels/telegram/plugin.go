// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - Telegram Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package telegram

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelstelegram "github.com/RealityLink-Tech/MoonHub/pkg/channels/telegram"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&TelegramPlugin{})
}

// TelegramPlugin implements ChannelPlugin interface for Telegram
type TelegramPlugin struct {
	ctx *plugin.RuntimeContext
}

// Metadata returns plugin metadata
func (p *TelegramPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-telegram",
		Name:        "Telegram",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "Telegram bot channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *TelegramPlugin) Init(ctx *plugin.RuntimeContext) error {
	p.ctx = ctx
	return nil
}

// Validate checks if the plugin can run with current config
func (p *TelegramPlugin) Validate(cfg *config.Config) error {
	if cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token == "" {
		return fmt.Errorf("telegram token required when enabled")
	}
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *TelegramPlugin) ChannelPrefix() string {
	return "telegram"
}

// IsEnabled checks if the channel is enabled in config
func (p *TelegramPlugin) IsEnabled(cfg *config.Config) bool {
	return cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token != ""
}

// CreateChannel instantiates the channel implementation
func (p *TelegramPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	return channelstelegram.NewTelegramChannel(cfg, bus)
}
