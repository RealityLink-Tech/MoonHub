// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System
// License: MIT

package adaptive_memory

import "time"

// EpisodicEventType defines the type of memory event
type EpisodicEventType string

const (
	// EventTypeCorrection is for user corrections (highest importance: 0.9)
	EventTypeCorrection EpisodicEventType = "correction"
	// EventTypePreferenceLearned is for learned user preferences (importance: 0.8)
	EventTypePreferenceLearned EpisodicEventType = "preference_learned"
	// EventTypeFactStored is for general facts (importance: 0.6)
	EventTypeFactStored EpisodicEventType = "fact_stored"
	// EventTypeTaskCompleted is for task completions (importance: 0.5)
	EventTypeTaskCompleted EpisodicEventType = "task_completed"
	// EventTypeDelegationResult is for delegation outcomes (importance: 0.5)
	EventTypeDelegationResult EpisodicEventType = "delegation_result"

	// --- New Event Types (Extended) ---

	// EventTypeUserFeedback is for explicit user feedback (importance: 0.85)
	EventTypeUserFeedback EpisodicEventType = "user_feedback"
	// EventTypeInsight is for agent-generated insights (importance: 0.7)
	EventTypeInsight EpisodicEventType = "insight"
	// EventTypeReminder is for scheduled reminders (importance: 0.6)
	EventTypeReminder EpisodicEventType = "reminder"
	// EventTypeErrorLearned is for error patterns learned (importance: 0.75)
	EventTypeErrorLearned EpisodicEventType = "error_learned"
	// EventTypeContextUpdate is for context updates (importance: 0.5)
	EventTypeContextUpdate EpisodicEventType = "context_update"
)

// Default importance scores by event type (Ebbinghaus-inspired)
var defaultImportance = map[EpisodicEventType]float64{
	EventTypeCorrection:        0.9,
	EventTypePreferenceLearned: 0.8,
	EventTypeFactStored:        0.6,
	EventTypeTaskCompleted:     0.5,
	EventTypeDelegationResult:  0.5,
	// New event types
	EventTypeUserFeedback:  0.85,
	EventTypeInsight:       0.7,
	EventTypeReminder:      0.6,
	EventTypeErrorLearned:  0.75,
	EventTypeContextUpdate: 0.5,
}

// GetDefaultImportance returns the default importance for an event type
func GetDefaultImportance(eventType EpisodicEventType) float64 {
	if importance, ok := defaultImportance[eventType]; ok {
		return importance
	}
	return 0.5
}

// EpisodicRecord represents a single memory event with metadata
type EpisodicRecord struct {
	ID             string            `json:"id"`
	UserID         string            `json:"user_id"`
	EventType      EpisodicEventType `json:"event_type"`
	Content        string            `json:"content"`        // Original content
	ContentTokens  string            `json:"content_tokens"` // Segmented for Chinese FTS5
	Outcome        *string           `json:"outcome,omitempty"`
	Importance     float64           `json:"importance"` // 0.0-1.0
	AccessCount    int               `json:"access_count"`
	CreatedAt      int64             `json:"created_at"`       // Unix timestamp (ms)
	LastAccessedAt int64             `json:"last_accessed_at"` // Unix timestamp (ms)
}

// MemorySearchResult represents a search hit with relevance scoring
type MemorySearchResult struct {
	ID             string  `json:"id"`
	Content        string  `json:"content"`
	RelevanceScore float64 `json:"relevance_score"`
	Source         string  `json:"source"` // "episodic" or "legacy"
}

// ConsolidationResult reports the outcome of memory consolidation
type ConsolidationResult struct {
	Decayed int `json:"decayed"`
	Pruned  int `json:"pruned"`
	Merged  int `json:"merged"`
}

// EventInput is the parameter structure for RecordEvent
type EventInput struct {
	Type       EpisodicEventType `json:"type"`
	Content    string            `json:"content"`
	Outcome    *string           `json:"outcome,omitempty"`
	Importance *float64          `json:"importance,omitempty"`
}

// Config holds configuration for the memory engine
type Config struct {
	DBPath                   string  `json:"db_path"`                    // SQLite database file path
	EnableChinese            bool    `json:"enable_chinese"`             // Enable Chinese segmentation
	FTSMaxResults            int     `json:"fts_max_results"`            // Max FTS5 search results (default: 50)
	ContextMaxResults        int     `json:"context_max_results"`        // Max results for context (default: 10)
	DecayOlderThanDays       int     `json:"decay_older_than_days"`      // Days before decay (default: 7)
	DecayFactor              float64 `json:"decay_factor"`               // Decay multiplier (default: 0.95)
	PruneImportanceMax       float64 `json:"prune_importance_max"`       // Max importance for pruning (default: 0.1)
	PruneAccessCountMax      int     `json:"prune_access_count_max"`     // Max access count for pruning (default: 0)
	PruneOlderThanDays       int     `json:"prune_older_than_days"`      // Days before pruning (default: 30)
	MergeSimilarityThreshold float64 `json:"merge_similarity_threshold"` // Jaccard threshold (default: 0.8)
}

// DefaultConfig returns sensible defaults
func DefaultConfig(dbPath string) Config {
	return Config{
		DBPath:                   dbPath,
		EnableChinese:            true,
		FTSMaxResults:            50,
		ContextMaxResults:        10,
		DecayOlderThanDays:       7,
		DecayFactor:              0.95,
		PruneImportanceMax:       0.1,
		PruneAccessCountMax:      0,
		PruneOlderThanDays:       30,
		MergeSimilarityThreshold: 0.8,
	}
}

// MemoryEngine defines the adaptive memory interface
type MemoryEngine interface {
	// RecordEvent stores a new episodic memory event
	RecordEvent(userID string, event EventInput) (string, error)

	// Search retrieves memories using hybrid scoring (FTS5 + temporal + importance)
	Search(userID, query string, limit int) ([]MemorySearchResult, error)

	// Consolidate performs memory maintenance (decay, prune, merge)
	Consolidate(userID string) (ConsolidationResult, error)

	// GetContextForAgent returns formatted memory context for LLM prompts
	// If query is provided, performs search-based retrieval
	// Otherwise returns high-importance recent memories
	GetContextForAgent(userID string, query *string) (string, error)

	// Reinforce strengthens a memory (bumps access count + timestamp)
	Reinforce(memoryID string) error

	// GetEvent retrieves a single episodic record by ID
	GetEvent(id string) (*EpisodicRecord, error)

	// GetEvents retrieves all events for a user (sorted by created_at DESC)
	GetEvents(userID string, limit int) ([]EpisodicRecord, error)

	// Close releases database resources
	Close() error
}

// Constants for time calculations
const (
	MsPerDay = 24 * 60 * 60 * 1000

	// Scoring weights
	WeightFTS5       = 0.4
	WeightTemporal   = 0.3
	WeightImportance = 0.3
)

// NowMs returns current time in milliseconds
func NowMs() int64 {
	return time.Now().UnixMilli()
}

// MsToTime converts milliseconds to time.Time
func MsToTime(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond))
}
