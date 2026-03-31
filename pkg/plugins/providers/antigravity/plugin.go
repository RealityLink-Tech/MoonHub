// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - Antigravity (Google Cloud Code Assist) Provider Plugin
//
// Copyright (c) 2026 MoonHub contributors

package antigravity

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
)

func init() {
	plugin.RegisterPlugin(&AntigravityPlugin{})
}

type AntigravityPlugin struct {
	ctx *plugin.RuntimeContext
}

func (p *AntigravityPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-provider-antigravity",
		Name:        "Antigravity",
		Type:        plugin.TypeProvider,
		Version:     "1.0.0",
		Description: "Google Cloud Code Assist",
		Priority:    100,
	}
}

func (p *AntigravityPlugin) ProviderName() string {
	return "antigravity"
}

func (p *AntigravityPlugin) Init(ctx *plugin.RuntimeContext) error {
	p.ctx = ctx
	return nil
}

func (p *AntigravityPlugin) Validate(cfg *config.Config) error {
	return nil
}

func (p *AntigravityPlugin) IsEnabled(cfg *config.Config) bool {
	for _, m := range cfg.ModelList {
		protocol, _ := providers.ExtractProtocol(m.Model)
		if protocol == "antigravity" {
			return true
		}
	}
	return cfg.Providers.Antigravity.APIKey != ""
}

func (p *AntigravityPlugin) CreateProvider(cfg *config.Config) (providers.LLMProvider, error) {
	model := cfg.Agents.Defaults.GetModelName()
	modelCfg, err := cfg.GetModelConfig(model)
	if err != nil {
		return nil, err
	}
	return p.CreateProviderFromModelConfig(modelCfg)
}

func (p *AntigravityPlugin) CreateProviderFromModelConfig(modelCfg *config.ModelConfig) (providers.LLMProvider, error) {
	protocol, _ := providers.ExtractProtocol(modelCfg.Model)
	if protocol != "antigravity" {
		return nil, providers.ErrSkipProvider
	}
	return providers.NewAntigravityProvider(), nil
}

func (p *AntigravityPlugin) SupportsProtocol(protocol string) bool {
	return protocol == "antigravity"
}

func (p *AntigravityPlugin) SupportsModel(modelName string) bool {
	protocol, _ := providers.ExtractProtocol(modelName)
	return protocol == "antigravity"
}
