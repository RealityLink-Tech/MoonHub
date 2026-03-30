// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System - SQLite Schema
// License: MIT

package adaptive_memory

// SchemaSQL contains the main table definitions
const SchemaSQL = `
-- Episodic Memory: timestamped events with outcomes
-- Optimized for Chinese text with dual content storage
CREATE TABLE IF NOT EXISTS episodic_memory (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	event_type TEXT NOT NULL,
	content TEXT NOT NULL,
	content_tokens TEXT NOT NULL,  -- Pre-segmented for Chinese FTS5
	outcome TEXT,
	importance REAL NOT NULL DEFAULT 0.5,
	access_count INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	last_accessed_at INTEGER NOT NULL
);

-- Indexes for efficient retrieval
CREATE INDEX IF NOT EXISTS idx_episodic_user ON episodic_memory(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_episodic_importance ON episodic_memory(user_id, importance DESC);
CREATE INDEX IF NOT EXISTS idx_episodic_access ON episodic_memory(user_id, last_accessed_at DESC);

-- Migration tracking
CREATE TABLE IF NOT EXISTS schema_version (
	version INTEGER PRIMARY KEY,
	applied_at INTEGER NOT NULL
);

-- Insert initial schema version if not exists
INSERT OR IGNORE INTO schema_version (version, applied_at) VALUES (1, strftime('%s', 'now') * 1000);
`

// FTSSchemaSQL contains the FTS5 virtual table definition
// Note: Using unicode61 tokenizer which works well with pre-segmented Chinese
const FTSSchemaSQL = `
-- FTS5 virtual table for semantic search
-- content_tokens stores pre-segmented Chinese text
-- Using 'unicode61' tokenizer - simpler than 'porter unicode61', better for Chinese
CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(
	id UNINDEXED,
	content,
	content_tokens,  -- Segmented Chinese version
	tags,
	tokenize='unicode61'
);
`

// TriggerSQL contains triggers to keep FTS5 index in sync
const TriggerSQL = `
-- Trigger: Insert into FTS5 when episodic_memory is inserted
CREATE TRIGGER IF NOT EXISTS trg_episodic_insert
AFTER INSERT ON episodic_memory
BEGIN
	INSERT INTO memory_fts(id, content, content_tokens, tags)
	VALUES (
		NEW.id,
		NEW.content,
		NEW.content_tokens,
		NEW.event_type || ' ' || NEW.user_id
	);
END;

-- Trigger: Delete from FTS5 when episodic_memory is deleted
CREATE TRIGGER IF NOT EXISTS trg_episodic_delete
AFTER DELETE ON episodic_memory
BEGIN
	DELETE FROM memory_fts WHERE id = OLD.id;
END;

-- Trigger: Update FTS5 when episodic_memory is updated
CREATE TRIGGER IF NOT EXISTS trg_episodic_update
AFTER UPDATE ON episodic_memory
BEGIN
	UPDATE memory_fts SET
		content = NEW.content,
		content_tokens = NEW.content_tokens,
		tags = NEW.event_type || ' ' || NEW.user_id
	WHERE id = NEW.id;
END;
`

// AllSQL combines all schema definitions
var AllSQL = []string{SchemaSQL, FTSSchemaSQL, TriggerSQL}

// MigrationSQL contains schema migrations for future versions
var MigrationSQL = map[int]string{
	// Example for future migrations:
	// 2: "ALTER TABLE episodic_memory ADD COLUMN embedding BLOB;",
}

// CurrentSchemaVersion is the current schema version
const CurrentSchemaVersion = 1
