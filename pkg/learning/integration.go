// MoonHub - Your ready-to-use AI assistant
// Self-Improving Behavioral Pattern Detection System
// License: MIT

package learning

import (
	"fmt"
	"strings"

	"github.com/RealityLink-Tech/MoonHub/pkg/adaptive_memory"
	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
)

// Integration provides integration helpers for the learning engine
// with MoonHub's agent system

// AgentIntegration wraps the learning engine for agent integration
type AgentIntegration struct {
	engine         *Engine
	adaptiveMemory *adaptive_memory.Engine
	userID         string
}

// NewAgentIntegration creates a new agent integration wrapper
func NewAgentIntegration(engine *Engine, adaptiveMemory *adaptive_memory.Engine, userID string) *AgentIntegration {
	return &AgentIntegration{
		engine:         engine,
		adaptiveMemory: adaptiveMemory,
		userID:         userID,
	}
}

// AnalyzeConversation analyzes a conversation after response generation
func (a *AgentIntegration) AnalyzeConversation(
	userMessage string,
	assistantMessage string,
	history []Message,
	toolCalls []ToolCall,
	toolResults []ToolResult,
) AnalysisResult {
	// Build analysis context
	ctx := AnalysisContext{
		UserID:            a.userID,
		SessionID:         "", // Would be passed from agent
		MessageHistory:    history,
		RecentToolCalls:   toolCalls,
		RecentToolResults: toolResults,
	}

	// Ensure last messages are included
	if len(ctx.MessageHistory) == 0 || ctx.MessageHistory[len(ctx.MessageHistory)-1].Content != userMessage {
		ctx.MessageHistory = append(ctx.MessageHistory, Message{
			Role:    "user",
			Content: userMessage,
		})
	}
	ctx.MessageHistory = append(ctx.MessageHistory, Message{
		Role:    "assistant",
		Content: assistantMessage,
	})

	// Run analysis
	result := a.engine.Analyze(ctx)

	// Sync significant patterns to adaptive memory
	if a.adaptiveMemory != nil {
		for _, signal := range result.Signals {
			if signal.Confidence >= 0.8 {
				eventType := adaptive_memory.EventTypePreferenceLearned
				if signal.Category == CategoryCorrectionExplicit {
					eventType = adaptive_memory.EventTypeCorrection
				} else if signal.Category == CategoryFailureIndicator {
					eventType = adaptive_memory.EventTypeErrorLearned
				} else if signal.Category == CategorySuccessIndicator {
					eventType = adaptive_memory.EventTypeUserFeedback
				}

				_, err := a.adaptiveMemory.RecordEvent(a.userID, adaptive_memory.EventInput{
					Type:       eventType,
					Content:    signal.Context,
					Importance: &signal.Confidence,
				})
				if err != nil {
					logger.DebugCF("learning", "Failed to record pattern to adaptive memory",
						map[string]any{"error": err.Error()})
				}
			}
		}
	}

	return result
}

// RecordToolCall records a tool call for tracking
func (a *AgentIntegration) RecordToolCall(toolName string, success bool, durationMs int64, context string) {
	record := ToolExecutionRecord{
		ToolName:    toolName,
		Success:     success,
		DurationMs:  durationMs,
		UserContext: context,
		Timestamp:   NowMs(),
	}
	a.engine.RecordToolExecution(record)
}

// RecordUserFeedback records user feedback on a tool result
func (a *AgentIntegration) RecordUserFeedback(toolName string, accepted bool) {
	if accepted {
		a.engine.RecordToolAcceptance(toolName)
	} else {
		a.engine.RecordToolRejection(toolName)
	}
}

// GetLearnedContext returns formatted context for prompt injection
func (a *AgentIntegration) GetLearnedContext() string {
	return a.engine.GetContextString()
}

// GetToolPreferences returns tool preference context
func (a *AgentIntegration) GetToolPreferences() string {
	ctx := a.engine.GetContext()
	if ctx.ToolPreferences == "" {
		return ""
	}
	return "## Tool Preferences\n\n" + ctx.ToolPreferences
}

// GetWorkflowPatterns returns workflow pattern context
func (a *AgentIntegration) GetWorkflowPatterns() string {
	ctx := a.engine.GetContext()
	if ctx.WorkflowPatterns == "" {
		return ""
	}
	return "## Workflow Patterns\n\n" + ctx.WorkflowPatterns
}

// GetRecentCorrections returns recent corrections context
func (a *AgentIntegration) GetRecentCorrections() string {
	ctx := a.engine.GetContext()
	if ctx.RecentCorrections == "" {
		return ""
	}
	return "## Recent Corrections\n\n" + ctx.RecentCorrections
}

// GetBehavioralInsights returns behavioral insights
func (a *AgentIntegration) GetBehavioralInsights() string {
	ctx := a.engine.GetContext()
	if ctx.BehavioralInsights == "" {
		return ""
	}
	return "## Behavioral Insights\n\n" + ctx.BehavioralInsights
}

// GetProactiveNotes returns proactive notes
func (a *AgentIntegration) GetProactiveNotes() string {
	ctx := a.engine.GetContext()
	if ctx.ProactiveNotes == "" {
		return ""
	}
	return "## Notes\n\n" + ctx.ProactiveNotes
}

// GetStats returns learning engine statistics
func (a *AgentIntegration) GetStats() Stats {
	return a.engine.GetStats()
}

// GetSuggestions returns active suggestions
func (a *AgentIntegration) GetSuggestions() []Suggestion {
	return a.engine.GetSuggestions()
}

// DismissSuggestion dismisses a suggestion
func (a *AgentIntegration) DismissSuggestion(id string) bool {
	return a.engine.DismissSuggestion(id)
}

// ApplySuggestion applies a suggestion
func (a *AgentIntegration) ApplySuggestion(id string) bool {
	return a.engine.ApplySuggestion(id)
}

// RunEvolution runs pattern evolution (should be called periodically)
func (a *AgentIntegration) RunEvolution() EvolutionResult {
	result := a.engine.Evolve()

	logger.DebugCF("learning", "Pattern evolution completed",
		map[string]any{
			"decayed":     result.PatternsDecayed,
			"merged":      result.PatternsMerged,
			"pruned":      result.PatternsPruned,
			"generalized": result.PatternsGeneralized,
		})

	return result
}

// GetPreferredTools returns a list of preferred tools based on user preference score
func (a *AgentIntegration) GetPreferredTools(minPreference float64) []string {
	toolUsage := a.engine.GetToolUsagePatterns()
	var preferred []string

	for _, tool := range toolUsage {
		if tool.UserPreference >= minPreference && tool.TotalCalls >= 3 {
			preferred = append(preferred, tool.ToolName)
		}
	}

	return preferred
}

// GetAvoidedTools returns a list of tools to avoid based on low preference or high failure rate
func (a *AgentIntegration) GetAvoidedTools() []string {
	toolUsage := a.engine.GetToolUsagePatterns()
	var avoided []string

	for _, tool := range toolUsage {
		if tool.UserPreference < -0.3 || (tool.TotalCalls >= 5 && tool.SuccessfulCalls < tool.TotalCalls/3) {
			avoided = append(avoided, tool.ToolName)
		}
	}

	return avoided
}

// ShouldUseTool returns a recommendation on whether to use a specific tool
func (a *AgentIntegration) ShouldUseTool(toolName string) (bool, string) {
	toolUsage := a.engine.GetToolUsagePatterns()

	for _, tool := range toolUsage {
		if tool.ToolName == toolName {
			// Check user preference
			if tool.UserPreference < -0.5 {
				return false, "User has shown strong negative preference for this tool"
			}

			// Check success rate
			if tool.TotalCalls >= 5 {
				successRate := float64(tool.SuccessfulCalls) / float64(tool.TotalCalls)
				if successRate < 0.3 {
					return false, fmt.Sprintf("Low success rate (%.0f%%)", successRate*100)
				}
			}

			// Check rejection rate
			if tool.TotalCalls >= 3 {
				rejectionRate := float64(tool.UserRejectedCalls) / float64(tool.TotalCalls)
				if rejectionRate > 0.5 {
					return false, "High user rejection rate"
				}
			}

			// Tool is acceptable
			if tool.UserPreference > 0.3 {
				return true, "User has shown positive preference for this tool"
			}

			return true, ""
		}
	}

	// No data available, allow by default
	return true, ""
}

// GetBehavioralScore returns the current behavioral score
func (a *AgentIntegration) GetBehavioralScore() BehavioralScore {
	return a.engine.GetBehavioralScore()
}

// Close closes the integration
func (a *AgentIntegration) Close() error {
	return a.engine.Close()
}

// BuildLearnedContextForPrompt builds a context section for LLM prompts
func (a *AgentIntegration) BuildLearnedContextForPrompt(query string) string {
	var sections []string

	// Get base context
	baseCtx := a.engine.GetContext()

	// Add preferences if relevant to query
	if baseCtx.Preferences != "" && a.isRelevantToQuery(query, baseCtx.Preferences) {
		sections = append(sections, "### User Preferences\n"+baseCtx.Preferences)
	}

	// Add tool preferences if query mentions tools
	if baseCtx.ToolPreferences != "" && strings.Contains(strings.ToLower(query), "tool") {
		sections = append(sections, "### Tool Preferences\n"+baseCtx.ToolPreferences)
	}

	// Add workflow patterns
	if baseCtx.WorkflowPatterns != "" {
		sections = append(sections, "### Workflow Patterns\n"+baseCtx.WorkflowPatterns)
	}

	// Add recent corrections (always relevant)
	if baseCtx.RecentCorrections != "" {
		sections = append(sections, "### Recent Corrections\n"+baseCtx.RecentCorrections)
	}

	// Add behavioral insights
	if baseCtx.BehavioralInsights != "" {
		sections = append(sections, "### Performance Insights\n"+baseCtx.BehavioralInsights)
	}

	// Add proactive notes
	if baseCtx.ProactiveNotes != "" {
		sections = append(sections, "### Notes\n"+baseCtx.ProactiveNotes)
	}

	if len(sections) == 0 {
		return ""
	}

	return "## Learning Context\n\n" + strings.Join(sections, "\n\n")
}

// isRelevantToQuery checks if context is relevant to the query
func (a *AgentIntegration) isRelevantToQuery(query string, context string) bool {
	// Simple keyword matching
	queryLower := strings.ToLower(query)
	contextLower := strings.ToLower(context)

	queryWords := strings.Fields(queryLower)
	contextWords := strings.Fields(contextLower)

	matchCount := 0
	for _, qWord := range queryWords {
		if len(qWord) < 3 {
			continue
		}
		for _, cWord := range contextWords {
			if qWord == cWord {
				matchCount++
				break
			}
		}
	}

	// Consider relevant if at least 2 words match
	return matchCount >= 2
}
