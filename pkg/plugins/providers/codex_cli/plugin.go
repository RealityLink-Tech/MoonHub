// MoonHub - Ultra-lightweight personal AI agent
// Plugin Architecture - Codex CLI Provider Plugin
//
// Copyright (c) 2026 MoonHub contributors

package codex_cli

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
)

func init() {
	plugin.RegisterPlugin(&CodexCliPlugin{})
}

type CodexCliPlugin struct {
	ctx *plugin.RuntimeContext
}

func (p *CodexCliPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-provider-codex_cli",
		Name:        "Codex CLI",
		Type:        plugin.TypeProvider,
		Version:     "1.0.0",
		Description: "Codex CLI subprocess",
		Priority:    100,
	}
}

func (p *CodexCliPlugin) ProviderName() string {
	return "codex_cli"
}

func (p *CodexCliPlugin) Init(ctx *plugin.RuntimeContext) error {
	p.ctx = ctx
	return nil
}

func (p *CodexCliPlugin) Validate(cfg *config.Config) error {
	return nil
}

func (p *CodexCliPlugin) IsEnabled(cfg *config.Config) bool {
	for _, m := range cfg.ModelList {
		protocol, _ := providers.ExtractProtocol(m.Model)
		if protocol == "codex-cli" || protocol == "codexcli" {
			return true
		}
	}
	return false
}

func (p *CodexCliPlugin) CreateProvider(cfg *config.Config) (providers.LLMProvider, error) {
	model := cfg.Agents.Defaults.GetModelName()
	modelCfg, err := cfg.GetModelConfig(model)
	if err != nil {
		return nil, err
	}
	return p.CreateProviderFromModelConfig(modelCfg)
}

func (p *CodexCliPlugin) CreateProviderFromModelConfig(modelCfg *config.ModelConfig) (providers.LLMProvider, error) {
	protocol, _ := providers.ExtractProtocol(modelCfg.Model)
	if protocol != "codex-cli" && protocol != "codexcli" {
		return nil, providers.ErrSkipProvider
	}
	workspace := modelCfg.Workspace
	if workspace == "" {
		workspace = "."
	}
	return providers.NewCodexCliProvider(workspace), nil
}

func (p *CodexCliPlugin) SupportsProtocol(protocol string) bool {
	return protocol == "codex-cli" || protocol == "codexcli"
}

func (p *CodexCliPlugin) SupportsModel(modelName string) bool {
	protocol, _ := providers.ExtractProtocol(modelName)
	return protocol == "codex-cli" || protocol == "codexcli"
}
