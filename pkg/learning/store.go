// MoonHub - Your ready-to-use AI assistant
// Self-Improving Behavioral Pattern Detection System
// License: MIT

package learning

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// Store persists learning data using SQLite
type Store struct {
	dbPath string
	db     *sql.DB
	mu     sync.RWMutex
}

// NewStore creates a new learning store
func NewStore(dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	store := &Store{
		dbPath: dbPath,
	}

	if err := store.initDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// initDB initializes the database schema
func (s *Store) initDB() error {
	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Create patterns table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS patterns (
			id TEXT PRIMARY KEY,
			category TEXT NOT NULL,
			source TEXT NOT NULL,
			pattern TEXT NOT NULL,
			confidence REAL NOT NULL,
			support INTEGER DEFAULT 0,
			contradiction INTEGER DEFAULT 0,
			first_observed INTEGER NOT NULL,
			last_observed INTEGER NOT NULL,
			observation_count INTEGER DEFAULT 1,
			triggers TEXT,  -- JSON array
			examples TEXT,  -- JSON array
			related_patterns TEXT,  -- JSON array
			supersedes TEXT,  -- JSON array
			decay_rate REAL DEFAULT 0.05,
			last_decay_at INTEGER,
			metadata TEXT  -- JSON object
		);

		CREATE INDEX IF NOT EXISTS idx_patterns_category ON patterns(category);
		CREATE INDEX IF NOT EXISTS idx_patterns_confidence ON patterns(confidence);
		CREATE INDEX IF NOT EXISTS idx_patterns_last_observed ON patterns(last_observed);

		-- FTS5 for pattern search
		CREATE VIRTUAL TABLE IF NOT EXISTS patterns_fts USING fts5(
			id UNINDEXED,
			pattern,
			content='patterns',
			content_rowid='rowid'
		);

		-- Tool usage table
		CREATE TABLE IF NOT EXISTS tool_usage (
			tool_name TEXT PRIMARY KEY,
			total_calls INTEGER DEFAULT 0,
			successful_calls INTEGER DEFAULT 0,
			failed_calls INTEGER DEFAULT 0,
			user_rejected_calls INTEGER DEFAULT 0,
			user_accepted_calls INTEGER DEFAULT 0,
			avg_duration_ms INTEGER DEFAULT 0,
			common_contexts TEXT,  -- JSON array
			user_preference REAL DEFAULT 0,
			usage_trend TEXT DEFAULT 'stable',
			last_used INTEGER
		);

		-- Suggestions table
		CREATE TABLE IF NOT EXISTS suggestions (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			confidence REAL NOT NULL,
			impact TEXT NOT NULL,
			based_on TEXT,  -- JSON array
			evidence TEXT,
			created_at INTEGER NOT NULL,
			dismissed INTEGER DEFAULT 0,
			applied INTEGER DEFAULT 0
		);

		-- Behavioral scores history
		CREATE TABLE IF NOT EXISTS score_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			overall REAL NOT NULL,
			response_quality REAL,
			tool_efficiency REAL,
			context_relevance REAL,
			correction_rate REAL,
			adaptation_speed REAL,
			sample_size INTEGER,
			calculated_at INTEGER NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// SavePattern saves a pattern to the store
func (s *Store) SavePattern(pattern EnhancedPattern) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	triggersJSON, err := json.Marshal(pattern.Triggers)
	if err != nil {
		triggersJSON = []byte("[]")
	}
	examplesJSON, err := json.Marshal(pattern.Examples)
	if err != nil {
		examplesJSON = []byte("[]")
	}
	relatedJSON, err := json.Marshal(pattern.RelatedPatterns)
	if err != nil {
		relatedJSON = []byte("[]")
	}
	supersedesJSON, err := json.Marshal(pattern.Supersedes)
	if err != nil {
		supersedesJSON = []byte("[]")
	}
	metadataJSON, err := json.Marshal(pattern.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO patterns (
			id, category, source, pattern, confidence, support, contradiction,
			first_observed, last_observed, observation_count, triggers, examples,
			related_patterns, supersedes, decay_rate, last_decay_at, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, pattern.ID, pattern.Category, pattern.Source, pattern.Pattern, pattern.Confidence,
		pattern.Support, pattern.Contradiction, pattern.FirstObserved, pattern.LastObserved,
		pattern.ObservationCount, string(triggersJSON), string(examplesJSON),
		string(relatedJSON), string(supersedesJSON), pattern.DecayRate, pattern.LastDecayAt,
		string(metadataJSON))

	if err != nil {
		return fmt.Errorf("failed to save pattern: %w", err)
	}

	// Update FTS index
	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO patterns_fts (id, pattern) VALUES (?, ?)
	`, pattern.ID, pattern.Pattern)

	return err
}

// GetPattern retrieves a pattern by ID
func (s *Store) GetPattern(id string) (*EnhancedPattern, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRow(`
		SELECT id, category, source, pattern, confidence, support, contradiction,
			first_observed, last_observed, observation_count, triggers, examples,
			related_patterns, supersedes, decay_rate, last_decay_at, metadata
		FROM patterns WHERE id = ?
	`, id)

	return s.scanPattern(row)
}

// GetAllPatterns retrieves all patterns
func (s *Store) GetAllPatterns() ([]EnhancedPattern, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, category, source, pattern, confidence, support, contradiction,
			first_observed, last_observed, observation_count, triggers, examples,
			related_patterns, supersedes, decay_rate, last_decay_at, metadata
		FROM patterns ORDER BY confidence DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query patterns: %w", err)
	}
	defer rows.Close()

	var patterns []EnhancedPattern
	for rows.Next() {
		pattern, err := s.scanPatternFromRows(rows)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, *pattern)
	}

	return patterns, nil
}

// GetPatternsByCategory retrieves patterns by category
func (s *Store) GetPatternsByCategory(category PatternCategory) ([]EnhancedPattern, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, category, source, pattern, confidence, support, contradiction,
			first_observed, last_observed, observation_count, triggers, examples,
			related_patterns, supersedes, decay_rate, last_decay_at, metadata
		FROM patterns WHERE category = ? ORDER BY confidence DESC
	`, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query patterns: %w", err)
	}
	defer rows.Close()

	var patterns []EnhancedPattern
	for rows.Next() {
		pattern, err := s.scanPatternFromRows(rows)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, *pattern)
	}

	return patterns, nil
}

// SearchPatterns searches patterns using FTS5
func (s *Store) SearchPatterns(query string, limit int) ([]EnhancedPattern, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Escape FTS5 special characters to prevent syntax errors
	escapedQuery := escapeFTS5Query(query)

	rows, err := s.db.Query(`
		SELECT p.id, p.category, p.source, p.pattern, p.confidence, p.support, p.contradiction,
			p.first_observed, p.last_observed, p.observation_count, p.triggers, p.examples,
			p.related_patterns, p.supersedes, p.decay_rate, p.last_decay_at, p.metadata
		FROM patterns p
		JOIN patterns_fts fts ON p.id = fts.id
		WHERE patterns_fts MATCH ?
		ORDER BY p.confidence DESC
		LIMIT ?
	`, escapedQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search patterns: %w", err)
	}
	defer rows.Close()

	var patterns []EnhancedPattern
	for rows.Next() {
		pattern, err := s.scanPatternFromRows(rows)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, *pattern)
	}

	return patterns, nil
}

// DeletePattern deletes a pattern by ID
func (s *Store) DeletePattern(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM patterns WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete pattern: %w", err)
	}

	_, err = s.db.Exec(`DELETE FROM patterns_fts WHERE id = ?`, id)
	return err
}

// DeletePatterns deletes multiple patterns by ID
func (s *Store) DeletePatterns(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, id := range ids {
		if _, err := tx.Exec(`DELETE FROM patterns WHERE id = ?`, id); err != nil {
			return fmt.Errorf("failed to delete pattern %s: %w", id, err)
		}
		if _, err := tx.Exec(`DELETE FROM patterns_fts WHERE id = ?`, id); err != nil {
			return fmt.Errorf("failed to delete pattern FTS %s: %w", id, err)
		}
	}

	return tx.Commit()
}

// SaveToolUsage saves tool usage statistics
func (s *Store) SaveToolUsage(usage ToolUsagePattern) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	contextsJSON, err := json.Marshal(usage.CommonContexts)
	if err != nil {
		contextsJSON = []byte("[]")
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO tool_usage (
			tool_name, total_calls, successful_calls, failed_calls,
			user_rejected_calls, user_accepted_calls, avg_duration_ms,
			common_contexts, user_preference, usage_trend, last_used
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, usage.ToolName, usage.TotalCalls, usage.SuccessfulCalls, usage.FailedCalls,
		usage.UserRejectedCalls, usage.UserAcceptedCalls, usage.AvgDurationMs,
		string(contextsJSON), usage.UserPreference, string(usage.UsageTrend), usage.LastUsed)

	return err
}

// GetAllToolUsage retrieves all tool usage statistics
func (s *Store) GetAllToolUsage() ([]ToolUsagePattern, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT tool_name, total_calls, successful_calls, failed_calls,
			user_rejected_calls, user_accepted_calls, avg_duration_ms,
			common_contexts, user_preference, usage_trend, last_used
		FROM tool_usage ORDER BY total_calls DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query tool usage: %w", err)
	}
	defer rows.Close()

	var usage []ToolUsagePattern
	for rows.Next() {
		var u ToolUsagePattern
		var contextsJSON string
		var trendStr string
		var lastUsed sql.NullInt64

		err := rows.Scan(&u.ToolName, &u.TotalCalls, &u.SuccessfulCalls, &u.FailedCalls,
			&u.UserRejectedCalls, &u.UserAcceptedCalls, &u.AvgDurationMs,
			&contextsJSON, &u.UserPreference, &trendStr, &lastUsed)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool usage: %w", err)
		}

		json.Unmarshal([]byte(contextsJSON), &u.CommonContexts)
		if u.CommonContexts == nil {
			u.CommonContexts = []string{}
		}
		u.UsageTrend = TrendDirection(trendStr)
		if lastUsed.Valid {
			u.LastUsed = lastUsed.Int64
		}

		usage = append(usage, u)
	}

	return usage, nil
}

// SaveSuggestion saves a suggestion
func (s *Store) SaveSuggestion(suggestion Suggestion) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	basedOnJSON, err := json.Marshal(suggestion.BasedOn)
	if err != nil {
		basedOnJSON = []byte("[]")
	}
	dismissed := 0
	if suggestion.Dismissed {
		dismissed = 1
	}
	applied := 0
	if suggestion.Applied {
		applied = 1
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO suggestions (
			id, type, title, description, confidence, impact, based_on,
			evidence, created_at, dismissed, applied
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, suggestion.ID, suggestion.Type, suggestion.Title, suggestion.Description,
		suggestion.Confidence, suggestion.Impact, string(basedOnJSON),
		suggestion.Evidence, suggestion.CreatedAt, dismissed, applied)

	return err
}

// GetActiveSuggestions retrieves non-dismissed, non-applied suggestions
func (s *Store) GetActiveSuggestions() ([]Suggestion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, type, title, description, confidence, impact, based_on,
			evidence, created_at, dismissed, applied
		FROM suggestions WHERE dismissed = 0 AND applied = 0
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query suggestions: %w", err)
	}
	defer rows.Close()

	var suggestions []Suggestion
	for rows.Next() {
		var s Suggestion
		var basedOnJSON string
		var dismissed, applied int

		err := rows.Scan(&s.ID, &s.Type, &s.Title, &s.Description, &s.Confidence,
			&s.Impact, &basedOnJSON, &s.Evidence, &s.CreatedAt, &dismissed, &applied)
		if err != nil {
			return nil, fmt.Errorf("failed to scan suggestion: %w", err)
		}

		json.Unmarshal([]byte(basedOnJSON), &s.BasedOn)
		if s.BasedOn == nil {
			s.BasedOn = []string{}
		}
		s.Dismissed = dismissed == 1
		s.Applied = applied == 1
		suggestions = append(suggestions, s)
	}

	return suggestions, nil
}

// SaveScoreHistory saves a behavioral score to history
func (s *Store) SaveScoreHistory(score BehavioralScore) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO score_history (
			overall, response_quality, tool_efficiency, context_relevance,
			correction_rate, adaptation_speed, sample_size, calculated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, score.Overall, score.Dimensions.ResponseQuality, score.Dimensions.ToolEfficiency,
		score.Dimensions.ContextRelevance, score.Dimensions.CorrectionRate,
		score.Dimensions.AdaptationSpeed, score.BasedOn.SampleSize, NowMs())

	return err
}

// GetRecentScores retrieves recent behavioral scores
func (s *Store) GetRecentScores(limit int) ([]BehavioralScore, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT overall, response_quality, tool_efficiency, context_relevance,
			correction_rate, adaptation_speed, sample_size, calculated_at
		FROM score_history ORDER BY calculated_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query score history: %w", err)
	}
	defer rows.Close()

	var scores []BehavioralScore
	for rows.Next() {
		var s BehavioralScore
		err := rows.Scan(&s.Overall, &s.Dimensions.ResponseQuality, &s.Dimensions.ToolEfficiency,
			&s.Dimensions.ContextRelevance, &s.Dimensions.CorrectionRate,
			&s.Dimensions.AdaptationSpeed, &s.BasedOn.SampleSize, &s.BasedOn.LastUpdated)
		if err != nil {
			return nil, fmt.Errorf("failed to scan score: %w", err)
		}
		scores = append(scores, s)
	}

	return scores, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Helper functions

func (s *Store) scanPattern(row *sql.Row) (*EnhancedPattern, error) {
	var p EnhancedPattern
	var triggersJSON, examplesJSON, relatedJSON, supersedesJSON, metadataJSON string

	err := row.Scan(&p.ID, &p.Category, &p.Source, &p.Pattern, &p.Confidence,
		&p.Support, &p.Contradiction, &p.FirstObserved, &p.LastObserved,
		&p.ObservationCount, &triggersJSON, &examplesJSON, &relatedJSON,
		&supersedesJSON, &p.DecayRate, &p.LastDecayAt, &metadataJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan pattern: %w", err)
	}

	json.Unmarshal([]byte(triggersJSON), &p.Triggers)
	json.Unmarshal([]byte(examplesJSON), &p.Examples)
	json.Unmarshal([]byte(relatedJSON), &p.RelatedPatterns)
	json.Unmarshal([]byte(supersedesJSON), &p.Supersedes)
	json.Unmarshal([]byte(metadataJSON), &p.Metadata)

	// Ensure slices and maps are not nil
	if p.Triggers == nil {
		p.Triggers = []string{}
	}
	if p.Examples == nil {
		p.Examples = []PatternExample{}
	}
	if p.RelatedPatterns == nil {
		p.RelatedPatterns = []string{}
	}
	if p.Supersedes == nil {
		p.Supersedes = []string{}
	}
	if p.Metadata == nil {
		p.Metadata = map[string]any{}
	}

	return &p, nil
}

func (s *Store) scanPatternFromRows(rows *sql.Rows) (*EnhancedPattern, error) {
	var p EnhancedPattern
	var triggersJSON, examplesJSON, relatedJSON, supersedesJSON, metadataJSON string

	err := rows.Scan(&p.ID, &p.Category, &p.Source, &p.Pattern, &p.Confidence,
		&p.Support, &p.Contradiction, &p.FirstObserved, &p.LastObserved,
		&p.ObservationCount, &triggersJSON, &examplesJSON, &relatedJSON,
		&supersedesJSON, &p.DecayRate, &p.LastDecayAt, &metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to scan pattern: %w", err)
	}

	json.Unmarshal([]byte(triggersJSON), &p.Triggers)
	json.Unmarshal([]byte(examplesJSON), &p.Examples)
	json.Unmarshal([]byte(relatedJSON), &p.RelatedPatterns)
	json.Unmarshal([]byte(supersedesJSON), &p.Supersedes)
	json.Unmarshal([]byte(metadataJSON), &p.Metadata)

	// Ensure slices and maps are not nil
	if p.Triggers == nil {
		p.Triggers = []string{}
	}
	if p.Examples == nil {
		p.Examples = []PatternExample{}
	}
	if p.RelatedPatterns == nil {
		p.RelatedPatterns = []string{}
	}
	if p.Supersedes == nil {
		p.Supersedes = []string{}
	}
	if p.Metadata == nil {
		p.Metadata = map[string]any{}
	}

	return &p, nil
}

// escapeFTS5Query escapes special characters in FTS5 queries
func escapeFTS5Query(query string) string {
	// FTS5 special characters that need to be removed or escaped
	// These characters have special meaning in FTS5 syntax
	replacer := strings.NewReplacer(
		`"`, "",
		`'`, "",
		`*`, "",
		`^`, "",
		`(`, "",
		`)`, "",
		`{`, "",
		`}`, "",
		`[`, "",
		`]`, "",
		`:`, "",
		`;`, "",
		`-`, " ",
	)
	return replacer.Replace(query)
}
