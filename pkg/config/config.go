package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/caarlos0/env/v11"

	"github.com/RealityLink-Tech/MoonHub/pkg/fileutil"
)

// rrCounter is a global counter for round-robin load balancing across models.
var rrCounter atomic.Uint64

// FlexibleStringSlice is a []string that also accepts JSON numbers,
// so allow_from can contain both "123" and 123.
// It also supports parsing comma-separated strings from environment variables,
// including both English (,) and Chinese (，) commas.
type FlexibleStringSlice []string

func (f *FlexibleStringSlice) UnmarshalJSON(data []byte) error {
	// Try []string first
	var ss []string
	if err := json.Unmarshal(data, &ss); err == nil {
		*f = ss
		return nil
	}

	// Try []interface{} to handle mixed types
	var raw []any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	result := make([]string, 0, len(raw))
	for _, v := range raw {
		switch val := v.(type) {
		case string:
			result = append(result, val)
		case float64:
			result = append(result, fmt.Sprintf("%.0f", val))
		default:
			result = append(result, fmt.Sprintf("%v", val))
		}
	}
	*f = result
	return nil
}

// UnmarshalText implements encoding.TextUnmarshaler to support env variable parsing.
// It handles comma-separated values with both English (,) and Chinese (，) commas.
func (f *FlexibleStringSlice) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*f = nil
		return nil
	}

	s := string(text)
	// Replace Chinese comma with English comma, then split
	s = strings.ReplaceAll(s, "，", ",")
	parts := strings.Split(s, ",")

	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	*f = result
	return nil
}

type Config struct {
	Agents         AgentsConfig         `json:"agents"`
	Bindings       []AgentBinding       `json:"bindings,omitempty"`
	Session        SessionConfig        `json:"session,omitempty"`
	Channels       ChannelsConfig       `json:"channels"`
	Providers      ProvidersConfig      `json:"providers,omitempty"`
	ModelList      []ModelConfig        `json:"model_list"` // New model-centric provider configuration
	Gateway        GatewayConfig        `json:"gateway"`
	Tools          ToolsConfig          `json:"tools"`
	Heartbeat      HeartbeatConfig      `json:"heartbeat"`
	Devices        DevicesConfig        `json:"devices"`
	Voice          VoiceConfig          `json:"voice"`
	AdaptiveMemory AdaptiveMemoryConfig `json:"adaptive_memory,omitempty"` // Adaptive memory system configuration
	Compactor      CompactorConfig      `json:"compactor,omitempty"`       // Context compactor configuration
	Delegation     DelegationConfig     `json:"delegation,omitempty"`      // Delegation system configuration
	Cloud          CloudConfig          `json:"cloud,omitempty"`           // Cloud directory and relay configuration
	// BuildInfo contains build-time version information
	BuildInfo BuildInfo `json:"build_info,omitempty"`
}

// AdaptiveMemoryConfig configures the adaptive memory system
type AdaptiveMemoryConfig struct {
	Enabled                  bool    `json:"enabled"                      env:"MOONHUB_ADAPTIVE_MEMORY_ENABLED"`
	EnableChinese            bool    `json:"enable_chinese"               env:"MOONHUB_ADAPTIVE_MEMORY_ENABLE_CHINESE"`
	FTSMaxResults            int     `json:"fts_max_results"              env:"MOONHUB_ADAPTIVE_MEMORY_FTS_MAX_RESULTS"`
	ContextMaxResults        int     `json:"context_max_results"          env:"MOONHUB_ADAPTIVE_MEMORY_CONTEXT_MAX_RESULTS"`
	DecayOlderThanDays       int     `json:"decay_older_than_days"        env:"MOONHUB_ADAPTIVE_MEMORY_DECAY_OLDER_THAN_DAYS"`
	DecayFactor              float64 `json:"decay_factor"                 env:"MOONHUB_ADAPTIVE_MEMORY_DECAY_FACTOR"`
	PruneImportanceMax       float64 `json:"prune_importance_max"         env:"MOONHUB_ADAPTIVE_MEMORY_PRUNE_IMPORTANCE_MAX"`
	PruneAccessCountMax      int     `json:"prune_access_count_max"       env:"MOONHUB_ADAPTIVE_MEMORY_PRUNE_ACCESS_COUNT_MAX"`
	PruneOlderThanDays       int     `json:"prune_older_than_days"        env:"MOONHUB_ADAPTIVE_MEMORY_PRUNE_OLDER_THAN_DAYS"`
	MergeSimilarityThreshold float64 `json:"merge_similarity_threshold"   env:"MOONHUB_ADAPTIVE_MEMORY_MERGE_SIMILARITY_THRESHOLD"`
}

// DefaultAdaptiveMemoryConfig returns the default adaptive memory configuration
func DefaultAdaptiveMemoryConfig() AdaptiveMemoryConfig {
	return AdaptiveMemoryConfig{
		Enabled:                  true,
		EnableChinese:            true,
		FTSMaxResults:            50,
		ContextMaxResults:        10,
		DecayOlderThanDays:       7,
		DecayFactor:              0.95,
		PruneImportanceMax:       0.1,
		PruneAccessCountMax:      0,
		PruneOlderThanDays:       30,
		MergeSimilarityThreshold: 0.8,
	}
}

// CompactorConfig configures the 4-layer context compactor system
type CompactorConfig struct {
	Enabled                  bool              `json:"enabled"                     env:"MOONHUB_COMPACTOR_ENABLED"`
	TriggerTokenPercent      int               `json:"trigger_token_percent"       env:"MOONHUB_COMPACTOR_TRIGGER_TOKEN_PERCENT"`
	KeepRecent               int               `json:"keep_recent"                 env:"MOONHUB_COMPACTOR_KEEP_RECENT"`
	TierBudgets              TierBudgetsConfig `json:"tier_budgets"`
	DedupEnabled             bool              `json:"dedup_enabled"               env:"MOONHUB_COMPACTOR_DEDUP_ENABLED"`
	DedupSimilarityThreshold float64           `json:"dedup_similarity_threshold"  env:"MOONHUB_COMPACTOR_DEDUP_SIMILARITY_THRESHOLD"`
	StripEmoji               bool              `json:"strip_emoji"                 env:"MOONHUB_COMPACTOR_STRIP_EMOJI"`
	RemoveDuplicateLines     bool              `json:"remove_duplicate_lines"      env:"MOONHUB_COMPACTOR_REMOVE_DUPLICATE_LINES"`
	NormalizeCJK             bool              `json:"normalize_cjk"               env:"MOONHUB_COMPACTOR_NORMALIZE_CJK"`
	SmartRuleSelection       bool              `json:"smart_rule_selection"        env:"MOONHUB_COMPACTOR_SMART_RULE_SELECTION"`
	ParallelProcessing       bool              `json:"parallel_processing"         env:"MOONHUB_COMPACTOR_PARALLEL_PROCESSING"`
	IncrementalCompaction    bool              `json:"incremental_compaction"      env:"MOONHUB_COMPACTOR_INCREMENTAL_COMPACTION"`
	SummarizationModel       string            `json:"summarization_model"         env:"MOONHUB_COMPACTOR_SUMMARIZATION_MODEL"`
}

// TierBudgetsConfig holds token budgets for each tier
type TierBudgetsConfig struct {
	L0 int `json:"l0"` // Ultra-compact (default: 200 tokens)
	L1 int `json:"l1"` // Working memory (default: 1000 tokens)
	L2 int `json:"l2"` // Full context (default: 3000 tokens)
}

// DefaultCompactorConfig returns the default compactor configuration
func DefaultCompactorConfig() CompactorConfig {
	return CompactorConfig{
		Enabled:                  true,
		TriggerTokenPercent:      70,
		KeepRecent:               10,
		TierBudgets:              TierBudgetsConfig{L0: 200, L1: 1000, L2: 3000},
		DedupEnabled:             true,
		DedupSimilarityThreshold: 0.6,
		StripEmoji:               true,
		RemoveDuplicateLines:     true,
		NormalizeCJK:             true,
		SmartRuleSelection:       true,
		ParallelProcessing:       true,
		IncrementalCompaction:    true,
	}
}

// DelegationConfig configures the sub-agent delegation system
type DelegationConfig struct {
	Enabled              bool     `json:"enabled"                env:"MOONHUB_DELEGATION_ENABLED"`
	DefaultSubAgentTools []string `json:"default_sub_agent_tools"`
	MaxActivePerUser     int      `json:"max_active_per_user"    env:"MOONHUB_DELEGATION_MAX_ACTIVE_PER_USER"`
	MaxConcurrentTasks   int      `json:"max_concurrent_tasks"   env:"MOONHUB_DELEGATION_MAX_CONCURRENT_TASKS"`
	RetentionDays        int      `json:"retention_days"         env:"MOONHUB_DELEGATION_RETENTION_DAYS"`
	ReuseThreshold       float64  `json:"reuse_threshold"        env:"MOONHUB_DELEGATION_REUSE_THRESHOLD"`
}

// DefaultDelegationConfig returns the default delegation configuration
func DefaultDelegationConfig() DelegationConfig {
	return DelegationConfig{
		Enabled:              false, // Disabled by default
		DefaultSubAgentTools: []string{"read_file", "list_dir", "web_search", "web_fetch", "memory_recall"},
		MaxActivePerUser:     10,
		MaxConcurrentTasks:   3,
		RetentionDays:        14,
		ReuseThreshold:       0.6,
	}
}

// CloudConfig configures cloud directory and relay services.
type CloudConfig struct {
	Enabled           bool   `json:"enabled"                       env:"MOONHUB_CLOUD_ENABLED"`
	DirectoryURL      string `json:"directory_url"                 env:"MOONHUB_CLOUD_DIRECTORY_URL"`
	RelayURL          string `json:"relay_url"                     env:"MOONHUB_CLOUD_RELAY_URL"`
	HeartbeatInterval int    `json:"heartbeat_interval,omitempty"  env:"MOONHUB_CLOUD_HEARTBEAT_INTERVAL"`
	RegisterOnBoot    bool   `json:"register_on_boot,omitempty"    env:"MOONHUB_CLOUD_REGISTER_ON_BOOT"`
}

// BuildInfo contains build-time version information
type BuildInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

// MarshalJSON implements custom JSON marshaling for Config
// to omit providers section when empty and session when empty
func (c Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	aux := &struct {
		Providers *ProvidersConfig `json:"providers,omitempty"`
		Session   *SessionConfig   `json:"session,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(&c),
	}

	// Only include providers if not empty
	if !c.Providers.IsEmpty() {
		aux.Providers = &c.Providers
	}

	// Only include session if not empty
	if c.Session.DMScope != "" || len(c.Session.IdentityLinks) > 0 {
		aux.Session = &c.Session
	}

	return json.Marshal(aux)
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
	List     []AgentConfig `json:"list,omitempty"`
}

// AgentModelConfig supports both string and structured model config.
// String format: "gpt-4" (just primary, no fallbacks)
// Object format: {"primary": "gpt-4", "fallbacks": ["claude-haiku"]}
type AgentModelConfig struct {
	Primary   string   `json:"primary,omitempty"`
	Fallbacks []string `json:"fallbacks,omitempty"`
}

func (m *AgentModelConfig) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		m.Primary = s
		m.Fallbacks = nil
		return nil
	}
	type raw struct {
		Primary   string   `json:"primary"`
		Fallbacks []string `json:"fallbacks"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	m.Primary = r.Primary
	m.Fallbacks = r.Fallbacks
	return nil
}

func (m AgentModelConfig) MarshalJSON() ([]byte, error) {
	if len(m.Fallbacks) == 0 && m.Primary != "" {
		return json.Marshal(m.Primary)
	}
	type raw struct {
		Primary   string   `json:"primary,omitempty"`
		Fallbacks []string `json:"fallbacks,omitempty"`
	}
	return json.Marshal(raw{Primary: m.Primary, Fallbacks: m.Fallbacks})
}

type AgentConfig struct {
	ID        string            `json:"id"`
	Default   bool              `json:"default,omitempty"`
	Name      string            `json:"name,omitempty"`
	Workspace string            `json:"workspace,omitempty"`
	Model     *AgentModelConfig `json:"model,omitempty"`
	Skills    []string          `json:"skills,omitempty"`
	Subagents *SubagentsConfig  `json:"subagents,omitempty"`
}

type SubagentsConfig struct {
	AllowAgents []string          `json:"allow_agents,omitempty"`
	Model       *AgentModelConfig `json:"model,omitempty"`
}

type PeerMatch struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type BindingMatch struct {
	Channel   string     `json:"channel"`
	AccountID string     `json:"account_id,omitempty"`
	Peer      *PeerMatch `json:"peer,omitempty"`
	GuildID   string     `json:"guild_id,omitempty"`
	TeamID    string     `json:"team_id,omitempty"`
}

type AgentBinding struct {
	AgentID string       `json:"agent_id"`
	Match   BindingMatch `json:"match"`
}

type SessionConfig struct {
	DMScope       string              `json:"dm_scope,omitempty"`
	IdentityLinks map[string][]string `json:"identity_links,omitempty"`
}

// RoutingConfig controls the intelligent model routing feature.
// When enabled, each incoming message is scored against structural features
// (message length, code blocks, tool call history, conversation depth, attachments).
// Messages scoring below Threshold are sent to LightModel; all others use the
// agent's primary model. This reduces cost and latency for simple tasks without
// requiring any keyword matching — all scoring is language-agnostic.
//
// V2 4-tier routing:
// When TierMapping is configured, the router uses 4 tiers instead of 2:
// - simple: greetings, trivial Q&A
// - moderate: short questions, simple tasks
// - complex: coding, long context
// - reasoning: deep analysis, multi-modal
type RoutingConfig struct {
	Enabled    bool    `json:"enabled"`
	LightModel string  `json:"light_model"` // model_name from model_list to use for simple tasks
	Threshold  float64 `json:"threshold"`   // complexity score in [0,1]; score >= threshold → primary model

	// V2 4-tier routing
	TierMapping    map[string]string     `json:"tier_mapping,omitempty"`    // tier -> model_name (e.g., {"simple": "gpt-3.5", "complex": "gpt-4"})
	TierBoundaries *TierBoundariesConfig `json:"tier_boundaries,omitempty"` // custom tier boundaries
}

// TierBoundariesConfig holds custom score boundaries for the 4-tier system.
// Each field defines the threshold between tiers. Pointer fields distinguish
// "JSON omitted / unset" from an explicit 0.0 (e.g. simple_moderate: 0).
// Omitted fields fall back to defaults: -0.05, 0.15, 0.35.
type TierBoundariesConfig struct {
	SimpleModerate   *float64 `json:"simple_moderate,omitempty"`
	ModerateComplex  *float64 `json:"moderate_complex,omitempty"`
	ComplexReasoning *float64 `json:"complex_reasoning,omitempty"`
}

// HasCustomBoundaries returns true if any cutpoint was set in JSON/config.
func (c *TierBoundariesConfig) HasCustomBoundaries() bool {
	if c == nil {
		return false
	}
	return c.SimpleModerate != nil || c.ModerateComplex != nil || c.ComplexReasoning != nil
}

type AgentDefaults struct {
	Workspace                 string         `json:"workspace"                       env:"MOONHUB_AGENTS_DEFAULTS_WORKSPACE"`
	RestrictToWorkspace       bool           `json:"restrict_to_workspace"           env:"MOONHUB_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE"`
	AllowReadOutsideWorkspace bool           `json:"allow_read_outside_workspace"    env:"MOONHUB_AGENTS_DEFAULTS_ALLOW_READ_OUTSIDE_WORKSPACE"`
	Provider                  string         `json:"provider"                        env:"MOONHUB_AGENTS_DEFAULTS_PROVIDER"`
	ModelName                 string         `json:"model_name"                      env:"MOONHUB_AGENTS_DEFAULTS_MODEL_NAME"`
	Model                     string         `json:"model,omitempty"                 env:"MOONHUB_AGENTS_DEFAULTS_MODEL"` // Deprecated: use model_name instead
	ModelFallbacks            []string       `json:"model_fallbacks,omitempty"`
	ImageModel                string         `json:"image_model,omitempty"           env:"MOONHUB_AGENTS_DEFAULTS_IMAGE_MODEL"`
	ImageModelFallbacks       []string       `json:"image_model_fallbacks,omitempty"`
	MaxTokens                 int            `json:"max_tokens"                      env:"MOONHUB_AGENTS_DEFAULTS_MAX_TOKENS"`
	Temperature               *float64       `json:"temperature,omitempty"           env:"MOONHUB_AGENTS_DEFAULTS_TEMPERATURE"`
	MaxToolIterations         int            `json:"max_tool_iterations"             env:"MOONHUB_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS"`
	SummarizeMessageThreshold int            `json:"summarize_message_threshold"     env:"MOONHUB_AGENTS_DEFAULTS_SUMMARIZE_MESSAGE_THRESHOLD"`
	SummarizeTokenPercent     int            `json:"summarize_token_percent"         env:"MOONHUB_AGENTS_DEFAULTS_SUMMARIZE_TOKEN_PERCENT"`
	MaxMediaSize              int            `json:"max_media_size,omitempty"        env:"MOONHUB_AGENTS_DEFAULTS_MAX_MEDIA_SIZE"`
	Routing                   *RoutingConfig `json:"routing,omitempty"`
}

const DefaultMaxMediaSize = 20 * 1024 * 1024 // 20 MB

func (d *AgentDefaults) GetMaxMediaSize() int {
	if d.MaxMediaSize > 0 {
		return d.MaxMediaSize
	}
	return DefaultMaxMediaSize
}

// GetModelName returns the effective model name for the agent defaults.
// It prefers the new "model_name" field but falls back to "model" for backward compatibility.
func (d *AgentDefaults) GetModelName() string {
	if d.ModelName != "" {
		return d.ModelName
	}
	return d.Model
}

type ChannelsConfig struct {
	WhatsApp   WhatsAppConfig   `json:"whatsapp"`
	Telegram   TelegramConfig   `json:"telegram"`
	Feishu     FeishuConfig     `json:"feishu"`
	Discord    DiscordConfig    `json:"discord"`
	MaixCam    MaixCamConfig    `json:"maixcam"`
	QQ         QQConfig         `json:"qq"`
	DingTalk   DingTalkConfig   `json:"dingtalk"`
	Slack      SlackConfig      `json:"slack"`
	Matrix     MatrixConfig     `json:"matrix"`
	LINE       LINEConfig       `json:"line"`
	OneBot     OneBotConfig     `json:"onebot"`
	WeCom      WeComConfig      `json:"wecom"`
	WeComApp   WeComAppConfig   `json:"wecom_app"`
	WeComAIBot WeComAIBotConfig `json:"wecom_aibot"`
	Pico       PicoConfig       `json:"pico"`
	MoonHub    MoonHubConfig    `json:"moonhub"`
	IRC        IRCConfig        `json:"irc"`
}

// GroupTriggerConfig controls when the bot responds in group chats.
type GroupTriggerConfig struct {
	MentionOnly bool     `json:"mention_only,omitempty"`
	Prefixes    []string `json:"prefixes,omitempty"`
}

// TypingConfig controls typing indicator behavior (Phase 10).
type TypingConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

// PlaceholderConfig controls placeholder message behavior (Phase 10).
type PlaceholderConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Text    string `json:"text,omitempty"`
}

type WhatsAppConfig struct {
	Enabled            bool                `json:"enabled"              env:"MOONHUB_CHANNELS_WHATSAPP_ENABLED"`
	BridgeURL          string              `json:"bridge_url"           env:"MOONHUB_CHANNELS_WHATSAPP_BRIDGE_URL"`
	UseNative          bool                `json:"use_native"           env:"MOONHUB_CHANNELS_WHATSAPP_USE_NATIVE"`
	SessionStorePath   string              `json:"session_store_path"   env:"MOONHUB_CHANNELS_WHATSAPP_SESSION_STORE_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"MOONHUB_CHANNELS_WHATSAPP_ALLOW_FROM"`
	ReasoningChannelID string              `json:"reasoning_channel_id" env:"MOONHUB_CHANNELS_WHATSAPP_REASONING_CHANNEL_ID"`
}

type TelegramConfig struct {
	Enabled            bool                `json:"enabled"                 env:"MOONHUB_CHANNELS_TELEGRAM_ENABLED"`
	Token              string              `json:"token"                   env:"MOONHUB_CHANNELS_TELEGRAM_TOKEN"`
	BaseURL            string              `json:"base_url"                env:"MOONHUB_CHANNELS_TELEGRAM_BASE_URL"`
	Proxy              string              `json:"proxy"                   env:"MOONHUB_CHANNELS_TELEGRAM_PROXY"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"MOONHUB_CHANNELS_TELEGRAM_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"MOONHUB_CHANNELS_TELEGRAM_REASONING_CHANNEL_ID"`
}

type FeishuConfig struct {
	Enabled             bool                `json:"enabled"                 env:"MOONHUB_CHANNELS_FEISHU_ENABLED"`
	AppID               string              `json:"app_id"                  env:"MOONHUB_CHANNELS_FEISHU_APP_ID"`
	AppSecret           string              `json:"app_secret"              env:"MOONHUB_CHANNELS_FEISHU_APP_SECRET"`
	EncryptKey          string              `json:"encrypt_key"             env:"MOONHUB_CHANNELS_FEISHU_ENCRYPT_KEY"`
	VerificationToken   string              `json:"verification_token"      env:"MOONHUB_CHANNELS_FEISHU_VERIFICATION_TOKEN"`
	AllowFrom           FlexibleStringSlice `json:"allow_from"              env:"MOONHUB_CHANNELS_FEISHU_ALLOW_FROM"`
	GroupTrigger        GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Placeholder         PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID  string              `json:"reasoning_channel_id"    env:"MOONHUB_CHANNELS_FEISHU_REASONING_CHANNEL_ID"`
	RandomReactionEmoji FlexibleStringSlice `json:"random_reaction_emoji"   env:"MOONHUB_CHANNELS_FEISHU_RANDOM_REACTION_EMOJI"`
}

type DiscordConfig struct {
	Enabled            bool                `json:"enabled"                 env:"MOONHUB_CHANNELS_DISCORD_ENABLED"`
	Token              string              `json:"token"                   env:"MOONHUB_CHANNELS_DISCORD_TOKEN"`
	Proxy              string              `json:"proxy"                   env:"MOONHUB_CHANNELS_DISCORD_PROXY"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"MOONHUB_CHANNELS_DISCORD_ALLOW_FROM"`
	MentionOnly        bool                `json:"mention_only"            env:"MOONHUB_CHANNELS_DISCORD_MENTION_ONLY"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"MOONHUB_CHANNELS_DISCORD_REASONING_CHANNEL_ID"`
}

type MaixCamConfig struct {
	Enabled            bool                `json:"enabled"              env:"MOONHUB_CHANNELS_MAIXCAM_ENABLED"`
	Host               string              `json:"host"                 env:"MOONHUB_CHANNELS_MAIXCAM_HOST"`
	Port               int                 `json:"port"                 env:"MOONHUB_CHANNELS_MAIXCAM_PORT"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"MOONHUB_CHANNELS_MAIXCAM_ALLOW_FROM"`
	ReasoningChannelID string              `json:"reasoning_channel_id" env:"MOONHUB_CHANNELS_MAIXCAM_REASONING_CHANNEL_ID"`
}

type QQConfig struct {
	Enabled            bool                `json:"enabled"                 env:"MOONHUB_CHANNELS_QQ_ENABLED"`
	AppID              string              `json:"app_id"                  env:"MOONHUB_CHANNELS_QQ_APP_ID"`
	AppSecret          string              `json:"app_secret"              env:"MOONHUB_CHANNELS_QQ_APP_SECRET"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"MOONHUB_CHANNELS_QQ_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	MaxMessageLength   int                 `json:"max_message_length"      env:"MOONHUB_CHANNELS_QQ_MAX_MESSAGE_LENGTH"`
	SendMarkdown       bool                `json:"send_markdown"           env:"MOONHUB_CHANNELS_QQ_SEND_MARKDOWN"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"MOONHUB_CHANNELS_QQ_REASONING_CHANNEL_ID"`
}

type DingTalkConfig struct {
	Enabled            bool                `json:"enabled"                 env:"MOONHUB_CHANNELS_DINGTALK_ENABLED"`
	ClientID           string              `json:"client_id"               env:"MOONHUB_CHANNELS_DINGTALK_CLIENT_ID"`
	ClientSecret       string              `json:"client_secret"           env:"MOONHUB_CHANNELS_DINGTALK_CLIENT_SECRET"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"MOONHUB_CHANNELS_DINGTALK_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"MOONHUB_CHANNELS_DINGTALK_REASONING_CHANNEL_ID"`
}

type SlackConfig struct {
	Enabled            bool                `json:"enabled"                 env:"MOONHUB_CHANNELS_SLACK_ENABLED"`
	BotToken           string              `json:"bot_token"               env:"MOONHUB_CHANNELS_SLACK_BOT_TOKEN"`
	AppToken           string              `json:"app_token"               env:"MOONHUB_CHANNELS_SLACK_APP_TOKEN"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"MOONHUB_CHANNELS_SLACK_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"MOONHUB_CHANNELS_SLACK_REASONING_CHANNEL_ID"`
}

type MatrixConfig struct {
	Enabled            bool                `json:"enabled"                  env:"MOONHUB_CHANNELS_MATRIX_ENABLED"`
	Homeserver         string              `json:"homeserver"               env:"MOONHUB_CHANNELS_MATRIX_HOMESERVER"`
	UserID             string              `json:"user_id"                  env:"MOONHUB_CHANNELS_MATRIX_USER_ID"`
	AccessToken        string              `json:"access_token"             env:"MOONHUB_CHANNELS_MATRIX_ACCESS_TOKEN"`
	DeviceID           string              `json:"device_id,omitempty"      env:"MOONHUB_CHANNELS_MATRIX_DEVICE_ID"`
	JoinOnInvite       bool                `json:"join_on_invite"           env:"MOONHUB_CHANNELS_MATRIX_JOIN_ON_INVITE"`
	MessageFormat      string              `json:"message_format,omitempty" env:"MOONHUB_CHANNELS_MATRIX_MESSAGE_FORMAT"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"               env:"MOONHUB_CHANNELS_MATRIX_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"     env:"MOONHUB_CHANNELS_MATRIX_REASONING_CHANNEL_ID"`
}

type LINEConfig struct {
	Enabled            bool                `json:"enabled"                 env:"MOONHUB_CHANNELS_LINE_ENABLED"`
	ChannelSecret      string              `json:"channel_secret"          env:"MOONHUB_CHANNELS_LINE_CHANNEL_SECRET"`
	ChannelAccessToken string              `json:"channel_access_token"    env:"MOONHUB_CHANNELS_LINE_CHANNEL_ACCESS_TOKEN"`
	WebhookHost        string              `json:"webhook_host"            env:"MOONHUB_CHANNELS_LINE_WEBHOOK_HOST"`
	WebhookPort        int                 `json:"webhook_port"            env:"MOONHUB_CHANNELS_LINE_WEBHOOK_PORT"`
	WebhookPath        string              `json:"webhook_path"            env:"MOONHUB_CHANNELS_LINE_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"MOONHUB_CHANNELS_LINE_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"MOONHUB_CHANNELS_LINE_REASONING_CHANNEL_ID"`
}

type OneBotConfig struct {
	Enabled            bool                `json:"enabled"                 env:"MOONHUB_CHANNELS_ONEBOT_ENABLED"`
	WSUrl              string              `json:"ws_url"                  env:"MOONHUB_CHANNELS_ONEBOT_WS_URL"`
	AccessToken        string              `json:"access_token"            env:"MOONHUB_CHANNELS_ONEBOT_ACCESS_TOKEN"`
	ReconnectInterval  int                 `json:"reconnect_interval"      env:"MOONHUB_CHANNELS_ONEBOT_RECONNECT_INTERVAL"`
	GroupTriggerPrefix []string            `json:"group_trigger_prefix"    env:"MOONHUB_CHANNELS_ONEBOT_GROUP_TRIGGER_PREFIX"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"MOONHUB_CHANNELS_ONEBOT_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"MOONHUB_CHANNELS_ONEBOT_REASONING_CHANNEL_ID"`
}

type WeComConfig struct {
	Enabled            bool                `json:"enabled"                 env:"MOONHUB_CHANNELS_WECOM_ENABLED"`
	Token              string              `json:"token"                   env:"MOONHUB_CHANNELS_WECOM_TOKEN"`
	EncodingAESKey     string              `json:"encoding_aes_key"        env:"MOONHUB_CHANNELS_WECOM_ENCODING_AES_KEY"`
	WebhookURL         string              `json:"webhook_url"             env:"MOONHUB_CHANNELS_WECOM_WEBHOOK_URL"`
	WebhookHost        string              `json:"webhook_host"            env:"MOONHUB_CHANNELS_WECOM_WEBHOOK_HOST"`
	WebhookPort        int                 `json:"webhook_port"            env:"MOONHUB_CHANNELS_WECOM_WEBHOOK_PORT"`
	WebhookPath        string              `json:"webhook_path"            env:"MOONHUB_CHANNELS_WECOM_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"MOONHUB_CHANNELS_WECOM_ALLOW_FROM"`
	ReplyTimeout       int                 `json:"reply_timeout"           env:"MOONHUB_CHANNELS_WECOM_REPLY_TIMEOUT"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"MOONHUB_CHANNELS_WECOM_REASONING_CHANNEL_ID"`
}

type WeComAppConfig struct {
	Enabled            bool                `json:"enabled"                 env:"MOONHUB_CHANNELS_WECOM_APP_ENABLED"`
	CorpID             string              `json:"corp_id"                 env:"MOONHUB_CHANNELS_WECOM_APP_CORP_ID"`
	CorpSecret         string              `json:"corp_secret"             env:"MOONHUB_CHANNELS_WECOM_APP_CORP_SECRET"`
	AgentID            int64               `json:"agent_id"                env:"MOONHUB_CHANNELS_WECOM_APP_AGENT_ID"`
	Token              string              `json:"token"                   env:"MOONHUB_CHANNELS_WECOM_APP_TOKEN"`
	EncodingAESKey     string              `json:"encoding_aes_key"        env:"MOONHUB_CHANNELS_WECOM_APP_ENCODING_AES_KEY"`
	WebhookHost        string              `json:"webhook_host"            env:"MOONHUB_CHANNELS_WECOM_APP_WEBHOOK_HOST"`
	WebhookPort        int                 `json:"webhook_port"            env:"MOONHUB_CHANNELS_WECOM_APP_WEBHOOK_PORT"`
	WebhookPath        string              `json:"webhook_path"            env:"MOONHUB_CHANNELS_WECOM_APP_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"MOONHUB_CHANNELS_WECOM_APP_ALLOW_FROM"`
	ReplyTimeout       int                 `json:"reply_timeout"           env:"MOONHUB_CHANNELS_WECOM_APP_REPLY_TIMEOUT"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"MOONHUB_CHANNELS_WECOM_APP_REASONING_CHANNEL_ID"`
}

type WeComAIBotConfig struct {
	Enabled            bool                `json:"enabled"              env:"MOONHUB_CHANNELS_WECOM_AIBOT_ENABLED"`
	Token              string              `json:"token"                env:"MOONHUB_CHANNELS_WECOM_AIBOT_TOKEN"`
	EncodingAESKey     string              `json:"encoding_aes_key"     env:"MOONHUB_CHANNELS_WECOM_AIBOT_ENCODING_AES_KEY"`
	WebhookPath        string              `json:"webhook_path"         env:"MOONHUB_CHANNELS_WECOM_AIBOT_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"MOONHUB_CHANNELS_WECOM_AIBOT_ALLOW_FROM"`
	ReplyTimeout       int                 `json:"reply_timeout"        env:"MOONHUB_CHANNELS_WECOM_AIBOT_REPLY_TIMEOUT"`
	MaxSteps           int                 `json:"max_steps"            env:"MOONHUB_CHANNELS_WECOM_AIBOT_MAX_STEPS"`       // Maximum streaming steps
	WelcomeMessage     string              `json:"welcome_message"      env:"MOONHUB_CHANNELS_WECOM_AIBOT_WELCOME_MESSAGE"` // Sent on enter_chat event; empty = no welcome
	ReasoningChannelID string              `json:"reasoning_channel_id" env:"MOONHUB_CHANNELS_WECOM_AIBOT_REASONING_CHANNEL_ID"`
}

type PicoConfig struct {
	Enabled         bool                `json:"enabled"                     env:"MOONHUB_CHANNELS_PICO_ENABLED"`
	Token           string              `json:"token"                       env:"MOONHUB_CHANNELS_PICO_TOKEN"`
	AllowTokenQuery bool                `json:"allow_token_query,omitempty"`
	AllowOrigins    []string            `json:"allow_origins,omitempty"`
	PingInterval    int                 `json:"ping_interval,omitempty"`
	ReadTimeout     int                 `json:"read_timeout,omitempty"`
	WriteTimeout    int                 `json:"write_timeout,omitempty"`
	MaxConnections  int                 `json:"max_connections,omitempty"`
	AllowFrom       FlexibleStringSlice `json:"allow_from"                  env:"MOONHUB_CHANNELS_PICO_ALLOW_FROM"`
	Placeholder     PlaceholderConfig   `json:"placeholder,omitempty"`
}

type MoonHubConfig struct {
	Enabled         bool                `json:"enabled"                     env:"MOONHUB_CHANNELS_MOONHUB_ENABLED"`
	Port            int                 `json:"port,omitempty"              env:"MOONHUB_CHANNELS_MOONHUB_PORT"`
	Token           string              `json:"token"                       env:"MOONHUB_CHANNELS_MOONHUB_TOKEN"`
	AllowFrom       FlexibleStringSlice `json:"allow_from"                  env:"MOONHUB_CHANNELS_MOONHUB_ALLOW_FROM"`
	PingInterval    int                 `json:"ping_interval,omitempty"`
	ReadTimeout     int                 `json:"read_timeout,omitempty"`
	MaxConnections  int                 `json:"max_connections,omitempty"`
}

type IRCConfig struct {
	Enabled            bool                `json:"enabled"                 env:"MOONHUB_CHANNELS_IRC_ENABLED"`
	Server             string              `json:"server"                  env:"MOONHUB_CHANNELS_IRC_SERVER"`
	TLS                bool                `json:"tls"                     env:"MOONHUB_CHANNELS_IRC_TLS"`
	Nick               string              `json:"nick"                    env:"MOONHUB_CHANNELS_IRC_NICK"`
	User               string              `json:"user,omitempty"          env:"MOONHUB_CHANNELS_IRC_USER"`
	RealName           string              `json:"real_name,omitempty"     env:"MOONHUB_CHANNELS_IRC_REAL_NAME"`
	Password           string              `json:"password"                env:"MOONHUB_CHANNELS_IRC_PASSWORD"`
	NickServPassword   string              `json:"nickserv_password"       env:"MOONHUB_CHANNELS_IRC_NICKSERV_PASSWORD"`
	SASLUser           string              `json:"sasl_user"               env:"MOONHUB_CHANNELS_IRC_SASL_USER"`
	SASLPassword       string              `json:"sasl_password"           env:"MOONHUB_CHANNELS_IRC_SASL_PASSWORD"`
	Channels           FlexibleStringSlice `json:"channels"                env:"MOONHUB_CHANNELS_IRC_CHANNELS"`
	RequestCaps        FlexibleStringSlice `json:"request_caps,omitempty"  env:"MOONHUB_CHANNELS_IRC_REQUEST_CAPS"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"MOONHUB_CHANNELS_IRC_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"MOONHUB_CHANNELS_IRC_REASONING_CHANNEL_ID"`
}

type HeartbeatConfig struct {
	Enabled  bool `json:"enabled"  env:"MOONHUB_HEARTBEAT_ENABLED"`
	Interval int  `json:"interval" env:"MOONHUB_HEARTBEAT_INTERVAL"` // minutes, min 5
}

type DevicesConfig struct {
	Enabled    bool `json:"enabled"     env:"MOONHUB_DEVICES_ENABLED"`
	MonitorUSB bool `json:"monitor_usb" env:"MOONHUB_DEVICES_MONITOR_USB"`
}

type VoiceConfig struct {
	EchoTranscription bool `json:"echo_transcription" env:"MOONHUB_VOICE_ECHO_TRANSCRIPTION"`
}

type ProvidersConfig struct {
	Anthropic     ProviderConfig       `json:"anthropic"`
	OpenAI        OpenAIProviderConfig `json:"openai"`
	LiteLLM       ProviderConfig       `json:"litellm"`
	OpenRouter    ProviderConfig       `json:"openrouter"`
	Groq          ProviderConfig       `json:"groq"`
	Zhipu         ProviderConfig       `json:"zhipu"`
	VLLM          ProviderConfig       `json:"vllm"`
	Gemini        ProviderConfig       `json:"gemini"`
	Nvidia        ProviderConfig       `json:"nvidia"`
	Ollama        ProviderConfig       `json:"ollama"`
	Moonshot      ProviderConfig       `json:"moonshot"`
	ShengSuanYun  ProviderConfig       `json:"shengsuanyun"`
	DeepSeek      ProviderConfig       `json:"deepseek"`
	Cerebras      ProviderConfig       `json:"cerebras"`
	Vivgrid       ProviderConfig       `json:"vivgrid"`
	VolcEngine    ProviderConfig       `json:"volcengine"`
	GitHubCopilot ProviderConfig       `json:"github_copilot"`
	Antigravity   ProviderConfig       `json:"antigravity"`
	Qwen          ProviderConfig       `json:"qwen"`
	Mistral       ProviderConfig       `json:"mistral"`
	Avian         ProviderConfig       `json:"avian"`
	Minimax       ProviderConfig       `json:"minimax"`
	LongCat       ProviderConfig       `json:"longcat"`
	ModelScope    ProviderConfig       `json:"modelscope"`
}

// IsEmpty checks if all provider configs are empty (no API keys or API bases set)
// Note: WebSearch is an optimization option and doesn't count as "non-empty"
func (p ProvidersConfig) IsEmpty() bool {
	return p.Anthropic.APIKey == "" && p.Anthropic.APIBase == "" &&
		p.OpenAI.APIKey == "" && p.OpenAI.APIBase == "" &&
		p.LiteLLM.APIKey == "" && p.LiteLLM.APIBase == "" &&
		p.OpenRouter.APIKey == "" && p.OpenRouter.APIBase == "" &&
		p.Groq.APIKey == "" && p.Groq.APIBase == "" &&
		p.Zhipu.APIKey == "" && p.Zhipu.APIBase == "" &&
		p.VLLM.APIKey == "" && p.VLLM.APIBase == "" &&
		p.Gemini.APIKey == "" && p.Gemini.APIBase == "" &&
		p.Nvidia.APIKey == "" && p.Nvidia.APIBase == "" &&
		p.Ollama.APIKey == "" && p.Ollama.APIBase == "" &&
		p.Moonshot.APIKey == "" && p.Moonshot.APIBase == "" &&
		p.ShengSuanYun.APIKey == "" && p.ShengSuanYun.APIBase == "" &&
		p.DeepSeek.APIKey == "" && p.DeepSeek.APIBase == "" &&
		p.Cerebras.APIKey == "" && p.Cerebras.APIBase == "" &&
		p.Vivgrid.APIKey == "" && p.Vivgrid.APIBase == "" &&
		p.VolcEngine.APIKey == "" && p.VolcEngine.APIBase == "" &&
		p.GitHubCopilot.APIKey == "" && p.GitHubCopilot.APIBase == "" &&
		p.Antigravity.APIKey == "" && p.Antigravity.APIBase == "" &&
		p.Qwen.APIKey == "" && p.Qwen.APIBase == "" &&
		p.Mistral.APIKey == "" && p.Mistral.APIBase == "" &&
		p.Avian.APIKey == "" && p.Avian.APIBase == "" &&
		p.Minimax.APIKey == "" && p.Minimax.APIBase == "" &&
		p.LongCat.APIKey == "" && p.LongCat.APIBase == "" &&
		p.ModelScope.APIKey == "" && p.ModelScope.APIBase == ""
}

// MarshalJSON implements custom JSON marshaling for ProvidersConfig
// to omit the entire section when empty
func (p ProvidersConfig) MarshalJSON() ([]byte, error) {
	if p.IsEmpty() {
		return []byte("null"), nil
	}
	type Alias ProvidersConfig
	return json.Marshal((*Alias)(&p))
}

type ProviderConfig struct {
	APIKey         string `json:"api_key"                   env:"MOONHUB_PROVIDERS_{{.Name}}_API_KEY"`
	APIBase        string `json:"api_base"                  env:"MOONHUB_PROVIDERS_{{.Name}}_API_BASE"`
	Proxy          string `json:"proxy,omitempty"           env:"MOONHUB_PROVIDERS_{{.Name}}_PROXY"`
	RequestTimeout int    `json:"request_timeout,omitempty" env:"MOONHUB_PROVIDERS_{{.Name}}_REQUEST_TIMEOUT"`
	AuthMethod     string `json:"auth_method,omitempty"     env:"MOONHUB_PROVIDERS_{{.Name}}_AUTH_METHOD"`
	ConnectMode    string `json:"connect_mode,omitempty"    env:"MOONHUB_PROVIDERS_{{.Name}}_CONNECT_MODE"` // only for Github Copilot, `stdio` or `grpc`
}

type OpenAIProviderConfig struct {
	ProviderConfig
	WebSearch bool `json:"web_search" env:"MOONHUB_PROVIDERS_OPENAI_WEB_SEARCH"`
}

// ModelConfig represents a model-centric provider configuration.
// It allows adding new providers (especially OpenAI-compatible ones) via configuration only.
// The model field uses protocol prefix format: [protocol/]model-identifier
// Supported protocols: openai, anthropic, antigravity, claude-cli, codex-cli, github-copilot
// Default protocol is "openai" if no prefix is specified.
type ModelConfig struct {
	// Required fields
	ModelName string `json:"model_name"` // User-facing alias for the model
	Model     string `json:"model"`      // Protocol/model-identifier (e.g., "openai/gpt-4o", "anthropic/claude-sonnet-4.6")

	// HTTP-based providers
	APIBase string `json:"api_base,omitempty"` // API endpoint URL
	APIKey  string `json:"api_key"`            // API authentication key
	Proxy   string `json:"proxy,omitempty"`    // HTTP proxy URL

	// Special providers (CLI-based, OAuth, etc.)
	AuthMethod  string `json:"auth_method,omitempty"`  // Authentication method: oauth, token
	ConnectMode string `json:"connect_mode,omitempty"` // Connection mode: stdio, grpc
	Workspace   string `json:"workspace,omitempty"`    // Workspace path for CLI-based providers

	// Optional optimizations
	RPM            int    `json:"rpm,omitempty"`              // Requests per minute limit
	MaxTokensField string `json:"max_tokens_field,omitempty"` // Field name for max tokens (e.g., "max_completion_tokens")
	RequestTimeout int    `json:"request_timeout,omitempty"`
	ThinkingLevel  string `json:"thinking_level,omitempty"` // Extended thinking: off|low|medium|high|xhigh|adaptive
}

// Validate checks if the ModelConfig has all required fields.
func (c *ModelConfig) Validate() error {
	if c.ModelName == "" {
		return fmt.Errorf("model_name is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

type GatewayConfig struct {
	Host string `json:"host" env:"MOONHUB_GATEWAY_HOST"`
	Port int    `json:"port" env:"MOONHUB_GATEWAY_PORT"`
}

type ToolDiscoveryConfig struct {
	Enabled          bool `json:"enabled"            env:"MOONHUB_TOOLS_DISCOVERY_ENABLED"`
	TTL              int  `json:"ttl"                env:"MOONHUB_TOOLS_DISCOVERY_TTL"`
	MaxSearchResults int  `json:"max_search_results" env:"MOONHUB_MAX_SEARCH_RESULTS"`
	UseBM25          bool `json:"use_bm25"           env:"MOONHUB_TOOLS_DISCOVERY_USE_BM25"`
	UseRegex         bool `json:"use_regex"          env:"MOONHUB_TOOLS_DISCOVERY_USE_REGEX"`
}

type ToolConfig struct {
	Enabled bool `json:"enabled" env:"ENABLED"`
}

type BraveConfig struct {
	Enabled    bool     `json:"enabled"     env:"MOONHUB_TOOLS_WEB_BRAVE_ENABLED"`
	APIKey     string   `json:"api_key"     env:"MOONHUB_TOOLS_WEB_BRAVE_API_KEY"`
	APIKeys    []string `json:"api_keys"    env:"MOONHUB_TOOLS_WEB_BRAVE_API_KEYS"`
	MaxResults int      `json:"max_results" env:"MOONHUB_TOOLS_WEB_BRAVE_MAX_RESULTS"`
}

type TavilyConfig struct {
	Enabled    bool     `json:"enabled"     env:"MOONHUB_TOOLS_WEB_TAVILY_ENABLED"`
	APIKey     string   `json:"api_key"     env:"MOONHUB_TOOLS_WEB_TAVILY_API_KEY"`
	APIKeys    []string `json:"api_keys"    env:"MOONHUB_TOOLS_WEB_TAVILY_API_KEYS"`
	BaseURL    string   `json:"base_url"    env:"MOONHUB_TOOLS_WEB_TAVILY_BASE_URL"`
	MaxResults int      `json:"max_results" env:"MOONHUB_TOOLS_WEB_TAVILY_MAX_RESULTS"`
}

type DuckDuckGoConfig struct {
	Enabled    bool `json:"enabled"     env:"MOONHUB_TOOLS_WEB_DUCKDUCKGO_ENABLED"`
	MaxResults int  `json:"max_results" env:"MOONHUB_TOOLS_WEB_DUCKDUCKGO_MAX_RESULTS"`
}

type PerplexityConfig struct {
	Enabled    bool     `json:"enabled"     env:"MOONHUB_TOOLS_WEB_PERPLEXITY_ENABLED"`
	APIKey     string   `json:"api_key"     env:"MOONHUB_TOOLS_WEB_PERPLEXITY_API_KEY"`
	APIKeys    []string `json:"api_keys"    env:"MOONHUB_TOOLS_WEB_PERPLEXITY_API_KEYS"`
	MaxResults int      `json:"max_results" env:"MOONHUB_TOOLS_WEB_PERPLEXITY_MAX_RESULTS"`
}

type SearXNGConfig struct {
	Enabled    bool   `json:"enabled"     env:"MOONHUB_TOOLS_WEB_SEARXNG_ENABLED"`
	BaseURL    string `json:"base_url"    env:"MOONHUB_TOOLS_WEB_SEARXNG_BASE_URL"`
	MaxResults int    `json:"max_results" env:"MOONHUB_TOOLS_WEB_SEARXNG_MAX_RESULTS"`
}

type GLMSearchConfig struct {
	Enabled bool   `json:"enabled"  env:"MOONHUB_TOOLS_WEB_GLM_ENABLED"`
	APIKey  string `json:"api_key"  env:"MOONHUB_TOOLS_WEB_GLM_API_KEY"`
	BaseURL string `json:"base_url" env:"MOONHUB_TOOLS_WEB_GLM_BASE_URL"`
	// SearchEngine specifies the search backend: "search_std" (default),
	// "search_pro", "search_pro_sogou", or "search_pro_quark".
	SearchEngine string `json:"search_engine" env:"MOONHUB_TOOLS_WEB_GLM_SEARCH_ENGINE"`
	MaxResults   int    `json:"max_results"   env:"MOONHUB_TOOLS_WEB_GLM_MAX_RESULTS"`
}

type WebToolsConfig struct {
	ToolConfig `                 envPrefix:"MOONHUB_TOOLS_WEB_"`
	Brave      BraveConfig      `                                json:"brave"`
	Tavily     TavilyConfig     `                                json:"tavily"`
	DuckDuckGo DuckDuckGoConfig `                                json:"duckduckgo"`
	Perplexity PerplexityConfig `                                json:"perplexity"`
	SearXNG    SearXNGConfig    `                                json:"searxng"`
	GLMSearch  GLMSearchConfig  `                                json:"glm_search"`
	// Proxy is an optional proxy URL for web tools (http/https/socks5/socks5h).
	// For authenticated proxies, prefer HTTP_PROXY/HTTPS_PROXY env vars instead of embedding credentials in config.
	Proxy           string `json:"proxy,omitempty"             env:"MOONHUB_TOOLS_WEB_PROXY"`
	FetchLimitBytes int64  `json:"fetch_limit_bytes,omitempty" env:"MOONHUB_TOOLS_WEB_FETCH_LIMIT_BYTES"`
}

type CronToolsConfig struct {
	ToolConfig         `    envPrefix:"MOONHUB_TOOLS_CRON_"`
	ExecTimeoutMinutes int `                                 env:"MOONHUB_TOOLS_CRON_EXEC_TIMEOUT_MINUTES" json:"exec_timeout_minutes"` // 0 means no timeout
}

type ExecConfig struct {
	ToolConfig          `         envPrefix:"MOONHUB_TOOLS_EXEC_"`
	EnableDenyPatterns  bool     `                                 env:"MOONHUB_TOOLS_EXEC_ENABLE_DENY_PATTERNS"  json:"enable_deny_patterns"`
	AllowRemote         bool     `                                 env:"MOONHUB_TOOLS_EXEC_ALLOW_REMOTE"          json:"allow_remote"`
	CustomDenyPatterns  []string `                                 env:"MOONHUB_TOOLS_EXEC_CUSTOM_DENY_PATTERNS"  json:"custom_deny_patterns"`
	CustomAllowPatterns []string `                                 env:"MOONHUB_TOOLS_EXEC_CUSTOM_ALLOW_PATTERNS" json:"custom_allow_patterns"`
	TimeoutSeconds      int      `                                 env:"MOONHUB_TOOLS_EXEC_TIMEOUT_SECONDS"       json:"timeout_seconds"` // 0 means use default (60s)
}

type SkillsToolsConfig struct {
	ToolConfig            `                       envPrefix:"MOONHUB_TOOLS_SKILLS_"`
	Registries            SkillsRegistriesConfig `                                   json:"registries"`
	Github                SkillsGithubConfig     `                                   json:"github"`
	MaxConcurrentSearches int                    `                                   json:"max_concurrent_searches" env:"MOONHUB_TOOLS_SKILLS_MAX_CONCURRENT_SEARCHES"`
	SearchCache           SearchCacheConfig      `                                   json:"search_cache"`
}

type MediaCleanupConfig struct {
	ToolConfig `    envPrefix:"MOONHUB_MEDIA_CLEANUP_"`
	MaxAge     int `                                    env:"MOONHUB_MEDIA_CLEANUP_MAX_AGE"  json:"max_age_minutes"`
	Interval   int `                                    env:"MOONHUB_MEDIA_CLEANUP_INTERVAL" json:"interval_minutes"`
}

type ReadFileToolConfig struct {
	Enabled         bool `json:"enabled"`
	MaxReadFileSize int  `json:"max_read_file_size"`
}

type ToolsConfig struct {
	AllowReadPaths  []string           `json:"allow_read_paths"  env:"MOONHUB_TOOLS_ALLOW_READ_PATHS"`
	AllowWritePaths []string           `json:"allow_write_paths" env:"MOONHUB_TOOLS_ALLOW_WRITE_PATHS"`
	Web             WebToolsConfig     `json:"web"`
	Cron            CronToolsConfig    `json:"cron"`
	Exec            ExecConfig         `json:"exec"`
	Skills          SkillsToolsConfig  `json:"skills"`
	MediaCleanup    MediaCleanupConfig `json:"media_cleanup"`
	MCP             MCPConfig          `json:"mcp"`
	Memory          ToolConfig         `json:"memory"                                                  envPrefix:"MOONHUB_TOOLS_MEMORY_"`
	AppendFile      ToolConfig         `json:"append_file"                                              envPrefix:"MOONHUB_TOOLS_APPEND_FILE_"`
	EditFile        ToolConfig         `json:"edit_file"                                                envPrefix:"MOONHUB_TOOLS_EDIT_FILE_"`
	FindSkills      ToolConfig         `json:"find_skills"                                              envPrefix:"MOONHUB_TOOLS_FIND_SKILLS_"`
	I2C             ToolConfig         `json:"i2c"                                                      envPrefix:"MOONHUB_TOOLS_I2C_"`
	InstallSkill    ToolConfig         `json:"install_skill"                                            envPrefix:"MOONHUB_TOOLS_INSTALL_SKILL_"`
	ListDir         ToolConfig         `json:"list_dir"                                                 envPrefix:"MOONHUB_TOOLS_LIST_DIR_"`
	Message         ToolConfig         `json:"message"                                                  envPrefix:"MOONHUB_TOOLS_MESSAGE_"`
	ReadFile        ReadFileToolConfig `json:"read_file"                                                envPrefix:"MOONHUB_TOOLS_READ_FILE_"`
	SendFile        ToolConfig         `json:"send_file"                                                envPrefix:"MOONHUB_TOOLS_SEND_FILE_"`
	Spawn           ToolConfig         `json:"spawn"                                                    envPrefix:"MOONHUB_TOOLS_SPAWN_"`
	SPI             ToolConfig         `json:"spi"                                                      envPrefix:"MOONHUB_TOOLS_SPI_"`
	Subagent        ToolConfig         `json:"subagent"                                                 envPrefix:"MOONHUB_TOOLS_SUBAGENT_"`
	WebFetch        ToolConfig         `json:"web_fetch"                                                envPrefix:"MOONHUB_TOOLS_WEB_FETCH_"`
	WriteFile       ToolConfig         `json:"write_file"                                               envPrefix:"MOONHUB_TOOLS_WRITE_FILE_"`
}

type SearchCacheConfig struct {
	MaxSize    int `json:"max_size"    env:"MOONHUB_SKILLS_SEARCH_CACHE_MAX_SIZE"`
	TTLSeconds int `json:"ttl_seconds" env:"MOONHUB_SKILLS_SEARCH_CACHE_TTL_SECONDS"`
}

type SkillsRegistriesConfig struct {
	ClawHub ClawHubRegistryConfig `json:"clawhub"`
}

type SkillsGithubConfig struct {
	Token string `json:"token,omitempty" env:"MOONHUB_TOOLS_SKILLS_GITHUB_AUTH_TOKEN"`
	Proxy string `json:"proxy,omitempty" env:"MOONHUB_TOOLS_SKILLS_GITHUB_PROXY"`
}

type ClawHubRegistryConfig struct {
	Enabled         bool   `json:"enabled"           env:"MOONHUB_SKILLS_REGISTRIES_CLAWHUB_ENABLED"`
	BaseURL         string `json:"base_url"          env:"MOONHUB_SKILLS_REGISTRIES_CLAWHUB_BASE_URL"`
	AuthToken       string `json:"auth_token"        env:"MOONHUB_SKILLS_REGISTRIES_CLAWHUB_AUTH_TOKEN"`
	SearchPath      string `json:"search_path"       env:"MOONHUB_SKILLS_REGISTRIES_CLAWHUB_SEARCH_PATH"`
	SkillsPath      string `json:"skills_path"       env:"MOONHUB_SKILLS_REGISTRIES_CLAWHUB_SKILLS_PATH"`
	DownloadPath    string `json:"download_path"     env:"MOONHUB_SKILLS_REGISTRIES_CLAWHUB_DOWNLOAD_PATH"`
	Timeout         int    `json:"timeout"           env:"MOONHUB_SKILLS_REGISTRIES_CLAWHUB_TIMEOUT"`
	MaxZipSize      int    `json:"max_zip_size"      env:"MOONHUB_SKILLS_REGISTRIES_CLAWHUB_MAX_ZIP_SIZE"`
	MaxResponseSize int    `json:"max_response_size" env:"MOONHUB_SKILLS_REGISTRIES_CLAWHUB_MAX_RESPONSE_SIZE"`
}

// MCPServerConfig defines configuration for a single MCP server
type MCPServerConfig struct {
	// Enabled indicates whether this MCP server is active
	Enabled bool `json:"enabled"`
	// Command is the executable to run (e.g., "npx", "python", "/path/to/server")
	Command string `json:"command"`
	// Args are the arguments to pass to the command
	Args []string `json:"args,omitempty"`
	// Env are environment variables to set for the server process (stdio only)
	Env map[string]string `json:"env,omitempty"`
	// EnvFile is the path to a file containing environment variables (stdio only)
	EnvFile string `json:"env_file,omitempty"`
	// Type is "stdio", "sse", or "http" (default: stdio if command is set, sse if url is set)
	Type string `json:"type,omitempty"`
	// URL is used for SSE/HTTP transport
	URL string `json:"url,omitempty"`
	// Headers are HTTP headers to send with requests (sse/http only)
	Headers map[string]string `json:"headers,omitempty"`
}

// MCPConfig defines configuration for all MCP servers
type MCPConfig struct {
	ToolConfig `                    envPrefix:"MOONHUB_TOOLS_MCP_"`
	Discovery  ToolDiscoveryConfig `                                json:"discovery"`
	// Servers is a map of server name to server configuration
	Servers map[string]MCPServerConfig `json:"servers,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	// Pre-scan the JSON to check how many model_list entries the user provided.
	// Go's JSON decoder reuses existing slice backing-array elements rather than
	// zero-initializing them, so fields absent from the user's JSON (e.g. api_base)
	// would silently inherit values from the DefaultConfig template at the same
	// index position. We only reset cfg.ModelList when the user actually provides
	// entries; when count is 0 we keep DefaultConfig's built-in list as fallback.
	var tmp Config
	if err := json.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}
	if len(tmp.ModelList) > 0 {
		cfg.ModelList = nil
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	// Migrate legacy channel config fields to new unified structures
	cfg.migrateChannelConfigs()

	// Auto-migrate: if only legacy providers config exists, convert to model_list
	if len(cfg.ModelList) == 0 && cfg.HasProvidersConfig() {
		cfg.ModelList = ConvertProvidersToModelList(cfg)
	}

	// Validate model_list for uniqueness and required fields
	if err := cfg.ValidateModelList(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) migrateChannelConfigs() {
	// Discord: mention_only -> group_trigger.mention_only
	if c.Channels.Discord.MentionOnly && !c.Channels.Discord.GroupTrigger.MentionOnly {
		c.Channels.Discord.GroupTrigger.MentionOnly = true
	}

	// OneBot: group_trigger_prefix -> group_trigger.prefixes
	if len(c.Channels.OneBot.GroupTriggerPrefix) > 0 &&
		len(c.Channels.OneBot.GroupTrigger.Prefixes) == 0 {
		c.Channels.OneBot.GroupTrigger.Prefixes = c.Channels.OneBot.GroupTriggerPrefix
	}
}

func SaveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	return fileutil.WriteFileAtomic(path, data, 0o600)
}

func (c *Config) WorkspacePath() string {
	return expandHome(c.Agents.Defaults.Workspace)
}

func (c *Config) GetAPIKey() string {
	if c.Providers.OpenRouter.APIKey != "" {
		return c.Providers.OpenRouter.APIKey
	}
	if c.Providers.Anthropic.APIKey != "" {
		return c.Providers.Anthropic.APIKey
	}
	if c.Providers.OpenAI.APIKey != "" {
		return c.Providers.OpenAI.APIKey
	}
	if c.Providers.Gemini.APIKey != "" {
		return c.Providers.Gemini.APIKey
	}
	if c.Providers.Zhipu.APIKey != "" {
		return c.Providers.Zhipu.APIKey
	}
	if c.Providers.Groq.APIKey != "" {
		return c.Providers.Groq.APIKey
	}
	if c.Providers.VLLM.APIKey != "" {
		return c.Providers.VLLM.APIKey
	}
	if c.Providers.ShengSuanYun.APIKey != "" {
		return c.Providers.ShengSuanYun.APIKey
	}
	if c.Providers.Cerebras.APIKey != "" {
		return c.Providers.Cerebras.APIKey
	}
	return ""
}

func (c *Config) GetAPIBase() string {
	if c.Providers.OpenRouter.APIKey != "" {
		if c.Providers.OpenRouter.APIBase != "" {
			return c.Providers.OpenRouter.APIBase
		}
		return "https://openrouter.ai/api/v1"
	}
	if c.Providers.Zhipu.APIKey != "" {
		return c.Providers.Zhipu.APIBase
	}
	if c.Providers.VLLM.APIKey != "" && c.Providers.VLLM.APIBase != "" {
		return c.Providers.VLLM.APIBase
	}
	return ""
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

// GetModelConfig returns the ModelConfig for the given model name.
// If multiple configs exist with the same model_name, it uses round-robin
// selection for load balancing. Returns an error if the model is not found.
func (c *Config) GetModelConfig(modelName string) (*ModelConfig, error) {
	matches := c.findMatches(modelName)
	if len(matches) == 0 {
		return nil, fmt.Errorf("model %q not found in model_list or providers", modelName)
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}

	// Multiple configs - use round-robin for load balancing
	idx := rrCounter.Add(1) % uint64(len(matches))
	return &matches[idx], nil
}

// findMatches finds all ModelConfig entries with the given model_name.
func (c *Config) findMatches(modelName string) []ModelConfig {
	var matches []ModelConfig
	for i := range c.ModelList {
		if c.ModelList[i].ModelName == modelName {
			matches = append(matches, c.ModelList[i])
		}
	}
	return matches
}

// HasProvidersConfig checks if any provider in the old providers config has configuration.
func (c *Config) HasProvidersConfig() bool {
	return !c.Providers.IsEmpty()
}

// ValidateModelList validates all ModelConfig entries in the model_list.
// It checks that each model config is valid.
// Note: Multiple entries with the same model_name are allowed for load balancing.
func (c *Config) ValidateModelList() error {
	for i := range c.ModelList {
		if err := c.ModelList[i].Validate(); err != nil {
			return fmt.Errorf("model_list[%d]: %w", i, err)
		}
	}
	return nil
}

func MergeAPIKeys(apiKey string, apiKeys []string) []string {
	seen := make(map[string]struct{})
	var all []string

	if k := strings.TrimSpace(apiKey); k != "" {
		if _, exists := seen[k]; !exists {
			seen[k] = struct{}{}
			all = append(all, k)
		}
	}

	for _, k := range apiKeys {
		if trimmed := strings.TrimSpace(k); trimmed != "" {
			if _, exists := seen[trimmed]; !exists {
				seen[trimmed] = struct{}{}
				all = append(all, trimmed)
			}
		}
	}

	return all
}

func (t *ToolsConfig) IsToolEnabled(name string) bool {
	switch name {
	case "web":
		return t.Web.Enabled
	case "cron":
		return t.Cron.Enabled
	case "exec":
		return t.Exec.Enabled
	case "skills":
		return t.Skills.Enabled
	case "media_cleanup":
		return t.MediaCleanup.Enabled
	case "append_file":
		return t.AppendFile.Enabled
	case "edit_file":
		return t.EditFile.Enabled
	case "find_skills":
		return t.FindSkills.Enabled
	case "i2c":
		return t.I2C.Enabled
	case "install_skill":
		return t.InstallSkill.Enabled
	case "list_dir":
		return t.ListDir.Enabled
	case "message":
		return t.Message.Enabled
	case "read_file":
		return t.ReadFile.Enabled
	case "spawn":
		return t.Spawn.Enabled
	case "spi":
		return t.SPI.Enabled
	case "subagent":
		return t.Subagent.Enabled
	case "web_fetch":
		return t.WebFetch.Enabled
	case "send_file":
		return t.SendFile.Enabled
	case "write_file":
		return t.WriteFile.Enabled
	case "mcp":
		return t.MCP.Enabled
	case "memory":
		return t.Memory.Enabled
	default:
		return true
	}
}
