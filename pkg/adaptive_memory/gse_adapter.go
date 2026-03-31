// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System - GSE Adapter for Chinese Segmentation
// License: MIT

//go:build !nogse

package adaptive_memory

import (
	"github.com/go-ego/gse"
)

// GSESegmenter wraps the gse library for Chinese segmentation
type GSESegmenter struct {
	segmenter gse.Segmenter
	loaded    bool
}

// NewGSESegmenter creates a new GSE segmenter
func NewGSESegmenter() (*GSESegmenter, error) {
	var seg gse.Segmenter
	// Load default dictionary
	seg.LoadDict()

	return &GSESegmenter{
		segmenter: seg,
		loaded:    true,
	}, nil
}

// Cut segments text into words
func (gs *GSESegmenter) Cut(text string, searchMode ...bool) []string {
	if !gs.loaded {
		return []string{text}
	}

	// Use search mode for better segmentation in search context
	if len(searchMode) > 0 && searchMode[0] {
		return gs.segmenter.CutSearch(text, true)
	}

	return gs.segmenter.Cut(text, true)
}

// Load loads the dictionary
func (gs *GSESegmenter) Load() error {
	if gs.loaded {
		return nil
	}

	gs.segmenter.LoadDict()
	gs.loaded = true
	return nil
}
