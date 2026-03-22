// MoonHub - Ultra-lightweight personal AI agent
// Plugin Architecture - Plugin Manager
//
// Copyright (c) 2026 MoonHub contributors

package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
	"github.com/RealityLink-Tech/MoonHub/pkg/media"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
	"github.com/RealityLink-Tech/MoonHub/pkg/tools"
)

// Manager orchestrates plugin lifecycle
type Manager struct {
	registry *Registry
	config   *config.Config
	ctx      *RuntimeContext
	// initialized instances
	channels  map[string]Channel
	providers map[string]providers.LLMProvider
	toolReg   *tools.ToolRegistry
	// state
	mu sync.RWMutex
}

// NewManager creates a new plugin manager
func NewManager(cfg *config.Config, messageBus *bus.MessageBus, store media.MediaStore) *Manager {
	return &Manager{
		registry: GlobalRegistry(),
		config:   cfg,
		ctx: &RuntimeContext{
			Config:     cfg,
			Bus:        messageBus,
			MediaStore: store,
		},
		channels:  make(map[string]Channel),
		providers: make(map[string]providers.LLMProvider),
		toolReg:   tools.NewToolRegistry(),
	}
}

// Initialize loads and initializes all enabled plugins
func (m *Manager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize tool plugins first (tools may be needed by channels/providers)
	if err := m.initTools(ctx); err != nil {
		return fmt.Errorf("initialize tools: %w", err)
	}

	// Initialize provider plugins
	if err := m.initProviders(ctx); err != nil {
		return fmt.Errorf("initialize providers: %w", err)
	}

	// Initialize channel plugins last (they may need providers)
	if err := m.initChannels(ctx); err != nil {
		return fmt.Errorf("initialize channels: %w", err)
	}

	return nil
}

// InitializeToolsOnly initializes only tool plugins.
// Used when channels and providers are managed separately (e.g., by gateway).
func (m *Manager) InitializeToolsOnly(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.initTools(ctx)
}

// initTools initializes all tool plugins
func (m *Manager) initTools(ctx context.Context) error {
	for id, tp := range m.registry.GetToolPlugins() {
		if err := tp.Init(m.ctx); err != nil {
			logger.WarnCF("plugin", "Tool plugin init failed", map[string]any{
				"id":    id,
				"error": err.Error(),
			})
			continue
		}
		for _, tool := range tp.CreateTools(m.ctx) {
			if tp.IsCore() {
				m.toolReg.Register(tool)
			} else {
				m.toolReg.RegisterHidden(tool)
			}
		}
	}
	return nil
}

// initProviders initializes all provider plugins
func (m *Manager) initProviders(ctx context.Context) error {
	for id, pp := range m.registry.GetProviderPlugins() {
		if !pp.IsEnabled(m.config) {
			continue
		}
		if err := pp.Init(m.ctx); err != nil {
			return fmt.Errorf("provider %s init failed: %w", id, err)
		}
		provider, err := pp.CreateProvider(m.config)
		if err != nil {
			if errors.Is(err, providers.ErrSkipProvider) {
				continue
			}
			return fmt.Errorf("create provider %s: %w", id, err)
		}
		m.providers[id] = provider
		logger.InfoCF("plugin", "Provider plugin initialized", map[string]any{
			"id":   id,
			"name": pp.Metadata().Name,
		})
	}
	return nil
}

// initChannels initializes all channel plugins
func (m *Manager) initChannels(ctx context.Context) error {
	for id, cp := range m.registry.GetChannelPlugins() {
		if !cp.IsEnabled(m.config) {
			continue
		}
		if err := cp.Init(m.ctx); err != nil {
			return fmt.Errorf("channel %s init failed: %w", id, err)
		}
		channel, err := cp.CreateChannel(m.config, m.ctx.Bus)
		if err != nil {
			return fmt.Errorf("create channel %s: %w", id, err)
		}
		m.channels[cp.ChannelPrefix()] = channel
		logger.InfoCF("plugin", "Channel plugin initialized", map[string]any{
			"id":     id,
			"name":   cp.Metadata().Name,
			"prefix": cp.ChannelPrefix(),
		})
	}
	return nil
}

// GetChannel returns an initialized channel by prefix
func (m *Manager) GetChannel(prefix string) (Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, ok := m.channels[prefix]
	return ch, ok
}

// GetProvider returns an initialized provider by name
func (m *Manager) GetProvider(name string) (providers.LLMProvider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.providers[name]
	return p, ok
}

// ToolRegistry returns the tool registry for agent use
func (m *Manager) ToolRegistry() *tools.ToolRegistry {
	return m.toolReg
}

// AllChannels returns all initialized channels
func (m *Manager) AllChannels() map[string]Channel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]Channel, len(m.channels))
	for k, v := range m.channels {
		result[k] = v
	}
	return result
}

// AllProviders returns all initialized providers
func (m *Manager) AllProviders() map[string]providers.LLMProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]providers.LLMProvider, len(m.providers))
	for k, v := range m.providers {
		result[k] = v
	}
	return result
}
