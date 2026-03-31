// MoonHub - Your ready-to-use AI assistant
// Context Compactor - SQLite Store
// License: MIT

package compactor

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SummaryStore handles SQLite database operations for tiered summaries
type SummaryStore struct {
	db *sql.DB
	mu sync.RWMutex

	// Prepared statements
	stmtUpsertSummary *sql.Stmt
	stmtGetSummary    *sql.Stmt
	stmtDeleteSummary *sql.Stmt
	stmtUpsertState   *sql.Stmt
	stmtGetState      *sql.Stmt
	stmtInsertHistory *sql.Stmt
}

// CompactionState tracks incremental compaction state
type CompactionState struct {
	SessionKey         string
	LastCompactionTime int64
	LastMessageIndex   int
	TotalCompactions   int
}

// CompactionHistoryRecord represents a single compaction history entry
type CompactionHistoryRecord struct {
	ID              int64
	SessionKey      string
	CompactionID    int64
	MessagesRemoved int
	DedupGroups     int
	RulesApplied    string // JSON array
	DurationMs      int64
	CreatedAt       int64
}

// Schema SQL statements
var schemaSQL = []string{
	`CREATE TABLE IF NOT EXISTS tiered_summaries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_key TEXT NOT NULL UNIQUE,
		l0_summary TEXT NOT NULL,
		l1_summary TEXT NOT NULL,
		l2_summary TEXT NOT NULL,
		messages_before INTEGER NOT NULL,
		messages_summarized INTEGER NOT NULL,
		tokens_before INTEGER NOT NULL,
		tokens_after INTEGER NOT NULL,
		compression_ratio REAL NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_tiered_summaries_session ON tiered_summaries(session_key)`,
	`CREATE TABLE IF NOT EXISTS compaction_state (
		session_key TEXT PRIMARY KEY,
		last_compaction_timestamp INTEGER NOT NULL,
		last_message_index INTEGER NOT NULL,
		total_compactions INTEGER NOT NULL DEFAULT 0
	)`,
	`CREATE TABLE IF NOT EXISTS compaction_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_key TEXT NOT NULL,
		compaction_id INTEGER NOT NULL,
		messages_removed INTEGER NOT NULL,
		dedup_groups INTEGER NOT NULL,
		rules_applied TEXT NOT NULL,
		duration_ms INTEGER NOT NULL,
		created_at INTEGER NOT NULL,
		FOREIGN KEY (compaction_id) REFERENCES tiered_summaries(id)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_compaction_history_session ON compaction_history(session_key)`,
}

// NewSummaryStore creates a new SQLite store for tiered summaries
func NewSummaryStore(dbPath string) (*SummaryStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set pragmas for performance
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-32000", // 32MB cache
		"PRAGMA busy_timeout=5000",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}

	store := &SummaryStore{db: db}

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
func (s *SummaryStore) initSchema() error {
	for _, sqlStmt := range schemaSQL {
		if _, err := s.db.Exec(sqlStmt); err != nil {
			return fmt.Errorf("schema initialization failed: %w", err)
		}
	}
	return nil
}

// prepareStatements prepares all SQL statements for reuse
func (s *SummaryStore) prepareStatements() error {
	var err error

	s.stmtUpsertSummary, err = s.db.Prepare(`
		INSERT INTO tiered_summaries (session_key, l0_summary, l1_summary, l2_summary,
			messages_before, messages_summarized, tokens_before, tokens_after, compression_ratio, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_key) DO UPDATE SET
			l0_summary = excluded.l0_summary,
			l1_summary = excluded.l1_summary,
			l2_summary = excluded.l2_summary,
			messages_before = excluded.messages_before,
			messages_summarized = excluded.messages_summarized,
			tokens_before = excluded.tokens_before,
			tokens_after = excluded.tokens_after,
			compression_ratio = excluded.compression_ratio,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return err
	}

	s.stmtGetSummary, err = s.db.Prepare(`
		SELECT l0_summary, l1_summary, l2_summary, messages_before, messages_summarized,
			tokens_before, tokens_after, compression_ratio
		FROM tiered_summaries WHERE session_key = ?
	`)
	if err != nil {
		return err
	}

	s.stmtDeleteSummary, err = s.db.Prepare(`DELETE FROM tiered_summaries WHERE session_key = ?`)
	if err != nil {
		return err
	}

	s.stmtUpsertState, err = s.db.Prepare(`
		INSERT INTO compaction_state (session_key, last_compaction_timestamp, last_message_index, total_compactions)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(session_key) DO UPDATE SET
			last_compaction_timestamp = excluded.last_compaction_timestamp,
			last_message_index = excluded.last_message_index,
			total_compactions = total_compactions + 1
	`)
	if err != nil {
		return err
	}

	s.stmtGetState, err = s.db.Prepare(`
		SELECT last_compaction_timestamp, last_message_index, total_compactions
		FROM compaction_state WHERE session_key = ?
	`)
	if err != nil {
		return err
	}

	s.stmtInsertHistory, err = s.db.Prepare(`
		INSERT INTO compaction_history (session_key, compaction_id, messages_removed, dedup_groups, rules_applied, duration_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}

	return nil
}

// UpsertSummary stores or updates a tiered summary
func (s *SummaryStore) UpsertSummary(ctx context.Context, sessionKey string, summary *TieredSummary, metrics *CompactionMetrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()
	_, err := s.stmtUpsertSummary.ExecContext(ctx,
		sessionKey,
		summary.L0,
		summary.L1,
		summary.L2,
		metrics.MessagesBefore,
		metrics.MessagesSummarized,
		metrics.TokensBefore,
		metrics.TokensAfter,
		metrics.CompressionRatio,
		now,
		now,
	)

	return err
}

// GetSummary retrieves a tiered summary for a session
func (s *SummaryStore) GetSummary(ctx context.Context, sessionKey string) (*TieredSummary, *CompactionMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.stmtGetSummary.QueryRowContext(ctx, sessionKey)

	var summary TieredSummary
	var metrics CompactionMetrics

	err := row.Scan(
		&summary.L0,
		&summary.L1,
		&summary.L2,
		&metrics.MessagesBefore,
		&metrics.MessagesSummarized,
		&metrics.TokensBefore,
		&metrics.TokensAfter,
		&metrics.CompressionRatio,
	)

	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	return &summary, &metrics, nil
}

// DeleteSummary removes a summary for a session
func (s *SummaryStore) DeleteSummary(ctx context.Context, sessionKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.stmtDeleteSummary.ExecContext(ctx, sessionKey)
	return err
}

// UpsertState stores or updates compaction state for incremental processing
func (s *SummaryStore) UpsertState(ctx context.Context, sessionKey string, state CompactionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.stmtUpsertState.ExecContext(ctx,
		sessionKey,
		state.LastCompactionTime,
		state.LastMessageIndex,
	)
	return err
}

// GetState retrieves compaction state for a session
func (s *SummaryStore) GetState(ctx context.Context, sessionKey string) (*CompactionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.stmtGetState.QueryRowContext(ctx, sessionKey)

	var state CompactionState
	state.SessionKey = sessionKey

	err := row.Scan(
		&state.LastCompactionTime,
		&state.LastMessageIndex,
		&state.TotalCompactions,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &state, nil
}

// InsertHistory records a compaction history entry
func (s *SummaryStore) InsertHistory(ctx context.Context, sessionKey string, compactionID int64, messagesRemoved, dedupGroups int, rulesApplied string, durationMs int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.stmtInsertHistory.ExecContext(ctx,
		sessionKey,
		compactionID,
		messagesRemoved,
		dedupGroups,
		rulesApplied,
		durationMs,
		time.Now().UnixMilli(),
	)
	return err
}

// GetLastCompactionID returns the last compaction ID for a session
func (s *SummaryStore) GetLastCompactionID(ctx context.Context, sessionKey string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var id int64
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM tiered_summaries WHERE session_key = ?`,
		sessionKey,
	).Scan(&id)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return id, nil
}

// Close closes the database connection
func (s *SummaryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close prepared statements
	if s.stmtUpsertSummary != nil {
		s.stmtUpsertSummary.Close()
	}
	if s.stmtGetSummary != nil {
		s.stmtGetSummary.Close()
	}
	if s.stmtDeleteSummary != nil {
		s.stmtDeleteSummary.Close()
	}
	if s.stmtUpsertState != nil {
		s.stmtUpsertState.Close()
	}
	if s.stmtGetState != nil {
		s.stmtGetState.Close()
	}
	if s.stmtInsertHistory != nil {
		s.stmtInsertHistory.Close()
	}

	return s.db.Close()
}
