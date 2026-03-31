// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - Claude CLI Provider Plugin
//
// Copyright (c) 2026 MoonHub contributors

package claude_cli

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
)

func init() {
	plugin.RegisterPlugin(&ClaudeCliPlugin{})
}

type ClaudeCliPlugin struct {
	ctx *plugin.RuntimeContext
}

func (p *ClaudeCliPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-provider-claude_cli",
		Name:        "Claude CLI",
		Type:        plugin.TypeProvider,
		Version:     "1.0.0",
		Description: "Claude CLI subprocess",
		Priority:    100,
	}
}

func (p *ClaudeCliPlugin) ProviderName() string {
	return "claude_cli"
}

func (p *ClaudeCliPlugin) Init(ctx *plugin.RuntimeContext) error {
	p.ctx = ctx
	return nil
}

func (p *ClaudeCliPlugin) Validate(cfg *config.Config) error {
	return nil
}

func (p *ClaudeCliPlugin) IsEnabled(cfg *config.Config) bool {
	for _, m := range cfg.ModelList {
		protocol, _ := providers.ExtractProtocol(m.Model)
		if protocol == "claude-cli" || protocol == "claudecli" {
			return true
		}
	}
	return false
}

func (p *ClaudeCliPlugin) CreateProvider(cfg *config.Config) (providers.LLMProvider, error) {
	model := cfg.Agents.Defaults.GetModelName()
	modelCfg, err := cfg.GetModelConfig(model)
	if err != nil {
		return nil, err
	}
	return p.CreateProviderFromModelConfig(modelCfg)
}

func (p *ClaudeCliPlugin) CreateProviderFromModelConfig(modelCfg *config.ModelConfig) (providers.LLMProvider, error) {
	protocol, _ := providers.ExtractProtocol(modelCfg.Model)
	if protocol != "claude-cli" && protocol != "claudecli" {
		return nil, providers.ErrSkipProvider
	}
	workspace := modelCfg.Workspace
	if workspace == "" {
		workspace = "."
	}
	return providers.NewClaudeCliProvider(workspace), nil
}

func (p *ClaudeCliPlugin) SupportsProtocol(protocol string) bool {
	return protocol == "claude-cli" || protocol == "claudecli"
}

func (p *ClaudeCliPlugin) SupportsModel(modelName string) bool {
	protocol, _ := providers.ExtractProtocol(modelName)
	return protocol == "claude-cli" || protocol == "claudecli"
}
