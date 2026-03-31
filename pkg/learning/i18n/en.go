// MoonHub - Your ready-to-use AI assistant
// English text resources for learning system
// License: MIT

package i18n

// initEnglish initializes English text resources
func initEnglish() TextResources {
	return TextResources{
		Suggestions: SuggestionsText{
			ToolLowSuccess: TitleDesc{
				Title:       "Review ${tool} usage",
				Description: "This tool has a low success rate. Consider using an alternative approach.",
			},
			ToolUserPreference: TitleDesc{
				Title:       "Tool preference noted",
				Description: "You seem to prefer not using this tool. I'll adjust future usage accordingly.",
			},
			ToolUsageTrend: TitleDesc{
				Title:       "Tool usage trend detected",
				Description: "I've noticed a pattern in how you use certain tools. Should I optimize this?",
			},
			WorkflowPattern: TitleDesc{
				Title:       "Workflow pattern detected",
				Description: "I've noticed you often ${pattern}. Should I make this a default behavior?",
			},
			ImproveAdaptation: TitleDesc{
				Title:       "Improve adaptation",
				Description: "I seem to be slow to adapt to your corrections. I'll try to be more responsive to your feedback.",
			},
			ResponseQuality: TitleDesc{
				Title:       "Response quality declining",
				Description: "Recent responses may not meet your expectations. I'll focus on improving quality.",
			},
			FrequentCorrection: TitleDesc{
				Title:       "Frequent corrections detected",
				Description: "You've been correcting me frequently. I'll pay closer attention to your preferences.",
			},
			ConfirmPreference: TitleDesc{
				Title:       "Confirm preference",
				Description: "Should I always follow this preference: \"${pattern}\"?",
			},
		},
		Context: ContextText{
			Preferences: "## Learned Preferences",
			ToolPrefs:   "## Tool Preferences",
			Workflow:    "## Workflow Patterns",
			WhatWorks:   "## What Works",
			Corrections: "## Recent Corrections",
			Insights:    "## Behavioral Insights",
			Notes:       "## Notes",
		},
	}
}
