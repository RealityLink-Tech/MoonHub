package routing

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
)

// defaultThreshold is used when the config threshold is zero or negative.
// At 0.35 a message needs at least one strong signal (code block, long text,
// or an attachment) before the heavy model is chosen.
const defaultThreshold = 0.35

// RouterConfig holds the validated model routing settings.
// It mirrors config.RoutingConfig but lives in pkg/routing to keep the
// dependency graph simple: pkg/agent resolves config → routing, not the reverse.
type RouterConfig struct {
	// LightModel is the model_name (from model_list) used for simple tasks.
	LightModel string

	// Threshold is the complexity score cutoff in [0, 1].
	// score >= Threshold → primary (heavy) model.
	// score <  Threshold → light model.
	Threshold float64
}

// Router selects the appropriate model tier for each incoming message.
// It is safe for concurrent use from multiple goroutines.
type Router struct {
	cfg        RouterConfig
	classifier Classifier
}

// New creates a Router with the given config and the default RuleClassifier.
// If cfg.Threshold is zero or negative, defaultThreshold (0.35) is used.
func New(cfg RouterConfig) *Router {
	if cfg.Threshold <= 0 {
		cfg.Threshold = defaultThreshold
	}
	return &Router{
		cfg:        cfg,
		classifier: &RuleClassifier{},
	}
}

// newWithClassifier creates a Router with a custom Classifier.
// Intended for unit tests that need to inject a deterministic scorer.
func newWithClassifier(cfg RouterConfig, c Classifier) *Router {
	if cfg.Threshold <= 0 {
		cfg.Threshold = defaultThreshold
	}
	return &Router{cfg: cfg, classifier: c}
}

// SelectModel returns the model to use for this conversation turn along with
// the computed complexity score (for logging and debugging).
//
//   - If score < cfg.Threshold: returns (cfg.LightModel, true, score)
//   - Otherwise:               returns (primaryModel, false, score)
//
// The caller is responsible for resolving the returned model name into
// provider candidates (see AgentInstance.LightCandidates).
func (r *Router) SelectModel(
	msg string,
	history []providers.Message,
	primaryModel string,
) (model string, usedLight bool, score float64) {
	features := ExtractFeatures(msg, history)
	score = r.classifier.Score(features)
	if score < r.cfg.Threshold {
		return r.cfg.LightModel, true, score
	}
	return primaryModel, false, score
}

// LightModel returns the configured light model name.
func (r *Router) LightModel() string {
	return r.cfg.LightModel
}

// Threshold returns the complexity threshold in use.
func (r *Router) Threshold() float64 {
	return r.cfg.Threshold
}

// TierModelMapping maps each query tier to a model name.
type TierModelMapping map[QueryTier]string

// RouterConfigV2 holds the 4-tier model routing settings.
// When TierMapping is configured, the router operates in V2 mode with
// four tiers (simple, moderate, complex, reasoning) instead of the
// legacy 2-tier (light/heavy) mode.
type RouterConfigV2 struct {
	// TierMapping maps each tier to a model name from model_list.
	TierMapping TierModelMapping

	// TierBoundaries defines custom score boundaries for each tier.
	// If nil, DefaultTierBoundaries is used.
	TierBoundaries map[QueryTier]TierBoundary

	// Legacy fields for backward compatibility

	// LightModel is the model for simple tasks (2-tier mode).
	LightModel string

	// Threshold is the complexity cutoff (2-tier mode).
	Threshold float64
}

// RouterV2 selects the appropriate model tier for each incoming message.
// It supports both 4-tier mode (when TierMapping is configured) and
// 2-tier legacy mode (when only LightModel/Threshold are configured).
type RouterV2 struct {
	cfg        RouterConfigV2
	classifier ClassifierV2
	legacy     *Router // For 2-tier mode fallback

	// Optional metrics collection
	metrics  MetricsCollector
	recorder *DecisionRecorder
}

// NewV2 creates a new RouterV2 with the given configuration.
// If cfg.TierMapping is non-empty, operates in 4-tier mode.
// Otherwise, falls back to 2-tier mode using LightModel and Threshold.
func NewV2(cfg RouterConfigV2) *RouterV2 {
	r := &RouterV2{
		cfg:        cfg,
		classifier: NewRuleClassifierV2(cfg.TierBoundaries),
		metrics:    GetGlobalMetrics(),
		recorder:   GetGlobalRecorder(),
	}

	// Set up legacy router for 2-tier mode if LightModel is configured
	if cfg.LightModel != "" {
		r.legacy = New(RouterConfig{
			LightModel: cfg.LightModel,
			Threshold:  cfg.Threshold,
		})
	}

	return r
}

// SetMetrics sets the metrics collector for this router.
func (r *RouterV2) SetMetrics(collector MetricsCollector) {
	if collector == nil {
		collector = &NoOpMetricsCollector{}
	}
	r.metrics = collector
}

// SetRecorder sets the decision recorder for this router.
func (r *RouterV2) SetRecorder(recorder *DecisionRecorder) {
	r.recorder = recorder
}

// NewV2WithClassifier creates a RouterV2 with a custom classifier (for testing).
// It wires the same metrics, recorder, and legacy 2-tier router as NewV2 so behavior
// matches production aside from the injected classifier.
func NewV2WithClassifier(cfg RouterConfigV2, classifier ClassifierV2) *RouterV2 {
	r := &RouterV2{
		cfg:        cfg,
		classifier: classifier,
		metrics:    GetGlobalMetrics(),
		recorder:   GetGlobalRecorder(),
	}
	if cfg.LightModel != "" {
		r.legacy = New(RouterConfig{
			LightModel: cfg.LightModel,
			Threshold:  cfg.Threshold,
		})
	}
	return r
}

// IsV2Enabled returns true if 4-tier mode is active (TierMapping is configured).
func (r *RouterV2) IsV2Enabled() bool {
	return len(r.cfg.TierMapping) > 0
}

// SelectModel returns the model, tier, and classification result for the message.
// In 4-tier mode, uses TierMapping. In 2-tier mode, delegates to legacy router.
// Privacy: The message content is only used for classification, never stored.
func (r *RouterV2) SelectModel(
	msg string,
	history []providers.Message,
	primaryModel string,
) (model string, tier QueryTier, result ClassificationResult) {
	features := ExtractFeatures(msg, history)
	result = r.classifier.Classify(features)
	tier = result.Tier

	// 4-tier mode
	if r.IsV2Enabled() {
		if mapped, ok := r.cfg.TierMapping[tier]; ok && mapped != "" {
			model = mapped
		} else {
			// Fallback to primary model if tier not mapped
			model = primaryModel
		}
	} else if r.legacy != nil {
		// 2-tier mode (backward compatibility)
		var usedLight bool
		model, usedLight, result.Score = r.legacy.SelectModel(msg, history, primaryModel)
		// Map 2-tier result to 4-tier
		if usedLight {
			tier = TierSimple
		} else {
			tier = TierComplex
		}
		result.Tier = tier
	} else {
		// No routing configured
		model = primaryModel
	}

	// Record metrics (privacy-safe: only statistical data)
	if r.metrics != nil {
		r.metrics.RecordClassification(result)
	}

	return model, tier, result
}

// SelectModelWithRecord is like SelectModel but also records the decision for visualization.
// The agentID should be a hashed/derived identifier, not the raw agent name.
func (r *RouterV2) SelectModelWithRecord(
	msg string,
	history []providers.Message,
	primaryModel string,
	agentID string,
) (model string, tier QueryTier, result ClassificationResult) {
	model, tier, result = r.SelectModel(msg, history, primaryModel)

	// Record decision for visualization (privacy-safe)
	if r.recorder != nil {
		r.recorder.Record(result, model, agentID)
	}

	return model, tier, result
}

// SelectModelLegacy provides backward-compatible 2-tier routing.
// Returns (model, usedLight, score) just like the legacy Router.SelectModel.
func (r *RouterV2) SelectModelLegacy(
	msg string,
	history []providers.Message,
	primaryModel string,
) (model string, usedLight bool, score float64) {
	if r.legacy != nil {
		return r.legacy.SelectModel(msg, history, primaryModel)
	}

	// Use V2 classifier but map to 2-tier
	features := ExtractFeatures(msg, history)
	result := r.classifier.Classify(features)
	score = result.Score

	// Map 4-tier to 2-tier: simple/moderate → light, complex/reasoning → heavy
	usedLight = result.Tier == TierSimple || result.Tier == TierModerate

	if usedLight && r.cfg.LightModel != "" {
		return r.cfg.LightModel, true, score
	}

	// In V2 mode, get the model from tier mapping
	if r.IsV2Enabled() {
		if mapped, ok := r.cfg.TierMapping[result.Tier]; ok && mapped != "" {
			return mapped, result.Tier == TierSimple || result.Tier == TierModerate, score
		}
	}

	return primaryModel, false, score
}

// GetTierMapping returns the tier-to-model mapping.
func (r *RouterV2) GetTierMapping() TierModelMapping {
	return r.cfg.TierMapping
}

// GetTierBoundaries returns the tier boundaries in use.
func (r *RouterV2) GetTierBoundaries() map[QueryTier]TierBoundary {
	return r.cfg.TierBoundaries
}

// LightModel returns the light model name (for 2-tier mode compatibility).
func (r *RouterV2) LightModel() string {
	if r.cfg.LightModel != "" {
		return r.cfg.LightModel
	}
	// In V2 mode, simple tier maps to light
	if mapped, ok := r.cfg.TierMapping[TierSimple]; ok {
		return mapped
	}
	return ""
}

// Threshold returns the complexity threshold (for 2-tier mode compatibility).
func (r *RouterV2) Threshold() float64 {
	if r.cfg.Threshold > 0 {
		return r.cfg.Threshold
	}
	// In V2 mode, return the boundary between moderate and complex
	if boundary, ok := r.cfg.TierBoundaries[TierComplex]; ok {
		return boundary.Min
	}
	return DefaultTierBoundaries[TierComplex].Min
}
