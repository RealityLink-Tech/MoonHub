// MoonHub - Your ready-to-use AI assistant
// Context Compactor - Main Orchestrator
// License: MIT

package compactor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
)

// CompactorEngine orchestrates the 4-layer compaction pipeline.
type CompactorEngine interface {
	// CompactIfNeeded runs compaction if threshold exceeded.
	// Returns CompactionResult if compaction occurred, nil otherwise.
	CompactIfNeeded(ctx context.Context, sessionKey string, messages []providers.Message, contextWindow int) (*CompactionResult, error)

	// ForceCompact forces compaction regardless of threshold.
	ForceCompact(ctx context.Context, sessionKey string, messages []providers.Message) (*CompactionResult, error)

	// GetTieredSummary loads persisted L0/L1/L2 summaries for the session.
	GetTieredSummary(ctx context.Context, sessionKey string) (*TieredSummary, error)

	// EstimateTokens estimates token count for text
	EstimateTokens(text string) int

	// EstimateTokensFromMessages estimates total tokens from messages
	EstimateTokensFromMessages(messages []providers.Message) int

	// Close releases resources
	Close() error
}

// CompactionResult contains the outcome of a compaction run
type CompactionResult struct {
	Summary         TieredSummary
	Metrics         CompactionMetrics
	MessagesRemoved int
	MessagesKept    int
}

// Compactor implements the 4-layer context compaction pipeline
type Compactor struct {
	config   Config
	provider providers.LLMProvider
	store    *SummaryStore
}

// New creates a new Compactor instance
func New(config Config, provider providers.LLMProvider) (*Compactor, error) {
	// Validate and set defaults
	config.Validate()

	// Create store if path is provided
	var store *SummaryStore
	var err error
	if config.DBPath != "" {
		store, err = NewSummaryStore(config.DBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create summary store: %w", err)
		}
	}

	return &Compactor{
		config:   config,
		provider: provider,
		store:    store,
	}, nil
}

// CompactIfNeeded runs compaction if threshold exceeded
func (c *Compactor) CompactIfNeeded(ctx context.Context, sessionKey string, messages []providers.Message, contextWindow int) (*CompactionResult, error) {
	if !c.config.Enabled {
		return nil, nil
	}

	// Estimate current token usage
	currentTokens := c.EstimateTokensFromMessages(messages)

	// Check if compaction should be triggered
	if !c.config.ShouldTrigger(currentTokens, contextWindow) {
		return nil, nil
	}

	return c.ForceCompact(ctx, sessionKey, messages)
}

// ForceCompact forces compaction regardless of threshold
func (c *Compactor) ForceCompact(ctx context.Context, sessionKey string, messages []providers.Message) (*CompactionResult, error) {
	startTime := time.Now()

	metrics := CompactionMetrics{
		MessagesBefore: len(messages),
		TokensBefore:   c.EstimateTokensFromMessages(messages),
	}

	// Get messages to compact (handle incremental compaction).
	// LastMessageIndex is reset to 0 after each run because the agent replaces session history
	// with the tail only; compactable range is always [0 : len(messages)-KeepRecent) when len > KeepRecent.
	messagesToCompact := messages
	if c.config.IncrementalCompaction && c.store != nil {
		state, err := c.store.GetState(ctx, sessionKey)
		if err == nil && state != nil {
			if state.LastMessageIndex < len(messages)-c.config.KeepRecent {
				keepFrom := state.LastMessageIndex
				keepTo := len(messages) - c.config.KeepRecent
				if keepTo > keepFrom {
					messagesToCompact = messages[keepFrom:keepTo]
				}
			}
		}
	}

	if len(messagesToCompact) == 0 {
		return nil, nil
	}

	// Layer 1: Rule-based pre-compression
	preCompressed, ruleStats := c.applyPreCompression(messagesToCompact)

	// Layer 2: Message deduplication
	var dedupResult DeduplicationResult
	if c.config.DedupEnabled {
		if c.config.ParallelProcessing {
			dedupResult = DeduplicateMessagesParallel(preCompressed, c.config.DedupSimilarityThreshold)
		} else {
			dedupResult = DeduplicateMessages(preCompressed, c.config.DedupSimilarityThreshold)
		}
	} else {
		dedupResult = DeduplicationResult{
			Messages:      preCompressed,
			GroupsRemoved: 0,
		}
	}

	metrics.DedupGroupsRemoved = dedupResult.GroupsRemoved
	metrics.MessagesSummarized = len(dedupResult.Messages)

	// Layer 3: LLM Summarization
	l2Summary, err := c.summarizeWithLLM(ctx, dedupResult.Messages)
	if err != nil {
		return nil, fmt.Errorf("LLM summarization failed: %w", err)
	}

	// Layer 4: Tiered summaries
	var summary *TieredSummary
	if c.config.ParallelProcessing {
		summary = GenerateTiersParallel(l2Summary, c.config.TierBudgets)
	} else {
		summary = GenerateTiers(l2Summary, c.config.TierBudgets)
	}

	// Calculate metrics
	metrics.MessagesKept = c.config.GetKeepRecentCount()
	metrics.TokensAfter = EstimateTokens(summary.L2)
	if metrics.TokensBefore > 0 {
		metrics.CompressionRatio = float64(metrics.TokensAfter) / float64(metrics.TokensBefore)
	} else {
		metrics.CompressionRatio = 1.0
	}
	metrics.DurationMs = time.Since(startTime).Milliseconds()

	// Get applied rules
	opts := c.getPreCompressOptionsFromProviders(messagesToCompact)
	metrics.RulesApplied = GetAppliedRules(opts, ruleStats)

	// Store summary if persistence is enabled
	if c.store != nil {
		if err := c.store.UpsertSummary(ctx, sessionKey, summary, &metrics); err != nil {
			logger.WarnCF("compactor", "Failed to store summary", map[string]any{
				"error":       err.Error(),
				"session_key": sessionKey,
			})
		}

		// Session history is replaced with the tail (KeepRecent) after compaction; incremental
		// compaction must treat the next run as starting from index 0 in the new slice.
		state := CompactionState{
			SessionKey:         sessionKey,
			LastCompactionTime: time.Now().UnixMilli(),
			LastMessageIndex:   0,
		}
		if err := c.store.UpsertState(ctx, sessionKey, state); err != nil {
			logger.WarnCF("compactor", "Failed to update compaction state", map[string]any{
				"error":       err.Error(),
				"session_key": sessionKey,
			})
		}

		rulesStr := "[]"
		if rulesJSON, errRules := json.Marshal(metrics.RulesApplied); errRules != nil {
			logger.WarnCF("compactor", "Failed to marshal rules for history", map[string]any{
				"error": errRules.Error(),
			})
		} else {
			rulesStr = string(rulesJSON)
		}
		compactionID, errID := c.store.GetLastCompactionID(ctx, sessionKey)
		if errID != nil {
			logger.WarnCF("compactor", "Failed to get compaction id for history", map[string]any{
				"error":       errID.Error(),
				"session_key": sessionKey,
			})
		} else if err := c.store.InsertHistory(ctx, sessionKey, compactionID,
			metrics.MessagesBefore-metrics.MessagesKept,
			metrics.DedupGroupsRemoved,
			rulesStr,
			metrics.DurationMs,
		); err != nil {
			logger.WarnCF("compactor", "Failed to insert compaction history", map[string]any{
				"error":       err.Error(),
				"session_key": sessionKey,
			})
		}
	}

	result := &CompactionResult{
		Summary:         *summary,
		Metrics:         metrics,
		MessagesRemoved: metrics.MessagesBefore - metrics.MessagesKept,
		MessagesKept:    metrics.MessagesKept,
	}

	return result, nil
}

// applyPreCompression applies Layer 1 rules to messages
func (c *Compactor) applyPreCompression(messages []providers.Message) ([]Message, RuleStats) {
	opts := c.getPreCompressOptionsFromProviders(messages)
	stats := AnalyzeContentFromProviders(messages)

	result := make([]Message, len(messages))

	if c.config.ParallelProcessing {
		var wg sync.WaitGroup
		for i, m := range messages {
			wg.Add(1)
			go func(idx int, msg providers.Message) {
				defer wg.Done()
				result[idx] = Message{
					Role:    msg.Role,
					Content: PreCompress(msg.Content, opts),
				}
			}(i, m)
		}
		wg.Wait()
	} else {
		for i, m := range messages {
			result[i] = Message{
				Role:    m.Role,
				Content: PreCompress(m.Content, opts),
			}
		}
	}

	return result, stats
}

// getPreCompressOptionsFromProviders returns pre-compression options from provider messages
func (c *Compactor) getPreCompressOptionsFromProviders(messages []providers.Message) PreCompressOptions {
	if c.config.SmartRuleSelection {
		contents := make([]string, len(messages))
		for i, m := range messages {
			contents[i] = m.Content
		}
		return SmartRuleSelection(contents)
	}

	return PreCompressOptions{
		StripEmoji:           c.config.StripEmoji,
		RemoveDuplicateLines: c.config.RemoveDuplicateLines,
		NormalizeCJK:         c.config.NormalizeCJK,
		CompressTables:       true,
		MergeBullets:         true,
	}
}

// summarizeWithLLM calls the LLM to generate a summary
func (c *Compactor) summarizeWithLLM(ctx context.Context, messages []Message) (string, error) {
	if c.provider == nil {
		return "", fmt.Errorf("no LLM provider configured")
	}

	// Build conversation content
	var sb strings.Builder
	for _, m := range messages {
		sb.WriteString(m.Role)
		sb.WriteString(": ")
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}
	summaryContent := sb.String()

	// Create summarization prompt
	systemPrompt := fmt.Sprintf(`You are a summarizer. Produce a concise summary of the following conversation.
Structure your summary with clear sections.
Preserve: key facts about the user (name, preferences, location),
important decisions made, corrections or clarifications,
and any open tasks or TODOs.
Keep your summary under %d tokens.

Format your response as:
## User Identity
[Key facts about the user]

## Decisions
[Important decisions made]

## Tasks
[Open tasks and TODOs]

## Preferences
[User preferences discovered]

## Context
[Relevant context and topics discussed]`, c.config.TierBudgets.L2)

	// Determine model to use
	model := c.config.SummarizationModel
	if model == "" {
		model = c.provider.GetDefaultModel()
	}

	// Call LLM
	resp, err := c.provider.Chat(ctx, []providers.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: summaryContent},
	}, nil, model, map[string]any{
		"max_tokens":  c.config.TierBudgets.L2 * 2, // Allow some buffer
		"temperature": 0.3,
	})

	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// GetTieredSummary loads persisted tiered summaries for the session.
func (c *Compactor) GetTieredSummary(ctx context.Context, sessionKey string) (*TieredSummary, error) {
	if c.store == nil {
		return nil, fmt.Errorf("no store configured")
	}

	summary, _, err := c.store.GetSummary(ctx, sessionKey)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

// EstimateTokens estimates token count for text
func (c *Compactor) EstimateTokens(text string) int {
	return EstimateTokens(text)
}

// EstimateTokensFromMessages estimates total tokens from provider messages
func (c *Compactor) EstimateTokensFromMessages(messages []providers.Message) int {
	total := 0
	for _, m := range messages {
		total += EstimateTokens(m.Content)
		// Add overhead for role and structure
		total += 4
	}
	return total
}

// Close releases resources
func (c *Compactor) Close() error {
	if c.store != nil {
		return c.store.Close()
	}
	return nil
}

// GetConfig returns the current configuration
func (c *Compactor) GetConfig() Config {
	return c.config
}

// AnalyzeContentFromProviders analyzes provider messages to determine rule applicability
func AnalyzeContentFromProviders(messages []providers.Message) RuleStats {
	stats := RuleStats{}
	for _, m := range messages {
		if HasCJK(m.Content) {
			stats.HasCJK = true
		}
		if HasTables(m.Content) {
			stats.HasTables = true
		}
		if emojiRegex.MatchString(m.Content) {
			stats.HasEmoji = true
		}
		// Check for duplicate lines
		lines := strings.Split(m.Content, "\n")
		seen := make(map[string]bool)
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				if seen[trimmed] {
					stats.HasDuplicates = true
					break
				}
				seen[trimmed] = true
			}
		}
	}
	return stats
}
