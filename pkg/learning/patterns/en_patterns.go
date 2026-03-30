// MoonHub - Your ready-to-use AI assistant
// English patterns for pattern detection
// License: MIT

package patterns

import "regexp"

// initEnglishPatterns initializes English regex patterns
func initEnglishPatterns() PatternSet {
	return PatternSet{
		// Positive patterns - user acceptance signals
		Positive: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(perfect|great|thanks|exactly|nice|awesome|good|yes|correct)`),
			regexp.MustCompile(`(?i)^(that'?s? (right|correct|it|what i (wanted|needed)))`),
			regexp.MustCompile(`(?i)^(love it|nailed it|spot on)`),
			regexp.MustCompile(`(?i)^(works|solved|fixed|helpful|useful)`),
			regexp.MustCompile(`(?i)^(excellent|amazing|fantastic|wonderful)`),
		},

		// Negative patterns - user rejection signals
		Negative: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(no|wrong|not what|that'?s? not|you misunderstood|incorrect)`),
			regexp.MustCompile(`(?i)^(that'?s? (wrong|incorrect|not right))`),
			regexp.MustCompile(`(?i)^(try again|redo|not quite|doesn'?t work)`),
			regexp.MustCompile(`(?i)^(unhelpful|useless|bad|terrible|awful)`),
			regexp.MustCompile(`(?i)^(failed|error|mistake|problem)`),
		},

		// Correction patterns - user preference/correction signals
		Correction: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(actually|i meant|i prefer|next time|please don'?t|instead)`),
			regexp.MustCompile(`(?i)^i (prefer|like|want|need) (.+)`),
			regexp.MustCompile(`(?i)^don'?t ([^,]+), (instead )?(.+)`),
			regexp.MustCompile(`(?i)^(remember|note) that i (.+)`),
			regexp.MustCompile(`(?i)^(but however|on second thought|let me clarify)`),
		},

		// Workflow preference patterns
		Workflow: []*regexp.Regexp{
			regexp.MustCompile(`(?i)always (run|do|use|check) (.+) (before|after|when) (.+)`),
			regexp.MustCompile(`(?i)never (run|do|use) (.+) (when|if|unless) (.+)`),
			regexp.MustCompile(`(?i)(prefer|like) (it|to) (be|use|have) (.+)`),
			regexp.MustCompile(`(?i)(typically|usually|often|frequently) (.+)`),
		},

		// Tool preference patterns
		ToolPref: []*regexp.Regexp{
			regexp.MustCompile(`(?i)use (.+) instead of (.+)`),
			regexp.MustCompile(`(?i)don'?t use (.+)`),
			regexp.MustCompile(`(?i)(always|never) (run|call|use) (.+)`),
			regexp.MustCompile(`(?i)(prefer|avoid|skip) (using|running) (.+)`),
		},
	}
}

// initEnglishSemantics initializes English semantic indicators
func initEnglishSemantics() SemanticIndicators {
	return SemanticIndicators{
		Positive: []SemanticIndicator{
			{Keywords: []string{"helpful", "useful", "worked", "solved"}, Weight: 0.7},
			{Keywords: []string{"exactly", "perfect", "great", "thanks"}, Weight: 0.8},
			{Keywords: []string{"love it", "nailed it", "spot on"}, Weight: 0.75},
		},
		Negative: []SemanticIndicator{
			{Keywords: []string{"didn't work", "not helpful", "useless"}, Weight: 0.8},
			{Keywords: []string{"wrong", "incorrect", "error", "mistake"}, Weight: 0.8},
			{Keywords: []string{"confused", "don't understand", "unclear"}, Weight: 0.6},
		},
	}
}
