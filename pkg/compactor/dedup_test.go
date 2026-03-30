// MoonHub - Your ready-to-use AI assistant
// Copyright (c) 2026 MoonHub contributors
// License: MIT

package compactor

import (
	"testing"
)

func TestComputeShingles(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		shingleSize int
		wantCount   int // minimum expected shingle count
	}{
		{
			name:        "simple text",
			text:        "hello world test",
			shingleSize: 3,
			wantCount:   1, // "hello world test" is 1 shingle
		},
		{
			name:        "longer text",
			text:        "the quick brown fox jumps over",
			shingleSize: 3,
			wantCount:   4, // multiple 3-word combinations
		},
		{
			name:        "short text",
			text:        "hi",
			shingleSize: 3,
			wantCount:   0, // not enough words for 3-shingle
		},
		{
			name:        "empty text",
			text:        "",
			shingleSize: 3,
			wantCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeShingles(tt.text, tt.shingleSize)
			if len(got) < tt.wantCount {
				t.Errorf("ComputeShingles got %d shingles, want at least %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestComputeShingles_Normalization(t *testing.T) {
	// Both should produce same shingles after normalization
	text1 := "Hello, World! Test."
	text2 := "hello world test"

	shingles1 := ComputeShingles(text1, 2)
	shingles2 := ComputeShingles(text2, 2)

	// After normalization, they should be similar
	// Note: punctuation is removed, so they may differ slightly
	if len(shingles1) == 0 && len(shingles2) > 0 {
		t.Error("ComputeShingles should normalize punctuation")
	}
}

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name    string
		setA    map[string]struct{}
		setB    map[string]struct{}
		wantMin float64
		wantMax float64
	}{
		{
			name:    "identical sets",
			setA:    map[string]struct{}{"a": {}, "b": {}, "c": {}},
			setB:    map[string]struct{}{"a": {}, "b": {}, "c": {}},
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name:    "no overlap",
			setA:    map[string]struct{}{"a": {}, "b": {}},
			setB:    map[string]struct{}{"c": {}, "d": {}},
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "partial overlap",
			setA:    map[string]struct{}{"a": {}, "b": {}, "c": {}},
			setB:    map[string]struct{}{"b": {}, "c": {}, "d": {}},
			wantMin: 0.3,
			wantMax: 0.7,
		},
		{
			name:    "one empty set",
			setA:    map[string]struct{}{},
			setB:    map[string]struct{}{"a": {}, "b": {}},
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "both empty",
			setA:    map[string]struct{}{},
			setB:    map[string]struct{}{},
			wantMin: 0.0,
			wantMax: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JaccardSimilarity(tt.setA, tt.setB)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("JaccardSimilarity() = %f, want between %f and %f",
					got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestDeduplicateMessages(t *testing.T) {
	tests := []struct {
		name              string
		messages          []Message
		threshold         float64
		wantCount         int
		wantGroupsRemoved int
	}{
		{
			name: "no duplicates",
			messages: []Message{
				{Role: "user", Content: "Hello world"},
				{Role: "assistant", Content: "Hi there"},
				{Role: "user", Content: "How are you?"},
			},
			threshold:         0.6,
			wantCount:         3,
			wantGroupsRemoved: 0,
		},
		{
			name: "exact duplicates",
			messages: []Message{
				{Role: "user", Content: "Hello world test message"},
				{Role: "user", Content: "Hello world test message"},
			},
			threshold:         0.6,
			wantCount:         1,
			wantGroupsRemoved: 1,
		},
		{
			name: "very similar messages",
			messages: []Message{
				{Role: "user", Content: "The quick brown fox jumps over the lazy dog and runs away"},
				{Role: "user", Content: "The quick brown fox jumps over the lazy dog and runs fast"},
			},
			threshold:         0.5,
			wantCount:         1,
			wantGroupsRemoved: 1,
		},
		{
			name: "different messages",
			messages: []Message{
				{Role: "user", Content: "Completely different content here"},
				{Role: "user", Content: "Totally unrelated message text"},
			},
			threshold:         0.6,
			wantCount:         2,
			wantGroupsRemoved: 0,
		},
		{
			name:              "empty input",
			messages:          []Message{},
			threshold:         0.6,
			wantCount:         0,
			wantGroupsRemoved: 0,
		},
		{
			name:              "single message",
			messages:          []Message{{Role: "user", Content: "Single"}},
			threshold:         0.6,
			wantCount:         1,
			wantGroupsRemoved: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeduplicateMessages(tt.messages, tt.threshold)
			if len(result.Messages) != tt.wantCount {
				t.Errorf("DeduplicateMessages got %d messages, want %d",
					len(result.Messages), tt.wantCount)
			}
			if result.GroupsRemoved != tt.wantGroupsRemoved {
				t.Errorf("DeduplicateMessages got %d groups removed, want %d",
					result.GroupsRemoved, tt.wantGroupsRemoved)
			}
		})
	}
}

func TestDeduplicateMessagesParallel(t *testing.T) {
	// Create a larger set of messages for parallel testing
	messages := make([]Message, 100)
	for i := 0; i < 100; i++ {
		if i%10 == 0 {
			// Every 10th message is a duplicate
			messages[i] = Message{Role: "user", Content: "This is a duplicate message for testing"}
		} else {
			messages[i] = Message{Role: "user", Content: "Unique message number " + string(rune('0'+i%10))}
		}
	}

	result := DeduplicateMessagesParallel(messages, 0.6)

	// Should have fewer messages than input due to deduplication
	if len(result.Messages) >= len(messages) {
		t.Errorf("DeduplicateMessagesParallel should reduce duplicates")
	}

	// Groups removed should be positive
	if result.GroupsRemoved <= 0 {
		t.Errorf("DeduplicateMessagesParallel should detect duplicate groups")
	}
}

func TestDeduplicateStrings(t *testing.T) {
	tests := []struct {
		name      string
		strings   []string
		threshold float64
		wantCount int
	}{
		{
			name: "no duplicates",
			strings: []string{
				"First unique string",
				"Second unique string",
				"Third unique string",
			},
			threshold: 0.6,
			wantCount: 3,
		},
		{
			name: "with duplicates",
			strings: []string{
				"The quick brown fox jumps over the lazy dog",
				"The quick brown fox jumps over the lazy dog",
				"Completely different text here",
			},
			threshold: 0.6,
			wantCount: 2,
		},
		{
			name:      "empty input",
			strings:   []string{},
			threshold: 0.6,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeduplicateStrings(tt.strings, tt.threshold)
			if len(result.Strings) != tt.wantCount {
				t.Errorf("DeduplicateStrings got %d strings, want %d",
					len(result.Strings), tt.wantCount)
			}
		})
	}
}

func TestDeduplicateMessages_KeepsNewer(t *testing.T) {
	// When messages are similar, the newer (later) one should be kept
	messages := []Message{
		{Role: "user", Content: "Hello world this is a test message"},
		{Role: "user", Content: "Hello world this is a test message with more"},
	}

	result := DeduplicateMessages(messages, 0.6)

	// Should have 1 message
	if len(result.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result.Messages))
	}

	// Should be the second (newer) message
	if len(result.Messages) > 0 && result.Messages[0].Content != messages[1].Content {
		t.Errorf("Expected newer message to be kept, got: %q", result.Messages[0].Content)
	}
}
