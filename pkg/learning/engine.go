// MoonHub - Your ready-to-use AI assistant
// Self-Improving Behavioral Pattern Detection System
// License: MIT

package learning

import (
	"fmt"
	"strings"
	"sync"
)

// Engine is the main learning engine that coordinates all components
type Engine struct {
	config    Config
	detector  *PatternDetector
	scorer    *BehavioralScorer
	evolver   *PatternEvolver
	suggester *Suggester
	tracker   *ToolTracker
	store     *Store

	mu              sync.RWMutex
	patterns        []EnhancedPattern
	toolUsage       []ToolUsagePattern
	behavioralScore BehavioralScore
	suggestions     []Suggestion
}

// NewEngine creates a new learning engine
func NewEngine(config Config) (*Engine, error) {
	// Initialize store
	store, err := NewStore(config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize store: %w", err)
	}

	engine := &Engine{
		config:    config,
		detector:  NewPatternDetector(config),
		scorer:    NewBehavioralScorer(config),
		evolver:   NewPatternEvolver(config),
		suggester: NewSuggester(config),
		tracker:   NewToolTracker(config),
		store:     store,
	}

	// Load existing data
	if err := engine.load(); err != nil {
		// Non-fatal - start fresh
		fmt.Printf("Warning: failed to load learning data: %v\n", err)
	}

	// Initialize behavioral score
	engine.behavioralScore = engine.scorer.CalculateScore(engine.patterns, engine.toolUsage)

	return engine, nil
}

// load loads patterns and tool usage from store
func (e *Engine) load() error {
	patterns, err := e.store.GetAllPatterns()
	if err != nil {
		return err
	}
	e.patterns = patterns

	toolUsage, err := e.store.GetAllToolUsage()
	if err != nil {
		return err
	}
	e.toolUsage = toolUsage

	return nil
}

// Analyze analyzes a conversation context and extracts patterns
func (e *Engine) Analyze(ctx AnalysisContext) AnalysisResult {
	result := AnalysisResult{}

	// Detect signals
	detectionCtx := DetectionContext{
		UserMessage:         "",
		AssistantMessage:    "",
		History:             ctx.MessageHistory,
		ToolCalls:           ctx.RecentToolCalls,
		PreviousToolResults: ctx.RecentToolResults,
	}

	// Get last user and assistant messages
	for i := len(ctx.MessageHistory) - 1; i >= 0; i-- {
		if ctx.MessageHistory[i].Role == "user" && detectionCtx.UserMessage == "" {
			detectionCtx.UserMessage = ctx.MessageHistory[i].Content
		}
		if ctx.MessageHistory[i].Role == "assistant" && detectionCtx.AssistantMessage == "" {
			detectionCtx.AssistantMessage = ctx.MessageHistory[i].Content
		}
		if detectionCtx.UserMessage != "" && detectionCtx.AssistantMessage != "" {
			break
		}
	}

	// Detect all signals
	signals := e.detector.DetectSignals(detectionCtx)

	// Filter signals by confidence
	var filteredSignals []DetectedSignal
	for _, signal := range signals {
		if signal.Confidence >= e.config.MinConfidence {
			filteredSignals = append(filteredSignals, signal)
		}
	}
	result.Signals = filteredSignals

	// Convert signals to patterns
	for _, signal := range filteredSignals {
		pattern := e.signalToPattern(signal)
		result.NewPatterns = append(result.NewPatterns, pattern)
	}

	// Store new patterns
	e.mu.Lock()
	for _, pattern := range result.NewPatterns {
		e.patterns = append(e.patterns, pattern)
		e.store.SavePattern(pattern)
	}
	e.mu.Unlock()

	// Update behavioral score
	if e.config.EnableBehavioralScoring {
		for _, signal := range filteredSignals {
			e.mu.Lock()
			e.behavioralScore = e.scorer.UpdateBehavioralScore(e.behavioralScore, signal)
			e.mu.Unlock()
		}
	}

	// Generate suggestions if enabled
	if e.config.EnableSuggestions {
		result.Suggestions = e.suggester.GenerateSuggestions(e.patterns, e.toolUsage, e.behavioralScore)
	}

	return result
}

// signalToPattern converts a detected signal to an enhanced pattern
func (e *Engine) signalToPattern(signal DetectedSignal) EnhancedPattern {
	now := NowMs()
	return EnhancedPattern{
		ID:               generateID(),
		Category:         signal.Category,
		Source:           signal.Source,
		Pattern:          signal.ExtractedPattern,
		Confidence:       signal.Confidence,
		Support:          1,
		Contradiction:    0,
		FirstObserved:    now,
		LastObserved:     now,
		ObservationCount: 1,
		Triggers:         []string{},
		Examples: []PatternExample{
			{
				Timestamp:  now,
				Outcome:    signal.Context,
				Confidence: signal.Confidence,
			},
		},
		RelatedPatterns: []string{},
		Supersedes:      []string{},
		DecayRate:       DefaultDecayRate,
		LastDecayAt:     now,
		Metadata:        map[string]any{},
	}
}

// RecordToolExecution records a tool execution for tracking
func (e *Engine) RecordToolExecution(record ToolExecutionRecord) {
	e.tracker.RecordExecution(record)

	// Update local cache
	e.mu.Lock()
	e.toolUsage = e.tracker.GetAllToolUsage()
	e.mu.Unlock()

	// Persist
	pattern := e.tracker.GetToolUsage(record.ToolName)
	if pattern != nil {
		e.store.SaveToolUsage(*pattern)
	}
}

// RecordToolRejection records when a user rejects a tool result
func (e *Engine) RecordToolRejection(toolName string) {
	e.tracker.RecordUserRejection(toolName)

	e.mu.Lock()
	e.toolUsage = e.tracker.GetAllToolUsage()
	e.mu.Unlock()

	pattern := e.tracker.GetToolUsage(toolName)
	if pattern != nil {
		e.store.SaveToolUsage(*pattern)
	}
}

// RecordToolAcceptance records when a user accepts a tool result
func (e *Engine) RecordToolAcceptance(toolName string) {
	e.tracker.RecordUserAcceptance(toolName)

	e.mu.Lock()
	e.toolUsage = e.tracker.GetAllToolUsage()
	e.mu.Unlock()

	pattern := e.tracker.GetToolUsage(toolName)
	if pattern != nil {
		e.store.SaveToolUsage(*pattern)
	}
}

// Evolve runs pattern evolution (decay, merge, prune)
func (e *Engine) Evolve() EvolutionResult {
	if !e.config.EnablePatternEvolution {
		return EvolutionResult{}
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	var result EvolutionResult
	e.patterns, result = e.evolver.Evolve(e.patterns)

	// Persist changes
	for _, pattern := range e.patterns {
		e.store.SavePattern(pattern)
	}

	// Update behavioral score after evolution
	e.behavioralScore = e.scorer.CalculateScore(e.patterns, e.toolUsage)

	return result
}

// GetContext returns the learned context for injection into prompts
func (e *Engine) GetContext() EnhancedLearnedContext {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ctx := EnhancedLearnedContext{}

	// Filter high-confidence patterns
	var highConfidence []EnhancedPattern
	for _, p := range e.patterns {
		if p.Confidence >= e.config.MinConfidence {
			highConfidence = append(highConfidence, p)
		}
	}

	// Build preferences section
	var preferences []string
	var toolPrefs []string
	var workflows []string

	for _, p := range highConfidence {
		switch p.Category {
		case CategoryPreferenceCommunication, CategoryPreferenceTool:
			preferences = append(preferences, "- "+p.Pattern)
			if p.Category == CategoryPreferenceTool {
				toolPrefs = append(toolPrefs, "- "+p.Pattern)
			}
		case CategoryPreferenceWorkflow:
			workflows = append(workflows, "- "+p.Pattern)
		}
	}

	if len(preferences) > 0 {
		ctx.Preferences = strings.Join(preferences, "\n")
	}
	if len(toolPrefs) > 0 {
		ctx.ToolPreferences = strings.Join(toolPrefs, "\n")
	}
	if len(workflows) > 0 {
		ctx.WorkflowPatterns = strings.Join(workflows, "\n")
	}

	// Build patterns section (what works)
	var successPatterns []string
	for _, p := range highConfidence {
		if p.Category == CategorySuccessIndicator {
			successPatterns = append(successPatterns, "- Works well: "+p.Pattern)
		}
	}
	if len(successPatterns) > 0 {
		ctx.Patterns = strings.Join(successPatterns, "\n")
	}

	// Build recent corrections section
	var corrections []EnhancedPattern
	for _, p := range e.patterns {
		if p.Category == CategoryCorrectionExplicit || p.Category == CategoryCorrectionImplicit {
			corrections = append(corrections, p)
		}
	}
	// Sort by last observed (newest first) and take last 5
	if len(corrections) > 5 {
		corrections = corrections[:5]
	}
	var correctionStrs []string
	for _, p := range corrections {
		correctionStrs = append(correctionStrs, "- "+p.Pattern)
	}
	if len(correctionStrs) > 0 {
		ctx.RecentCorrections = strings.Join(correctionStrs, "\n")
	}

	// Build behavioral insights
	if e.config.EnableBehavioralScoring {
		ctx.BehavioralInsights = fmt.Sprintf(
			"Overall performance: %.0f%% | Response quality: %.0f%% | Tool efficiency: %.0f%%",
			e.behavioralScore.Overall*100,
			e.behavioralScore.Dimensions.ResponseQuality*100,
			e.behavioralScore.Dimensions.ToolEfficiency*100,
		)
	}

	// Add proactive notes from high-impact suggestions
	if e.config.EnableSuggestions {
		var activeSuggestions []Suggestion
		for _, s := range e.suggestions {
			if !s.Dismissed && !s.Applied && s.Impact == "high" {
				activeSuggestions = append(activeSuggestions, s)
			}
		}
		if len(activeSuggestions) > 0 {
			var notes []string
			for _, s := range activeSuggestions[:2] { // Top 2 high-impact suggestions
				notes = append(notes, "- "+s.Title)
			}
			ctx.ProactiveNotes = strings.Join(notes, "\n")
		}
	}

	return ctx
}

// GetContextString returns the context as a formatted string for prompt injection
func (e *Engine) GetContextString() string {
	ctx := e.GetContext()
	var parts []string

	if ctx.Preferences != "" {
		parts = append(parts, "## Learned Preferences\n\n"+ctx.Preferences)
	}
	if ctx.ToolPreferences != "" {
		parts = append(parts, "## Tool Preferences\n\n"+ctx.ToolPreferences)
	}
	if ctx.WorkflowPatterns != "" {
		parts = append(parts, "## Workflow Patterns\n\n"+ctx.WorkflowPatterns)
	}
	if ctx.Patterns != "" {
		parts = append(parts, "## What Works\n\n"+ctx.Patterns)
	}
	if ctx.RecentCorrections != "" {
		parts = append(parts, "## Recent Corrections\n\n"+ctx.RecentCorrections)
	}
	if ctx.BehavioralInsights != "" {
		parts = append(parts, "## Behavioral Insights\n\n"+ctx.BehavioralInsights)
	}
	if ctx.ProactiveNotes != "" {
		parts = append(parts, "## Notes\n\n"+ctx.ProactiveNotes)
	}

	if len(parts) == 0 {
		return ""
	}

	return "\n\n---\n\n# Learning Context\n\n" + strings.Join(parts, "\n\n")
}

// GetBehavioralScore returns the current behavioral score
func (e *Engine) GetBehavioralScore() BehavioralScore {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.behavioralScore
}

// GetAllPatterns returns all patterns
func (e *Engine) GetAllPatterns() []EnhancedPattern {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.patterns == nil {
		return []EnhancedPattern{}
	}
	return e.patterns
}

// GetPatternsByCategory returns patterns filtered by category
func (e *Engine) GetPatternsByCategory(category PatternCategory) []EnhancedPattern {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := []EnhancedPattern{}
	for _, p := range e.patterns {
		if p.Category == category {
			result = append(result, p)
		}
	}
	return result
}

// GetToolUsagePatterns returns all tool usage patterns
func (e *Engine) GetToolUsagePatterns() []ToolUsagePattern {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.toolUsage == nil {
		return []ToolUsagePattern{}
	}
	return e.toolUsage
}

// GetSuggestions returns current suggestions
func (e *Engine) GetSuggestions() []Suggestion {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.suggester.GetActiveSuggestions(e.suggestions)
}

// DismissSuggestion dismisses a suggestion
func (e *Engine) DismissSuggestion(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range e.suggestions {
		if e.suggestions[i].ID == id {
			e.suggestions[i].Dismissed = true
			e.store.SaveSuggestion(e.suggestions[i])
			return true
		}
	}
	return false
}

// ApplySuggestion marks a suggestion as applied
func (e *Engine) ApplySuggestion(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range e.suggestions {
		if e.suggestions[i].ID == id {
			e.suggestions[i].Applied = true
			e.store.SaveSuggestion(e.suggestions[i])
			return true
		}
	}
	return false
}

// GetStats returns statistics about the learning engine
func (e *Engine) GetStats() Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := Stats{
		TotalPatterns:      len(e.patterns),
		PatternsByCategory: make(map[PatternCategory]int),
		BehavioralScore:    e.behavioralScore,
		ToolUsagePatterns:  len(e.toolUsage),
	}

	var totalConfidence float64
	highConfidence := 0

	for _, p := range e.patterns {
		stats.PatternsByCategory[p.Category]++
		totalConfidence += p.Confidence
		if p.Confidence >= e.config.MinConfidence {
			highConfidence++
		}
	}

	stats.HighConfidencePatterns = highConfidence
	if len(e.patterns) > 0 {
		stats.AvgConfidence = totalConfidence / float64(len(e.patterns))
	}

	stats.PendingSuggestions = len(e.suggester.GetActiveSuggestions(e.suggestions))

	return stats
}

// SearchPatterns searches for patterns matching a query
func (e *Engine) SearchPatterns(query string, limit int) ([]EnhancedPattern, error) {
	return e.store.SearchPatterns(query, limit)
}

// DeletePattern deletes a pattern by ID
func (e *Engine) DeletePattern(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.store.DeletePattern(id); err != nil {
		return err
	}

	// Remove from local cache
	var filtered []EnhancedPattern
	for _, p := range e.patterns {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}
	e.patterns = filtered

	return nil
}

// Close closes the learning engine and releases resources
func (e *Engine) Close() error {
	return e.store.Close()
}

// RefreshSuggestions regenerates suggestions based on current patterns
func (e *Engine) RefreshSuggestions() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.config.EnableSuggestions {
		e.suggestions = e.suggester.GenerateSuggestions(e.patterns, e.toolUsage, e.behavioralScore)

		// Persist new suggestions
		for _, s := range e.suggestions {
			e.store.SaveSuggestion(s)
		}
	}
}

// UpdateTrend updates tool usage trends
func (e *Engine) UpdateTrend() {
	e.tracker.UpdateTrends()

	e.mu.Lock()
	e.toolUsage = e.tracker.GetAllToolUsage()
	e.mu.Unlock()
}
