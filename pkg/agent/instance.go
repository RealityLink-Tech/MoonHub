package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/compactor"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/memory"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
	"github.com/RealityLink-Tech/MoonHub/pkg/routing"
	"github.com/RealityLink-Tech/MoonHub/pkg/session"
	"github.com/RealityLink-Tech/MoonHub/pkg/shield"
	"github.com/RealityLink-Tech/MoonHub/pkg/tools"
)

// AgentInstance represents a fully configured agent with its own workspace,
// session manager, context builder, and tool registry.
type AgentInstance struct {
	ID                        string
	Name                      string
	Model                     string
	Fallbacks                 []string
	Workspace                 string
	MaxIterations             int
	MaxTokens                 int
	Temperature               float64
	ThinkingLevel             ThinkingLevel
	ContextWindow             int
	SummarizeMessageThreshold int
	SummarizeTokenPercent     int
	Provider                  providers.LLMProvider
	Sessions                  session.SessionStore
	ContextBuilder            *ContextBuilder
	Tools                     *tools.ToolRegistry
	Subagents                 *config.SubagentsConfig
	SkillsFilter              []string
	Candidates                []providers.FallbackCandidate

	// Router is non-nil when model routing is configured and the light model
	// was successfully resolved. It scores each incoming message and decides
	// whether to route to LightCandidates or stay with Candidates.
	Router *routing.Router

	// RouterV2 is the 4-tier router. When non-nil and TierCandidates
	// is populated, the agent uses 4-tier routing (simple/moderate/complex/reasoning).
	RouterV2 *routing.RouterV2

	// TierCandidates holds the resolved provider candidates for each tier.
	// Pre-computed at agent creation to avoid repeated model_list lookups at runtime.
	// When this is populated, RouterV2 is used instead of Router.
	TierCandidates map[routing.QueryTier][]providers.FallbackCandidate

	// LightCandidates holds the resolved provider candidates for the light model.
	// Pre-computed at agent creation to avoid repeated model_list lookups at runtime.
	LightCandidates []providers.FallbackCandidate

	// Compactor is the 4-layer context compaction engine for managing conversation history.
	// When enabled, it replaces the legacy summarization with tiered summaries.
	Compactor compactor.CompactorEngine

	// CompactorTriggerTokenPercent is the token-usage threshold (percent of ContextWindow) at which
	// async compaction runs when the compactor is enabled. Sourced from config compactor.trigger_token_percent.
	CompactorTriggerTokenPercent int

	// Shield is the threat evaluation engine for runtime security enforcement.
	// It evaluates tool calls and other events against threat definitions from SHIELD.md.
	Shield *shield.ShieldEngine

	// ApprovalManager manages approval requests for actions that require user confirmation.
	ApprovalManager *shield.ApprovalManager

	// Delegation is the sub-agent delegation system (nil when disabled or init failed).
	Delegation *DelegationIntegration
}

// NewAgentInstance creates an agent instance from config.
func NewAgentInstance(
	agentCfg *config.AgentConfig,
	defaults *config.AgentDefaults,
	cfg *config.Config,
	provider providers.LLMProvider,
) *AgentInstance {
	workspace := resolveAgentWorkspace(agentCfg, defaults)
	os.MkdirAll(workspace, 0o755)

	model := resolveAgentModel(agentCfg, defaults)
	fallbacks := resolveAgentFallbacks(agentCfg, defaults)

	restrict := defaults.RestrictToWorkspace
	readRestrict := restrict && !defaults.AllowReadOutsideWorkspace

	// Compile path whitelist patterns from config.
	allowReadPaths := compilePatterns(cfg.Tools.AllowReadPaths)
	allowWritePaths := compilePatterns(cfg.Tools.AllowWritePaths)

	toolsRegistry := tools.NewToolRegistry()

	if cfg.Tools.IsToolEnabled("read_file") {
		maxReadFileSize := cfg.Tools.ReadFile.MaxReadFileSize
		toolsRegistry.Register(tools.NewReadFileTool(workspace, readRestrict, maxReadFileSize, allowReadPaths))
	}
	if cfg.Tools.IsToolEnabled("write_file") {
		toolsRegistry.Register(tools.NewWriteFileTool(workspace, restrict, allowWritePaths))
	}
	if cfg.Tools.IsToolEnabled("list_dir") {
		toolsRegistry.Register(tools.NewListDirTool(workspace, readRestrict, allowReadPaths))
	}
	if cfg.Tools.IsToolEnabled("exec") {
		execTool, err := tools.NewExecToolWithConfig(workspace, restrict, cfg)
		if err != nil {
			log.Fatalf("Critical error: unable to initialize exec tool: %v", err)
		}
		toolsRegistry.Register(execTool)
	}

	if cfg.Tools.IsToolEnabled("edit_file") {
		toolsRegistry.Register(tools.NewEditFileTool(workspace, restrict, allowWritePaths))
	}
	if cfg.Tools.IsToolEnabled("append_file") {
		toolsRegistry.Register(tools.NewAppendFileTool(workspace, restrict, allowWritePaths))
	}

	sessionsDir := filepath.Join(workspace, "sessions")
	sessions := initSessionStore(sessionsDir)

	mcpDiscoveryActive := cfg.Tools.MCP.Enabled && cfg.Tools.MCP.Discovery.Enabled
	contextBuilder := NewContextBuilder(workspace).WithToolDiscovery(
		mcpDiscoveryActive && cfg.Tools.MCP.Discovery.UseBM25,
		mcpDiscoveryActive && cfg.Tools.MCP.Discovery.UseRegex,
	)

	agentID := routing.DefaultAgentID
	agentName := ""
	var subagents *config.SubagentsConfig
	var skillsFilter []string

	if agentCfg != nil {
		agentID = routing.NormalizeAgentID(agentCfg.ID)
		agentName = agentCfg.Name
		subagents = agentCfg.Subagents
		skillsFilter = agentCfg.Skills
	}

	maxIter := defaults.MaxToolIterations
	if maxIter == 0 {
		maxIter = 20
	}

	maxTokens := defaults.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
	}

	temperature := 0.7
	if defaults.Temperature != nil {
		temperature = *defaults.Temperature
	}

	var thinkingLevelStr string
	if mc, err := cfg.GetModelConfig(model); err == nil {
		thinkingLevelStr = mc.ThinkingLevel
	}
	thinkingLevel := parseThinkingLevel(thinkingLevelStr)

	summarizeMessageThreshold := defaults.SummarizeMessageThreshold
	if summarizeMessageThreshold == 0 {
		summarizeMessageThreshold = 20
	}

	summarizeTokenPercent := defaults.SummarizeTokenPercent
	if summarizeTokenPercent == 0 {
		summarizeTokenPercent = 75
	}

	// Resolve fallback candidates
	modelCfg := providers.ModelConfig{
		Primary:   model,
		Fallbacks: fallbacks,
	}
	resolveFromModelList := func(raw string) (string, bool) {
		ensureProtocol := func(model string) string {
			model = strings.TrimSpace(model)
			if model == "" {
				return ""
			}
			if strings.Contains(model, "/") {
				return model
			}
			return "openai/" + model
		}

		raw = strings.TrimSpace(raw)
		if raw == "" {
			return "", false
		}

		if cfg != nil {
			if mc, err := cfg.GetModelConfig(raw); err == nil && mc != nil && strings.TrimSpace(mc.Model) != "" {
				return ensureProtocol(mc.Model), true
			}

			for i := range cfg.ModelList {
				fullModel := strings.TrimSpace(cfg.ModelList[i].Model)
				if fullModel == "" {
					continue
				}
				if fullModel == raw {
					return ensureProtocol(fullModel), true
				}
				_, modelID := providers.ExtractProtocol(fullModel)
				if modelID == raw {
					return ensureProtocol(fullModel), true
				}
			}
		}

		return "", false
	}

	candidates := providers.ResolveCandidatesWithLookup(modelCfg, defaults.Provider, resolveFromModelList)

	// Model routing setup: pre-resolve model candidates at creation time
	// to avoid repeated model_list lookups on every incoming message.
	var router *routing.Router
	var routerV2 *routing.RouterV2
	var lightCandidates []providers.FallbackCandidate
	var tierCandidates map[routing.QueryTier][]providers.FallbackCandidate

	rc := defaults.Routing
	if rc != nil && rc.Enabled {
		// V2 4-tier routing when tier_mapping is configured
		if len(rc.TierMapping) > 0 {
			tierCandidates = make(map[routing.QueryTier][]providers.FallbackCandidate)
			tierMapping := make(routing.TierModelMapping)
			allTiersFailed := true

			for tierStr, modelName := range rc.TierMapping {
				tier, ok := routing.TryParseTier(tierStr)
				if !ok {
					log.Printf("routing: unknown tier key %q in tier_mapping — skipping (expected simple|moderate|complex|reasoning)", tierStr)
					continue
				}

				tierCfg := providers.ModelConfig{Primary: modelName}
				resolved := providers.ResolveCandidatesWithLookup(tierCfg, defaults.Provider, resolveFromModelList)
				if len(resolved) > 0 {
					tierCandidates[tier] = resolved
					tierMapping[tier] = modelName
					allTiersFailed = false
				} else {
					log.Printf("routing: tier %q model %q not found in model_list — tier disabled", tierStr, modelName)
				}
			}

			if !allTiersFailed && len(tierMapping) > 0 {
				routerV2 = routing.NewV2(routing.RouterConfigV2{
					TierMapping:    tierMapping,
					TierBoundaries: convertTierBoundaries(rc.TierBoundaries),
				})
				var have, miss []string
				for _, t := range routing.AllTiers() {
					if _, ok := tierMapping[t]; ok {
						have = append(have, string(t))
					} else {
						miss = append(miss, string(t))
					}
				}
				log.Printf("routing: V2 4-tier routing enabled for agent %q (%d tiers mapped: %v; unmapped: %v)",
					agentID, len(tierMapping), have, miss)
			} else {
				log.Printf("routing: tier_mapping configured but no tiers resolved — routing disabled for agent %q", agentID)
			}
		} else if rc.LightModel != "" {
			// Legacy 2-tier routing
			lightModelCfg := providers.ModelConfig{Primary: rc.LightModel}
			resolved := providers.ResolveCandidatesWithLookup(lightModelCfg, defaults.Provider, resolveFromModelList)
			if len(resolved) > 0 {
				router = routing.New(routing.RouterConfig{
					LightModel: rc.LightModel,
					Threshold:  rc.Threshold,
				})
				lightCandidates = resolved
			} else {
				log.Printf("routing: light_model %q not found in model_list — routing disabled for agent %q",
					rc.LightModel, agentID)
			}
		}
	}

	triggerPct := cfg.Compactor.TriggerTokenPercent
	if triggerPct <= 0 {
		triggerPct = 70
	}
	if triggerPct > 95 {
		triggerPct = 95
	}

	// Initialize compactor if enabled
	var compactorEngine compactor.CompactorEngine
	if cfg.Compactor.Enabled && provider != nil {
		compactorCfg := compactor.Config{
			Enabled:             cfg.Compactor.Enabled,
			DBPath:              filepath.Join(workspace, "compactor.db"),
			TriggerTokenPercent: cfg.Compactor.TriggerTokenPercent,
			KeepRecent:          cfg.Compactor.KeepRecent,
			TierBudgets: compactor.TierBudgets{
				L0: cfg.Compactor.TierBudgets.L0,
				L1: cfg.Compactor.TierBudgets.L1,
				L2: cfg.Compactor.TierBudgets.L2,
			},
			DedupEnabled:             cfg.Compactor.DedupEnabled,
			DedupSimilarityThreshold: cfg.Compactor.DedupSimilarityThreshold,
			StripEmoji:               cfg.Compactor.StripEmoji,
			RemoveDuplicateLines:     cfg.Compactor.RemoveDuplicateLines,
			NormalizeCJK:             cfg.Compactor.NormalizeCJK,
			SmartRuleSelection:       cfg.Compactor.SmartRuleSelection,
			ParallelProcessing:       cfg.Compactor.ParallelProcessing,
			IncrementalCompaction:    cfg.Compactor.IncrementalCompaction,
			SummarizationModel:       cfg.Compactor.SummarizationModel,
		}
		var compactorErr error
		compactorEngine, compactorErr = compactor.New(compactorCfg, provider)
		if compactorErr != nil {
			log.Printf("compactor: initialization failed: %v; using legacy summarization", compactorErr)
		}
	}

	// Initialize shield engine for runtime threat evaluation
	var shieldEngine *shield.ShieldEngine
	shieldPath := filepath.Join(workspace, "SHIELD.md")
	if _, err := os.Stat(shieldPath); err == nil {
		var loadErr error
		shieldEngine, loadErr = shield.NewEngineFromFile(shieldPath)
		if loadErr != nil {
			log.Printf("shield: failed to read/parse %s: %v; using default threat feed", shieldPath, loadErr)
			shieldEngine = shield.NewEngineWithDefaults()
		} else {
			log.Printf("shield: loaded threat feed from %s", shieldPath)
		}
	} else {
		shieldEngine = shield.NewEngineWithDefaults()
		log.Printf("shield: using default threat feed (%d threats)", shieldEngine.GetThreatCount())
	}

	// Initialize approval manager for require_approval actions
	approvalManager := shield.NewApprovalManager(5 * time.Minute)

	var delegationIntegration *DelegationIntegration
	if cfg.Delegation.Enabled {
		di, derr := NewDelegationIntegration(workspace, provider, model, cfg.Delegation)
		if derr != nil {
			log.Printf("delegation: initialization failed: %v", derr)
		} else {
			delegationIntegration = di
			if delegationIntegration.IsEnabled() {
				delegationIntegration.RegisterTools(toolsRegistry)
			}
		}
	}

	return &AgentInstance{
		ID:                           agentID,
		Name:                         agentName,
		Model:                        model,
		Fallbacks:                    fallbacks,
		Workspace:                    workspace,
		MaxIterations:                maxIter,
		MaxTokens:                    maxTokens,
		Temperature:                  temperature,
		ThinkingLevel:                thinkingLevel,
		ContextWindow:                maxTokens,
		SummarizeMessageThreshold:    summarizeMessageThreshold,
		SummarizeTokenPercent:        summarizeTokenPercent,
		Provider:                     provider,
		Sessions:                     sessions,
		ContextBuilder:               contextBuilder,
		Tools:                        toolsRegistry,
		Subagents:                    subagents,
		SkillsFilter:                 skillsFilter,
		Candidates:                   candidates,
		Router:                       router,
		RouterV2:                     routerV2,
		TierCandidates:               tierCandidates,
		LightCandidates:              lightCandidates,
		Compactor:                    compactorEngine,
		CompactorTriggerTokenPercent: triggerPct,
		Shield:                       shieldEngine,
		ApprovalManager:              approvalManager,
		Delegation:                   delegationIntegration,
	}
}

// resolveAgentWorkspace determines the workspace directory for an agent.
func resolveAgentWorkspace(agentCfg *config.AgentConfig, defaults *config.AgentDefaults) string {
	if agentCfg != nil && strings.TrimSpace(agentCfg.Workspace) != "" {
		return expandHome(strings.TrimSpace(agentCfg.Workspace))
	}
	// Use the configured default workspace (respects MOONHUB_HOME)
	if agentCfg == nil || agentCfg.Default || agentCfg.ID == "" || routing.NormalizeAgentID(agentCfg.ID) == "main" {
		return expandHome(defaults.Workspace)
	}
	// For named agents without explicit workspace, use default workspace with agent ID suffix
	id := routing.NormalizeAgentID(agentCfg.ID)
	return filepath.Join(expandHome(defaults.Workspace), "..", "workspace-"+id)
}

// resolveAgentModel resolves the primary model for an agent.
func resolveAgentModel(agentCfg *config.AgentConfig, defaults *config.AgentDefaults) string {
	if agentCfg != nil && agentCfg.Model != nil && strings.TrimSpace(agentCfg.Model.Primary) != "" {
		return strings.TrimSpace(agentCfg.Model.Primary)
	}
	return defaults.GetModelName()
}

// resolveAgentFallbacks resolves the fallback models for an agent.
func resolveAgentFallbacks(agentCfg *config.AgentConfig, defaults *config.AgentDefaults) []string {
	if agentCfg != nil && agentCfg.Model != nil && agentCfg.Model.Fallbacks != nil {
		return agentCfg.Model.Fallbacks
	}
	return defaults.ModelFallbacks
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			fmt.Printf("Warning: invalid path pattern %q: %v\n", p, err)
			continue
		}
		compiled = append(compiled, re)
	}
	return compiled
}

// Close releases resources held by the agent's session store, compactor, and approval manager.
func (a *AgentInstance) Close() error {
	var errs []error

	if a.Sessions != nil {
		if err := a.Sessions.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if a.Compactor != nil {
		if err := a.Compactor.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if a.ApprovalManager != nil {
		a.ApprovalManager.Close()
	}

	if a.Delegation != nil {
		if err := a.Delegation.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// convertTierBoundaries converts config.TierBoundariesConfig to routing tier boundaries.
// Returns nil if the config is nil or no cutpoint field was set.
// Invalid cutpoints (not strictly increasing within [-1,1]) log a warning and yield nil.
func convertTierBoundaries(cfg *config.TierBoundariesConfig) map[routing.QueryTier]routing.TierBoundary {
	if cfg == nil || !cfg.HasCustomBoundaries() {
		return nil
	}

	sm := -0.05
	if cfg.SimpleModerate != nil {
		sm = *cfg.SimpleModerate
	}
	mc := 0.15
	if cfg.ModerateComplex != nil {
		mc = *cfg.ModerateComplex
	}
	cr := 0.35
	if cfg.ComplexReasoning != nil {
		cr = *cfg.ComplexReasoning
	}

	if err := routing.ValidateTierCutpoints(sm, mc, cr); err != nil {
		log.Printf("routing: invalid tier_boundaries: %v — using default boundaries", err)
		return nil
	}

	return map[routing.QueryTier]routing.TierBoundary{
		routing.TierSimple:    {Min: -1.0, Max: sm},
		routing.TierModerate:  {Min: sm, Max: mc},
		routing.TierComplex:   {Min: mc, Max: cr},
		routing.TierReasoning: {Min: cr, Max: 1.0},
	}
}

// initSessionStore creates the session persistence backend.
// It uses the JSONL store by default and auto-migrates legacy JSON sessions.
// Falls back to SessionManager if the JSONL store cannot be initialized or
// if migration fails (which indicates the store cannot write reliably).
func initSessionStore(dir string) session.SessionStore {
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		log.Printf("memory: init store: %v; using json sessions", err)
		return session.NewSessionManager(dir)
	}

	if n, merr := memory.MigrateFromJSON(context.Background(), dir, store); merr != nil {
		// Migration failure means the store could not write data.
		// Fall back to SessionManager to avoid a split state where
		// some sessions are in JSONL and others remain in JSON.
		log.Printf("memory: migration failed: %v; falling back to json sessions", merr)
		store.Close()
		return session.NewSessionManager(dir)
	} else if n > 0 {
		log.Printf("memory: migrated %d session(s) to jsonl", n)
	}

	return session.NewJSONLBackend(store)
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}
