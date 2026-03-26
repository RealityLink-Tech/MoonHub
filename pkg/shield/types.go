// Package shield provides runtime threat evaluation and enforcement for MoonHub.
// It parses threat definitions from SHIELD.md and evaluates agent actions against them.
package shield

import "time"

// ShieldAction represents the enforcement action to take when a threat matches.
type ShieldAction string

const (
	// ActionBlock prevents the action from executing and returns an error.
	ActionBlock ShieldAction = "block"
	// ActionRequireApproval flags the action for user approval (future use).
	ActionRequireApproval ShieldAction = "require_approval"
	// ActionLog allows the action but logs the threat detection.
	ActionLog ShieldAction = "log"
)

// ThreatSeverity represents the severity level of a threat.
type ThreatSeverity string

const (
	SeverityCritical ThreatSeverity = "critical"
	SeverityHigh     ThreatSeverity = "high"
	SeverityMedium   ThreatSeverity = "medium"
	SeverityLow      ThreatSeverity = "low"
)

// ThreatCategory represents the category of threat.
type ThreatCategory string

const (
	CategoryTool    ThreatCategory = "tool"
	CategorySkill   ThreatCategory = "skill"
	CategoryMemory  ThreatCategory = "memory"
	CategoryNetwork ThreatCategory = "network"
	CategoryMessage ThreatCategory = "message"
	CategoryFile    ThreatCategory = "file"
	CategoryPrompt  ThreatCategory = "prompt"
	CategoryOther   ThreatCategory = "other"
)

// ShieldScope represents the event scope being evaluated.
type ShieldScope string

const (
	ScopeToolCall      ShieldScope = "tool.call"
	ScopeSkillInstall  ShieldScope = "skill.install"
	ScopeSkillExecute  ShieldScope = "skill.execute"
	ScopeNetworkEgress ShieldScope = "network.egress"
	ScopeSecretsRead   ShieldScope = "secrets.read"
	ScopePrompt        ShieldScope = "prompt"
	ScopeFile          ShieldScope = "file"
)

// ThreatEntry represents a parsed threat definition from SHIELD.md.
type ThreatEntry struct {
	// ID is the unique identifier for the threat (e.g., THREAT-001).
	ID string `json:"id"`
	// Fingerprint is a short identifier for the threat pattern.
	Fingerprint string `json:"fingerprint"`
	// Category classifies the threat type.
	Category ThreatCategory `json:"category"`
	// Severity indicates the threat's potential impact.
	Severity ThreatSeverity `json:"severity"`
	// Confidence is the detection confidence (0.0 to 1.0).
	Confidence float64 `json:"confidence"`
	// Action is the enforcement action to take.
	Action ShieldAction `json:"action"`
	// Title is a short human-readable title.
	Title string `json:"title"`
	// Description provides detailed information about the threat.
	Description string `json:"description"`
	// RecommendationAgent contains the condition directives (BLOCK/APPROVE/LOG lines).
	RecommendationAgent string `json:"recommendation_agent"`
	// ExpiresAt is when this threat entry expires (optional).
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// Revoked indicates if this threat is no longer active.
	Revoked bool `json:"revoked"`
	// RevokedAt is when this threat was revoked.
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// ShieldEvent represents an event to evaluate against threats.
type ShieldEvent struct {
	// Scope is the type of event being evaluated.
	Scope ShieldScope `json:"scope"`
	// ToolName is the name of the tool being called (for tool.call scope).
	ToolName string `json:"tool_name,omitempty"`
	// ToolArgs are the arguments passed to the tool.
	ToolArgs map[string]any `json:"tool_args,omitempty"`
	// Domain is the target domain (for network.egress scope).
	Domain string `json:"domain,omitempty"`
	// SecretPath is the path being accessed (for secrets.read scope).
	SecretPath string `json:"secret_path,omitempty"`
	// SkillName is the skill being invoked (for skill.* scopes).
	SkillName string `json:"skill_name,omitempty"`
	// InputText is the user input (for prompt scope).
	InputText string `json:"input_text,omitempty"`
	// FilePath is the file path being accessed (for file operations).
	FilePath string `json:"file_path,omitempty"`
	// URL is the target URL (for network.egress scope).
	URL string `json:"url,omitempty"`
	// ApprovalID is the approval request ID (if approval was requested).
	ApprovalID string `json:"approval_id,omitempty"`
}

// ShieldDecision represents the enforcement decision after evaluation.
type ShieldDecision struct {
	// Action is the enforcement action to take.
	Action ShieldAction `json:"action"`
	// Scope is the event scope that was evaluated.
	Scope ShieldScope `json:"scope"`
	// ThreatID is the ID of the matched threat (if any).
	ThreatID string `json:"threat_id,omitempty"`
	// Fingerprint is the fingerprint of the matched threat.
	Fingerprint string `json:"fingerprint,omitempty"`
	// MatchedOn describes what was matched.
	MatchedOn string `json:"matched_on,omitempty"`
	// MatchValue is the specific value that triggered the match.
	MatchValue string `json:"match_value,omitempty"`
	// Reason provides a human-readable explanation.
	Reason string `json:"reason"`
	// ApprovalRequest is the approval request details (for require_approval action).
	ApprovalRequest *ApprovalRequest `json:"approval_request,omitempty"`
}

// Directive represents a parsed BLOCK/APPROVE/LOG directive from recommendation_agent.
type Directive struct {
	// Action is the enforcement action.
	Action ShieldAction `json:"action"`
	// Condition is the condition string to evaluate.
	Condition string `json:"condition"`
}

// MatchResult represents a successful threat match.
type MatchResult struct {
	// Threat is the matched threat entry.
	Threat ThreatEntry `json:"threat"`
	// Directive is the specific directive that matched.
	Directive Directive `json:"directive"`
	// MatchedOn describes what was matched.
	MatchedOn string `json:"matched_on"`
	// MatchValue is the specific value that triggered the match.
	MatchValue string `json:"match_value"`
}

// Engine is the interface for shield evaluation.
type Engine interface {
	// Evaluate evaluates an event against all active threats.
	Evaluate(event ShieldEvent) ShieldDecision
	// IsActive returns true if there are active threats loaded.
	IsActive() bool
	// GetThreats returns all loaded threat entries.
	GetThreats() []ThreatEntry
	// GetThreatCount returns the number of active threats.
	GetThreatCount() int
}

// ValidCategories returns all valid threat categories.
func ValidCategories() map[ThreatCategory]bool {
	return map[ThreatCategory]bool{
		CategoryTool:    true,
		CategorySkill:   true,
		CategoryMemory:  true,
		CategoryNetwork: true,
		CategoryMessage: true,
		CategoryFile:    true,
		CategoryPrompt:  true,
		CategoryOther:   true,
	}
}

// ValidSeverities returns all valid threat severities.
func ValidSeverities() map[ThreatSeverity]bool {
	return map[ThreatSeverity]bool{
		SeverityCritical: true,
		SeverityHigh:     true,
		SeverityMedium:   true,
		SeverityLow:      true,
	}
}

// ValidActions returns all valid enforcement actions.
func ValidActions() map[ShieldAction]bool {
	return map[ShieldAction]bool{
		ActionBlock:           true,
		ActionRequireApproval: true,
		ActionLog:             true,
	}
}
