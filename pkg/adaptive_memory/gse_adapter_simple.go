// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System - Simple GSE Adapter (No external dependencies)
// License: MIT

//go:build !nogse

package adaptive_memory

import "strings"

// SimpleSegmenter provides basic segmentation without external dependencies
type SimpleSegmenter struct{}

// NewSimpleSegmenter creates a simple segmenter
func NewSimpleSegmenter() *SimpleSegmenter {
	return &SimpleSegmenter{}
}

// Cut segments text into words (simple space-based segmentation)
func (s *SimpleSegmenter) Cut(text string, searchMode ...bool) []string {
	words := strings.Fields(text)
	var result []string
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w != "" {
			result = append(result, w)
		}
	}
	return result
}

// Load is a no-op for simple segmenter
func (s *SimpleSegmenter) Load() error {
	return nil
}
