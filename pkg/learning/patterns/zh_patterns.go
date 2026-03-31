// MoonHub - Your ready-to-use AI assistant
// Chinese patterns for pattern detection
// License: MIT

package patterns

import "regexp"

// initChinesePatterns initializes Chinese regex patterns
func initChinesePatterns() PatternSet {
	return PatternSet{
		// ========== 正面反馈模式 = ==========

		// 匡程度肯定 - 用户明确表示满意
		Positive: []*regexp.Regexp{
			// 基础肯定
			regexp.MustCompile(`(?i)^(好的?|行|可以|没问题|OK|ok)$`),
			regexp.MustCompile(`(?i)^(很好?|棒|赞|牛|给力|厉害|牛逼|666)$`),
			regexp.MustCompile(`(?i)^(完美|太棒了?|太好了?|太赞了?|绝了)$`),
			regexp.MustCompile(`(?i)^(谢谢|感谢|多谢|谢了|辛苦了)$`),
			regexp.MustCompile(`(?i)^(对|是的|没错|正确|正是|就是)$`),
			regexp.MustCompile(`(?i)^(正是我想要的?|这就是我要的|符合预期)$`),
			// 满意表达
			regexp.MustCompile(`(?i)^(靠谱|稳|到位|舒服|顺滑)$`),
			regexp.MustCompile(`(?i)^(就这个|这个对|这个好|要这个)$`),
			regexp.MustCompile(`(?i)^(继续|好的继续|没问题继续)$`),
			// 简短肯定
			regexp.MustCompile(`(?i)^(嗯|嗯嗯|嗯呐|昂|噢了)$`),
			// 高级赞扬
			regexp.MustCompile(`(?i)^(太棒了?|太牛了?绝了?给力)`),
			regexp.MustCompile(`(?i)^(省事|方便|快捷|高效)`),
			regexp.MustCompile(`(?i)^(专业|稳当|完美)`),
		},

		// ========== 负面反馈模式 = ==========

		Negative: []*regexp.Regexp{
			// 基础否定
			regexp.MustCompile(`(?i)^(不对|错了?|不是|不行|不可以)$`),
			regexp.MustCompile(`(?i)^(错误|不正确|有误|有问题)$`),
			regexp.MustCompile(`(?i)^(不是这个|不是我要的|理解错了?|会错意了?)$`),
			regexp.MustCompile(`(?i)^(再试(一次)?|重来|重新来|换个)$`),
			regexp.MustCompile(`(?i)^(不好|太差|差劲|垃圾|废)$`),
			// 委婉拒绝
			regexp.MustCompile(`(?i)^(不太对|不太行|差点意思|差点火候)$`),
			regexp.MustCompile(`(?i)^(一般|还行吧|马马虎虎|凑合)$`),
			regexp.MustCompile(`(?i)^(没帮上忙?|没解决|还是有问题)$`),
			// 沮丧表达
			regexp.MustCompile(`(?i)^(浪费时间|没意思|无语)$`),
		},

		// ========== 纠正/偏好模式 = ==========
		Correction: []*regexp.Regexp{
			// 澄清意图
			regexp.MustCompile(`(?i)^(其实|实际上|说真的|老实说)`),
			regexp.MustCompile(`(?i)^(我的意思(是|不是说)|我是说)`),
			regexp.MustCompile(`(?i)^(我更(喜欢|希望|想要|倾向于))`),
			regexp.MustCompile(`(?i)^(下次|以后|之后|以后记得)`),
			regexp.MustCompile(`(?i)^(不要|别|不用|不需要)`),
			regexp.MustCompile(`(?i)^(相反|反过来|倒过来|反过来)`),
			regexp.MustCompile(`(?i)^(记住(我|我喜欢|我想要)|记得)`),
			regexp.MustCompile(`(?i)^(最好是|最好是能|希望)`),
			regexp.MustCompile(`(?i)^(能不能|可以不可以|麻烦)`),
			regexp.MustCompile(`(?i)^(改(成|为)|换成|修改成)`),
			regexp.MustCompile(`(?i)^(你(应该|最好|能不能))`),
		},

		// ========== 工作流偏好模式 = ==========
		Workflow: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(每次|总是|一直|老)`),
			regexp.MustCompile(`(?i)(从不|从没|从来不)`),
			regexp.MustCompile(`(?i)(当.+时|在.+之前|在.+之后)`),
			regexp.MustCompile(`(?i)(喜欢用|习惯用|常用|爱用)`),
			regexp.MustCompile(`(?i)(先.+再|先.+然后|先.+接着)`),
			regexp.MustCompile(`(?i)(必须|一定要|务必)`),
		},

		// ========== 工具偏好模式 = ==========
		ToolPref: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(用.+代替|换成|改用)`),
			regexp.MustCompile(`(?i)(不要用|别用|不用)`),
			regexp.MustCompile(`(?i)(总是用|一直用|每次都用)`),
			regexp.MustCompile(`(?i)(从不用|从不使用|从来不用)`),
			regexp.MustCompile(`(?i)(优先(使用|选|用))`),
		},
	}
}

// initChineseSemantics initializes Chinese semantic indicators
func initChineseSemantics() SemanticIndicators {
	return SemanticIndicators{
		Positive: []SemanticIndicator{
			{Keywords: []string{"有帮助", "有用", "实用", "好用"}, Weight: 0.7},
			{Keywords: []string{"解决了", "搞定了", "完成了", "修好了"}, Weight: 0.8},
			{Keywords: []string{"谢谢", "感谢", "多谢", "辛苦了"}, Weight: 0.6},
			{Keywords: []string{"太棒了", "太好了", "太牛了", "给力"}, Weight: 0.75},
			{Keywords: []string{"省事", "方便", "快捷", "高效"}, Weight: 0.65},
			{Keywords: []string{"靠谱", "稳", "专业", "到位"}, Weight: 0.7},
		},
		Negative: []SemanticIndicator{
			{Keywords: []string{"没帮助", "没用", "无效", "白费"}, Weight: 0.8},
			{Keywords: []string{"错了", "不对", "有误", "出错了"}, Weight: 0.8},
			{Keywords: []string{"失败", "不行", "没成功", "搞不定"}, Weight: 0.75},
			{Keywords: []string{"困惑", "糊涂", "搞不懂", "不明白"}, Weight: 0.6},
			{Keywords: []string{"麻烦", "复杂", "繁琐", "费劲"}, Weight: 0.55},
			{Keywords: []string{"慢", "卡", "久等", "等很久"}, Weight: 0.5},
		},
	}
}
