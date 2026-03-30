// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System - SQLite Store
// License: MIT

package adaptive_memory

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// Store handles SQLite database operations for adaptive memory
type Store struct {
	db     *sql.DB
	config Config
	mu     sync.RWMutex

	// Prepared statements for performance
	stmtInsert       *sql.Stmt
	stmtUpdate       *sql.Stmt
	stmtGetByID      *sql.Stmt
	stmtGetByUser    *sql.Stmt
	stmtDelete       *sql.Stmt
	stmtSearch       *sql.Stmt
	stmtSearchByUser *sql.Stmt
	stmtReinforce    *sql.Stmt
	stmtDecay        *sql.Stmt
	stmtPrune        *sql.Stmt
	stmtGetForMerge  *sql.Stmt
	stmtGetOlderThan *sql.Stmt
	stmtGetForPrune  *sql.Stmt
}

// NewStore creates a new SQLite store
func NewStore(config Config) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(config.DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set pragmas for performance
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000", // 64MB cache
		"PRAGMA busy_timeout=5000",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}

	store := &Store{
		db:     db,
		config: config,
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Prepare statements
	if err := store.prepareStatements(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to prepare statements: %w", err)
	}

	return store, nil
}

// initSchema initializes the database schema
func (s *Store) initSchema() error {
	for _, sqlStmt := range AllSQL {
		if _, err := s.db.Exec(sqlStmt); err != nil {
			return fmt.Errorf("schema initialization failed: %w", err)
		}
	}
	return nil
}

// prepareStatements prepares all SQL statements for reuse
func (s *Store) prepareStatements() error {
	var err error

	s.stmtInsert, err = s.db.Prepare(`
		INSERT INTO episodic_memory (id, user_id, event_type, content, content_tokens, outcome, importance, access_count, created_at, last_accessed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}

	s.stmtUpdate, err = s.db.Prepare(`
		UPDATE episodic_memory
		SET content = ?, content_tokens = ?, outcome = ?, importance = ?, access_count = ?, last_accessed_at = ?
		WHERE id = ?
	`)
	if err != nil {
		return err
	}

	s.stmtGetByID, err = s.db.Prepare(`
		SELECT id, user_id, event_type, content, content_tokens, outcome, importance, access_count, created_at, last_accessed_at
		FROM episodic_memory WHERE id = ?
	`)
	if err != nil {
		return err
	}

	s.stmtGetByUser, err = s.db.Prepare(`
		SELECT id, user_id, event_type, content, content_tokens, outcome, importance, access_count, created_at, last_accessed_at
		FROM episodic_memory WHERE user_id = ?
		ORDER BY created_at DESC LIMIT ?
	`)
	if err != nil {
		return err
	}

	s.stmtDelete, err = s.db.Prepare(`DELETE FROM episodic_memory WHERE id = ?`)
	if err != nil {
		return err
	}

	s.stmtSearch, err = s.db.Prepare(`
		SELECT e.id, e.content, e.importance, e.access_count, e.created_at, e.last_accessed_at, f.rank
		FROM episodic_memory e
		JOIN memory_fts f ON e.id = f.id
		WHERE memory_fts MATCH ?
		ORDER BY f.rank
		LIMIT ?
	`)
	if err != nil {
		return err
	}

	s.stmtSearchByUser, err = s.db.Prepare(`
		SELECT e.id, e.user_id, e.event_type, e.content, e.importance, e.access_count, e.created_at, e.last_accessed_at, f.rank
		FROM episodic_memory e
		JOIN memory_fts f ON e.id = f.id
		WHERE e.user_id = ? AND memory_fts MATCH ?
		ORDER BY f.rank
		LIMIT ?
	`)
	if err != nil {
		return err
	}

	s.stmtGetOlderThan, err = s.db.Prepare(`
		SELECT id, user_id, event_type, content, content_tokens, outcome, importance, access_count, created_at, last_accessed_at
		FROM episodic_memory
		WHERE user_id = ? AND last_accessed_at < ?
		ORDER BY last_accessed_at ASC
	`)
	if err != nil {
		return err
	}

	s.stmtGetForPrune, err = s.db.Prepare(`
		SELECT id, user_id, event_type, content, content_tokens, outcome, importance, access_count, created_at, last_accessed_at
		FROM episodic_memory
		WHERE user_id = ? AND importance < ? AND access_count <= ? AND created_at < ?
	`)
	if err != nil {
		return err
	}

	s.stmtReinforce, err = s.db.Prepare(`
		UPDATE episodic_memory
		SET access_count = access_count + 1, last_accessed_at = ?
		WHERE id = ?
	`)
	if err != nil {
		return err
	}

	s.stmtDecay, err = s.db.Prepare(`
		UPDATE episodic_memory
		SET importance = importance * ?
		WHERE user_id = ? AND last_accessed_at < ? AND importance > 0
	`)
	if err != nil {
		return err
	}

	s.stmtPrune, err = s.db.Prepare(`
		DELETE FROM episodic_memory
		WHERE user_id = ? AND importance < ? AND access_count <= ? AND created_at < ?
	`)
	if err != nil {
		return err
	}

	s.stmtGetForMerge, err = s.db.Prepare(`
		SELECT id, content FROM episodic_memory
		WHERE user_id = ? AND created_at > ?
		ORDER BY created_at DESC
	`)
	if err != nil {
		return err
	}

	return nil
}

// Insert stores a new episodic record
func (s *Store) Insert(ctx context.Context, record *EpisodicRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Insert into main table
	_, err := s.stmtInsert.ExecContext(ctx,
		record.ID,
		record.UserID,
		record.EventType,
		record.Content,
		record.ContentTokens,
		record.Outcome,
		record.Importance,
		record.AccessCount,
		record.CreatedAt,
		record.LastAccessedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert record: %w", err)
	}

	// Insert into FTS5
	tags := string(record.EventType) + " " + record.UserID
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO memory_fts (id, content, content_tokens, tags) VALUES (?, ?, ?, ?)",
		record.ID, record.Content, record.ContentTokens, tags,
	)
	if err != nil {
		return fmt.Errorf("failed to insert into FTS5: %w", err)
	}

	return nil
}

// GetByID retrieves a single record by ID
func (s *Store) GetByID(ctx context.Context, id string) (*EpisodicRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.stmtGetByID.QueryRowContext(ctx, id)
	return s.scanRecord(row)
}

// GetByUser retrieves all records for a user
func (s *Store) GetByUser(ctx context.Context, userID string, limit int) ([]EpisodicRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = s.config.FTSMaxResults
	}

	rows, err := s.stmtGetByUser.QueryContext(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var records []EpisodicRecord
	for rows.Next() {
		record, err := s.scanRecordFromRows(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}

	return records, rows.Err()
}

// Delete removes a record by ID
func (s *Store) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete from main table
	_, err := s.stmtDelete.ExecContext(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	// Delete from FTS5
	_, err = s.db.ExecContext(ctx, "DELETE FROM memory_fts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete from FTS5: %w", err)
	}

	return nil
}

// Reinforce increments access count and updates last_accessed_at
func (s *Store) Reinforce(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.stmtReinforce.ExecContext(ctx, NowMs(), id)
	if err != nil {
		return fmt.Errorf("failed to reinforce record: %w", err)
	}

	return nil
}

// SearchFTS performs FTS5 search and returns raw results
func (s *Store) SearchFTS(ctx context.Context, query string, limit int) ([]FTSSearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = s.config.FTSMaxResults
	}

	rows, err := s.stmtSearch.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("FTS5 search failed: %w", err)
	}
	defer rows.Close()

	var results []FTSSearchResult
	for rows.Next() {
		var r FTSSearchResult
		var outcome sql.NullString
		err := rows.Scan(
			&r.ID, &r.Content, &r.Importance, &r.AccessCount,
			&r.CreatedAt, &r.LastAccessedAt, &r.FTSRank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		_ = outcome // not used in search results
		results = append(results, r)
	}

	return results, rows.Err()
}

// FTSSearchResult represents a raw FTS5 search result
type FTSSearchResult struct {
	ID             string
	Content        string
	Importance     float64
	AccessCount    int64
	CreatedAt      int64
	LastAccessedAt int64
	FTSRank        float64
}

// Decay reduces importance of old records
func (s *Store) Decay(ctx context.Context, userID string, olderThanMs int64, factor float64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.stmtDecay.ExecContext(ctx, factor, userID, olderThanMs)
	if err != nil {
		return 0, fmt.Errorf("decay failed: %w", err)
	}

	return result.RowsAffected()
}

// Prune removes low-importance records
func (s *Store) Prune(ctx context.Context, userID string, importanceMax float64, accessCountMax int, olderThanMs int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.stmtPrune.ExecContext(ctx, userID, importanceMax, accessCountMax, olderThanMs)
	if err != nil {
		return 0, fmt.Errorf("prune failed: %w", err)
	}

	affected, _ := result.RowsAffected()

	// Also clean up FTS5
	if affected > 0 {
		s.db.ExecContext(ctx,
			"DELETE FROM memory_fts WHERE id NOT IN (SELECT id FROM episodic_memory)",
		)
	}

	return affected, nil
}

// GetOlderThan retrieves records not accessed since cutoffTime (for decay)
func (s *Store) GetOlderThan(ctx context.Context, userID string, cutoffTime int64) ([]EpisodicRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.stmtGetOlderThan.QueryContext(ctx, userID, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get older records: %w", err)
	}
	defer rows.Close()

	var records []EpisodicRecord
	for rows.Next() {
		record, err := s.scanRecordFromRows(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}

	return records, rows.Err()
}

// GetForPrune retrieves records matching prune criteria (low importance, low access, old)
func (s *Store) GetForPrune(ctx context.Context, userID string, importanceMax float64, accessCountMax int, olderThanMs int64) ([]EpisodicRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.stmtGetForPrune.QueryContext(ctx, userID, importanceMax, accessCountMax, olderThanMs)
	if err != nil {
		return nil, fmt.Errorf("failed to get prune candidates: %w", err)
	}
	defer rows.Close()

	var records []EpisodicRecord
	for rows.Next() {
		record, err := s.scanRecordFromRows(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}

	return records, rows.Err()
}

// Search performs FTS5 search filtered by user, returns records with FTS rank in Importance for scoring
// queryStr is the segmented/search-ready query string (e.g. from SmartSegment)
func (s *Store) Search(ctx context.Context, userID string, queryStr string, limit int) ([]EpisodicRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ftsQuery := strings.TrimSpace(queryStr)
	if ftsQuery == "" {
		return nil, nil
	}

	if limit <= 0 {
		limit = s.config.FTSMaxResults
	}

	rows, err := s.stmtSearchByUser.QueryContext(ctx, userID, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer rows.Close()

	var records []EpisodicRecord
	for rows.Next() {
		var r EpisodicRecord
		var ftsRank float64
		err := rows.Scan(
			&r.ID, &r.UserID, &r.EventType, &r.Content, &r.Importance, &r.AccessCount,
			&r.CreatedAt, &r.LastAccessedAt, &ftsRank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		// Store FTS rank in Importance for engine's hybrid scoring (engine uses it for absRank)
		r.Importance = ftsRank
		records = append(records, r)
	}

	return records, rows.Err()
}

// GetForMerge retrieves records for similarity-based merging
func (s *Store) GetForMerge(ctx context.Context, userID string, sinceMs int64) ([]MergeCandidate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.stmtGetForMerge.QueryContext(ctx, userID, sinceMs)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge candidates: %w", err)
	}
	defer rows.Close()

	var candidates []MergeCandidate
	for rows.Next() {
		var c MergeCandidate
		if err := rows.Scan(&c.ID, &c.Content); err != nil {
			return nil, fmt.Errorf("failed to scan merge candidate: %w", err)
		}
		candidates = append(candidates, c)
	}

	return candidates, rows.Err()
}

// MergeCandidate represents a record for similarity comparison
type MergeCandidate struct {
	ID      string
	Content string
}

// UpdateImportance updates the importance of a record
func (s *Store) UpdateImportance(ctx context.Context, id string, importance float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx,
		"UPDATE episodic_memory SET importance = ? WHERE id = ?",
		importance, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update importance: %w", err)
	}

	return nil
}

// GetHighImportance retrieves records with importance >= threshold
func (s *Store) GetHighImportance(ctx context.Context, userID string, threshold float64, limit int) ([]EpisodicRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = s.config.ContextMaxResults
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, event_type, content, content_tokens, outcome, importance, access_count, created_at, last_accessed_at
		 FROM episodic_memory
		 WHERE user_id = ? AND importance >= ?
		 ORDER BY last_accessed_at DESC
		 LIMIT ?`,
		userID, threshold, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get high importance records: %w", err)
	}
	defer rows.Close()

	var records []EpisodicRecord
	for rows.Next() {
		record, err := s.scanRecordFromRows(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}

	return records, rows.Err()
}

// Close releases database resources
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close prepared statements
	stmts := []*sql.Stmt{
		s.stmtInsert, s.stmtUpdate, s.stmtGetByID, s.stmtGetByUser,
		s.stmtDelete, s.stmtSearch, s.stmtSearchByUser, s.stmtReinforce, s.stmtDecay,
		s.stmtPrune, s.stmtGetForMerge, s.stmtGetOlderThan, s.stmtGetForPrune,
	}
	for _, stmt := range stmts {
		if stmt != nil {
			stmt.Close()
		}
	}

	return s.db.Close()
}

// scanRecord scans a single record from a row
func (s *Store) scanRecord(row *sql.Row) (*EpisodicRecord, error) {
	var record EpisodicRecord
	var outcome sql.NullString

	err := row.Scan(
		&record.ID, &record.UserID, &record.EventType, &record.Content,
		&record.ContentTokens, &outcome, &record.Importance, &record.AccessCount,
		&record.CreatedAt, &record.LastAccessedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan record: %w", err)
	}

	if outcome.Valid {
		record.Outcome = &outcome.String
	}

	return &record, nil
}

// scanRecordFromRows scans a single record from rows
func (s *Store) scanRecordFromRows(rows *sql.Rows) (*EpisodicRecord, error) {
	var record EpisodicRecord
	var outcome sql.NullString

	err := rows.Scan(
		&record.ID, &record.UserID, &record.EventType, &record.Content,
		&record.ContentTokens, &outcome, &record.Importance, &record.AccessCount,
		&record.CreatedAt, &record.LastAccessedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan record: %w", err)
	}

	if outcome.Valid {
		record.Outcome = &outcome.String
	}

	return &record, nil
}
