// MoonHub - Your ready-to-use AI assistant
// Copyright (c) 2026 MoonHub contributors
// License: MIT

package compactor

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		minTokens int
		maxTokens int
	}{
		{
			name:      "empty string",
			input:     "",
			minTokens: 0,
			maxTokens: 0,
		},
		{
			name:      "simple English",
			input:     "Hello world",
			minTokens: 2,
			maxTokens: 4,
		},
		{
			name:      "pure Chinese",
			input:     "你好世界",
			minTokens: 2,
			maxTokens: 4,
		},
		{
			name:      "mixed content",
			input:     "Hello 你好 World 世界",
			minTokens: 4,
			maxTokens: 8,
		},
		{
			name:      "long English text",
			input:     strings.Repeat("hello ", 100),
			minTokens: 80,
			maxTokens: 160,
		},
		{
			name:      "long Chinese text",
			input:     strings.Repeat("你好", 100),
			minTokens: 100,
			maxTokens: 150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.input)
			if got < tt.minTokens || got > tt.maxTokens {
				t.Errorf("EstimateTokens(%q) = %d, want between %d and %d",
					tt.input, got, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestEstimateTokens_CJKRatio(t *testing.T) {
	// CJK text should have higher token density (fewer chars per token)
	englishText := "This is a simple English sentence with many words"
	cjkText := "这是一个简单的中文句子包含很多词语"

	engTokens := EstimateTokens(englishText)
	cjkTokens := EstimateTokens(cjkText)

	// CJK should have more tokens for similar character counts
	// because chars/token ratio is lower (~1.5 vs ~4)
	if cjkTokens < engTokens/2 {
		t.Errorf("CJK token estimation too low: got %d for CJK vs %d for English",
			cjkTokens, engTokens)
	}
}

func TestEstimateTokensFromMessages(t *testing.T) {
	messages := []string{
		"Hello world",
		"你好世界",
		"Testing 123",
	}

	total := EstimateTokensFromMessages(messages)
	if total == 0 {
		t.Error("EstimateTokensFromMessages returned 0 for non-empty messages")
	}

	// Should be sum of individual estimates
	expectedTotal := 0
	for _, m := range messages {
		expectedTotal += EstimateTokens(m)
	}
	if total != expectedTotal {
		t.Errorf("EstimateTokensFromMessages = %d, want %d", total, expectedTotal)
	}
}

func TestTruncateToTokenBudget(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		maxTokens  int
		wantLength int // approximate
	}{
		{
			name:       "under budget",
			input:      "Short text",
			maxTokens:  100,
			wantLength: 10, // unchanged
		},
		{
			name:       "over budget English",
			input:      strings.Repeat("Hello world ", 100),
			maxTokens:  20,
			wantLength: 80, // ~20 tokens * 4 chars
		},
		{
			name:       "over budget Chinese",
			input:      strings.Repeat("你好世界", 100),
			maxTokens:  30,
			wantLength: 45, // ~30 tokens * 1.5 chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateToTokenBudget(tt.input, tt.maxTokens)

			// Verify token budget is respected
			gotTokens := EstimateTokens(got)
			if gotTokens > tt.maxTokens {
				t.Errorf("TruncateToTokenBudget result has %d tokens, max is %d",
					gotTokens, tt.maxTokens)
			}

			// For under-budget case, should be unchanged
			if tt.name == "under budget" && got != tt.input {
				t.Errorf("TruncateToTokenBudget changed under-budget text")
			}
		})
	}
}

func TestTruncateToTokenBudget_WordBoundary(t *testing.T) {
	// Long text that needs truncation
	longText := "This is a sentence. And another one. More text here. Final part."
	result := TruncateToTokenBudget(longText, 5)

	// Should not cut in the middle of a word (when possible)
	// Result should end at a space or period
	if len(result) > 0 && result[len(result)-1] != ' ' && result[len(result)-1] != '.' {
		// Allow some flexibility for short results
		if len(result) > 10 {
			t.Errorf("TruncateToTokenBudget may have cut mid-word: %q", result)
		}
	}
}

func TestIsCJK(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'a', false},
		{'Z', false},
		{'0', false},
		{' ', false},
		{'中', true},
		{'文', true},
		{'あ', true},  // Hiragana
		{'ア', true},  // Katakana
		{'한', true},  // Hangul
		{'。', false}, // CJK punctuation is not in the ranges
	}

	for _, tt := range tests {
		got := isCJK(tt.r)
		if got != tt.want {
			t.Errorf("isCJK(%q) = %v, want %v", tt.r, got, tt.want)
		}
	}
}
