// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - Web Tools Plugin (web_search, web_fetch)
//
// Copyright (c) 2026 MoonHub contributors

package web

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
	"github.com/RealityLink-Tech/MoonHub/pkg/tools"
)

func init() {
	plugin.RegisterPlugin(&WebToolPlugin{})
}

type WebToolPlugin struct {
	ctx *plugin.RuntimeContext
}

func (p *WebToolPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-tool-web",
		Name:        "Web",
		Type:        plugin.TypeTool,
		Version:     "1.0.0",
		Description: "Web search and fetch tools",
		Priority:    100,
	}
}

func (p *WebToolPlugin) Init(ctx *plugin.RuntimeContext) error {
	p.ctx = ctx
	return nil
}

func (p *WebToolPlugin) Validate(cfg *config.Config) error {
	return nil
}

func (p *WebToolPlugin) CreateTools(ctx *plugin.RuntimeContext) []tools.Tool {
	cfg := ctx.Config
	var result []tools.Tool

	if cfg.Tools.IsToolEnabled("web") {
		searchTool, err := tools.NewWebSearchTool(tools.WebSearchToolOptions{
			BraveAPIKeys:         config.MergeAPIKeys(cfg.Tools.Web.Brave.APIKey, cfg.Tools.Web.Brave.APIKeys),
			BraveMaxResults:      cfg.Tools.Web.Brave.MaxResults,
			BraveEnabled:         cfg.Tools.Web.Brave.Enabled,
			TavilyAPIKeys:        config.MergeAPIKeys(cfg.Tools.Web.Tavily.APIKey, cfg.Tools.Web.Tavily.APIKeys),
			TavilyBaseURL:        cfg.Tools.Web.Tavily.BaseURL,
			TavilyMaxResults:     cfg.Tools.Web.Tavily.MaxResults,
			TavilyEnabled:        cfg.Tools.Web.Tavily.Enabled,
			DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
			DuckDuckGoEnabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
			PerplexityAPIKeys: config.MergeAPIKeys(
				cfg.Tools.Web.Perplexity.APIKey,
				cfg.Tools.Web.Perplexity.APIKeys,
			),
			PerplexityMaxResults: cfg.Tools.Web.Perplexity.MaxResults,
			PerplexityEnabled:    cfg.Tools.Web.Perplexity.Enabled,
			SearXNGBaseURL:       cfg.Tools.Web.SearXNG.BaseURL,
			SearXNGMaxResults:    cfg.Tools.Web.SearXNG.MaxResults,
			SearXNGEnabled:       cfg.Tools.Web.SearXNG.Enabled,
			GLMSearchAPIKey:      cfg.Tools.Web.GLMSearch.APIKey,
			GLMSearchBaseURL:     cfg.Tools.Web.GLMSearch.BaseURL,
			GLMSearchEngine:      cfg.Tools.Web.GLMSearch.SearchEngine,
			GLMSearchMaxResults:  cfg.Tools.Web.GLMSearch.MaxResults,
			GLMSearchEnabled:     cfg.Tools.Web.GLMSearch.Enabled,
			Proxy:                cfg.Tools.Web.Proxy,
		})
		if err == nil && searchTool != nil {
			result = append(result, searchTool)
		}
	}

	if cfg.Tools.IsToolEnabled("web_fetch") {
		fetchTool, err := tools.NewWebFetchToolWithProxy(50000, cfg.Tools.Web.Proxy, cfg.Tools.Web.FetchLimitBytes)
		if err == nil {
			result = append(result, fetchTool)
		}
	}

	return result
}

func (p *WebToolPlugin) IsCore() bool {
	return true
}
