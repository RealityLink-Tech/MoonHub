// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - Anthropic Provider Plugin
//
// Copyright (c) 2026 MoonHub contributors

package anthropic

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
)

func init() {
	plugin.RegisterPlugin(&AnthropicPlugin{})
}

type AnthropicPlugin struct {
	ctx *plugin.RuntimeContext
}

func (p *AnthropicPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-provider-anthropic",
		Name:        "Anthropic",
		Type:        plugin.TypeProvider,
		Version:     "1.0.0",
		Description: "Anthropic Claude (OAuth or API key)",
		Priority:    100,
	}
}

func (p *AnthropicPlugin) ProviderName() string {
	return "anthropic"
}

func (p *AnthropicPlugin) Init(ctx *plugin.RuntimeContext) error {
	p.ctx = ctx
	return nil
}

func (p *AnthropicPlugin) Validate(cfg *config.Config) error {
	return nil
}

func (p *AnthropicPlugin) IsEnabled(cfg *config.Config) bool {
	for _, m := range cfg.ModelList {
		protocol, _ := providers.ExtractProtocol(m.Model)
		if protocol == "anthropic" {
			return true
		}
	}
	return cfg.Providers.Anthropic.APIKey != "" || cfg.Providers.Anthropic.AuthMethod != ""
}

func (p *AnthropicPlugin) CreateProvider(cfg *config.Config) (providers.LLMProvider, error) {
	model := cfg.Agents.Defaults.GetModelName()
	modelCfg, err := cfg.GetModelConfig(model)
	if err != nil {
		return nil, err
	}
	return p.CreateProviderFromModelConfig(modelCfg)
}

func (p *AnthropicPlugin) CreateProviderFromModelConfig(modelCfg *config.ModelConfig) (providers.LLMProvider, error) {
	protocol, _ := providers.ExtractProtocol(modelCfg.Model)
	if protocol != "anthropic" {
		return nil, providers.ErrSkipProvider
	}
	if modelCfg.AuthMethod == "oauth" || modelCfg.AuthMethod == "token" {
		return providers.CreateClaudeProviderFromAuthStore()
	}
	if modelCfg.APIKey == "" {
		return nil, fmt.Errorf("api_key is required for anthropic protocol (model: %s)", modelCfg.Model)
	}
	apiBase := modelCfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.anthropic.com/v1"
	}
	return providers.NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
		modelCfg.APIKey,
		apiBase,
		modelCfg.Proxy,
		modelCfg.MaxTokensField,
		modelCfg.RequestTimeout,
	), nil
}

func (p *AnthropicPlugin) SupportsProtocol(protocol string) bool {
	return protocol == "anthropic"
}

func (p *AnthropicPlugin) SupportsModel(modelName string) bool {
	protocol, _ := providers.ExtractProtocol(modelName)
	return protocol == "anthropic"
}
