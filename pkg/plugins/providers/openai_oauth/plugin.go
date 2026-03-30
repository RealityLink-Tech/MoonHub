// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - OpenAI OAuth/Codex Provider Plugin
//
// Copyright (c) 2026 MoonHub contributors

package openai_oauth

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
)

func init() {
	plugin.RegisterPlugin(&OpenAIOAuthPlugin{})
}

type OpenAIOAuthPlugin struct {
	ctx *plugin.RuntimeContext
}

func (p *OpenAIOAuthPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-provider-openai_oauth",
		Name:        "OpenAI OAuth/Codex",
		Type:        plugin.TypeProvider,
		Version:     "1.0.0",
		Description: "OpenAI with OAuth/token auth (Codex-style)",
		Priority:    110,
	}
}

func (p *OpenAIOAuthPlugin) ProviderName() string {
	return "openai_oauth"
}

func (p *OpenAIOAuthPlugin) Init(ctx *plugin.RuntimeContext) error {
	p.ctx = ctx
	return nil
}

func (p *OpenAIOAuthPlugin) Validate(cfg *config.Config) error {
	return nil
}

func (p *OpenAIOAuthPlugin) IsEnabled(cfg *config.Config) bool {
	for _, m := range cfg.ModelList {
		protocol, _ := providers.ExtractProtocol(m.Model)
		if protocol == "openai" && (m.AuthMethod == "oauth" || m.AuthMethod == "token") {
			return true
		}
	}
	return false
}

func (p *OpenAIOAuthPlugin) CreateProvider(cfg *config.Config) (providers.LLMProvider, error) {
	model := cfg.Agents.Defaults.GetModelName()
	modelCfg, err := cfg.GetModelConfig(model)
	if err != nil {
		return nil, err
	}
	return p.CreateProviderFromModelConfig(modelCfg)
}

func (p *OpenAIOAuthPlugin) CreateProviderFromModelConfig(modelCfg *config.ModelConfig) (providers.LLMProvider, error) {
	protocol, _ := providers.ExtractProtocol(modelCfg.Model)
	if protocol != "openai" {
		return nil, providers.ErrSkipProvider
	}
	if modelCfg.AuthMethod != "oauth" && modelCfg.AuthMethod != "token" {
		return nil, providers.ErrSkipProvider // let openai_compat handle it
	}
	return providers.CreateCodexProviderFromAuthStore()
}

func (p *OpenAIOAuthPlugin) SupportsProtocol(protocol string) bool {
	return protocol == "openai"
}

func (p *OpenAIOAuthPlugin) SupportsModel(modelName string) bool {
	protocol, _ := providers.ExtractProtocol(modelName)
	return protocol == "openai"
}
