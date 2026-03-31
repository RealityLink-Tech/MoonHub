// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System - Core Engine
// License: MIT

package adaptive_memory

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// Engine implements the MemoryEngine interface
type Engine struct {
	store        *Store
	segmenter    *ChineseSegmenter
	consolidator *Consolidator
	config       Config
}

// NewMemoryEngine creates a new memory engine
func NewMemoryEngine(config Config) (*Engine, error) {
	store, err := NewStore(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	segmenter := NewChineseSegmenter(config.EnableChinese)

	// Initialize segmenter if Chinese is enabled
	if config.EnableChinese {
		gseSeg, err := NewGSESegmenter()
		if err == nil {
			segmenter.SetSegmenter(gseSeg)
		} else {
			// Fallback to simple space-based segmentation when GSE fails
			segmenter.SetSegmenter(NewSimpleSegmenter())
		}
	}

	consolidator := NewConsolidator(store, config)

	return &Engine{
		store:        store,
		segmenter:    segmenter,
		consolidator: consolidator,
		config:       config,
	}, nil
}

// searchStoreLimit chooses how many FTS rows to fetch before hybrid re-ranking.
// Pulling FTSMaxResults on every query is wasteful when the caller only needs a small top-K;
// we oversample modestly so temporal/importance scoring can reorder within the candidate set.
func searchStoreLimit(requested, ftsCap int) int {
	if ftsCap <= 0 {
		return requested
	}
	if requested <= 0 {
		return ftsCap
	}
	n := requested * 4
	if n < requested+16 {
		n = requested + 16
	}
	if n > ftsCap {
		n = ftsCap
	}
	// Ensure at least `requested` candidates when the cap allows (for small top-K).
	if n < requested && requested <= ftsCap {
		n = requested
	}
	return n
}

// RecordEvent stores a new episodic memory event
func (e *Engine) RecordEvent(userID string, event EventInput) (string, error) {
	// Generate ID
	id := uuid.New().String()

	// Determine importance
	importance := GetDefaultImportance(event.Type)
	if event.Importance != nil {
		importance = *event.Importance
	}

	// Segment content for Chinese
	_, contentTokens := e.segmenter.PrepareForFTS(event.Content)

	// Create record
	now := NowMs()
	record := &EpisodicRecord{
		ID:             id,
		UserID:         userID,
		EventType:      event.Type,
		Content:        event.Content,
		ContentTokens:  contentTokens,
		Outcome:        event.Outcome,
		Importance:     importance,
		AccessCount:    0,
		CreatedAt:      now,
		LastAccessedAt: now,
	}

	// Insert into store
	err := e.store.Insert(context.Background(), record)
	if err != nil {
		return "", fmt.Errorf("failed to record event: %w", err)
	}

	return id, nil
}

// Search retrieves memories using hybrid scoring (FTS5 + temporal + importance)
func (e *Engine) Search(userID, query string, limit int) ([]MemorySearchResult, error) {
	if limit <= 0 {
		limit = e.config.FTSMaxResults
	}

	// Sanitize query for FTS5
	ftsQuery := SanitizeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	// Segment query for Chinese
	queryTokens := e.segmenter.SmartSegment(query)

	storeLimit := searchStoreLimit(limit, e.config.FTSMaxResults)
	records, err := e.store.Search(context.Background(), userID, queryTokens, storeLimit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	now := NowMs()

	// Find max FTS rank for normalization (Importance holds rank until we build results)
	maxAbsRank := 0.0
	for i := range records {
		if r := records[i].Importance; r > maxAbsRank {
			maxAbsRank = r
		}
	}

	results := make([]MemorySearchResult, len(records))
	for i := range records {
		record := &records[i]
		temporalScore := ComputeTemporalScore(record.LastAccessedAt, int64(record.AccessCount), now)
		ftsRank := NormalizeFTSRank(record.Importance, maxAbsRank)
		actualImportance := GetDefaultImportance(record.EventType)
		relevance := ComputeRelevanceScore(ftsRank, temporalScore, actualImportance)
		results[i] = MemorySearchResult{
			ID:             record.ID,
			Content:        record.Content,
			RelevanceScore: relevance,
			Source:         "episodic",
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].RelevanceScore > results[j].RelevanceScore
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetContextForAgent returns formatted memory context for LLM prompts
func (e *Engine) GetContextForAgent(userID string, query *string) (string, error) {
	var results []MemorySearchResult
	var err error

	if query != nil && *query != "" {
		// Search-based retrieval
		results, err = e.Search(userID, *query, e.config.ContextMaxResults)
		if err != nil {
			return "", err
		}
	}

	// Also get high-importance recent memories
	highImportance, err := e.store.GetHighImportance(context.Background(), userID, 0.7, e.config.ContextMaxResults)
	if err != nil {
		return "", err
	}

	// Merge results (deduplicate by ID)
	seenIDs := make(map[string]bool)
	var allResults []MemorySearchResult

	for _, r := range results {
		if !seenIDs[r.ID] {
			seenIDs[r.ID] = true
			allResults = append(allResults, r)
		}
	}

	for _, record := range highImportance {
		if !seenIDs[record.ID] {
			seenIDs[record.ID] = true
			// Calculate relevance for high-importance records
			temporalScore := ComputeTemporalScore(record.LastAccessedAt, int64(record.AccessCount), NowMs())
			relevance := ComputeRelevanceScore(0.5, temporalScore, record.Importance)

			allResults = append(allResults, MemorySearchResult{
				ID:             record.ID,
				Content:        record.Content,
				RelevanceScore: relevance,
				Source:         "episodic",
			})
		}
	}

	if len(allResults) == 0 {
		return "", nil
	}

	// Sort by relevance
	sortResultsByRelevance(allResults)

	// Limit final results
	if len(allResults) > e.config.ContextMaxResults {
		allResults = allResults[:e.config.ContextMaxResults]
	}

	// Format for LLM context
	var sb strings.Builder
	sb.WriteString("## 记忆上下文\n\n")
	sb.WriteString("以下是相关的记忆信息，用于辅助回答：\n\n")

	for i, result := range allResults {
		sb.WriteString(fmt.Sprintf("%d. %s (相关性: %.2f)\n", i+1, result.Content, result.RelevanceScore))
	}

	return sb.String(), nil
}

// Reinforce strengthens a memory (bumps access count + timestamp)
func (e *Engine) Reinforce(memoryID string) error {
	return e.store.Reinforce(context.Background(), memoryID)
}

// GetEvent retrieves a single episodic record by ID
func (e *Engine) GetEvent(id string) (*EpisodicRecord, error) {
	return e.store.GetByID(context.Background(), id)
}

// GetEvents retrieves all events for a user (sorted by created_at DESC)
func (e *Engine) GetEvents(userID string, limit int) ([]EpisodicRecord, error) {
	return e.store.GetByUser(context.Background(), userID, limit)
}

// Consolidate performs memory maintenance (decay, prune, merge)
func (e *Engine) Consolidate(userID string) (ConsolidationResult, error) {
	return e.consolidator.Consolidate(context.Background(), userID)
}

// Close releases database resources
func (e *Engine) Close() error {
	return e.store.Close()
}

// sortResultsByRelevance sorts results by relevance score in descending order
func sortResultsByRelevance(results []MemorySearchResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].RelevanceScore > results[j].RelevanceScore
	})
}
