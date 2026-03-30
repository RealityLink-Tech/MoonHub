// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - GitHub Copilot Provider Plugin
//
// Copyright (c) 2026 MoonHub contributors

package github_copilot

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
)

func init() {
	plugin.RegisterPlugin(&GitHubCopilotPlugin{})
}

type GitHubCopilotPlugin struct {
	ctx *plugin.RuntimeContext
}

func (p *GitHubCopilotPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-provider-github_copilot",
		Name:        "GitHub Copilot",
		Type:        plugin.TypeProvider,
		Version:     "1.0.0",
		Description: "GitHub Copilot agent",
		Priority:    100,
	}
}

func (p *GitHubCopilotPlugin) ProviderName() string {
	return "github_copilot"
}

func (p *GitHubCopilotPlugin) Init(ctx *plugin.RuntimeContext) error {
	p.ctx = ctx
	return nil
}

func (p *GitHubCopilotPlugin) Validate(cfg *config.Config) error {
	return nil
}

func (p *GitHubCopilotPlugin) IsEnabled(cfg *config.Config) bool {
	for _, m := range cfg.ModelList {
		protocol, _ := providers.ExtractProtocol(m.Model)
		if protocol == "github-copilot" || protocol == "copilot" {
			return true
		}
	}
	return cfg.Providers.GitHubCopilot.APIBase != ""
}

func (p *GitHubCopilotPlugin) CreateProvider(cfg *config.Config) (providers.LLMProvider, error) {
	model := cfg.Agents.Defaults.GetModelName()
	modelCfg, err := cfg.GetModelConfig(model)
	if err != nil {
		return nil, err
	}
	return p.CreateProviderFromModelConfig(modelCfg)
}

func (p *GitHubCopilotPlugin) CreateProviderFromModelConfig(modelCfg *config.ModelConfig) (providers.LLMProvider, error) {
	protocol, modelID := providers.ExtractProtocol(modelCfg.Model)
	if protocol != "github-copilot" && protocol != "copilot" {
		return nil, providers.ErrSkipProvider
	}
	apiBase := modelCfg.APIBase
	if apiBase == "" {
		apiBase = "localhost:4321"
	}
	connectMode := modelCfg.ConnectMode
	if connectMode == "" {
		connectMode = "grpc"
	}
	provider, err := providers.NewGitHubCopilotProvider(apiBase, connectMode, modelID)
	if err != nil {
		return nil, fmt.Errorf("github copilot: %w", err)
	}
	return provider, nil
}

func (p *GitHubCopilotPlugin) SupportsProtocol(protocol string) bool {
	return protocol == "github-copilot" || protocol == "copilot"
}

func (p *GitHubCopilotPlugin) SupportsModel(modelName string) bool {
	protocol, _ := providers.ExtractProtocol(modelName)
	return protocol == "github-copilot" || protocol == "copilot"
}
