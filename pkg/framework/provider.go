// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - Provider Plugin Interface
//
// Copyright (c) 2026 MoonHub contributors

package plugin

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
)

// ProviderPlugin extends Plugin with provider-specific functionality.
// Provider plugins add LLM providers (OpenAI, Anthropic, Gemini, etc.)
type ProviderPlugin interface {
	Plugin

	// ProviderName returns the provider identifier (e.g., "anthropic", "openai")
	ProviderName() string

	// CreateProvider instantiates the provider from full config (used for IsEnabled checks).
	// For actual provider creation, use CreateProviderFromModelConfig.
	CreateProvider(cfg *config.Config) (providers.LLMProvider, error)

	// CreateProviderFromModelConfig creates a provider from model-centric config.
	// Used by the factory when routing by protocol.
	CreateProviderFromModelConfig(modelCfg *config.ModelConfig) (providers.LLMProvider, error)

	// IsEnabled checks if the provider has valid configuration
	IsEnabled(cfg *config.Config) bool

	// SupportsProtocol returns true if this plugin handles the given protocol (e.g., "openai", "anthropic").
	// Used for routing in CreateProviderFromConfig.
	SupportsProtocol(protocol string) bool

	// SupportsModel checks if this provider can handle a given model name (e.g., "openai/gpt-4o")
	SupportsModel(modelName string) bool
}
