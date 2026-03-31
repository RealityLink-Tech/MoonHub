// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - Plugin Registry
//
// Copyright (c) 2026 MoonHub contributors

package plugin

import (
	"sort"
	"sync"

	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
)

// Registry manages all registered plugins
type Registry struct {
	mu        sync.RWMutex
	plugins   map[string]Plugin
	channels  map[string]ChannelPlugin
	providers map[string]ProviderPlugin
	tools     map[string]ToolPlugin
}

// globalRegistry is the singleton registry instance
var globalRegistry = NewRegistry()

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		plugins:   make(map[string]Plugin),
		channels:  make(map[string]ChannelPlugin),
		providers: make(map[string]ProviderPlugin),
		tools:     make(map[string]ToolPlugin),
	}
}

// GlobalRegistry returns the singleton registry
func GlobalRegistry() *Registry {
	return globalRegistry
}

// Register registers any plugin type
func (r *Registry) Register(p Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()

	meta := p.Metadata()
	r.plugins[meta.ID] = p

	switch meta.Type {
	case TypeChannel:
		if cp, ok := p.(ChannelPlugin); ok {
			r.channels[meta.ID] = cp
			logger.InfoCF("plugin", "Registered channel plugin", map[string]any{
				"id":      meta.ID,
				"name":    meta.Name,
				"version": meta.Version,
			})
		}
	case TypeProvider:
		if pp, ok := p.(ProviderPlugin); ok {
			r.providers[meta.ID] = pp
			logger.InfoCF("plugin", "Registered provider plugin", map[string]any{
				"id":      meta.ID,
				"name":    meta.Name,
				"version": meta.Version,
			})
		}
	case TypeTool:
		if tp, ok := p.(ToolPlugin); ok {
			r.tools[meta.ID] = tp
			logger.InfoCF("plugin", "Registered tool plugin", map[string]any{
				"id":      meta.ID,
				"name":    meta.Name,
				"version": meta.Version,
			})
		}
	}
}

// RegisterPlugin is a convenience function for the global registry
func RegisterPlugin(p Plugin) {
	globalRegistry.Register(p)
}

// GetPlugin returns a plugin by ID
func (r *Registry) GetPlugin(id string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[id]
	return p, ok
}

// GetChannelPlugins returns all registered channel plugins
func (r *Registry) GetChannelPlugins() map[string]ChannelPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]ChannelPlugin, len(r.channels))
	for k, v := range r.channels {
		result[k] = v
	}
	return result
}

// GetProviderPlugins returns all registered provider plugins
func (r *Registry) GetProviderPlugins() map[string]ProviderPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]ProviderPlugin, len(r.providers))
	for k, v := range r.providers {
		result[k] = v
	}
	return result
}

// GetToolPlugins returns all registered tool plugins
func (r *Registry) GetToolPlugins() map[string]ToolPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]ToolPlugin, len(r.tools))
	for k, v := range r.tools {
		result[k] = v
	}
	return result
}

// ListPlugins returns a list of all registered plugin IDs
func (r *Registry) ListPlugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.plugins))
	for id := range r.plugins {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ListChannelPlugins returns a sorted list of channel plugin IDs
func (r *Registry) ListChannelPlugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.channels))
	for id := range r.channels {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ListProviderPlugins returns a sorted list of provider plugin IDs
func (r *Registry) ListProviderPlugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ListToolPlugins returns a sorted list of tool plugin IDs
func (r *Registry) ListToolPlugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.tools))
	for id := range r.tools {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Count returns the total number of registered plugins
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.plugins)
}

// CountChannels returns the number of registered channel plugins
func (r *Registry) CountChannels() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.channels)
}

// CountProviders returns the number of registered provider plugins
func (r *Registry) CountProviders() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
}

// CountTools returns the number of registered tool plugins
func (r *Registry) CountTools() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}
