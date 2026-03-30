// MoonHub - Your ready-to-use AI assistant
// Chinese text resources for learning system
// License: MIT

package i18n

// initChinese initializes Chinese text resources
func initChinese() TextResources {
	return TextResources{
		Suggestions: SuggestionsText{
			ToolLowSuccess: TitleDesc{
				Title:       "检查 ${tool} 使用情况",
				Description: "该工具成功率较低，建议考虑其他方案。",
			},
			ToolUserPreference: TitleDesc{
				Title:       "工具偏好已记录",
				Description: "检测到您不太喜欢使用此工具，之后会相应调整使用策略。",
			},
			ToolUsageTrend: TitleDesc{
				Title:       "工具使用趋势已发现",
				Description: "我注意到您使用某些工具的模式，是否需要优化?",
			},
			WorkflowPattern: TitleDesc{
				Title:       "工作流模式已发现",
				Description: "我注意到您经常 ${pattern}，是否将其设为默认行为?",
			},
			ImproveAdaptation: TitleDesc{
				Title:       "改进适应能力",
				Description: "我似乎对您的纠正反应较慢，之后会更积极地响应您的反馈。",
			},
			ResponseQuality: TitleDesc{
				Title:       "响应质量下降",
				Description: "最近的响应可能未达到您的预期，我会专注于提高质量。",
			},
			FrequentCorrection: TitleDesc{
				Title:       "频繁纠正已检测",
				Description: "您最近经常纠正我，我会更加关注您的偏好。",
			},
			ConfirmPreference: TitleDesc{
				Title:       "确认偏好设置",
				Description: "是否将此偏好设为默认行为：「${pattern}」?",
			},
		},
		Context: ContextText{
			Preferences: "## 已学习的偏好",
			ToolPrefs:   "## 工具偏好",
			Workflow:    "## 工作流模式",
			WhatWorks:   "## 有效做法",
			Corrections: "## 最近修正",
			Insights:    "## 行为洞察",
			Notes:       "## 备注",
		},
	}
}
