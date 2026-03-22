// MoonHub - Ultra-lightweight personal AI agent
// Plugin Architecture - Pico Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package pico

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelspico "github.com/RealityLink-Tech/MoonHub/pkg/channels/pico"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&PicoPlugin{})
}

// PicoPlugin implements ChannelPlugin interface for Pico
type PicoPlugin struct{}

// Metadata returns plugin metadata
func (p *PicoPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-pico",
		Name:        "Pico",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "Pico VR device channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *PicoPlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

// Validate checks if the plugin can run with current config
func (p *PicoPlugin) Validate(cfg *config.Config) error {
	if cfg.Channels.Pico.Enabled && cfg.Channels.Pico.Token == "" {
		return fmt.Errorf("pico token required when enabled")
	}
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *PicoPlugin) ChannelPrefix() string {
	return "pico"
}

// IsEnabled checks if the channel is enabled in config
func (p *PicoPlugin) IsEnabled(cfg *config.Config) bool {
	return cfg.Channels.Pico.Enabled && cfg.Channels.Pico.Token != ""
}

// CreateChannel instantiates the channel implementation
func (p *PicoPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	return channelspico.NewPicoChannel(cfg.Channels.Pico, bus)
}
