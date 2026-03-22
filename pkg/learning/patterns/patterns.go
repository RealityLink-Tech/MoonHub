// MoonHub - Ultra-lightweight personal AI agent
// Pattern management for multi-language support
// License: MIT

package patterns

import (
	"regexp"

	"github.com/RealityLink-Tech/MoonHub/pkg/learning/i18n"
)

// SemanticIndicator represents a set of keywords with associated weight
type SemanticIndicator struct {
	Keywords []string
	Weight   float64
}

// PatternSet contains regex patterns for detection
type PatternSet struct {
	Positive   []*regexp.Regexp
	Negative   []*regexp.Regexp
	Correction []*regexp.Regexp
	Workflow   []*regexp.Regexp
	ToolPref   []*regexp.Regexp
}

// SemanticIndicators contains semantic keyword indicators
type SemanticIndicators struct {
	Positive []SemanticIndicator
	Negative []SemanticIndicator
}

// Manager manages patterns for multiple languages
type Manager struct {
	patterns  map[i18n.Language]PatternSet
	semantics map[i18n.Language]SemanticIndicators
}

// NewManager creates a new pattern manager
func NewManager() *Manager {
	m := &Manager{
		patterns:  make(map[i18n.Language]PatternSet),
		semantics: make(map[i18n.Language]SemanticIndicators),
	}

	// Initialize all language patterns
	m.patterns[i18n.LangEn] = initEnglishPatterns()
	m.patterns[i18n.LangZh] = initChinesePatterns()

	// Initialize all language semantics
	m.semantics[i18n.LangEn] = initEnglishSemantics()
	m.semantics[i18n.LangZh] = initChineseSemantics()

	return m
}

// GetPatterns returns patterns for a specific language
func (m *Manager) GetPatterns(lang i18n.Language) PatternSet {
	if lang == i18n.LangAuto {
		// Return combined patterns for auto mode
		return m.combinePatterns()
	}
	return m.patterns[lang]
}

// GetSemantics returns semantic indicators for a specific language
func (m *Manager) GetSemantics(lang i18n.Language) SemanticIndicators {
	if lang == i18n.LangAuto {
		// Return combined semantics for auto mode
		return m.combineSemantics()
	}
	return m.semantics[lang]
}

// GetAllPatterns returns patterns for all languages combined
func (m *Manager) GetAllPatterns() PatternSet {
	return m.combinePatterns()
}

// GetAllSemantics returns semantic indicators for all languages combined
func (m *Manager) GetAllSemantics() SemanticIndicators {
	return m.combineSemantics()
}

// combinePatterns combines patterns from all languages
func (m *Manager) combinePatterns() PatternSet {
	combined := PatternSet{
		Positive:   append(m.patterns[i18n.LangEn].Positive, m.patterns[i18n.LangZh].Positive...),
		Negative:   append(m.patterns[i18n.LangEn].Negative, m.patterns[i18n.LangZh].Negative...),
		Correction: append(m.patterns[i18n.LangEn].Correction, m.patterns[i18n.LangZh].Correction...),
		Workflow:   append(m.patterns[i18n.LangEn].Workflow, m.patterns[i18n.LangZh].Workflow...),
		ToolPref:   append(m.patterns[i18n.LangEn].ToolPref, m.patterns[i18n.LangZh].ToolPref...),
	}
	return combined
}

// combineSemantics combines semantic indicators from all languages
func (m *Manager) combineSemantics() SemanticIndicators {
	combined := SemanticIndicators{
		Positive: append(m.semantics[i18n.LangEn].Positive, m.semantics[i18n.LangZh].Positive...),
		Negative: append(m.semantics[i18n.LangEn].Negative, m.semantics[i18n.LangZh].Negative...),
	}
	return combined
}
