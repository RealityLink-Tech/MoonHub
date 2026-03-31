// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System - Scoring Algorithms
// License: MIT

package adaptive_memory

import (
	"math"
	"regexp"
	"strings"
)

// ComputeTemporalScore implements Ebbinghaus forgetting curve with access frequency bonus
// Formula: e^(-0.05 * days_since_last_access) * (1 + 0.02 * access_count)
func ComputeTemporalScore(lastAccessedAt, accessCount, now int64) float64 {
	daysSinceAccess := float64(now-lastAccessedAt) / float64(MsPerDay)
	if daysSinceAccess < 0 {
		daysSinceAccess = 0
	}

	// Ebbinghaus decay
	decay := math.Exp(-0.05 * daysSinceAccess)

	// Access frequency bonus - more access = stronger memory
	accessBonus := 1.0 + 0.02*float64(accessCount)

	score := decay * accessBonus

	// Clamp to [0.0, 1.0]
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// NormalizeFTSRank converts FTS5 bm25 rank to 0.0-1.0 range
// FTS5 bm25 returns negative values where MORE negative = better match
func NormalizeFTSRank(rank float64, maxAbsRank float64) float64 {
	if maxAbsRank == 0 {
		return 0
	}
	absRank := math.Abs(rank)
	normalized := absRank / maxAbsRank
	if normalized > 1.0 {
		normalized = 1.0
	}
	return normalized
}

// ComputeRelevanceScore combines FTS5 rank, temporal score, and importance
// Formula: (fts5_rank * 0.4) + (temporal_score * 0.3) + (importance * 0.3)
func ComputeRelevanceScore(ftsRank, temporalScore, importance float64) float64 {
	return ftsRank*WeightFTS5 + temporalScore*WeightTemporal + importance*WeightImportance
}

// SanitizeFTSQuery cleans a query string for FTS5 MATCH syntax
// Handles Chinese characters and removes special FTS5 characters
func SanitizeFTSQuery(query string) string {
	// Convert to lowercase
	query = strings.ToLower(query)

	// Remove FTS5 special characters but keep Chinese
	// FTS5 special chars: * " ' ( ) { } [ ] ^ ~ : -
	re := regexp.MustCompile(`[^\p{L}\p{N}\s\p{Lo}\p{Z}\p{So}\p{Cc}\p{C}\p{Pc}0-9a-zA-Z]`)
	query = re.ReplaceAllString(query, " ")

	// Split into tokens
	tokens := strings.Fields(query)

	// Filter short tokens (but keep Chinese characters which can be meaningful as single chars)
	var filtered []string
	for _, token := range tokens {
		// Keep tokens longer than 1 char, or Chinese characters
		if len(token) > 1 || containsChinese(token) {
			filtered = append(filtered, token)
		}
	}

	if len(filtered) == 0 {
		return ""
	}

	// Join with OR for FTS5 matching
	return strings.Join(filtered, " OR ")
}

// containsChinese checks if string contains Chinese characters
func containsChinese(s string) bool {
	for _, r := range s {
		if r >= '\u4e00' && r <= '\u9fff' {
			return true
		}
	}
	return false
}

// ContentSimilarity computes Jaccard similarity between two strings
// Used for merge detection during consolidation
func ContentSimilarity(a, b string) float64 {
	// Tokenize both strings
	tokensA := tokenizeForSimilarity(a)
	tokensB := tokenizeForSimilarity(b)

	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0
	}

	// Compute intersection
	intersection := 0
	for token := range tokensA {
		if tokensB[token] {
			intersection++
		}
	}

	// Compute union
	union := len(tokensA) + len(tokensB) - intersection

	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// tokenizeForSimilarity tokenizes text for similarity comparison
// Handles both English and Chinese
func tokenizeForSimilarity(text string) map[string]bool {
	text = strings.ToLower(text)

	// Remove punctuation but keep Chinese characters
	re := regexp.MustCompile(`[^\p{L}\p{N}\s]`)
	text = re.ReplaceAllString(text, " ")

	tokens := make(map[string]bool)
	for _, token := range strings.Fields(text) {
		// For Chinese, we can also split into individual characters for better matching
		if containsChinese(token) {
			// Add the whole token
			if len(token) > 1 {
				tokens[token] = true
			}
			// Also add individual Chinese characters
			for _, r := range token {
				if r >= '\u4e00' && r <= '\u9fff' {
					tokens[string(r)] = true
				}
			}
		} else {
			// For non-Chinese, only add tokens with length > 2
			if len(token) > 2 {
				tokens[token] = true
			}
		}
	}

	return tokens
}

// DaysSinceCreated calculates days since the record was created
func DaysSinceCreated(createdAt, now int64) int {
	return int((now - createdAt) / MsPerDay)
}

// DaysSinceAccessed calculates days since the record was last accessed
func DaysSinceAccessed(lastAccessedAt, now int64) int {
	return int((now - lastAccessedAt) / MsPerDay)
}
