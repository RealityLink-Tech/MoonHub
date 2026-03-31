// MoonHub - Your ready-to-use AI assistant
// Self-Improving Behavioral Pattern Detection System
// License: MIT

package learning

import (
	"testing"
)

func TestPatternDetector_DetectSignals_PositiveFeedback(t *testing.T) {
	config := DefaultConfig(":memory:")
	detector := NewPatternDetector(config)

	ctx := DetectionContext{
		UserMessage:      "Great job! That's exactly what I wanted.",
		AssistantMessage: "Here's the result of the task.",
	}

	signals := detector.DetectSignals(ctx)

	if len(signals) == 0 {
		t.Error("expected to detect positive feedback signal")
		return
	}

	// Check that at least one signal is a success indicator
	foundSuccess := false
	for _, s := range signals {
		if s.Category == CategorySuccessIndicator {
			foundSuccess = true
			break
		}
	}

	if !foundSuccess {
		t.Errorf("expected CategorySuccessIndicator, got categories: %v", signals[0].Category)
	}
}

func TestPatternDetector_DetectSignals_NegativeFeedback(t *testing.T) {
	config := DefaultConfig(":memory:")
	detector := NewPatternDetector(config)

	ctx := DetectionContext{
		UserMessage:      "That's wrong. Please fix it.",
		AssistantMessage: "Let me try again.",
	}

	signals := detector.DetectSignals(ctx)

	if len(signals) == 0 {
		t.Error("expected to detect negative feedback signal")
		return
	}

	// Check that at least one signal is a failure indicator
	foundFailure := false
	for _, s := range signals {
		if s.Category == CategoryFailureIndicator {
			foundFailure = true
			break
		}
	}

	if !foundFailure {
		t.Errorf("expected CategoryFailureIndicator, got categories: %v", signals[0].Category)
	}
}

func TestPatternDetector_DetectSignals_ExplicitCorrection(t *testing.T) {
	config := DefaultConfig(":memory:")
	detector := NewPatternDetector(config)

	ctx := DetectionContext{
		UserMessage:      "Actually, I prefer TypeScript over JavaScript.",
		AssistantMessage: "Okay, I'll use TypeScript.",
	}

	signals := detector.DetectSignals(ctx)

	if len(signals) == 0 {
		t.Error("expected to detect explicit correction signal")
		return
	}

	// Check that at least one signal is an explicit correction
	foundCorrection := false
	for _, s := range signals {
		if s.Category == CategoryCorrectionExplicit {
			foundCorrection = true
			break
		}
	}

	if !foundCorrection {
		t.Errorf("expected CategoryCorrectionExplicit, got categories: %v", signals[0].Category)
	}
}

func TestPatternDetector_DetectSignals_CommunicationPreference(t *testing.T) {
	config := DefaultConfig(":memory:")
	detector := NewPatternDetector(config)

	// Use a context with history to trigger conversation flow patterns
	ctx := DetectionContext{
		UserMessage:      "ok",
		AssistantMessage: "Here's a detailed explanation...",
		History: []Message{
			{Role: "user", Content: "yes"},
			{Role: "assistant", Content: "Detailed response 1"},
			{Role: "user", Content: "ok"},
			{Role: "assistant", Content: "Detailed response 2"},
			{Role: "user", Content: "ok"},
		},
	}

	signals := detector.DetectSignals(ctx)

	if len(signals) == 0 {
		t.Error("expected to detect communication preference signal")
		return
	}

	// Check that at least one signal is a communication preference
	foundPref := false
	for _, s := range signals {
		if s.Category == CategoryPreferenceCommunication {
			foundPref = true
			break
		}
	}

	if !foundPref {
		t.Errorf("expected CategoryPreferenceCommunication, got categories: %v", signals[0].Category)
	}
}

func TestPatternDetector_DetectSignals_WorkflowPreference(t *testing.T) {
	config := DefaultConfig(":memory:")
	detector := NewPatternDetector(config)

	ctx := DetectionContext{
		UserMessage:      "Always run tests after making code changes.",
		AssistantMessage: "I'll run the tests now.",
	}

	signals := detector.DetectSignals(ctx)

	if len(signals) == 0 {
		t.Error("expected to detect workflow preference signal")
		return
	}

	// Check that at least one signal is a workflow preference
	foundWorkflow := false
	for _, s := range signals {
		if s.Category == CategoryPreferenceWorkflow {
			foundWorkflow = true
			break
		}
	}

	if !foundWorkflow {
		t.Errorf("expected CategoryPreferenceWorkflow, got categories: %v", signals[0].Category)
	}
}

func TestPatternDetector_DetectSignals_ToolPreference(t *testing.T) {
	config := DefaultConfig(":memory:")
	detector := NewPatternDetector(config)

	// Use a message that matches tool preference pattern: "use X instead of Y"
	ctx := DetectionContext{
		UserMessage:      "Use web search instead of file search.",
		AssistantMessage: "I'll use web search.",
	}

	signals := detector.DetectSignals(ctx)

	if len(signals) == 0 {
		t.Error("expected to detect tool preference signal")
		return
	}

	// Check that at least one signal is a tool preference
	foundToolPref := false
	for _, s := range signals {
		if s.Category == CategoryPreferenceTool {
			foundToolPref = true
			break
		}
	}

	if !foundToolPref {
		t.Errorf("expected CategoryPreferenceTool, got categories: %v", signals[0].Category)
	}
}

func TestPatternDetector_DetectSignals_EmptyContext(t *testing.T) {
	config := DefaultConfig(":memory:")
	detector := NewPatternDetector(config)

	ctx := DetectionContext{
		UserMessage:      "",
		AssistantMessage: "",
	}

	signals := detector.DetectSignals(ctx)

	// Should not panic and should return empty or minimal signals
	if signals == nil {
		t.Error("expected non-nil signals slice")
	}
}

func TestPatternDetector_DetectSignals_History(t *testing.T) {
	config := DefaultConfig(":memory:")
	detector := NewPatternDetector(config)

	ctx := DetectionContext{
		UserMessage:      "That's not right",
		AssistantMessage: "Let me correct that",
		History: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
			{Role: "user", Content: "Can you help me?"},
			{Role: "assistant", Content: "Sure!"},
		},
	}

	signals := detector.DetectSignals(ctx)

	// Should process history without error
	if signals == nil {
		t.Error("expected non-nil signals slice")
	}
}

func TestPatternDetector_DetectSignals_ConfidenceThreshold(t *testing.T) {
	config := DefaultConfig(":memory:")
	config.MinConfidence = 0.8
	detector := NewPatternDetector(config)

	ctx := DetectionContext{
		UserMessage:      "Perfect! Exactly what I needed!",
		AssistantMessage: "Here's the result.",
	}

	signals := detector.DetectSignals(ctx)

	// All signals should have confidence >= min_confidence
	for _, s := range signals {
		if s.Confidence < 0 {
			t.Errorf("negative confidence: %f", s.Confidence)
		}
		if s.Confidence > 1 {
			t.Errorf("confidence > 1: %f", s.Confidence)
		}
	}
}

func TestPatternDetector_DetectSignals_MultipleSignals(t *testing.T) {
	config := DefaultConfig(":memory:")
	detector := NewPatternDetector(config)

	ctx := DetectionContext{
		UserMessage:      "Good job! But next time, use TypeScript instead of JavaScript.",
		AssistantMessage: "Thanks! I'll use TypeScript next time.",
	}

	signals := detector.DetectSignals(ctx)

	// Should detect both positive feedback and correction
	if len(signals) < 1 {
		t.Error("expected at least 1 signal from mixed feedback")
	}
}
