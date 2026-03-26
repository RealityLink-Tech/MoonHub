package shield

import (
	"fmt"
	"os"
	"sync"

	"github.com/RealityLink-Tech/MoonHub/pkg/zones"
)

const (
	// ConfidenceThreshold is the minimum confidence to enforce the threat's action.
	// Below this threshold, actions are downgraded to require_approval (unless critical).
	ConfidenceThreshold = 0.85
)

// ActionPriority defines the priority order for actions.
// Higher value = stronger action.
var ActionPriority = map[ShieldAction]int{
	ActionLog:             0,
	ActionRequireApproval: 1,
	ActionBlock:           2,
}

// ShieldEngine is the main threat evaluation engine.
type ShieldEngine struct {
	mu      sync.RWMutex
	threats []ThreatEntry
	matcher *Matcher
}

// NewEngine creates a new shield engine from parsed threat content.
func NewEngine(content string) *ShieldEngine {
	parser := NewParser()
	threats := parser.Parse(content)

	return &ShieldEngine{
		threats: threats,
		matcher: NewMatcher(),
	}
}

// NewEngineFromFile creates a shield engine from a SHIELD.md file.
// Returns a default engine if the file doesn't exist or can't be read.
func NewEngineFromFile(path string) (*ShieldEngine, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	engine := NewEngine(string(content))
	return engine, nil
}

// NewEngineWithDefaults creates a shield engine with default threats.
func NewEngineWithDefaults() *ShieldEngine {
	return NewEngine(DefaultThreatsContent)
}

// NewEngineFromFileOrDefault creates a shield engine from file, or falls back to defaults.
func NewEngineFromFileOrDefault(path string) *ShieldEngine {
	engine, err := NewEngineFromFile(path)
	if err != nil {
		// File doesn't exist or can't be read, use defaults
		return NewEngineWithDefaults()
	}
	return engine
}

// Evaluate evaluates an event against all active threats and returns a decision.
func (e *ShieldEngine) Evaluate(event ShieldEvent) ShieldDecision {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// If no threats loaded, return default log decision
	if len(e.threats) == 0 {
		return ShieldDecision{
			Action: ActionLog,
			Scope:  event.Scope,
			Reason: "no threats loaded",
		}
	}

	// Match event against all threats
	matches := e.matcher.MatchEvent(event, e.threats)

	// No matches, return default log decision
	if len(matches) == 0 {
		return ShieldDecision{
			Action: ActionLog,
			Scope:  event.Scope,
			Reason: "no threat matched",
		}
	}

	// Find the strongest action across all matches
	var strongestAction ShieldAction = ActionLog
	var strongestMatch *MatchResult

	for i := range matches {
		match := &matches[i]

		// Apply confidence threshold
		action := e.applyConfidenceThreshold(match.Threat)

		// Compare with current strongest
		if ActionPriority[action] > ActionPriority[strongestAction] {
			strongestAction = action
			strongestMatch = match
		}
	}

	// Build decision from strongest match
	if strongestMatch != nil {
		return ShieldDecision{
			Action:      strongestAction,
			Scope:       event.Scope,
			ThreatID:    strongestMatch.Threat.ID,
			Fingerprint: strongestMatch.Threat.Fingerprint,
			MatchedOn:   strongestMatch.MatchedOn,
			MatchValue:  strongestMatch.MatchValue,
			Reason:      e.buildReason(strongestMatch, strongestAction),
		}
	}

	return ShieldDecision{
		Action: strongestAction,
		Scope:  event.Scope,
		Reason: "threat detected",
	}
}

// applyConfidenceThreshold adjusts the action based on confidence level.
func (e *ShieldEngine) applyConfidenceThreshold(threat ThreatEntry) ShieldAction {
	// High confidence: use the threat's action directly
	if threat.Confidence >= ConfidenceThreshold {
		return threat.Action
	}

	// Low confidence with critical severity: still block
	if threat.Severity == SeverityCritical && threat.Action == ActionBlock {
		return ActionBlock
	}

	// Low confidence: downgrade to require_approval (except log stays log)
	if threat.Action == ActionLog {
		return ActionLog
	}

	return ActionRequireApproval
}

// buildReason builds a human-readable reason for the decision.
func (e *ShieldEngine) buildReason(match *MatchResult, action ShieldAction) string {
	threat := match.Threat

	var actionVerb string
	switch action {
	case ActionBlock:
		actionVerb = "Blocked"
	case ActionRequireApproval:
		actionVerb = "Requires approval for"
	case ActionLog:
		actionVerb = "Logged"
	}

	return fmt.Sprintf("%s by threat %s (%s): matched %s=%s",
		actionVerb,
		threat.ID,
		threat.Title,
		match.MatchedOn,
		match.MatchValue,
	)
}

// IsActive returns true if there are active threats loaded.
func (e *ShieldEngine) IsActive() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.threats) > 0
}

// GetThreats returns all loaded threat entries.
func (e *ShieldEngine) GetThreats() []ThreatEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.threats
}

// GetThreatCount returns the number of active threats.
func (e *ShieldEngine) GetThreatCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.threats)
}

// Reload reloads threats from new content.
func (e *ShieldEngine) Reload(content string) {
	parser := NewParser()
	threats := parser.Parse(content)

	e.mu.Lock()
	defer e.mu.Unlock()
	e.threats = threats
}

// ReloadFromFile reloads threats from a file.
func (e *ShieldEngine) ReloadFromFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	e.Reload(string(content))
	return nil
}

// AddThreat adds a new threat to the engine.
func (e *ShieldEngine) AddThreat(threat ThreatEntry) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.threats = append(e.threats, threat)
}

// RemoveThreat removes a threat by ID.
func (e *ShieldEngine) RemoveThreat(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, threat := range e.threats {
		if threat.ID == id {
			e.threats = append(e.threats[:i], e.threats[i+1:]...)
			return true
		}
	}
	return false
}

// EvaluateZoneAccess checks whether a zone access request should be allowed.
// This is used by the moonhub channel to enforce data partition rules.
// It returns a ShieldDecision with ActionBlock if access is denied.
func EvaluateZoneAccess(zone zones.Zone, rel zones.Relationship, access zones.AccessLevel, resource string) ShieldDecision {
	allowed := zone.Allow(rel, access)
	if !allowed {
		return ShieldDecision{
			Action:     ActionBlock,
			Scope:      ScopeFile,
			MatchedOn:  "zone:" + zone.String(),
			MatchValue: resource,
			Reason:     fmt.Sprintf("access denied: %s %s to %s zone resource %q", rel, access, zone, resource),
		}
	}
	return ShieldDecision{
		Action:     ActionLog,
		Scope:      ScopeFile,
		MatchedOn:  "zone:" + zone.String(),
		MatchValue: resource,
		Reason:     fmt.Sprintf("access allowed: %s %s to %s zone resource %q", rel, access, zone, resource),
	}
}
