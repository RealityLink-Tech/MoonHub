// MoonHub - Your ready-to-use AI assistant
// Self-Improving Behavioral Pattern Detection System
// License: MIT

package learning

import (
	"math"
	"strings"
)

// PatternEvolver handles pattern evolution: decay, merge, prune, and generalize
type PatternEvolver struct {
	config Config
}

// NewPatternEvolver creates a new pattern evolver
func NewPatternEvolver(config Config) *PatternEvolver {
	return &PatternEvolver{
		config: config,
	}
}

// Evolve runs all evolution operations on patterns
func (e *PatternEvolver) Evolve(patterns []EnhancedPattern) ([]EnhancedPattern, EvolutionResult) {
	result := EvolutionResult{}

	// Step 1: Decay confidence based on time
	patterns = e.Decay(patterns)
	result.PatternsDecayed = len(patterns)

	// Step 2: Merge similar patterns
	var merged []string
	patterns, merged = e.Merge(patterns)
	result.PatternsMerged = len(merged)

	// Step 3: Generalize patterns
	var generalized int
	patterns, generalized = e.Generalize(patterns)
	result.PatternsGeneralized = generalized

	// Step 4: Prune low-value patterns
	pruned := e.Prune(patterns)
	result.PatternsPruned = len(pruned)

	// Remove pruned patterns
	patterns = e.removePatterns(patterns, pruned)

	return patterns, result
}

// Decay applies temporal decay to pattern confidence
func (e *PatternEvolver) Decay(patterns []EnhancedPattern) []EnhancedPattern {
	now := NowMs()

	for i := range patterns {
		daysSinceUpdate := DaysSince(patterns[i].LastObserved)

		// Ebbinghaus-inspired decay
		decayRate := patterns[i].DecayRate
		if decayRate == 0 {
			decayRate = DefaultDecayRate
		}

		decayFactor := math.Exp(-decayRate * daysSinceUpdate)

		// Increase contradiction weight over time
		contradictionPenalty := 0.0
		if patterns[i].Contradiction > 0 {
			contradictionPenalty = math.Min(0.3, float64(patterns[i].Contradiction)*0.05)
		}

		patterns[i].Confidence = math.Max(0.1, patterns[i].Confidence*decayFactor-contradictionPenalty)
		patterns[i].LastDecayAt = now
	}

	return patterns
}

// Merge combines similar patterns into generalized patterns
func (e *PatternEvolver) Merge(patterns []EnhancedPattern) ([]EnhancedPattern, []string) {
	var merged []EnhancedPattern
	var mergedIDs []string
	processed := make(map[string]bool)

	threshold := e.config.MergeSimilarityThreshold
	if threshold == 0 {
		threshold = 0.8
	}

	for i := range patterns {
		if processed[patterns[i].ID] {
			continue
		}

		var similar []EnhancedPattern
		similar = append(similar, patterns[i])

		for j := range patterns {
			if i == j || processed[patterns[j].ID] {
				continue
			}

			similarity := e.calculateSimilarity(patterns[i].Pattern, patterns[j].Pattern)

			if similarity > threshold && patterns[i].Category == patterns[j].Category {
				similar = append(similar, patterns[j])
			}
		}

		if len(similar) > 1 {
			// Merge similar patterns
			mergedPattern := e.createMergedPattern(similar)
			merged = append(merged, mergedPattern)

			for _, p := range similar {
				processed[p.ID] = true
				mergedIDs = append(mergedIDs, p.ID)
			}
		}
	}

	// Add non-merged patterns
	for _, p := range patterns {
		if !processed[p.ID] {
			merged = append(merged, p)
		}
	}

	return merged, mergedIDs
}

// Prune removes low-value patterns
func (e *PatternEvolver) Prune(patterns []EnhancedPattern) []string {
	var pruned []string
	now := NowMs()
	thirtyDays := int64(e.config.PruneOlderThanDays) * MsPerDay
	if thirtyDays == 0 {
		thirtyDays = 30 * MsPerDay
	}

	for _, p := range patterns {
		// Prune if: low confidence + few observations + old
		if p.Confidence < 0.15 &&
			p.ObservationCount < 2 &&
			(now-p.LastObserved) > thirtyDays {
			pruned = append(pruned, p.ID)
		}
	}

	return pruned
}

// Generalize creates generalized patterns from related specific patterns
func (e *PatternEvolver) Generalize(patterns []EnhancedPattern) ([]EnhancedPattern, int) {
	var generalized []EnhancedPattern
	generalizedCount := 0
	processed := make(map[string]bool)

	// Group patterns by category
	byCategory := make(map[PatternCategory][]EnhancedPattern)
	for _, p := range patterns {
		byCategory[p.Category] = append(byCategory[p.Category], p)
	}

	// Try to generalize within each category
	for _, categoryPatterns := range byCategory {
		if len(categoryPatterns) < 2 {
			continue
		}

		// Find common words/themes
		commonTheme := e.extractCommonTheme(categoryPatterns)
		if commonTheme == "" {
			continue
		}

		// Check if we already have a pattern with this theme
		alreadyExists := false
		for _, p := range patterns {
			if p.Category == CategoryBehavioralTrend && strings.Contains(p.Pattern, commonTheme) {
				alreadyExists = true
				break
			}
		}

		if alreadyExists {
			continue
		}

		// Create generalized pattern
		generalizedPattern := EnhancedPattern{
			ID:               generateID(),
			Category:         CategoryBehavioralTrend,
			Source:           SourceCrossSession,
			Pattern:          "General pattern: " + commonTheme,
			Confidence:       e.calculateMinConfidence(categoryPatterns) * 0.9,
			Support:          e.calculateTotalSupport(categoryPatterns),
			Contradiction:    e.calculateTotalContradiction(categoryPatterns),
			FirstObserved:    e.calculateMinFirstObserved(categoryPatterns),
			LastObserved:     NowMs(),
			ObservationCount: e.calculateTotalObservations(categoryPatterns),
			Triggers:         e.mergeTriggers(categoryPatterns),
			Examples:         e.mergeExamples(categoryPatterns),
			RelatedPatterns:  e.collectIDs(categoryPatterns),
			DecayRate:        DefaultDecayRate * 0.6, // Slower decay for generalized patterns
			LastDecayAt:      NowMs(),
			Metadata: map[string]any{
				"generalized_from": e.collectIDs(categoryPatterns),
			},
		}

		generalized = append(generalized, generalizedPattern)
		generalizedCount++

		for _, p := range categoryPatterns {
			processed[p.ID] = true
		}
	}

	// Add non-generalized patterns
	for _, p := range patterns {
		if !processed[p.ID] {
			generalized = append(generalized, p)
		}
	}

	return generalized, generalizedCount
}

// Helper functions

func (e *PatternEvolver) calculateSimilarity(a, b string) float64 {
	// Simple Jaccard similarity
	wordsA := strings.Fields(strings.ToLower(a))
	wordsB := strings.Fields(strings.ToLower(b))

	setA := make(map[string]bool)
	for _, w := range wordsA {
		setA[w] = true
	}

	setB := make(map[string]bool)
	for _, w := range wordsB {
		setB[w] = true
	}

	intersection := 0
	for w := range setA {
		if setB[w] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func (e *PatternEvolver) createMergedPattern(patterns []EnhancedPattern) EnhancedPattern {
	// Find the strongest pattern as base
	base := patterns[0]
	for _, p := range patterns {
		if p.Confidence > base.Confidence {
			base = p
		}
	}

	// Merge confidence (weighted average)
	var totalWeight float64
	var weightedConfidence float64
	for _, p := range patterns {
		weight := float64(p.ObservationCount) + 1
		weightedConfidence += p.Confidence * weight
		totalWeight += weight
	}

	mergedConfidence := weightedConfidence / totalWeight

	// Merge examples
	examples := e.mergeExamples(patterns)
	if len(examples) > e.config.MaxExamples {
		examples = examples[:e.config.MaxExamples]
	}

	return EnhancedPattern{
		ID:               generateID(),
		Category:         base.Category,
		Source:           base.Source,
		Pattern:          base.Pattern,
		Confidence:       mergedConfidence,
		Support:          e.calculateTotalSupport(patterns),
		Contradiction:    e.calculateTotalContradiction(patterns),
		FirstObserved:    e.calculateMinFirstObserved(patterns),
		LastObserved:     NowMs(),
		ObservationCount: e.calculateTotalObservations(patterns),
		Triggers:         e.mergeTriggers(patterns),
		Examples:         examples,
		RelatedPatterns:  e.collectIDs(patterns),
		Supersedes:       e.collectIDs(patterns),
		DecayRate:        base.DecayRate,
		LastDecayAt:      NowMs(),
		Metadata: map[string]any{
			"merged_from": e.collectIDs(patterns),
		},
	}
}

func (e *PatternEvolver) extractCommonTheme(patterns []EnhancedPattern) string {
	// Extract common words (appearing in at least 50% of patterns)
	wordFreq := make(map[string]int)
	fillerWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "can": true, "to": true, "of": true,
		"in": true, "for": true, "on": true, "with": true, "at": true,
		"by": true, "from": true, "i": true, "you": true, "it": true,
	}

	for _, p := range patterns {
		words := strings.Fields(strings.ToLower(p.Pattern))
		seen := make(map[string]bool)
		for _, w := range words {
			w = strings.Trim(w, ".,!?;:\"'")
			if len(w) > 2 && !fillerWords[w] && !seen[w] {
				wordFreq[w]++
				seen[w] = true
			}
		}
	}

	threshold := len(patterns) / 2
	if threshold < 1 {
		threshold = 1
	}

	var commonWords []string
	for word, count := range wordFreq {
		if count >= threshold {
			commonWords = append(commonWords, word)
		}
	}

	if len(commonWords) == 0 {
		return ""
	}

	return strings.Join(commonWords, " ")
}

func (e *PatternEvolver) removePatterns(patterns []EnhancedPattern, ids []string) []EnhancedPattern {
	removeSet := make(map[string]bool)
	for _, id := range ids {
		removeSet[id] = true
	}

	var result []EnhancedPattern
	for _, p := range patterns {
		if !removeSet[p.ID] {
			result = append(result, p)
		}
	}

	return result
}

func (e *PatternEvolver) calculateMinConfidence(patterns []EnhancedPattern) float64 {
	if len(patterns) == 0 {
		return 0
	}
	min := patterns[0].Confidence
	for _, p := range patterns {
		if p.Confidence < min {
			min = p.Confidence
		}
	}
	return min
}

func (e *PatternEvolver) calculateTotalSupport(patterns []EnhancedPattern) int {
	total := 0
	for _, p := range patterns {
		total += p.Support
	}
	return total
}

func (e *PatternEvolver) calculateTotalContradiction(patterns []EnhancedPattern) int {
	total := 0
	for _, p := range patterns {
		total += p.Contradiction
	}
	return total
}

func (e *PatternEvolver) calculateMinFirstObserved(patterns []EnhancedPattern) int64 {
	if len(patterns) == 0 {
		return 0
	}
	min := patterns[0].FirstObserved
	for _, p := range patterns {
		if p.FirstObserved < min {
			min = p.FirstObserved
		}
	}
	return min
}

func (e *PatternEvolver) calculateTotalObservations(patterns []EnhancedPattern) int {
	total := 0
	for _, p := range patterns {
		total += p.ObservationCount
	}
	return total
}

func (e *PatternEvolver) mergeTriggers(patterns []EnhancedPattern) []string {
	triggerSet := make(map[string]bool)
	for _, p := range patterns {
		for _, t := range p.Triggers {
			triggerSet[t] = true
		}
	}
	var result []string
	for t := range triggerSet {
		result = append(result, t)
	}
	return result
}

func (e *PatternEvolver) mergeExamples(patterns []EnhancedPattern) []PatternExample {
	var result []PatternExample
	for _, p := range patterns {
		result = append(result, p.Examples...)
	}
	// Sort by timestamp (newest first) and limit
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

func (e *PatternEvolver) collectIDs(patterns []EnhancedPattern) []string {
	ids := make([]string, len(patterns))
	for i, p := range patterns {
		ids[i] = p.ID
	}
	return ids
}

// generateID generates a unique ID for a pattern
func generateID() string {
	return randomString(16)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
