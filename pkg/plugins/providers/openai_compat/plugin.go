// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - OpenAI-compatible HTTP Provider Plugin
//
// Copyright (c) 2026 MoonHub contributors

package openai_compat

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
)

// HTTP protocols supported by this plugin (OpenAI-compatible APIs)
var httpProtocols = map[string]bool{
	"openai": true, "litellm": true, "openrouter": true, "groq": true,
	"zhipu": true, "gemini": true, "nvidia": true, "ollama": true,
	"moonshot": true, "shengsuanyun": true, "deepseek": true,
	"cerebras": true, "vivgrid": true, "volcengine": true, "vllm": true,
	"qwen": true, "mistral": true, "avian": true, "minimax": true,
	"longcat": true, "modelscope": true,
}

func init() {
	plugin.RegisterPlugin(&OpenAICompatPlugin{})
}

type OpenAICompatPlugin struct {
	ctx *plugin.RuntimeContext
}

func (p *OpenAICompatPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-provider-openai_compat",
		Name:        "OpenAI-compatible HTTP",
		Type:        plugin.TypeProvider,
		Version:     "1.0.0",
		Description: "OpenAI-compatible HTTP providers (OpenAI, LiteLLM, Groq, Ollama, etc.)",
		Priority:    100,
	}
}

func (p *OpenAICompatPlugin) ProviderName() string {
	return "openai_compat"
}

func (p *OpenAICompatPlugin) Init(ctx *plugin.RuntimeContext) error {
	p.ctx = ctx
	return nil
}

func (p *OpenAICompatPlugin) Validate(cfg *config.Config) error {
	return nil
}

func (p *OpenAICompatPlugin) IsEnabled(cfg *config.Config) bool {
	// Enabled if any model in model_list uses an HTTP protocol
	for _, m := range cfg.ModelList {
		protocol, _ := providers.ExtractProtocol(m.Model)
		if httpProtocols[protocol] {
			return true
		}
	}
	return cfg.HasProvidersConfig()
}

func (p *OpenAICompatPlugin) CreateProvider(cfg *config.Config) (providers.LLMProvider, error) {
	model := cfg.Agents.Defaults.GetModelName()
	modelCfg, err := cfg.GetModelConfig(model)
	if err != nil {
		return nil, err
	}
	return p.CreateProviderFromModelConfig(modelCfg)
}

func (p *OpenAICompatPlugin) CreateProviderFromModelConfig(modelCfg *config.ModelConfig) (providers.LLMProvider, error) {
	protocol, _ := providers.ExtractProtocol(modelCfg.Model)
	if !httpProtocols[protocol] {
		return nil, fmt.Errorf("openai_compat plugin does not support protocol %q", protocol)
	}
	// OpenAI with OAuth/token uses Codex - let openai_oauth plugin handle it
	if protocol == "openai" && (modelCfg.AuthMethod == "oauth" || modelCfg.AuthMethod == "token") {
		return nil, providers.ErrSkipProvider
	}
	if modelCfg.APIKey == "" && modelCfg.APIBase == "" {
		return nil, fmt.Errorf("api_key or api_base is required for HTTP-based protocol %q", protocol)
	}
	apiBase := modelCfg.APIBase
	if apiBase == "" {
		apiBase = getDefaultAPIBase(protocol)
	}
	return providers.NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
		modelCfg.APIKey,
		apiBase,
		modelCfg.Proxy,
		modelCfg.MaxTokensField,
		modelCfg.RequestTimeout,
	), nil
}

func (p *OpenAICompatPlugin) SupportsProtocol(protocol string) bool {
	return httpProtocols[protocol]
}

func (p *OpenAICompatPlugin) SupportsModel(modelName string) bool {
	protocol, _ := providers.ExtractProtocol(modelName)
	return httpProtocols[protocol]
}

func getDefaultAPIBase(protocol string) string {
	switch protocol {
	case "openai":
		return "https://api.openai.com/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "litellm":
		return "http://localhost:4000/v1"
	case "groq":
		return "https://api.groq.com/openai/v1"
	case "zhipu":
		return "https://open.bigmodel.cn/api/paas/v4"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta"
	case "nvidia":
		return "https://integrate.api.nvidia.com/v1"
	case "ollama":
		return "http://localhost:11434/v1"
	case "moonshot":
		return "https://api.moonshot.cn/v1"
	case "shengsuanyun":
		return "https://router.shengsuanyun.com/api/v1"
	case "deepseek":
		return "https://api.deepseek.com/v1"
	case "cerebras":
		return "https://api.cerebras.ai/v1"
	case "vivgrid":
		return "https://api.vivgrid.com/v1"
	case "volcengine":
		return "https://ark.cn-beijing.volces.com/api/v3"
	case "qwen":
		return "https://dashscope.aliyuncs.com/compatible-mode/v1"
	case "vllm":
		return "http://localhost:8000/v1"
	case "mistral":
		return "https://api.mistral.ai/v1"
	case "avian":
		return "https://api.avian.io/v1"
	case "minimax":
		return "https://api.minimaxi.com/v1"
	case "longcat":
		return "https://api.longcat.chat/openai"
	case "modelscope":
		return "https://api-inference.modelscope.cn/v1"
	default:
		return ""
	}
}

// Ensure OpenAICompatPlugin implements plugin.ProviderPlugin
var _ plugin.ProviderPlugin = (*OpenAICompatPlugin)(nil)
