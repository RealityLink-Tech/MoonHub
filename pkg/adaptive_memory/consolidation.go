// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System - Memory Consolidation
// License: MIT

package adaptive_memory

import (
	"context"
	"fmt"
	"time"
)

// Consolidator handles memory consolidation operations
type Consolidator struct {
	store  *Store
	config Config
}

// NewConsolidator creates a new consolidator
func NewConsolidator(store *Store, config Config) *Consolidator {
	return &Consolidator{
		store:  store,
		config: config,
	}
}

// Consolidate performs all consolidation operations: decay, prune, merge
func (c *Consolidator) Consolidate(ctx context.Context, userID string) (ConsolidationResult, error) {
	result := ConsolidationResult{}

	// 1. Decay: Reduce importance of old, unaccessed memories
	decayed, err := c.decayOldMemories(ctx, userID)
	if err != nil {
		return result, fmt.Errorf("decay failed: %w", err)
	}
	result.Decayed = decayed

	// 2. Prune: Delete low-importance, unaccessed old memories
	pruned, err := c.pruneMemories(ctx, userID)
	if err != nil {
		return result, fmt.Errorf("prune failed: %w", err)
	}
	result.Pruned = pruned

	// 3. Merge: Combine similar memories
	merged, err := c.mergeSimilarMemories(ctx, userID)
	if err != nil {
		return result, fmt.Errorf("merge failed: %w", err)
	}
	result.Merged = merged

	return result, nil
}

// decayOldMemories reduces importance of memories not accessed in N days
func (c *Consolidator) decayOldMemories(ctx context.Context, userID string) (int, error) {
	cutoffTime := time.Now().UnixMilli() - int64(c.config.DecayOlderThanDays)*MsPerDay

	// Get all memories older than cutoff time
	records, err := c.store.GetOlderThan(ctx, userID, cutoffTime)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, record := range records {
		// Skip if already very low importance
		if record.Importance < 0.1 {
			continue
		}

		// Apply decay factor
		newImportance := record.Importance * c.config.DecayFactor
		if newImportance < 0.1 {
			newImportance = 0.1 // Minimum threshold
		}

		err := c.store.UpdateImportance(ctx, record.ID, newImportance)
		if err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// pruneMemories deletes very low importance, unaccessed old memories
func (c *Consolidator) pruneMemories(ctx context.Context, userID string) (int, error) {
	cutoffTime := time.Now().UnixMilli() - int64(c.config.PruneOlderThanDays)*MsPerDay

	// Get memories matching prune criteria
	records, err := c.store.GetForPrune(ctx, userID,
		c.config.PruneImportanceMax,
		c.config.PruneAccessCountMax,
		cutoffTime)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, record := range records {
		err := c.store.Delete(ctx, record.ID)
		if err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// mergeSimilarMemories detects and merges highly similar content
func (c *Consolidator) mergeSimilarMemories(ctx context.Context, userID string) (int, error) {
	// Get all memories for comparison
	records, err := c.store.GetByUser(ctx, userID, 1000)
	if err != nil {
		return 0, err
	}

	count := 0
	merged := make(map[string]bool)

	// Compare each pair of memories
	for i := 0; i < len(records); i++ {
		if merged[records[i].ID] {
			continue
		}

		for j := i + 1; j < len(records); j++ {
			if merged[records[j].ID] {
				continue
			}

			// Calculate similarity
			similarity := ContentSimilarity(records[i].Content, records[j].Content)
			if similarity > c.config.MergeSimilarityThreshold {
				// Keep the newer one, delete the older one
				older, newer := records[i], records[j]
				if older.CreatedAt > newer.CreatedAt {
					older, newer = newer, older
				}

				// Bump importance of newer record
				bumpedImportance := newer.Importance + older.Importance*0.2
				if bumpedImportance > 1.0 {
					bumpedImportance = 1.0
				}

				err := c.store.UpdateImportance(ctx, newer.ID, bumpedImportance)
				if err != nil {
					continue
				}

				// Delete older record
				err = c.store.Delete(ctx, older.ID)
				if err != nil {
					continue
				}

				merged[older.ID] = true
				count++
			}
		}
	}

	return count, nil
}
