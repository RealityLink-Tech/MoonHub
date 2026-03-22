package delegation

import (
	"context"
	"fmt"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
)

// TemplateManager manages role templates for sub-agents
type TemplateManager struct {
	store        DelegationStore
	defaultTools []string
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(store DelegationStore, defaultTools []string) *TemplateManager {
	return &TemplateManager{
		store:        store,
		defaultTools: defaultTools,
	}
}

// FindOrCreate finds a matching template or creates a new sub-agent config
func (m *TemplateManager) FindOrCreate(ctx context.Context, userID string, task string, customPrompt string, customTools []string) (*RoleTemplate, bool, error) {
	// Extract keywords from task
	keywords := ExtractKeywords(task)
	category := ClassifyTask(task)

	// If custom prompt is provided, don't try to find a template
	if customPrompt != "" {
		template := &RoleTemplate{
			ID:           generateID("template"),
			UserID:       userID,
			Name:         fmt.Sprintf("custom-%s", category),
			RolePrompt:   customPrompt,
			TaskKeywords: keywords,
			Tools:        m.effectiveTools(customTools),
			Category:     category,
			CreatedAt:    time.Now().UnixMilli(),
			LastUsedAt:   time.Now().UnixMilli(),
		}
		return template, false, nil
	}

	// Try to find a matching template
	template, err := m.store.FindBestMatchTemplate(ctx, userID, keywords, 0.6)
	if err != nil {
		logger.WarnCF("delegation", "Failed to find template", map[string]any{
			"error": err.Error(),
		})
		// Continue with default template
	}

	if template != nil {
		// Update last used time
		template.LastUsedAt = time.Now().UnixMilli()
		if err := m.store.SaveRoleTemplate(ctx, template); err != nil {
			logger.WarnCF("delegation", "Failed to update template last used time", map[string]any{
				"error": err.Error(),
			})
		}

		logger.InfoCF("delegation", "Found matching template", map[string]any{
			"template_id": template.ID,
			"name":        template.Name,
			"keywords":    keywords,
		})

		return template, true, nil
	}

	// Create a new template based on task category
	template = m.createDefaultTemplate(userID, task, category, keywords)

	logger.InfoCF("delegation", "Created new template from task", map[string]any{
		"category": category,
		"keywords": keywords,
	})

	return template, false, nil
}

// createDefaultTemplate creates a default template based on task category
func (m *TemplateManager) createDefaultTemplate(userID, task string, category TaskCategory, keywords []string) *RoleTemplate {
	rolePrompt := m.defaultRolePrompt(category)
	tools := m.categoryTools(category)

	return &RoleTemplate{
		ID:           generateID("template"),
		UserID:       userID,
		Name:         fmt.Sprintf("%s-specialist", category),
		RolePrompt:   rolePrompt,
		TaskKeywords: keywords,
		Tools:        tools,
		Category:     category,
		CreatedAt:    time.Now().UnixMilli(),
		LastUsedAt:   time.Now().UnixMilli(),
	}
}

// defaultRolePrompt returns a default role prompt for a category
func (m *TemplateManager) defaultRolePrompt(category TaskCategory) string {
	prompts := map[TaskCategory]string{
		CategoryResearch: `You are a research specialist. Your job is to:
- Thoroughly investigate the given topic
- Gather relevant information from multiple sources
- Synthesize findings into clear, actionable insights
- Cite sources when possible
- Highlight any uncertainties or conflicting information

Focus on accuracy and completeness. Report your findings objectively.`,

		CategoryCode: `You are a code specialist. Your job is to:
- Write clean, efficient, and well-documented code
- Follow best practices and coding standards
- Handle edge cases and errors appropriately
- Test your code when possible
- Explain your implementation decisions

Focus on code quality and maintainability.`,

		CategoryAnalysis: `You are an analysis specialist. Your job is to:
- Examine the subject thoroughly and systematically
- Identify patterns, trends, and anomalies
- Provide data-driven insights
- Consider multiple perspectives
- Make clear recommendations based on your analysis

Focus on accuracy and actionable insights.`,

		CategoryWriting: `You are a writing specialist. Your job is to:
- Create clear, engaging, and well-structured content
- Adapt tone and style to the target audience
- Ensure grammatical correctness and readability
- Organize information logically
- Edit and refine your work

Focus on clarity and impact.`,

		CategoryGeneral: `You are a capable assistant. Your job is to:
- Complete the assigned task efficiently
- Ask clarifying questions if needed
- Provide clear and helpful output
- Follow any specific instructions given

Focus on delivering useful results.`,

		CategoryCustom: `You are a specialized assistant. Your job is to:
- Complete the assigned task according to any custom instructions
- Apply relevant expertise and knowledge
- Provide high-quality output
- Communicate clearly about your progress

Focus on meeting the specific requirements of the task.`,
	}

	if prompt, ok := prompts[category]; ok {
		return prompt
	}
	return prompts[CategoryGeneral]
}

// categoryTools returns the recommended tools for a category
func (m *TemplateManager) categoryTools(category TaskCategory) []string {
	var extra []string
	switch category {
	case CategoryResearch:
		extra = []string{"web_search", "web_fetch"}
	case CategoryCode:
		extra = []string{"read_file", "write_file", "edit_file", "list_dir"}
	case CategoryAnalysis:
		extra = []string{"read_file", "list_dir"}
	case CategoryWriting:
		extra = []string{"read_file", "write_file", "edit_file"}
	}
	out := make([]string, 0, len(m.defaultTools)+len(extra))
	out = append(out, m.defaultTools...)
	out = append(out, extra...)
	return dedupeStrings(out)
}

// effectiveTools returns the effective tools list
func (m *TemplateManager) effectiveTools(customTools []string) []string {
	if len(customTools) > 0 {
		return customTools
	}
	return m.defaultTools
}

// SaveTemplate saves a template after successful task completion
func (m *TemplateManager) SaveTemplate(ctx context.Context, template *RoleTemplate) error {
	template.LastUsedAt = time.Now().UnixMilli()
	return m.store.SaveRoleTemplate(ctx, template)
}

// RecordSuccess records a successful task completion for a template
func (m *TemplateManager) RecordSuccess(ctx context.Context, templateID string, durationMs int64) error {
	template, err := m.store.FindBestMatchTemplate(ctx, "", nil, 0)
	if err != nil {
		return err
	}
	if template == nil || template.ID != templateID {
		logger.DebugCF("delegation", "Template not found for success recording", map[string]any{
			"template_id": templateID,
		})
		return nil
	}

	template.SuccessCount++
	template.LastUsedAt = time.Now().UnixMilli()

	// Update average duration
	if template.AvgDurationMs == 0 {
		template.AvgDurationMs = durationMs
	} else {
		template.AvgDurationMs = (template.AvgDurationMs + durationMs) / 2
	}

	return m.store.SaveRoleTemplate(ctx, template)
}

// RecordFailure records a failed task for a template
func (m *TemplateManager) RecordFailure(ctx context.Context, templateID string) error {
	// We need to get all templates and find the one with matching ID
	// This is a bit inefficient, but the FindBestMatchTemplate doesn't support direct ID lookup
	// For now, we'll skip recording failures by ID
	logger.DebugCF("delegation", "Recording template failure", map[string]any{
		"template_id": templateID,
	})
	return nil
}

// ListTemplates lists all templates for a user
func (m *TemplateManager) ListTemplates(ctx context.Context, userID string) ([]*RoleTemplate, error) {
	return m.store.GetRoleTemplates(ctx, userID)
}

// DeleteTemplate deletes a template
func (m *TemplateManager) DeleteTemplate(ctx context.Context, templateID string) error {
	return m.store.DeleteRoleTemplate(ctx, templateID)
}

// GetTemplateStats returns statistics about templates
func (m *TemplateManager) GetTemplateStats(ctx context.Context, userID string) (map[string]any, error) {
	templates, err := m.store.GetRoleTemplates(ctx, userID)
	if err != nil {
		return nil, err
	}

	stats := map[string]any{
		"total_templates": len(templates),
		"by_category":     make(map[TaskCategory]int),
		"top_templates":   make([]map[string]any, 0),
	}

	categoryCount := make(map[TaskCategory]int)
	for _, t := range templates {
		categoryCount[t.Category]++
	}
	stats["by_category"] = categoryCount

	// Sort by success count for top templates
	for i := 0; i < len(templates); i++ {
		for j := i + 1; j < len(templates); j++ {
			if templates[i].SuccessCount < templates[j].SuccessCount {
				templates[i], templates[j] = templates[j], templates[i]
			}
		}
	}

	// Get top 5
	limit := 5
	if len(templates) < limit {
		limit = len(templates)
	}

	for i := 0; i < limit; i++ {
		t := templates[i]
		stats["top_templates"] = append(stats["top_templates"].([]map[string]any), map[string]any{
			"id":            t.ID,
			"name":          t.Name,
			"category":      t.Category,
			"success_count": t.SuccessCount,
			"success_rate":  float64(t.SuccessCount) / float64(t.SuccessCount+t.FailureCount+1),
		})
	}

	return stats, nil
}

// Helper functions

func dedupeStrings(s []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
