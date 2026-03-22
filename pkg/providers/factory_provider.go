// MoonHub - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 MoonHub contributors

package providers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	anthropicmessages "github.com/RealityLink-Tech/MoonHub/pkg/providers/anthropic_messages"
)

// ErrSkipProvider is returned by a plugin when it does not handle the given config
// (e.g., openai_oauth when auth is API key). The resolver should try the next plugin.
var ErrSkipProvider = errors.New("plugin does not handle this config")

// PluginProviderResolver is a function that attempts to create a provider from a plugin.
// It is set by the gateway to avoid circular imports (plugin -> providers).
// Returns (provider, modelID, nil) on success, or (nil, "", err) when no plugin handles the config.
// When err is ErrSkipProvider, the resolver should try the next plugin.
type PluginProviderResolver func(modelCfg *config.ModelConfig) (LLMProvider, string, error)

// pluginResolver is set by the gateway at startup
var pluginResolver PluginProviderResolver

// SetPluginProviderResolver sets the resolver used by CreateProviderFromConfig.
// Must be called before CreateProvider. The gateway calls this with a function
// that checks plugin.GlobalRegistry().GetProviderPlugins().
func SetPluginProviderResolver(resolver PluginProviderResolver) {
	pluginResolver = resolver
}

// createClaudeAuthProvider creates a Claude provider using OAuth credentials from auth store.
func createClaudeAuthProvider() (LLMProvider, error) {
	cred, err := getCredential("anthropic")
	if err != nil {
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for anthropic. Run: moonhub auth login --provider anthropic")
	}
	return NewClaudeProviderWithTokenSource(cred.AccessToken, createClaudeTokenSource()), nil
}

// createCodexAuthProvider creates a Codex provider using OAuth credentials from auth store.
func createCodexAuthProvider() (LLMProvider, error) {
	return CreateCodexProviderFromAuthStore()
}

// CreateCodexProviderFromAuthStore creates a Codex provider from OAuth credentials.
// Exported for use by provider plugins.
func CreateCodexProviderFromAuthStore() (LLMProvider, error) {
	cred, err := getCredential("openai")
	if err != nil {
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for openai. Run: moonhub auth login --provider openai")
	}
	return NewCodexProviderWithTokenSource(cred.AccessToken, cred.AccountID, createCodexTokenSource()), nil
}

// CreateClaudeProviderFromAuthStore creates a Claude provider from OAuth credentials.
// Exported for use by provider plugins.
func CreateClaudeProviderFromAuthStore() (LLMProvider, error) {
	return createClaudeAuthProvider()
}

// ExtractProtocol extracts the protocol prefix and model identifier from a model string.
// If no prefix is specified, it defaults to "openai".
// Examples:
//   - "openai/gpt-4o" -> ("openai", "gpt-4o")
//   - "anthropic/claude-sonnet-4.6" -> ("anthropic", "claude-sonnet-4.6")
//   - "gpt-4o" -> ("openai", "gpt-4o")  // default protocol
func ExtractProtocol(model string) (protocol, modelID string) {
	model = strings.TrimSpace(model)
	protocol, modelID, found := strings.Cut(model, "/")
	if !found {
		return "openai", model
	}
	return protocol, modelID
}

// CreateProviderFromConfig creates a provider based on the ModelConfig.
// It uses the protocol prefix in the Model field to determine which provider to create.
// If SetPluginProviderResolver was called, it tries plugins first.
// Supported protocols: openai, litellm, anthropic, anthropic-messages, antigravity,
// claude-cli, codex-cli, github-copilot
// Returns the provider, the model ID (without protocol prefix), and any error.
func CreateProviderFromConfig(cfg *config.ModelConfig) (LLMProvider, string, error) {
	if cfg == nil {
		return nil, "", fmt.Errorf("config is nil")
	}

	if cfg.Model == "" {
		return nil, "", fmt.Errorf("model is required")
	}

	protocol, modelID := ExtractProtocol(cfg.Model)

	// Try plugin resolver first (set by gateway to avoid circular imports)
	if pluginResolver != nil {
		provider, id, err := pluginResolver(cfg)
		if err == nil {
			return provider, id, nil
		}
		if !errors.Is(err, ErrSkipProvider) {
			return nil, "", err
		}
		// No plugin handled it — fall through to built-in factory
	}

	switch protocol {
	case "openai":
		// OpenAI with OAuth/token auth (Codex-style)
		if cfg.AuthMethod == "oauth" || cfg.AuthMethod == "token" {
			provider, err := createCodexAuthProvider()
			if err != nil {
				return nil, "", err
			}
			return provider, modelID, nil
		}
		// OpenAI with API key
		if cfg.APIKey == "" && cfg.APIBase == "" {
			return nil, "", fmt.Errorf("api_key or api_base is required for HTTP-based protocol %q", protocol)
		}
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = getDefaultAPIBase(protocol)
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			cfg.APIKey,
			apiBase,
			cfg.Proxy,
			cfg.MaxTokensField,
			cfg.RequestTimeout,
		), modelID, nil

	case "litellm", "openrouter", "groq", "zhipu", "gemini", "nvidia",
		"ollama", "moonshot", "shengsuanyun", "deepseek", "cerebras",
		"vivgrid", "volcengine", "vllm", "qwen", "mistral", "avian",
		"minimax", "longcat", "modelscope":
		// All other OpenAI-compatible HTTP providers
		if cfg.APIKey == "" && cfg.APIBase == "" {
			return nil, "", fmt.Errorf("api_key or api_base is required for HTTP-based protocol %q", protocol)
		}
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = getDefaultAPIBase(protocol)
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			cfg.APIKey,
			apiBase,
			cfg.Proxy,
			cfg.MaxTokensField,
			cfg.RequestTimeout,
		), modelID, nil

	case "anthropic":
		if cfg.AuthMethod == "oauth" || cfg.AuthMethod == "token" {
			// Use OAuth credentials from auth store
			provider, err := createClaudeAuthProvider()
			if err != nil {
				return nil, "", err
			}
			return provider, modelID, nil
		}
		// Use API key with HTTP API
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = "https://api.anthropic.com/v1"
		}
		if cfg.APIKey == "" {
			return nil, "", fmt.Errorf("api_key is required for anthropic protocol (model: %s)", cfg.Model)
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			cfg.APIKey,
			apiBase,
			cfg.Proxy,
			cfg.MaxTokensField,
			cfg.RequestTimeout,
		), modelID, nil

	case "anthropic-messages":
		// Anthropic Messages API with native format (HTTP-based, no SDK)
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = "https://api.anthropic.com/v1"
		}
		if cfg.APIKey == "" {
			return nil, "", fmt.Errorf("api_key is required for anthropic-messages protocol (model: %s)", cfg.Model)
		}
		return anthropicmessages.NewProviderWithTimeout(
			cfg.APIKey,
			apiBase,
			cfg.RequestTimeout,
		), modelID, nil

	case "antigravity":
		return NewAntigravityProvider(), modelID, nil

	case "claude-cli", "claudecli":
		workspace := cfg.Workspace
		if workspace == "" {
			workspace = "."
		}
		return NewClaudeCliProvider(workspace), modelID, nil

	case "codex-cli", "codexcli":
		workspace := cfg.Workspace
		if workspace == "" {
			workspace = "."
		}
		return NewCodexCliProvider(workspace), modelID, nil

	case "github-copilot", "copilot":
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = "localhost:4321"
		}
		connectMode := cfg.ConnectMode
		if connectMode == "" {
			connectMode = "grpc"
		}
		provider, err := NewGitHubCopilotProvider(apiBase, connectMode, modelID)
		if err != nil {
			return nil, "", err
		}
		return provider, modelID, nil

	default:
		return nil, "", fmt.Errorf("unknown protocol %q in model %q", protocol, cfg.Model)
	}
}

// getDefaultAPIBase returns the default API base URL for a given protocol.
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
