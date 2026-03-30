// MoonHub - Your ready-to-use AI assistant
// Context Compactor - Configuration
// License: MIT

package compactor

// Config holds compactor configuration
type Config struct {
	// Enabled turns on the 4-layer compactor
	Enabled bool

	// DBPath is the path to the SQLite database
	DBPath string

	// TriggerTokenPercent triggers when tokens exceed this % of context window
	// Default: 70 (trigger when 70% of context window is used)
	TriggerTokenPercent int

	// KeepRecent messages to preserve after compaction
	// Default: 10
	KeepRecent int

	// TierBudgets for L0/L1/L2 summaries
	TierBudgets TierBudgets

	// Dedup settings
	DedupEnabled             bool
	DedupSimilarityThreshold float64

	// PreCompression settings
	StripEmoji           bool
	RemoveDuplicateLines bool
	NormalizeCJK         bool
	SmartRuleSelection   bool

	// ParallelProcessing enables parallel layer processing
	ParallelProcessing bool

	// IncrementalCompaction only processes new messages since last compaction
	IncrementalCompaction bool

	// SummarizationModel is the model to use for summarization (optional, uses default if empty)
	SummarizationModel string
}

// DefaultConfig returns the default compactor configuration
func DefaultConfig() Config {
	return Config{
		Enabled:                  true,
		TriggerTokenPercent:      70,
		KeepRecent:               10,
		TierBudgets:              DefaultTierBudgets(),
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

// Validate validates the configuration and sets defaults for missing values
func (c *Config) Validate() {
	if c.TriggerTokenPercent <= 0 {
		c.TriggerTokenPercent = 70
	}
	if c.TriggerTokenPercent > 95 {
		c.TriggerTokenPercent = 95
	}

	if c.KeepRecent <= 0 {
		c.KeepRecent = 10
	}

	if c.TierBudgets.L0 <= 0 {
		c.TierBudgets.L0 = 200
	}
	if c.TierBudgets.L1 <= 0 {
		c.TierBudgets.L1 = 1000
	}
	if c.TierBudgets.L2 <= 0 {
		c.TierBudgets.L2 = 3000
	}

	if c.DedupSimilarityThreshold <= 0 {
		c.DedupSimilarityThreshold = 0.6
	}
	if c.DedupSimilarityThreshold > 1 {
		c.DedupSimilarityThreshold = 1.0
	}
}

// CompactionMetrics tracks compaction statistics
type CompactionMetrics struct {
	MessagesBefore     int
	MessagesSummarized int
	MessagesKept       int
	TokensBefore       int
	TokensAfter        int
	CompressionRatio   float64
	DedupGroupsRemoved int
	RulesApplied       []string
	DurationMs         int64
}

// ShouldTrigger returns true if compaction should be triggered based on token usage
func (c *Config) ShouldTrigger(currentTokens, contextWindow int) bool {
	if !c.Enabled {
		return false
	}

	threshold := int(float64(contextWindow) * float64(c.TriggerTokenPercent) / 100.0)
	return currentTokens >= threshold
}

// GetKeepRecentCount returns the number of messages to keep after compaction
func (c *Config) GetKeepRecentCount() int {
	if c.KeepRecent <= 0 {
		return 10
	}
	return c.KeepRecent
}

// GetTierBudgets returns validated tier budgets
func (c *Config) GetTierBudgets() TierBudgets {
	budgets := c.TierBudgets
	if budgets.L0 <= 0 {
		budgets.L0 = 200
	}
	if budgets.L1 <= 0 {
		budgets.L1 = 1000
	}
	if budgets.L2 <= 0 {
		budgets.L2 = 3000
	}
	return budgets
}
