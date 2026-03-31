// MoonHub - Your ready-to-use AI assistant
// Self-Improving Behavioral Pattern Detection System
// License: MIT

package learning

import (
	"sort"
)

// Suggester generates proactive suggestions based on detected patterns
type Suggester struct {
	config Config
}

// NewSuggester creates a new suggester
func NewSuggester(config Config) *Suggester {
	return &Suggester{
		config: config,
	}
}

// GenerateSuggestions generates proactive suggestions from patterns and tool usage
func (s *Suggester) GenerateSuggestions(
	patterns []EnhancedPattern,
	toolUsage []ToolUsagePattern,
	score BehavioralScore,
) []Suggestion {
	var suggestions []Suggestion

	// 1. Tool optimization suggestions
	suggestions = append(suggestions, s.generateToolSuggestions(toolUsage)...)

	// 2. Workflow optimization suggestions
	suggestions = append(suggestions, s.generateWorkflowSuggestions(patterns)...)

	// 3. Behavioral improvement suggestions
	suggestions = append(suggestions, s.generateBehavioralSuggestions(score, patterns)...)

	// 4. Preference consolidation suggestions
	suggestions = append(suggestions, s.generatePreferenceSuggestions(patterns)...)

	// Sort by impact and confidence
	sort.Slice(suggestions, func(i, j int) bool {
		impactOrder := map[string]int{"high": 3, "medium": 2, "low": 1}
		impactDiff := impactOrder[suggestions[i].Impact] - impactOrder[suggestions[j].Impact]
		if impactDiff != 0 {
			return impactDiff > 0
		}
		return suggestions[i].Confidence > suggestions[j].Confidence
	})

	// Return top suggestions
	maxSuggestions := 5
	if len(suggestions) < maxSuggestions {
		maxSuggestions = len(suggestions)
	}

	return suggestions[:maxSuggestions]
}

// generateToolSuggestions generates suggestions based on tool usage patterns
func (s *Suggester) generateToolSuggestions(toolUsage []ToolUsagePattern) []Suggestion {
	var suggestions []Suggestion

	for _, tool := range toolUsage {
		// High failure rate
		if tool.TotalCalls >= 5 && tool.SuccessfulCalls < tool.TotalCalls/2 {
			suggestions = append(suggestions, Suggestion{
				ID:          generateID(),
				Type:        "tool_usage",
				Title:       "Review " + tool.ToolName + " usage",
				Description: "This tool has a low success rate. Consider using an alternative approach.",
				Confidence:  0.8,
				Impact:      "high",
				BasedOn:     []string{},
				Evidence:    "Low success rate detected",
				CreatedAt:   NowMs(),
				Dismissed:   false,
				Applied:     false,
			})
		}

		// High rejection rate
		if tool.TotalCalls >= 3 && float64(tool.UserRejectedCalls)/float64(tool.TotalCalls) > 0.3 {
			suggestions = append(suggestions, Suggestion{
				ID:          generateID(),
				Type:        "tool_usage",
				Title:       "User preference: " + tool.ToolName,
				Description: "You often reject results from this tool. Should I use it less frequently?",
				Confidence:  0.7,
				Impact:      "medium",
				BasedOn:     []string{},
				Evidence:    "High rejection rate detected",
				CreatedAt:   NowMs(),
				Dismissed:   false,
				Applied:     false,
			})
		}

		// Declining usage trend
		if tool.UsageTrend == TrendDeclining && tool.TotalCalls >= 5 {
			suggestions = append(suggestions, Suggestion{
				ID:          generateID(),
				Type:        "tool_usage",
				Title:       "Tool usage declining: " + tool.ToolName,
				Description: "Usage of this tool has been declining. Consider if it's still needed.",
				Confidence:  0.6,
				Impact:      "low",
				BasedOn:     []string{},
				Evidence:    "Declining usage trend",
				CreatedAt:   NowMs(),
				Dismissed:   false,
				Applied:     false,
			})
		}
	}

	return suggestions
}

// generateWorkflowSuggestions generates suggestions based on workflow patterns
func (s *Suggester) generateWorkflowSuggestions(patterns []EnhancedPattern) []Suggestion {
	var suggestions []Suggestion

	// Find workflow patterns with high support
	for _, pattern := range patterns {
		if pattern.Category == CategoryPreferenceWorkflow && pattern.Support >= 3 && pattern.Confidence > 0.7 {
			suggestions = append(suggestions, Suggestion{
				ID:          generateID(),
				Type:        "workflow",
				Title:       "Workflow pattern detected",
				Description: "I've noticed you often " + pattern.Pattern + ". Should I make this a default behavior?",
				Confidence:  pattern.Confidence,
				Impact:      "medium",
				BasedOn:     []string{pattern.ID},
				Evidence:    "Observed multiple times",
				CreatedAt:   NowMs(),
				Dismissed:   false,
				Applied:     false,
			})
		}
	}

	return suggestions
}

// generateBehavioralSuggestions generates suggestions based on behavioral scores
func (s *Suggester) generateBehavioralSuggestions(score BehavioralScore, patterns []EnhancedPattern) []Suggestion {
	var suggestions []Suggestion

	// Low adaptation speed
	if score.Dimensions.AdaptationSpeed < 0.5 {
		suggestions = append(suggestions, Suggestion{
			ID:          generateID(),
			Type:        "optimization",
			Title:       "Improve adaptation",
			Description: "I seem to be slow to adapt to your corrections. I'll try to be more responsive to your feedback.",
			Confidence:  0.6,
			Impact:      "medium",
			BasedOn:     []string{},
			Evidence:    "Low adaptation speed score",
			CreatedAt:   NowMs(),
			Dismissed:   false,
			Applied:     false,
		})
	}

	// Declining response quality
	if score.Trends.ResponseQuality == TrendDeclining {
		suggestions = append(suggestions, Suggestion{
			ID:          generateID(),
			Type:        "optimization",
			Title:       "Response quality declining",
			Description: "My response quality seems to be declining recently. Let me know if there's something specific I should focus on.",
			Confidence:  0.5,
			Impact:      "high",
			BasedOn:     []string{},
			Evidence:    "Based on recent feedback patterns",
			CreatedAt:   NowMs(),
			Dismissed:   false,
			Applied:     false,
		})
	}

	// High correction rate
	correctionCount := 0
	for _, p := range patterns {
		if p.Category == CategoryCorrectionExplicit || p.Category == CategoryCorrectionImplicit {
			correctionCount++
		}
	}
	if correctionCount > 5 {
		suggestions = append(suggestions, Suggestion{
			ID:          generateID(),
			Type:        "optimization",
			Title:       "Frequent corrections detected",
			Description: "You've been correcting me frequently. Would you like to provide more specific preferences to help me understand you better?",
			Confidence:  0.6,
			Impact:      "high",
			BasedOn:     []string{},
			Evidence:    "Multiple corrections in recent history",
			CreatedAt:   NowMs(),
			Dismissed:   false,
			Applied:     false,
		})
	}

	return suggestions
}

// generatePreferenceSuggestions generates suggestions for consolidating preferences
func (s *Suggester) generatePreferenceSuggestions(patterns []EnhancedPattern) []Suggestion {
	var suggestions []Suggestion

	// Find strong preferences that aren't yet formalized
	for _, pattern := range patterns {
		if (pattern.Category == CategoryPreferenceCommunication ||
			pattern.Category == CategoryPreferenceTool ||
			pattern.Category == CategoryPreferenceWorkflow) &&
			pattern.Confidence > 0.8 && pattern.Support >= 3 {
			suggestions = append(suggestions, Suggestion{
				ID:          generateID(),
				Type:        "preference",
				Title:       "Confirm preference",
				Description: "Should I always follow this preference: \"" + pattern.Pattern + "\"?",
				Confidence:  pattern.Confidence,
				Impact:      "low",
				BasedOn:     []string{pattern.ID},
				Evidence:    "Consistently observed",
				CreatedAt:   NowMs(),
				Dismissed:   false,
				Applied:     false,
			})
		}
	}

	return suggestions
}

// DismissSuggestion marks a suggestion as dismissed
func (s *Suggester) DismissSuggestion(suggestions []Suggestion, id string) []Suggestion {
	for i := range suggestions {
		if suggestions[i].ID == id {
			suggestions[i].Dismissed = true
			break
		}
	}
	return suggestions
}

// ApplySuggestion marks a suggestion as applied
func (s *Suggester) ApplySuggestion(suggestions []Suggestion, id string) []Suggestion {
	for i := range suggestions {
		if suggestions[i].ID == id {
			suggestions[i].Applied = true
			break
		}
	}
	return suggestions
}

// GetActiveSuggestions returns only non-dismissed, non-applied suggestions
func (s *Suggester) GetActiveSuggestions(suggestions []Suggestion) []Suggestion {
	var active []Suggestion
	for _, sug := range suggestions {
		if !sug.Dismissed && !sug.Applied {
			active = append(active, sug)
		}
	}
	return active
}

// GetHighImpactSuggestions returns only high-impact suggestions
func (s *Suggester) GetHighImpactSuggestions(suggestions []Suggestion) []Suggestion {
	var high []Suggestion
	for _, sug := range suggestions {
		if sug.Impact == "high" && !sug.Dismissed && !sug.Applied {
			high = append(high, sug)
		}
	}
	return high
}
