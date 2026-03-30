// MoonHub - Your ready-to-use AI assistant
// Self-Improving Behavioral Pattern Detection System
// License: MIT

package learning

import "time"

// PatternCategory defines the category of a behavioral pattern
type PatternCategory string

const (
	// User preference patterns
	CategoryPreferenceCommunication PatternCategory = "preference_communication" // "I prefer concise responses"
	CategoryPreferenceWorkflow      PatternCategory = "preference_workflow"      // "Always run tests after code changes"
	CategoryPreferenceTool          PatternCategory = "preference_tool"          // "Use TypeScript instead of JavaScript"

	// Feedback patterns
	CategorySuccessIndicator PatternCategory = "success_indicator" // Positive feedback on specific approach
	CategoryFailureIndicator PatternCategory = "failure_indicator" // Negative feedback on specific approach

	// Correction patterns
	CategoryCorrectionExplicit PatternCategory = "correction_explicit" // Direct correction from user
	CategoryCorrectionImplicit PatternCategory = "correction_implicit" // Inferred correction from behavior

	// Aggregated patterns
	CategoryBehavioralTrend PatternCategory = "behavioral_trend" // Aggregated pattern over time
)

// PatternSource defines where a pattern was detected
type PatternSource string

const (
	SourceExplicitFeedback PatternSource = "explicit_feedback"         // User said "good job" or "wrong"
	SourceImplicitBehavior PatternSource = "implicit_behavior"         // User accepted/rejected tool result
	SourceToolUsageSuccess PatternSource = "tool_usage_success"        // Tool completed successfully
	SourceToolUsageFailure PatternSource = "tool_usage_failure"        // Tool failed or was rejected
	SourceConversationFlow PatternSource = "conversation_flow"         // Natural conversation patterns
	SourceCrossSession     PatternSource = "cross_session_correlation" // Pattern detected across sessions
)

// EnhancedPattern represents a detected behavioral pattern with metadata
type EnhancedPattern struct {
	ID               string           `json:"id"`
	Category         PatternCategory  `json:"category"`
	Source           PatternSource    `json:"source"`
	Pattern          string           `json:"pattern"`           // Human-readable pattern description
	Confidence       float64          `json:"confidence"`        // 0.0 - 1.0
	Support          int              `json:"support"`           // Number of supporting instances
	Contradiction    int              `json:"contradiction"`     // Number of contradicting instances
	FirstObserved    int64            `json:"first_observed"`    // Unix timestamp (ms)
	LastObserved     int64            `json:"last_observed"`     // Unix timestamp (ms)
	ObservationCount int              `json:"observation_count"` // Total observations
	Triggers         []string         `json:"triggers"`          // Contexts where this pattern applies
	Examples         []PatternExample `json:"examples"`          // Supporting examples (max 10)
	RelatedPatterns  []string         `json:"related_patterns"`  // IDs of semantically related patterns
	Supersedes       []string         `json:"supersedes"`        // IDs of patterns this one replaces
	DecayRate        float64          `json:"decay_rate"`        // Custom decay rate (default: 0.05)
	LastDecayAt      int64            `json:"last_decay_at"`     // Last decay timestamp
	Metadata         map[string]any   `json:"metadata"`          // Additional metadata
}

// PatternExample represents a single example supporting a pattern
type PatternExample struct {
	Timestamp        int64    `json:"timestamp"`
	UserMessage      string   `json:"user_message"`
	AssistantMessage string   `json:"assistant_message"`
	ToolCalls        []string `json:"tool_calls,omitempty"`
	Outcome          string   `json:"outcome"` // positive, negative, neutral
	Confidence       float64  `json:"confidence"`
}

// TrendDirection indicates the direction of a trend
type TrendDirection string

const (
	TrendImproving TrendDirection = "improving"
	TrendStable    TrendDirection = "stable"
	TrendDeclining TrendDirection = "declining"
)

// BehavioralScore represents multi-dimensional behavioral scoring
type BehavioralScore struct {
	Overall float64 `json:"overall"` // Combined score 0.0 - 1.0

	Dimensions struct {
		ResponseQuality  float64 `json:"response_quality"`  // User satisfaction with responses
		ToolEfficiency   float64 `json:"tool_efficiency"`   // Success rate of tool usage
		ContextRelevance float64 `json:"context_relevance"` // How well context is used
		CorrectionRate   float64 `json:"correction_rate"`   // Frequency of user corrections (lower is better)
		AdaptationSpeed  float64 `json:"adaptation_speed"`  // How quickly agent adapts
	} `json:"dimensions"`

	Trends struct {
		ResponseQuality  TrendDirection `json:"response_quality"`
		ToolEfficiency   TrendDirection `json:"tool_efficiency"`
		ContextRelevance TrendDirection `json:"context_relevance"`
	} `json:"trends"`

	BasedOn struct {
		SampleSize  int   `json:"sample_size"`
		TimeWindow  int64 `json:"time_window"`  // ms
		LastUpdated int64 `json:"last_updated"` // Unix timestamp (ms)
	} `json:"based_on"`
}

// ToolUsagePattern tracks tool usage statistics
type ToolUsagePattern struct {
	ToolName          string         `json:"tool_name"`
	TotalCalls        int            `json:"total_calls"`
	SuccessfulCalls   int            `json:"successful_calls"`
	FailedCalls       int            `json:"failed_calls"`
	UserRejectedCalls int            `json:"user_rejected_calls"` // Tool ran but user didn't like result
	UserAcceptedCalls int            `json:"user_accepted_calls"` // Tool ran and user liked result
	AvgDurationMs     int64          `json:"avg_duration_ms"`
	P50DurationMs     int64          `json:"p50_duration_ms"`
	P95DurationMs     int64          `json:"p95_duration_ms"`
	CommonContexts    []string       `json:"common_contexts"` // When is this tool typically used?
	CommonArgs        map[string]any `json:"common_args"`     // Common argument patterns
	UserPreference    float64        `json:"user_preference"` // -1.0 to 1.0 (avoid to prefer)
	UsageTrend        TrendDirection `json:"usage_trend"`
	LastUsed          int64          `json:"last_used"` // Unix timestamp (ms)
}

// PatternContradiction represents a detected contradiction between patterns
type PatternContradiction struct {
	ID                string                   `json:"id"`
	PatternA          string                   `json:"pattern_a"`          // Pattern ID
	PatternB          string                   `json:"pattern_b"`          // Pattern ID
	ContradictionType string                   `json:"contradiction_type"` // direct, contextual, temporal
	Severity          float64                  `json:"severity"`           // 0.0 - 1.0
	Resolution        *ContradictionResolution `json:"resolution,omitempty"`
	DetectedAt        int64                    `json:"detected_at"` // Unix timestamp (ms)
}

// ContradictionResolution represents how a contradiction was resolved
type ContradictionResolution struct {
	Strategy       string `json:"strategy"`                  // keep_newer, keep_stronger, merge, contextualize, ask_user
	Winner         string `json:"winner,omitempty"`          // Pattern ID that won
	MergedPattern  string `json:"merged_pattern,omitempty"`  // New pattern ID if merged
	ContextualNote string `json:"contextual_note,omitempty"` // Explanation of context-dependent applicability
	ResolvedAt     int64  `json:"resolved_at"`               // Unix timestamp (ms)
}

// ProactiveSuggestion represents a suggestion for improvement
type ProactiveSuggestion struct {
	ID              string           `json:"id"`
	Type            string           `json:"type"` // optimization, preference, workflow, tool_usage, correction
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Confidence      float64          `json:"confidence"`
	Impact          string           `json:"impact"`   // high, medium, low
	BasedOn         []string         `json:"based_on"` // Pattern IDs
	Evidence        string           `json:"evidence"` // Human-readable explanation
	SuggestedAction *SuggestedAction `json:"suggested_action,omitempty"`
	CreatedAt       int64            `json:"created_at"` // Unix timestamp (ms)
	Dismissed       bool             `json:"dismissed"`
	Applied         bool             `json:"applied"`
}

// Suggestion is an alias for ProactiveSuggestion for convenience
type Suggestion = ProactiveSuggestion

// SuggestedAction represents an action that can be taken on a suggestion
type SuggestedAction struct {
	Type    string         `json:"type"` // update_preference, modify_tool_usage, change_workflow
	Details map[string]any `json:"details"`
}

// DetectedSignal represents a signal detected from user interaction
type DetectedSignal struct {
	Type             PatternCategory `json:"type"` // positive, negative, correction, preference, implicit
	Confidence       float64         `json:"confidence"`
	Source           PatternSource   `json:"source"`
	Category         PatternCategory `json:"category"`
	Context          string          `json:"context"`
	ExtractedPattern string          `json:"extracted_pattern,omitempty"`
	Timestamp        int64           `json:"timestamp"` // Unix timestamp (ms)
}

// AnalysisContext provides context for pattern analysis
type AnalysisContext struct {
	UserID            string
	SessionID         string
	MessageHistory    []Message
	RecentToolCalls   []ToolCall
	RecentToolResults []ToolResult
	CurrentPatterns   []EnhancedPattern
	BehavioralScore   BehavioralScore
}

// AnalysisResult contains the results of pattern analysis
type AnalysisResult struct {
	Signals        []DetectedSignal       `json:"signals"`
	NewPatterns    []EnhancedPattern      `json:"new_patterns"`
	PatternUpdates []PatternUpdate        `json:"pattern_updates"`
	Contradictions []PatternContradiction `json:"contradictions,omitempty"`
	Suggestions    []ProactiveSuggestion  `json:"suggestions"`
}

// PatternUpdate represents an update to an existing pattern
type PatternUpdate struct {
	PatternID string         `json:"pattern_id"`
	Updates   map[string]any `json:"updates"`
	Reason    string         `json:"reason"`
}

// Message represents a simplified message for analysis
type Message struct {
	Role    string `json:"role"` // user, assistant, system, tool
	Content string `json:"content"`
}

// ToolCall represents a simplified tool call for analysis
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	Tool       string `json:"tool"` // Simplified name for detection context
	Success    bool   `json:"success"`
	Result     string `json:"result"`
	DurationMs int64  `json:"duration_ms"`
}

// EvolutionResult contains the results of pattern evolution
type EvolutionResult struct {
	PatternsDecayed        int `json:"patterns_decayed"`
	PatternsMerged         int `json:"patterns_merged"`
	PatternsPruned         int `json:"patterns_pruned"`
	PatternsGeneralized    int `json:"patterns_generalized"`
	ContradictionsResolved int `json:"contradictions_resolved"`
}

// Config holds configuration for the learning engine
type Config struct {
	DBPath                       string  `json:"db_path"`
	Language                     string  `json:"language"`                       // Default: "auto" (options: "en", "zh", "auto")
	MinConfidence                float64 `json:"min_confidence"`                 // Default: 0.7
	MaxExamples                  int     `json:"max_examples"`                   // Default: 10
	EnableSemanticDetection      bool    `json:"enable_semantic_detection"`      // Default: true
	EnableImplicitSignals        bool    `json:"enable_implicit_signals"`        // Default: true
	EnableBehavioralScoring      bool    `json:"enable_behavioral_scoring"`      // Default: true
	EnablePatternEvolution       bool    `json:"enable_pattern_evolution"`       // Default: false (enable after testing)
	EnableContradictionDetection bool    `json:"enable_contradiction_detection"` // Default: false
	EnableSuggestions            bool    `json:"enable_suggestions"`             // Default: false
	DecayOlderThanDays           int     `json:"decay_older_than_days"`          // Default: 7
	PruneOlderThanDays           int     `json:"prune_older_than_days"`          // Default: 30
	MergeSimilarityThreshold     float64 `json:"merge_similarity_threshold"`     // Default: 0.8
}

// DefaultConfig returns sensible defaults
func DefaultConfig(dbPath string) Config {
	return Config{
		Language:                     "auto",
		DBPath:                       dbPath,
		MinConfidence:                0.7,
		MaxExamples:                  10,
		EnableSemanticDetection:      true,
		EnableImplicitSignals:        true,
		EnableBehavioralScoring:      true,
		EnablePatternEvolution:       false,
		EnableContradictionDetection: false,
		EnableSuggestions:            false,
		DecayOlderThanDays:           7,
		PruneOlderThanDays:           30,
		MergeSimilarityThreshold:     0.8,
	}
}

// EnhancedLearnedContext represents the context assembled from learned patterns
type EnhancedLearnedContext struct {
	Preferences        string `json:"preferences"`
	Patterns           string `json:"patterns"`
	RecentCorrections  string `json:"recent_corrections"`
	ToolPreferences    string `json:"tool_preferences"`
	WorkflowPatterns   string `json:"workflow_patterns"`
	BehavioralInsights string `json:"behavioral_insights"`
	ProactiveNotes     string `json:"proactive_notes"`
}

// Stats contains statistics about the learning engine
type Stats struct {
	TotalPatterns          int                     `json:"total_patterns"`
	HighConfidencePatterns int                     `json:"high_confidence_patterns"`
	PatternsByCategory     map[PatternCategory]int `json:"patterns_by_category"`
	AvgConfidence          float64                 `json:"avg_confidence"`
	BehavioralScore        BehavioralScore         `json:"behavioral_score"`
	ToolUsagePatterns      int                     `json:"tool_usage_patterns"`
	PendingContradictions  int                     `json:"pending_contradictions"`
	PendingSuggestions     int                     `json:"pending_suggestions"`
}

// Constants for time calculations
const (
	MsPerDay = 24 * 60 * 60 * 1000

	// Default decay rate (Ebbinghaus-inspired)
	DefaultDecayRate = 0.05

	// Scoring weights
	WeightFTS5       = 0.4
	WeightTemporal   = 0.3
	WeightImportance = 0.3

	// Signal confidence levels
	ConfidencePositive   = 0.8
	ConfidenceNegative   = 0.85
	ConfidenceCorrection = 0.95
	ConfidenceImplicit   = 0.7
)

// NowMs returns current time in milliseconds
func NowMs() int64 {
	return time.Now().UnixMilli()
}

// MsToTime converts milliseconds to time.Time
func MsToTime(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond))
}

// DaysSince calculates days since a timestamp
func DaysSince(timestampMs int64) float64 {
	if timestampMs == 0 {
		return 0
	}
	return float64(NowMs()-timestampMs) / float64(MsPerDay)
}

// MinInt returns the minimum of two integers
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MinFloat64 returns the minimum of two float64 values
func MinFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// MaxFloat64 returns the maximum of two float64 values
func MaxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
