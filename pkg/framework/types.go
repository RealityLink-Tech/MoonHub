// MoonHub - Ultra-lightweight personal AI agent
// Plugin Architecture - Core Types
//
// Copyright (c) 2026 MoonHub contributors

package plugin

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/media"
)

// PluginType defines the category of plugin
type PluginType string

const (
	// TypeChannel indicates a channel plugin (Telegram, Discord, etc.)
	TypeChannel PluginType = "channel"
	// TypeProvider indicates a provider plugin (OpenAI, Anthropic, etc.)
	TypeProvider PluginType = "provider"
	// TypeTool indicates a tool plugin (web search, filesystem, etc.)
	TypeTool PluginType = "tool"
)

// Metadata contains common plugin information
type Metadata struct {
	// ID is the unique identifier for the plugin (e.g., "moonhub-channel-telegram")
	ID string `json:"id"`
	// Name is the human-readable name (e.g., "Telegram")
	Name string `json:"name"`
	// Type indicates the plugin category
	Type PluginType `json:"type"`
	// Version follows semantic versioning (e.g., "1.0.0")
	Version string `json:"version"`
	// Description provides a brief summary of the plugin
	Description string `json:"description"`
	// Author is optional attribution
	Author string `json:"author,omitempty"`
	// Priority affects loading order (higher = loaded later)
	Priority int `json:"priority,omitempty"`
}

// Plugin is the base interface all plugins must implement
type Plugin interface {
	// Metadata returns plugin metadata
	Metadata() Metadata

	// Init initializes the plugin with runtime context
	Init(ctx *RuntimeContext) error

	// Validate checks if the plugin can run with current config
	Validate(cfg *config.Config) error
}

// RuntimeContext provides dependencies injected at initialization
type RuntimeContext struct {
	// Config is the application configuration
	Config *config.Config
	// Bus is the message bus for inter-component communication
	Bus *bus.MessageBus
	// Workspace is the path to the workspace directory
	Workspace string
	// MediaStore handles media file storage
	MediaStore media.MediaStore
}
