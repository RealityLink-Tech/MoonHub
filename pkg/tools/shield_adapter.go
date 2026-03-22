package tools

import "github.com/RealityLink-Tech/MoonHub/pkg/shield"

// shieldEngineAdapter bridges *shield.ShieldEngine to ShieldEvaluator without a
// circular import from pkg/shield to pkg/tools.
type shieldEngineAdapter struct {
	eng *shield.ShieldEngine
}

func (a *shieldEngineAdapter) IsActive() bool {
	return a.eng != nil && a.eng.IsActive()
}

func (a *shieldEngineAdapter) Evaluate(ev ShieldEvent) ShieldDecision {
	d := a.eng.Evaluate(shield.ShieldEvent{
		Scope:     shield.ShieldScope(ev.Scope),
		ToolName:  ev.ToolName,
		ToolArgs:  ev.ToolArgs,
		Domain:    ev.Domain,
		URL:       ev.URL,
		SkillName: ev.SkillName,
	})
	return ShieldDecision{
		Action:     ShieldAction(d.Action),
		Scope:      ShieldScope(d.Scope),
		ThreatID:   d.ThreatID,
		Reason:     d.Reason,
		MatchedOn:  d.MatchedOn,
		MatchValue: d.MatchValue,
	}
}

// NewShieldEvaluatorFromEngine returns a ShieldEvaluator backed by eng, or nil if eng is nil.
func NewShieldEvaluatorFromEngine(eng *shield.ShieldEngine) ShieldEvaluator {
	if eng == nil {
		return nil
	}
	return &shieldEngineAdapter{eng: eng}
}
