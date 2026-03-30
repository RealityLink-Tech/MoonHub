// MoonHub - Your ready-to-use AI assistant
// Copyright (c) 2026 MoonHub contributors
// License: MIT

package compactor

import (
	"strings"
	"unicode"
)

// CJK ranges for detecting Chinese, Japanese, Korean characters
var cjkRanges = []*unicode.RangeTable{
	unicode.Han,
	unicode.Hiragana,
	unicode.Katakana,
	unicode.Hangul,
}

// isCJK checks if a rune is a CJK character
func isCJK(r rune) bool {
	for _, rt := range cjkRanges {
		if unicode.Is(rt, r) {
			return true
		}
	}
	return false
}

// EstimateTokens estimates token count for a given text.
// Uses a character-based heuristic calibrated against common tokenizers:
//   - English text: ~4 characters per token
//   - CJK text: ~1.5 characters per token
//   - Mixed: weighted average
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	var cjkChars, totalChars int
	for _, r := range text {
		totalChars++
		if isCJK(r) {
			cjkChars++
		}
	}

	asciiChars := totalChars - cjkChars

	// ASCII at ~4 chars/token, CJK at ~1.5 chars/token
	asciiTokens := float64(asciiChars) / 4.0
	cjkTokens := float64(cjkChars) / 1.5

	return int(asciiTokens + cjkTokens + 0.5) // Round up
}

// EstimateTokensFromMessages estimates total tokens from a slice of messages
func EstimateTokensFromMessages(messages []string) int {
	total := 0
	for _, m := range messages {
		total += EstimateTokens(m)
	}
	return total
}

// TruncateToTokenBudget truncates text to fit within a token budget.
// Uses an adaptive chars-per-token ratio based on content composition.
func TruncateToTokenBudget(text string, maxTokens int) string {
	if EstimateTokens(text) <= maxTokens {
		return text
	}

	// Detect CJK density to choose an adaptive chars-per-token ratio
	runes := []rune(text)
	var cjkCount int
	for _, r := range runes {
		if isCJK(r) {
			cjkCount++
		}
	}

	cjkRatio := 0.0
	if len(runes) > 0 {
		cjkRatio = float64(cjkCount) / float64(len(runes))
	}

	// Blend between ~1.5 (pure CJK) and ~4 (pure ASCII)
	charsPerToken := 4.0
	if cjkRatio > 0.3 {
		charsPerToken = 1.5 + (1-cjkRatio)*2.5
	}

	// Initial estimate: slice by runes
	charLimit := int(float64(maxTokens) * charsPerToken)
	if charLimit > len(runes) {
		charLimit = len(runes)
	}
	truncated := string(runes[:charLimit])

	// Cut at last newline or space for clean boundary
	lastNewline := strings.LastIndex(truncated, "\n")
	lastSpace := strings.LastIndex(truncated, " ")
	cutPoint := max(lastNewline, lastSpace)

	if cutPoint > len(truncated)/2 {
		truncated = truncated[:cutPoint]
	}

	// Iteratively trim if still over budget
	for EstimateTokens(truncated) > maxTokens && len(truncated) > 0 {
		overBy := EstimateTokens(truncated) - maxTokens
		// Remove roughly overBy * charsPerToken runes
		trimBy := max(1, int(float64(overBy)*charsPerToken))

		runes = []rune(truncated)
		newLen := len(runes) - trimBy
		if newLen < 0 {
			newLen = 0
		}
		truncated = string(runes[:newLen])

		// Re-cut at word boundary
		lastSpace = strings.LastIndex(truncated, " ")
		lastNewline = strings.LastIndex(truncated, "\n")
		cutPoint = max(lastSpace, lastNewline)
		if cutPoint > len(truncated)/2 {
			truncated = truncated[:cutPoint]
		}
	}

	return truncated
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
