// MoonHub - Your ready-to-use AI assistant
// Copyright (c) 2026 MoonHub contributors
// License: MIT

package compactor

import (
	"regexp"
	"strings"
	"sync"
)

// Message represents a chat message for deduplication
type Message struct {
	Role    string
	Content string
}

// ComputeShingles extracts word-level n-gram shingles from text.
// Uses word-level shingles for better semantic comparison.
func ComputeShingles(text string, shingleSize int) map[string]struct{} {
	// Normalize text: lowercase, remove non-alphanumeric
	text = strings.ToLower(text)
	// Replace non-alphanumeric with space
	re := regexp.MustCompile(`[^a-z0-9\s]`)
	text = re.ReplaceAllString(text, " ")

	// Split into words, filter short words
	words := strings.Fields(text)
	var filtered []string
	for _, w := range words {
		if len(w) > 1 {
			filtered = append(filtered, w)
		}
	}

	shingles := make(map[string]struct{})
	for i := 0; i <= len(filtered)-shingleSize; i++ {
		shingle := strings.Join(filtered[i:i+shingleSize], " ")
		shingles[shingle] = struct{}{}
	}

	return shingles
}

// JaccardSimilarity computes Jaccard similarity between two shingle sets.
// Returns 0.0 (no overlap) to 1.0 (identical).
func JaccardSimilarity(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	// Iterate over smaller set for efficiency
	smaller, larger := a, b
	if len(b) < len(a) {
		smaller, larger = b, a
	}

	intersection := 0
	for shingle := range smaller {
		if _, exists := larger[shingle]; exists {
			intersection++
		}
	}

	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// shingledMessage holds a message with pre-computed shingles
type shingledMessage struct {
	index    int
	message  Message
	shingles map[string]struct{}
}

// DeduplicationResult contains the result of deduplication
type DeduplicationResult struct {
	Messages      []Message
	GroupsRemoved int
}

// DeduplicateMessages removes near-duplicate messages based on shingle similarity.
//
// When two messages exceed the similarity threshold, the earlier one
// is dropped (keeping the more recent/complete version).
//
// Returns the deduplicated messages and the count of groups removed.
func DeduplicateMessages(messages []Message, similarityThreshold float64) DeduplicationResult {
	if len(messages) <= 1 {
		return DeduplicationResult{
			Messages:      messages,
			GroupsRemoved: 0,
		}
	}

	// Pre-compute shingles for all messages
	shingled := make([]shingledMessage, len(messages))
	for i, msg := range messages {
		shingled[i] = shingledMessage{
			index:    i,
			message:  msg,
			shingles: ComputeShingles(msg.Content, 3),
		}
	}

	// Mark duplicates (earlier message in each similar pair is dropped)
	dropped := make(map[int]bool)
	groupsRemoved := 0

	for i := 0; i < len(shingled); i++ {
		if dropped[i] {
			continue
		}

		for j := i + 1; j < len(shingled); j++ {
			if dropped[j] {
				continue
			}

			similarity := JaccardSimilarity(shingled[i].shingles, shingled[j].shingles)
			if similarity >= similarityThreshold {
				// Drop the earlier (less recent) message
				dropped[i] = true
				// Count one group per first duplicate detected (not per message)
				groupsRemoved++
				break // This message is already marked, move on
			}
		}
	}

	// Filter out dropped messages
	result := make([]Message, 0, len(messages)-len(dropped))
	for _, s := range shingled {
		if !dropped[s.index] {
			result = append(result, s.message)
		}
	}

	return DeduplicationResult{
		Messages:      result,
		GroupsRemoved: groupsRemoved,
	}
}

// DeduplicateMessagesParallel removes near-duplicate messages using parallel processing.
// This is an optimized version for large message sets.
func DeduplicateMessagesParallel(messages []Message, similarityThreshold float64) DeduplicationResult {
	if len(messages) <= 1 {
		return DeduplicationResult{
			Messages:      messages,
			GroupsRemoved: 0,
		}
	}

	// Pre-compute shingles for all messages in parallel
	shingled := make([]shingledMessage, len(messages))
	var wg sync.WaitGroup

	for i, msg := range messages {
		wg.Add(1)
		go func(idx int, m Message) {
			defer wg.Done()
			shingled[idx] = shingledMessage{
				index:    idx,
				message:  m,
				shingles: ComputeShingles(m.Content, 3),
			}
		}(i, msg)
	}
	wg.Wait()

	// Mark duplicates with mutex protection
	dropped := make(map[int]bool)
	var mu sync.Mutex
	groupsRemoved := 0

	for i := 0; i < len(shingled); i++ {
		if dropped[i] {
			continue
		}

		for j := i + 1; j < len(shingled); j++ {
			if dropped[j] {
				continue
			}

			similarity := JaccardSimilarity(shingled[i].shingles, shingled[j].shingles)
			if similarity >= similarityThreshold {
				mu.Lock()
				dropped[i] = true
				groupsRemoved++
				mu.Unlock()
				break
			}
		}
	}

	// Filter out dropped messages
	result := make([]Message, 0, len(messages)-len(dropped))
	for _, s := range shingled {
		if !dropped[s.index] {
			result = append(result, s.message)
		}
	}

	return DeduplicationResult{
		Messages:      result,
		GroupsRemoved: groupsRemoved,
	}
}

// StringDeduplicationResult contains the result of string deduplication
type StringDeduplicationResult struct {
	Strings       []string
	GroupsRemoved int
}

// DeduplicateStrings removes near-duplicate strings based on shingle similarity.
func DeduplicateStrings(texts []string, similarityThreshold float64) StringDeduplicationResult {
	messages := make([]Message, len(texts))
	for i, t := range texts {
		messages[i] = Message{Content: t}
	}

	result := DeduplicateMessages(messages, similarityThreshold)

	strings := make([]string, len(result.Messages))
	for i, m := range result.Messages {
		strings[i] = m.Content
	}

	return StringDeduplicationResult{
		Strings:       strings,
		GroupsRemoved: result.GroupsRemoved,
	}
}
