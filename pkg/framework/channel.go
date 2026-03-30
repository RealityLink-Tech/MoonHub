// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - Channel Plugin Interface
//
// Copyright (c) 2026 MoonHub contributors

package plugin

import (
	"context"
	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
)

// Channel is the interface for messaging channels
// Defined here to avoid circular imports
type Channel interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Send(ctx context.Context, msg bus.OutboundMessage) error
	IsRunning() bool
}

// ChannelPlugin extends Plugin with channel-specific functionality.
// Channel plugins connect external messaging platforms (Telegram, Discord, etc.)
type ChannelPlugin interface {
	Plugin

	// ChannelPrefix returns the userId prefix this channel owns (e.g., "telegram", "discord")
	ChannelPrefix() string

	// CreateChannel instantiates the channel implementation
	CreateChannel(cfg *config.Config, bus *bus.MessageBus) (Channel, error)

	// IsEnabled checks if the channel is enabled in config
	IsEnabled(cfg *config.Config) bool
}
