// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System - Chinese Segmentation
// License: MIT

package adaptive_memory

import (
	"regexp"
	"strings"
	"sync"
)

// ChineseSegmenter handles Chinese text segmentation
// Uses lazy loading to minimize memory footprint
type ChineseSegmenter struct {
	segmenter ChineseSegmenterInterface
	once      sync.Once
	enabled   bool
	mu        sync.RWMutex
}

// ChineseSegmenterInterface defines the interface for Chinese segmentation
// This allows us to mock in tests
type ChineseSegmenterInterface interface {
	// Cut segments text into words
	Cut(text string, searchMode ...bool) []string
	// Load loads the dictionary
	Load() error
}

// NewChineseSegmenter creates a Chinese segmenter instance
func NewChineseSegmenter(enabled bool) *ChineseSegmenter {
	return &ChineseSegmenter{
		enabled: enabled,
	}
}

// Init lazily loads the segmenter (heavy operation ~2-3MB)
func (cs *ChineseSegmenter) Init() error {
	if !cs.enabled {
		return nil
	}

	var initErr error
	cs.once.Do(func() {
		// We'll set the segmenter via SetSegmenter
		// This is done to avoid direct dependency on gse in this file
		// The actual segmenter is set via SetSegmenter
	})
	return initErr
}

// SetSegmenter sets the segmenter implementation
// This is called by the engine initialization
func (cs *ChineseSegmenter) SetSegmenter(seg ChineseSegmenterInterface) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.segmenter = seg
}

// Segment splits Chinese text into space-separated tokens for FTS5
func (cs *ChineseSegmenter) Segment(text string) string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if !cs.enabled || cs.segmenter == nil {
		return text // Return as-is if disabled or not initialized
	}

	// Use gse to segment
	segments := cs.segmenter.Cut(text, true)
	return strings.Join(segments, " ")
}

// SmartSegment segments if Chinese detected, otherwise returns original
func (cs *ChineseSegmenter) SmartSegment(text string) string {
	if ContainsChinese(text) {
		return cs.Segment(text)
	}
	return text
}

// PrepareForFTS creates both original and segmented versions
func (cs *ChineseSegmenter) PrepareForFTS(content string) (original, segmented string) {
	original = content
	segmented = cs.SmartSegment(content)
	return
}

// chineseRegex matches Chinese characters (CJK Unified Ideographs)
var chineseRegex = regexp.MustCompile(`[\p{Han}]`)

// ContainsChinese checks if text has Chinese characters
func ContainsChinese(text string) bool {
	return chineseRegex.MatchString(text)
}

// TokenizeForSearch tokenizes text for search query
// Handles both Chinese and English
func TokenizeForSearch(text string) []string {
	text = strings.ToLower(text)

	// Remove punctuation
	re := regexp.MustCompile(`[^\p{L}\p{N}\s]`)
	text = re.ReplaceAllString(text, " ")

	var tokens []string
	for _, token := range strings.Fields(text) {
		// Skip short tokens
		if len(token) > 1 {
			tokens = append(tokens, token)
		}
	}

	return tokens
}
