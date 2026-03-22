package delegation

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
)

// TimeoutEstimator provides adaptive timeout estimation based on historical data
type TimeoutEstimator struct {
	store           DelegationStore
	defaultTimeouts map[TaskCategory]time.Duration
	maxTimeout      time.Duration
	minTimeout      time.Duration
}

// NewTimeoutEstimator creates a new timeout estimator
func NewTimeoutEstimator(store DelegationStore) *TimeoutEstimator {
	return &TimeoutEstimator{
		store: store,
		defaultTimeouts: map[TaskCategory]time.Duration{
			CategoryResearch: 5 * time.Minute,
			CategoryCode:     10 * time.Minute,
			CategoryAnalysis: 7 * time.Minute,
			CategoryWriting:  8 * time.Minute,
			CategoryGeneral:  5 * time.Minute,
			CategoryCustom:   5 * time.Minute,
		},
		maxTimeout: 30 * time.Minute,
		minTimeout: 30 * time.Second,
	}
}

// SetDefaultTimeout sets the default timeout for a category
func (e *TimeoutEstimator) SetDefaultTimeout(category TaskCategory, duration time.Duration) {
	e.defaultTimeouts[category] = duration
}

// SetLimits sets the min and max timeout limits
func (e *TimeoutEstimator) SetLimits(min, max time.Duration) {
	e.minTimeout = min
	e.maxTimeout = max
}

// Estimate calculates an appropriate timeout for a task
func (e *TimeoutEstimator) Estimate(ctx context.Context, userID string, task string, category TaskCategory) TimeoutEstimate {
	// Get historical metrics
	metrics, err := e.store.GetTaskMetrics(ctx, userID, category, 20)
	if err != nil {
		logger.WarnCF("delegation", "Failed to get task metrics for timeout estimation", map[string]any{
			"error": err.Error(),
		})
		return e.defaultEstimate(category)
	}

	if len(metrics) < 3 {
		// Not enough data, use default
		return e.defaultEstimate(category)
	}

	// Calculate weighted average (more recent = higher weight)
	var totalWeight float64
	var weightedSum float64
	successCount := 0

	for i, m := range metrics {
		if m.Success {
			successCount++
		}
		weight := float64(i + 1) // Older metrics have lower weight
		weightedSum += float64(m.DurationMs) * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return e.defaultEstimate(category)
	}

	avgMs := weightedSum / totalWeight
	confidence := float64(successCount) / float64(len(metrics))

	// Apply confidence-based buffer
	// Lower confidence = more buffer time
	buffer := 1.5 - (confidence * 0.5) // 1.0 to 1.5x
	estimatedMs := avgMs * buffer

	duration := time.Duration(estimatedMs) * time.Millisecond

	// Clamp to limits
	if duration < e.minTimeout {
		duration = e.minTimeout
	}
	if duration > e.maxTimeout {
		duration = e.maxTimeout
	}

	logger.DebugCF("delegation", "Estimated timeout", map[string]any{
		"category":   category,
		"avg_ms":     avgMs,
		"buffer":     buffer,
		"confidence": confidence,
		"estimate":   duration.String(),
	})

	return TimeoutEstimate{
		Duration:   duration,
		Category:   category,
		Confidence: confidence,
	}
}

// defaultEstimate returns the default timeout for a category
func (e *TimeoutEstimator) defaultEstimate(category TaskCategory) TimeoutEstimate {
	duration, ok := e.defaultTimeouts[category]
	if !ok {
		duration = e.defaultTimeouts[CategoryGeneral]
	}

	return TimeoutEstimate{
		Duration:   duration,
		Category:   category,
		Confidence: 0.5, // Default confidence
	}
}

// ClassifyTask determines the category of a task based on its content
func ClassifyTask(task string) TaskCategory {
	task = strings.ToLower(task)

	// Code-related patterns
	codePatterns := []string{
		"write code", "implement", "debug", "fix bug", "refactor",
		"create function", "create class", "add method", "script",
		"program", "coding", "development", "build", "compile",
	}
	for _, p := range codePatterns {
		if strings.Contains(task, p) {
			return CategoryCode
		}
	}

	// Code-specific detection: code blocks or file extensions
	if strings.Contains(task, "```") ||
		regexp.MustCompile(`\.(go|py|js|ts|java|cpp|c|rs|rb)`).MatchString(task) {
		return CategoryCode
	}

	// Research-related patterns
	researchPatterns := []string{
		"research", "investigate", "explore", "find information",
		"look up", "search", "gather", "analyze data",
		"compare", "evaluate options", "survey",
	}
	for _, p := range researchPatterns {
		if strings.Contains(task, p) {
			return CategoryResearch
		}
	}

	// Analysis-related patterns
	analysisPatterns := []string{
		"analyze", "review", "examine", "assess", "audit",
		"evaluate", "check", "verify", "validate", "inspect",
		"summarize findings", "report on",
	}
	for _, p := range analysisPatterns {
		if strings.Contains(task, p) {
			return CategoryAnalysis
		}
	}

	// Writing-related patterns
	writingPatterns := []string{
		"write", "draft", "compose", "create document", "generate",
		"prepare report", "document", "article", "blog", "email",
		"message", "proposal", "specification",
	}
	for _, p := range writingPatterns {
		if strings.Contains(task, p) {
			return CategoryWriting
		}
	}

	return CategoryGeneral
}

// ExtractKeywords extracts keywords from a task for template matching
func ExtractKeywords(task string) []string {
	task = strings.ToLower(task)

	// Remove common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "shall": true, "can": true,
		"need": true, "to": true, "of": true, "in": true, "for": true,
		"on": true, "with": true, "at": true, "by": true, "from": true,
		"as": true, "into": true, "through": true, "during": true,
		"before": true, "after": true, "above": true, "below": true,
		"between": true, "under": true, "again": true, "further": true,
		"then": true, "once": true, "here": true, "there": true,
		"when": true, "where": true, "why": true, "how": true, "all": true,
		"each": true, "few": true, "more": true, "most": true, "other": true,
		"some": true, "such": true, "no": true, "nor": true, "not": true,
		"only": true, "own": true, "same": true, "so": true, "than": true,
		"too": true, "very": true, "just": true, "and": true, "but": true,
		"if": true, "or": true, "because": true, "until": true, "while": true,
		"please": true, "help": true, "me": true, "i": true, "you": true,
		"we": true, "they": true, "it": true, "this": true, "that": true,
		"these": true, "those": true, "what": true, "which": true, "who": true,
	}

	// Extract words
	words := regexp.MustCompile(`[a-z]+`).FindAllString(task, -1)

	var keywords []string
	seen := make(map[string]bool)

	for _, word := range words {
		if len(word) < 3 || stopWords[word] || seen[word] {
			continue
		}
		seen[word] = true
		keywords = append(keywords, word)
	}

	// Also extract specific technical terms
	techPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\b(api|sdk|cli|http|json|yaml|xml|sql|git|docker)\b`),
		regexp.MustCompile(`\b(golang|python|javascript|typescript|rust|java)\b`),
		regexp.MustCompile(`\b(test|debug|deploy|build|compile|lint)\b`),
		regexp.MustCompile(`\b(database|server|client|backend|frontend)\b`),
		regexp.MustCompile(`\b(auth|login|logout|token|session)\b`),
	}

	for _, pattern := range techPatterns {
		matches := pattern.FindAllString(task, -1)
		for _, m := range matches {
			if !seen[m] {
				seen[m] = true
				keywords = append(keywords, m)
			}
		}
	}

	return keywords
}

// EstimateWithHint estimates timeout considering a user-provided hint
func (e *TimeoutEstimator) EstimateWithHint(ctx context.Context, userID, task string, category TaskCategory, hintSeconds int) TimeoutEstimate {
	// If hint is provided and reasonable, use it
	if hintSeconds > 0 {
		hint := time.Duration(hintSeconds) * time.Second
		if hint >= e.minTimeout && hint <= e.maxTimeout {
			return TimeoutEstimate{
				Duration:   hint,
				Category:   category,
				Confidence: 1.0, // User hint = high confidence
			}
		}
	}

	return e.Estimate(ctx, userID, task, category)
}

// RecordMetric records a task execution metric for future estimates
func (e *TimeoutEstimator) RecordMetric(ctx context.Context, userID string, task string, duration time.Duration, success bool) error {
	category := ClassifyTask(task)

	// Truncate task hint
	taskHint := task
	if len(taskHint) > 100 {
		taskHint = taskHint[:100]
	}

	metric := &TaskMetricRecord{
		ID:           generateID("metric"),
		UserID:       userID,
		TaskCategory: category,
		TaskHint:     taskHint,
		DurationMs:   duration.Milliseconds(),
		Success:      success,
		CreatedAt:    time.Now().UnixMilli(),
	}

	return e.store.SaveTaskMetric(ctx, metric)
}
