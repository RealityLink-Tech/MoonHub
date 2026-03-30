// MoonHub - Your ready-to-use AI assistant
// Internationalization (i18n) support for learning system
// License: MIT

package i18n

import "strings"

// Language represents a supported language
type Language string

const (
	// LangEn is English
	LangEn Language = "en"
	// LangZh is Chinese (Simplified)
	LangZh Language = "zh"
	// LangAuto automatically detects based on content
	LangAuto Language = "auto"
)

// TitleDesc represents a title and description pair
type TitleDesc struct {
	Title       string
	Description string
}

// SuggestionsText contains all suggestion-related text
type SuggestionsText struct {
	ToolLowSuccess     TitleDesc
	ToolUserPreference TitleDesc
	ToolUsageTrend     TitleDesc
	WorkflowPattern    TitleDesc
	ImproveAdaptation  TitleDesc
	ResponseQuality    TitleDesc
	FrequentCorrection TitleDesc
	ConfirmPreference  TitleDesc
}

// ContextText contains all context-related text
type ContextText struct {
	Preferences string
	ToolPrefs   string
	Workflow    string
	WhatWorks   string
	Corrections string
	Insights    string
	Notes       string
}

// TextResources contains all localized text resources
type TextResources struct {
	Suggestions SuggestionsText
	Context     ContextText
}

// I18n manages internationalization
type I18n struct {
	lang  Language
	texts map[Language]TextResources
}

// New creates a new I18n instance
func New(lang string) *I18n {
	i := &I18n{
		lang:  parseLanguage(lang),
		texts: make(map[Language]TextResources),
	}

	// Initialize all language resources
	i.texts[LangEn] = initEnglish()
	i.texts[LangZh] = initChinese()

	return i
}

// parseLanguage parses a language string into a Language constant
func parseLanguage(lang string) Language {
	switch strings.ToLower(lang) {
	case "zh", "chinese", "cn", "zh-cn":
		return LangZh
	case "en", "english":
		return LangEn
	default:
		return LangAuto
	}
}

// GetLanguage returns the current language setting
func (i *I18n) GetLanguage() Language {
	return i.lang
}

// Texts returns the text resources for the configured language
func (i *I18n) Texts() TextResources {
	if i.lang == LangAuto {
		// Default to English for auto mode
		return i.texts[LangEn]
	}
	return i.texts[i.lang]
}

// TextsForLang returns text resources for a specific language
func (i *I18n) TextsForLang(lang Language) TextResources {
	return i.texts[lang]
}

// AllTexts returns text resources for all languages (for multi-language support)
func (i *I18n) AllTexts() map[Language]TextResources {
	return i.texts
}

// DetectLanguage attempts to detect the language from the given text
func DetectLanguage(text string) Language {
	// Simple heuristic: check for Chinese characters
	chineseCount := 0
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			chineseCount++
		}
	}

	// If more than 20% of characters are Chinese, consider it Chinese
	if float64(chineseCount)/float64(len([]rune(text))) > 0.2 {
		return LangZh
	}
	return LangEn
}
