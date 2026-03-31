// MoonHub - Your ready-to-use AI assistant
// Self-Improving Behavioral Pattern Detection System
// License: MIT

package learning

import (
	"sort"
	"sync"
	"time"
)

// ToolTracker tracks tool usage patterns and statistics
type ToolTracker struct {
	config Config
	mu     sync.RWMutex
	stats  map[string]*ToolUsagePattern
}

// ToolExecutionRecord represents a single tool execution
type ToolExecutionRecord struct {
	ToolName    string
	Success     bool
	DurationMs  int64
	UserContext string
	Timestamp   int64
}

// NewToolTracker creates a new tool tracker
func NewToolTracker(config Config) *ToolTracker {
	return &ToolTracker{
		config: config,
		stats:  make(map[string]*ToolUsagePattern),
	}
}

// RecordExecution records a tool execution
func (t *ToolTracker) RecordExecution(record ToolExecutionRecord) {
	t.mu.Lock()
	defer t.mu.Unlock()

	pattern, exists := t.stats[record.ToolName]
	if !exists {
		pattern = &ToolUsagePattern{
			ToolName:       record.ToolName,
			CommonContexts: []string{},
			UsageTrend:     TrendStable,
		}
		t.stats[record.ToolName] = pattern
	}

	// Update statistics
	pattern.TotalCalls++
	if record.Success {
		pattern.SuccessfulCalls++
	} else {
		pattern.FailedCalls++
	}
	pattern.LastUsed = record.Timestamp

	// Update average duration (exponential moving average)
	if pattern.AvgDurationMs == 0 {
		pattern.AvgDurationMs = record.DurationMs
	} else {
		alpha := 0.2 // Smoothing factor
		pattern.AvgDurationMs = int64(float64(alpha)*float64(record.DurationMs) + (1-alpha)*float64(pattern.AvgDurationMs))
	}

	// Track common contexts (keep top 5)
	if record.UserContext != "" {
		pattern.CommonContexts = appendIfNotExists(pattern.CommonContexts, record.UserContext, 5)
	}
}

// RecordUserRejection records when a user rejects a tool result
func (t *ToolTracker) RecordUserRejection(toolName string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	pattern, exists := t.stats[toolName]
	if !exists {
		pattern = &ToolUsagePattern{
			ToolName:   toolName,
			UsageTrend: TrendStable,
		}
		t.stats[toolName] = pattern
	}

	pattern.UserRejectedCalls++
	pattern.UserPreference = t.calculatePreference(pattern)
}

// RecordUserAcceptance records when a user accepts a tool result
func (t *ToolTracker) RecordUserAcceptance(toolName string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	pattern, exists := t.stats[toolName]
	if !exists {
		pattern = &ToolUsagePattern{
			ToolName:   toolName,
			UsageTrend: TrendStable,
		}
		t.stats[toolName] = pattern
	}

	pattern.UserAcceptedCalls++
	pattern.UserPreference = t.calculatePreference(pattern)
}

// calculatePreference calculates user preference score (-1.0 to 1.0)
func (t *ToolTracker) calculatePreference(pattern *ToolUsagePattern) float64 {
	if pattern.TotalCalls == 0 {
		return 0
	}

	rejectionRate := float64(pattern.UserRejectedCalls) / float64(pattern.TotalCalls)
	acceptanceRate := float64(pattern.UserAcceptedCalls) / float64(pattern.TotalCalls)

	// Simple formula: acceptance increases preference, rejection decreases it
	// Range is naturally -1 to 1 since both rates are 0-1
	preference := acceptanceRate - rejectionRate

	return MaxFloat64(-1, MinFloat64(1, preference))
}

// GetToolUsage returns usage pattern for a specific tool
func (t *ToolTracker) GetToolUsage(toolName string) *ToolUsagePattern {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if pattern, exists := t.stats[toolName]; exists {
		copy := *pattern
		return &copy
	}
	return nil
}

// GetAllToolUsage returns all tool usage patterns
func (t *ToolTracker) GetAllToolUsage() []ToolUsagePattern {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]ToolUsagePattern, 0, len(t.stats))
	for _, pattern := range t.stats {
		result = append(result, *pattern)
	}

	// Sort by total calls (most used first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCalls > result[j].TotalCalls
	})

	return result
}

// GetProblematicTools returns tools with low success rates
func (t *ToolTracker) GetProblematicTools(minCalls int, maxSuccessRate float64) []ToolUsagePattern {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []ToolUsagePattern
	for _, pattern := range t.stats {
		if pattern.TotalCalls >= minCalls {
			successRate := float64(pattern.SuccessfulCalls) / float64(pattern.TotalCalls)
			if successRate < maxSuccessRate {
				result = append(result, *pattern)
			}
		}
	}

	return result
}

// GetPreferredTools returns tools with high user preference
func (t *ToolTracker) GetPreferredTools(minCalls int, minPreference float64) []ToolUsagePattern {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []ToolUsagePattern
	for _, pattern := range t.stats {
		if pattern.TotalCalls >= minCalls && pattern.UserPreference >= minPreference {
			result = append(result, *pattern)
		}
	}

	return result
}

// UpdateTrends updates usage trends based on historical data
func (t *ToolTracker) UpdateTrends() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := NowMs()
	oneDayAgo := now - MsPerDay
	threeDaysAgo := now - (3 * MsPerDay)

	for _, pattern := range t.stats {
		// Simple trend analysis based on last used time
		// In a full implementation, this would analyze call frequency over time
		if pattern.LastUsed > oneDayAgo {
			pattern.UsageTrend = TrendImproving
		} else if pattern.LastUsed < threeDaysAgo {
			pattern.UsageTrend = TrendDeclining
		} else {
			pattern.UsageTrend = TrendStable
		}
	}
}

// GetStats returns statistics about tool usage
func (t *ToolTracker) GetStats() map[string]any {
	t.mu.RLock()
	defer t.mu.RUnlock()

	totalCalls := 0
	totalSuccess := 0
	totalFailed := 0

	for _, pattern := range t.stats {
		totalCalls += pattern.TotalCalls
		totalSuccess += pattern.SuccessfulCalls
		totalFailed += pattern.FailedCalls
	}

	var overallSuccessRate float64
	if totalCalls > 0 {
		overallSuccessRate = float64(totalSuccess) / float64(totalCalls)
	}

	return map[string]any{
		"unique_tools":         len(t.stats),
		"total_calls":          totalCalls,
		"successful_calls":     totalSuccess,
		"failed_calls":         totalFailed,
		"overall_success_rate": overallSuccessRate,
	}
}

// LoadFromStore loads tool usage data from the store
func (t *ToolTracker) LoadFromStore(stats map[string]*ToolUsagePattern) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for k, v := range stats {
		t.stats[k] = v
	}
}

// Helper functions

func appendIfNotExists(slice []string, item string, maxLen int) []string {
	// Check if already exists
	for _, s := range slice {
		if s == item {
			return slice
		}
	}

	// Append new item
	slice = append(slice, item)

	// Trim if exceeds max length
	if len(slice) > maxLen {
		slice = slice[len(slice)-maxLen:]
	}

	return slice
}

// GetToolUsageSince returns tool usage since a specific timestamp
func (t *ToolTracker) GetToolUsageSince(timestamp int64) []ToolExecutionRecord {
	// This would require storing individual records
	// For now, return empty as we only aggregate stats
	return nil
}

// GetMostUsedTools returns the top N most used tools
func (t *ToolTracker) GetMostUsedTools(n int) []ToolUsagePattern {
	all := t.GetAllToolUsage()
	if len(all) <= n {
		return all
	}
	return all[:n]
}

// GetLeastUsedTools returns the top N least used tools (with at least one call)
func (t *ToolTracker) GetLeastUsedTools(n int) []ToolUsagePattern {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []ToolUsagePattern
	for _, pattern := range t.stats {
		if pattern.TotalCalls > 0 {
			result = append(result, *pattern)
		}
	}

	// Sort by total calls (least used first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCalls < result[j].TotalCalls
	})

	if len(result) <= n {
		return result
	}
	return result[:n]
}

// Reset clears all tool usage statistics
func (t *ToolTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.stats = make(map[string]*ToolUsagePattern)
}

// Export exports tool usage data for persistence
func (t *ToolTracker) Export() map[string]*ToolUsagePattern {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*ToolUsagePattern, len(t.stats))
	for k, v := range t.stats {
		copy := *v
		result[k] = &copy
	}
	return result
}

// GetLastUsed returns the timestamp when a tool was last used
func (t *ToolTracker) GetLastUsed(toolName string) time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if pattern, exists := t.stats[toolName]; exists {
		return MsToTime(pattern.LastUsed)
	}
	return time.Time{}
}
