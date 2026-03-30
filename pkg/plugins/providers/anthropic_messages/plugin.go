// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - Anthropic Messages API Provider Plugin
//
// Copyright (c) 2026 MoonHub contributors

package anthropic_messages

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
	anthropicmessages "github.com/RealityLink-Tech/MoonHub/pkg/providers/anthropic_messages"
)

func init() {
	plugin.RegisterPlugin(&AnthropicMessagesPlugin{})
}

type AnthropicMessagesPlugin struct {
	ctx *plugin.RuntimeContext
}

func (p *AnthropicMessagesPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-provider-anthropic_messages",
		Name:        "Anthropic Messages",
		Type:        plugin.TypeProvider,
		Version:     "1.0.0",
		Description: "Anthropic Messages API (HTTP-based, native format)",
		Priority:    100,
	}
}

func (p *AnthropicMessagesPlugin) ProviderName() string {
	return "anthropic_messages"
}

func (p *AnthropicMessagesPlugin) Init(ctx *plugin.RuntimeContext) error {
	p.ctx = ctx
	return nil
}

func (p *AnthropicMessagesPlugin) Validate(cfg *config.Config) error {
	return nil
}

func (p *AnthropicMessagesPlugin) IsEnabled(cfg *config.Config) bool {
	for _, m := range cfg.ModelList {
		protocol, _ := providers.ExtractProtocol(m.Model)
		if protocol == "anthropic-messages" {
			return true
		}
	}
	return false
}

func (p *AnthropicMessagesPlugin) CreateProvider(cfg *config.Config) (providers.LLMProvider, error) {
	model := cfg.Agents.Defaults.GetModelName()
	modelCfg, err := cfg.GetModelConfig(model)
	if err != nil {
		return nil, err
	}
	return p.CreateProviderFromModelConfig(modelCfg)
}

func (p *AnthropicMessagesPlugin) CreateProviderFromModelConfig(modelCfg *config.ModelConfig) (providers.LLMProvider, error) {
	protocol, _ := providers.ExtractProtocol(modelCfg.Model)
	if protocol != "anthropic-messages" {
		return nil, providers.ErrSkipProvider
	}
	if modelCfg.APIKey == "" {
		return nil, fmt.Errorf("api_key is required for anthropic-messages protocol (model: %s)", modelCfg.Model)
	}
	apiBase := modelCfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.anthropic.com/v1"
	}
	return anthropicmessages.NewProviderWithTimeout(
		modelCfg.APIKey,
		apiBase,
		modelCfg.RequestTimeout,
	), nil
}

func (p *AnthropicMessagesPlugin) SupportsProtocol(protocol string) bool {
	return protocol == "anthropic-messages"
}

func (p *AnthropicMessagesPlugin) SupportsModel(modelName string) bool {
	protocol, _ := providers.ExtractProtocol(modelName)
	return protocol == "anthropic-messages"
}
