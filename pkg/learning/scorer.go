// MoonHub - Your ready-to-use AI assistant
// Self-Improving Behavioral Pattern Detection System
// License: MIT

package learning

import (
	"math"
)

// BehavioralScorer calculates multi-dimensional behavioral scores
type BehavioralScorer struct {
	config Config
}

// NewBehavioralScorer creates a new behavioral scorer
func NewBehavioralScorer(config Config) *BehavioralScorer {
	return &BehavioralScorer{
		config: config,
	}
}

// CalculateScore calculates a comprehensive behavioral score from patterns and tool usage
func (s *BehavioralScorer) CalculateScore(patterns []EnhancedPattern, toolUsage []ToolUsagePattern) BehavioralScore {
	now := NowMs()
	oneWeekAgo := now - (7 * MsPerDay)

	// Filter recent patterns
	var recentPatterns []EnhancedPattern
	for _, p := range patterns {
		if p.LastObserved > oneWeekAgo {
			recentPatterns = append(recentPatterns, p)
		}
	}

	score := BehavioralScore{
		Overall: 0.5, // Start with neutral
	}
	score.BasedOn.SampleSize = len(recentPatterns)
	score.BasedOn.TimeWindow = 7 * MsPerDay
	score.BasedOn.LastUpdated = now

	// Calculate response quality from positive/negative patterns
	var positivePatterns, negativePatterns, correctionPatterns []EnhancedPattern
	for _, p := range recentPatterns {
		switch p.Category {
		case CategorySuccessIndicator:
			positivePatterns = append(positivePatterns, p)
		case CategoryFailureIndicator:
			negativePatterns = append(negativePatterns, p)
		case CategoryCorrectionExplicit, CategoryCorrectionImplicit:
			correctionPatterns = append(correctionPatterns, p)
		}
	}

	totalFeedback := len(positivePatterns) + len(negativePatterns)
	if totalFeedback > 0 {
		score.Dimensions.ResponseQuality = float64(len(positivePatterns)) / float64(totalFeedback)
	} else {
		score.Dimensions.ResponseQuality = 0.5
	}

	// Calculate tool efficiency
	score.Dimensions.ToolEfficiency = s.calculateToolUsageScore(toolUsage)

	// Calculate context relevance (patterns with high support are being used)
	if len(recentPatterns) > 0 {
		highSupportCount := 0
		for _, p := range recentPatterns {
			if p.Support >= 3 {
				highSupportCount++
			}
		}
		score.Dimensions.ContextRelevance = float64(highSupportCount) / float64(len(recentPatterns))
	} else {
		score.Dimensions.ContextRelevance = 0.5
	}

	// Calculate correction rate (lower is better)
	if len(recentPatterns) > 0 {
		correctionRate := float64(len(correctionPatterns)) / float64(len(recentPatterns))
		score.Dimensions.CorrectionRate = math.Max(0, 1-correctionRate)
	} else {
		score.Dimensions.CorrectionRate = 0.8 // Default to high when no corrections
	}

	// Calculate adaptation speed (how quickly confidence increases after corrections)
	oneDayAgo := now - MsPerDay
	var recentCorrections []EnhancedPattern
	for _, p := range correctionPatterns {
		if p.LastObserved > oneDayAgo {
			recentCorrections = append(recentCorrections, p)
		}
	}
	if len(recentCorrections) > 0 {
		score.Dimensions.AdaptationSpeed = math.Min(1, 1/float64(len(recentCorrections)+1))
	} else {
		score.Dimensions.AdaptationSpeed = 0.8
	}

	// Calculate overall score
	score.Overall = (score.Dimensions.ResponseQuality*0.3 +
		score.Dimensions.ToolEfficiency*0.25 +
		score.Dimensions.ContextRelevance*0.2 +
		score.Dimensions.CorrectionRate*0.15 +
		score.Dimensions.AdaptationSpeed*0.1)

	// Set trends (would require historical data for accurate calculation)
	score.Trends.ResponseQuality = TrendStable
	score.Trends.ToolEfficiency = TrendStable
	score.Trends.ContextRelevance = TrendStable

	return score
}

// calculateToolUsageScore calculates efficiency score from tool usage patterns
func (s *BehavioralScorer) calculateToolUsageScore(toolUsage []ToolUsagePattern) float64 {
	if len(toolUsage) == 0 {
		return 0.5
	}

	var totalScore, totalWeight float64

	for _, tool := range toolUsage {
		// Calculate success rate
		var successRate float64
		if tool.TotalCalls > 0 {
			successRate = float64(tool.SuccessfulCalls) / float64(tool.TotalCalls)
		} else {
			successRate = 0.5
		}

		// Calculate rejection rate
		var rejectionRate float64
		if tool.TotalCalls > 0 {
			rejectionRate = float64(tool.UserRejectedCalls) / float64(tool.TotalCalls)
		}

		// Normalize user preference (-1 to 1 -> 0 to 1)
		preferenceScore := (tool.UserPreference + 1) / 2

		// Weight by usage frequency (log scale)
		weight := math.Log10(float64(tool.TotalCalls)+1) + 1

		// Combined score
		toolScore := successRate*0.4 + (1-rejectionRate)*0.3 + preferenceScore*0.3

		totalScore += toolScore * weight
		totalWeight += weight
	}

	if totalWeight > 0 {
		return totalScore / totalWeight
	}
	return 0.5
}

// CalculateTrend determines trend direction from a series of values
func (s *BehavioralScorer) CalculateTrend(values []float64) TrendDirection {
	if len(values) < 3 {
		return TrendStable
	}

	// Compare recent values to older values
	recent := values[len(values)-3:]
	older := values[0 : len(values)-3]

	if len(older) == 0 {
		return TrendStable
	}

	var recentSum, olderSum float64
	for _, v := range recent {
		recentSum += v
	}
	for _, v := range older {
		olderSum += v
	}

	recentAvg := recentSum / float64(len(recent))
	olderAvg := olderSum / float64(len(older))

	if olderAvg == 0 {
		return TrendStable
	}

	change := (recentAvg - olderAvg) / olderAvg

	if change > 0.1 {
		return TrendImproving
	}
	if change < -0.1 {
		return TrendDeclining
	}
	return TrendStable
}

// UpdateBehavioralScore updates a behavioral score with new data
func (s *BehavioralScorer) UpdateBehavioralScore(current BehavioralScore, newSignal DetectedSignal) BehavioralScore {
	// Weighted update based on signal confidence
	weight := newSignal.Confidence * 0.1 // Small update factor

	switch newSignal.Category {
	case CategorySuccessIndicator:
		current.Dimensions.ResponseQuality = updateWeightedAvg(
			current.Dimensions.ResponseQuality, 1.0, weight)
	case CategoryFailureIndicator:
		current.Dimensions.ResponseQuality = updateWeightedAvg(
			current.Dimensions.ResponseQuality, 0.0, weight)
	case CategoryCorrectionExplicit, CategoryCorrectionImplicit:
		current.Dimensions.CorrectionRate = updateWeightedAvg(
			current.Dimensions.CorrectionRate, 0.0, weight)
	}

	// Recalculate overall
	current.Overall = (current.Dimensions.ResponseQuality*0.3 +
		current.Dimensions.ToolEfficiency*0.25 +
		current.Dimensions.ContextRelevance*0.2 +
		current.Dimensions.CorrectionRate*0.15 +
		current.Dimensions.AdaptationSpeed*0.1)

	current.BasedOn.LastUpdated = NowMs()

	return current
}

// updateWeightedAvg performs a weighted average update
func updateWeightedAvg(current, newValue, weight float64) float64 {
	return current*(1-weight) + newValue*weight
}
